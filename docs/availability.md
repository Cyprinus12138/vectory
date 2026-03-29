# Availability Design

This document summarizes the mechanisms Vectory uses to maintain service availability in cluster mode.

---

## Node Lifecycle and Health

### Keepalive Lease

Every node registers itself in etcd with a keepalive lease (TTL configured via `cluster_mode.ttl`). The etcd client continuously renews the lease in the background. If a node crashes or loses connectivity, the lease expires and etcd automatically deletes the node's metadata key --- the cluster detects the departure without any explicit unregister call.

### Status State Machine

Each node progresses through a well-defined set of statuses:

```
Start → Init → Healthy
                  ↕
              Unhealthy
                  ↓
             Rebalancing → Inactive
```

| Status | Meaning | Routable? |
|--------|---------|-----------|
| `Start` | Process launched, not yet initialized | No |
| `Init` | Loading indices | No |
| `Healthy` | Fully operational | Yes (full weight) |
| `Unhealthy` | Degraded but serving | Yes (10% weight) |
| `Rebalancing` | Shard redistribution in progress | Yes (10% weight) |
| `Inactive` | Shutting down | No |

Status transitions are broadcast to etcd via `StatusUpdating` channel. Other nodes consume these updates through the cluster watch and adjust routing weights accordingly.

### Grace Period

On startup, a configurable `grace_period` delays the node from accepting traffic. This allows enough time for the hash ring to stabilize and for shards to finish loading before the node is marked healthy.

---

## Shard Distribution

### Consistent Hash Ring

Shard-to-node assignment is determined by a consistent hash ring (`github.com/serialx/hashring`). Each shard key (`{indexName}:{shardId}:{replicaId}`) is hashed to determine its owning node.

The hash ring is rebuilt dynamically:
- **Node join**: `SyncCluster` watch fires `IsCreate()` → node added to ring → rebalance triggered
- **Node leave**: watch fires `DELETE` event → node removed from ring → rebalance triggered
- **Node inactive**: watch fires `IsModify()` with `Inactive` status → node removed from ring → rebalance triggered

### Rebalancing

When topology changes, `Rebalance()` on every surviving node:

1. Iterates all configured shards across all indices
2. Checks `NeedLoad(shardKey)` against the current hash ring
3. Loads newly assigned shards (respecting a pending-shard mechanism to avoid duplicate loads)
4. Deletes shards no longer owned (calls `Delete()`, which also frees any staged data)

The node transitions to `Rebalancing` status during this process, reducing its routing weight to 10%.

---

## Replica-Based Failover

When an index is configured with `replicas > 1`, each shard has multiple replica keys (e.g., `idx:0:0`, `idx:0:1`). These replicas are independently hashed onto different nodes in the ring.

### gRPC Resolver and Weighted Round-Robin

For remote shard queries, Vectory uses a custom gRPC resolver (`internal/cluster/resolver.go`) combined with a weighted round-robin balancer (`internal/cluster/balancer.go`):

1. `SearchProcessor` calls `IndexManager.Search()` for each unique shard
2. If the shard is not local, `Route()` returns the owning node's address
3. The gRPC resolver discovers all replica nodes for the target shard
4. The balancer distributes requests based on node weight:
   - `Healthy` nodes get full weight (scaled by inverse CPU load)
   - `Rebalancing`/`Unhealthy` nodes get 10% weight
   - `Init`/`Start`/`Inactive` nodes get zero weight

If a node holding one replica dies, its keepalive lease expires, it is removed from the hash ring, and subsequent requests are routed to surviving replicas automatically.

---

## Search Path Resilience

### Local vs Remote Dispatch

`IndexManager.Search()` iterates the unique shards of an index:

- **Local shard available**: search is performed directly via `FaissIndex.Search()` under `RLock`
- **Remote shard**: the request is forwarded via gRPC `Cluster.SearchShard()` to the owning node

If a shard is neither loaded locally nor routable (no healthy replica exists), the search returns an error for that shard.

### Thread Safety

`FaissIndex` wraps all search operations with `sync.RWMutex`:
- `Search()`, `VectorCount()`, `MetricType()`, `InputDim()`, `Revision()` hold `RLock`
- `Reload()` and `CommitStaged()` acquire `WLock` only for the pointer swap

Searches are never blocked during download or staging --- only the brief pointer swap acquires the write lock.

---

## Revision Consistency

In cluster mode, index reloads use a two-phase protocol to prevent nodes from serving mixed revisions during a rolling update. See [cluster-revision-consistency.md](cluster-revision-consistency.md) for the full design.

Summary:

1. **Stage**: each node downloads and loads the new revision into memory but continues serving the old one
2. **Signal**: once all local shards of an index are staged, the node writes a readiness key to etcd (ephemeral, attached to keepalive lease)
3. **Commit**: when all owning nodes have signaled readiness for the same revision, a commit key is written
4. **Swap**: all nodes watch the commit prefix and atomically swap to the new revision

If a node dies during staging, its ephemeral ready key is automatically deleted, the hash ring recomputes owners, and the commit proceeds with the surviving nodes.

---

## Graceful Shutdown

`core.Vectory.Stop()` executes an ordered shutdown:

1. Sets node status to `Inactive` (removed from hash ring, stops receiving new requests)
2. Stops the cron scheduler (no new reload cycles)
3. Gracefully stops the gRPC server (drains in-flight RPCs)
4. Shuts down the HTTP server with a context deadline
5. Unregisters from etcd (immediate cleanup, though the keepalive lease would handle it regardless)
6. Closes the etcd client connection

This sequence ensures in-flight requests complete before the node is fully removed from the cluster.

---

## Relevant Source Files

| File | Role |
|------|------|
| `internal/cluster/node.go` | Keepalive lease, hash ring, node status, routing |
| `internal/cluster/resolver.go` | gRPC service discovery for replicas |
| `internal/cluster/balancer.go` | Weighted round-robin load balancing |
| `internal/engine/manager.go` | Shard loading, rebalancing, revision coordination |
| `internal/engine/faiss.go` | Thread-safe index access, staged reload |
| `internal/engine/index.go` | `StagedReloader` interface, shard key generation |
| `internal/core/vectory.go` | Startup orchestration, graceful shutdown |
| `internal/processor/search.go` | Local/remote shard dispatch |
| `pkg/pkg.go` | Node status state machine |
