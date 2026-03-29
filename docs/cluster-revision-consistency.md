# Cluster-Wide Revision Consistency

## Problem

In cluster mode, each node independently reloads index data on a cron schedule. When a new revision appears, nodes download and swap at slightly different times. During this window, some nodes serve revision N while others serve revision N+1, producing inconsistent search results across shards of the same index.

## Solution: Two-Phase Reload

Vectory uses a **stage-then-commit** protocol coordinated through etcd. A node downloads and loads the new revision into memory but continues serving the old one. Only when every node owning a shard of the index has staged the same revision does the cluster-wide switch happen atomically.

Single-mode deployments are unaffected --- reload swaps immediately as before.

---

## State Machine

Each `FaissIndex` shard transitions through the following states during a reload cycle in cluster mode:

```
                 cron fires
                     |
                     v
              +-----------+
              |  Serving   |  (revision N, search traffic served here)
              |  rev N     |
              +-----+-----+
                    |
                    | StageReload() downloads rev N+1,
                    | loads FAISS index into memory
                    v
              +-----------+
              |  Staged    |  (serving rev N, rev N+1 ready in memory)
              |  rev N+1   |
              +-----+-----+
                    |
                    | CommitStaged() triggered by
                    | cluster-wide commit signal
                    v
              +-----------+
              |  Serving   |  (revision N+1, old index freed)
              |  rev N+1   |
              +-----------+

    At any point, DiscardStaged() can abort:

              +-----------+
              |  Staged    | --DiscardStaged()--> Serving rev N (staged data freed)
              |  rev N+1   |
              +-----------+
```

Key properties:

- **Search is never blocked.** Read traffic holds `RLock` on the serving index throughout. The staged index is a separate field.
- **Staging is idempotent.** If a newer revision N+2 appears before N+1 is committed, `StageReload` replaces the staged data and re-signals readiness.
- **Commit is atomic per-shard.** `CommitStaged` acquires `WLock`, swaps the pointer, releases the lock, then frees the old index.

---

## Coordination Protocol

### etcd Key Layout

All keys are scoped under the cluster root (typically `/vectory/{clusterName}`).

| Key | Purpose | Lifecycle |
|-----|---------|-----------|
| `{root}/revision/{indexName}/ready/{nodeId}` | Node declares it has staged a revision | Ephemeral (attached to keepalive lease) |
| `{root}/revision/{indexName}/commit` | Signals all nodes to commit | Persistent until next cycle |

The ready key is attached to the node's keepalive lease, so it is automatically deleted if the node crashes.

### Flow

```
  Node A (owns shard 0)             Node B (owns shard 1)            etcd
  ────────────────────               ────────────────────             ────

  1. cron fires                      1. cron fires
     StageReload() for shard 0          StageReload() for shard 1
     downloads + loads rev 42           downloads + loads rev 42
     still serving rev 41               still serving rev 41

  2. signalReady("idx", 42)          2. signalReady("idx", 42)
     all local shards staged?            all local shards staged?
     yes -> PUT ready/A = 42             yes -> PUT ready/B = 42
                                                                     ready/A = 42
                                                                     ready/B = 42

  3. checkAndCommit()                3. checkAndCommit()
     reads ready/* keys                 reads ready/* keys
     owners = {A, B}                    owners = {A, B}
     all match rev 42!                  all match rev 42!
     PUT commit = 42                    (or B writes it; both safe)
                                                                     commit = 42

  4. watchCommits sees event         4. watchCommits sees event
     commitIndex("idx", 42)             commitIndex("idx", 42)
     shard 0: CommitStaged(42)          shard 1: CommitStaged(42)
     now serving rev 42                 now serving rev 42
     DELETE ready/A                     DELETE ready/B
```

Both nodes serve revision 41 throughout steps 1--3. The switch to revision 42 happens only at step 4, after every node has confirmed readiness.

### Determining Owning Nodes

`nodesOwningIndex(indexName)` iterates all shard x replica keys of the index through the consistent hash ring and collects the distinct set of node IDs. Only these nodes need to report readiness for the commit to proceed.

---

## Edge Cases

### Node crashes during staging

The ready key is attached to the keepalive lease. When the node dies, etcd automatically deletes the key. The remaining nodes detect the topology change through the existing `SyncCluster` watch, rebalance reassigns the shards, and `nodesOwningIndex` recomputes the owner set --- the commit can then proceed with the surviving nodes.

### Node joins during staging

A new node receives its shards via `Rebalance`, starts its own cron-based reload, stages the new revision, and signals readiness. The commit cannot proceed until the new node is also ready, which is the correct behavior.

### Revision N+1 appears before N is committed

When a shard's cron fires again, `StageReload` discards the previously staged data, stages N+1, and re-signals readiness with the new revision. `checkAndCommit` requires all nodes to agree on the same revision, so mixed-revision ready states never trigger a premature commit.

### Shard removed during staging

`Rebalance` calls `index.Delete()` on removed shards, which internally calls `DiscardStaged()` to free any staged data before releasing the serving index.

### Multiple nodes write the commit key

Both nodes may independently detect that all ready keys are present and write the commit key. This is safe --- the value is identical and etcd `Put` is idempotent. The watch fires once per node regardless.

---

## Relevant Source Files

| File | Role |
|------|------|
| `internal/engine/faiss.go` | `StageReload`, `CommitStaged`, `DiscardStaged`, `reloadCallback` |
| `internal/engine/index.go` | `StagedReloader` interface definition |
| `internal/engine/manager.go` | `signalReady`, `checkAndCommit`, `commitIndex`, `watchCommits` |
| `internal/cluster/node.go` | `GetOwnerNode` (hash ring lookup), `KeepAliveLeaseID` |
| `internal/config/path.go` | `GetRevisionReadyPath`, `GetRevisionCommitPath` |
