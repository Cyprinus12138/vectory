# Vectory Usage Guide

## Overview

Vectory is a distributed vector similarity search service. It loads FAISS index files, exposes a search API over gRPC and HTTP, and can operate either as a single standalone node or as a cluster of nodes coordinated through etcd. Search requests are routed across shards — locally if the shard lives on the current node, or via a gRPC call to the owning node if not.

---

## Prerequisites

### Runtime dependency — FAISS native library

Vectory links against `libfaiss_c.so`. This library is not distributed as a Go module; it must be present on the host at link and runtime. The `Dockerfile` handles this by compiling FAISS from source in a builder stage and copying the shared library into the final image. Outside Docker, you must build and install FAISS manually before `make build` will succeed.

### Private Go modules

The project depends on a forked version of `spf13/viper` hosted at `github.com/Cyprinus12138/*`. Before running `go mod download` or `go build`, set:

```bash
export GOPRIVATE="github.com/Cyprinus12138/*"
```

---

## Building

```bash
make build          # compiles binary to bin/server/main
make gen            # regenerates proto/gen/go/ from proto/vectory/*.proto
make build_image    # builds Docker image
```

`make gen` requires `protoc` (recommended: libprotoc 3.6.1), `protoc-gen-go` v1.28, and `protoc-gen-go-grpc` v1.2.

---

## Configuration

The server reads `./etc/config.yml` on startup. The file has three required top-level sections.

### `cluster` — service identity and cluster behaviour

```yaml
cluster:
  cluster_name: "example"       # logical name of this cluster; also used as the etcd key namespace
  grpc_enabled: true            # expose gRPC on RPC_PORT in addition to HTTP

  cluster_mode:
    enabled: true               # false = single-node mode, no etcd required
    endpoints:                  # etcd endpoints (used when enabled: true)
      - "http://10.0.0.1:2379"
      - "http://10.0.0.2:2379"
      - "http://10.0.0.3:2379"
    ttl: 5                      # etcd lease TTL in seconds; node is removed if it stops renewing
    grace_period: 15            # seconds to wait for cluster sync before accepting traffic
    lb_mode: "none"             # load balancing mode: "none" | "cpu"

  admin:
    enable_metrics: true        # expose Prometheus metrics on GET /metrics
    enable_pprof: true          # expose pprof profiling endpoints
    health_check_endpoint: "/health_check"
```

### `env` — runtime environment (for local development only)

```yaml
env:
  PORT: 6060       # HTTP listener port
  RPC_PORT: 8999   # gRPC listener port
  POD_IP: 127.0.0.1
```

In production these three values must be injected as real environment variables, not via the config file. The config-file `env` section exists only as a fallback for local runs. The `POD_IP` value is what the node advertises to etcd as its address; in Kubernetes this must be `status.podIP`.

Two additional environment variables are available but not shown in the default config:

| Variable | Default | Purpose |
|---|---|---|
| `IDX_PATH` | `./etc/index` | Directory containing index manifests (single mode only) |
| `ETCD_ROOT` | `/vectory/<cluster_name>` | Root key prefix under which all etcd state is stored |

### `logger`

```yaml
logger:
  level: "debug"
  default_path: "./log/data.log"
  level_path:
    debug:  "./log/data.log"
    info:   "./log/info.log"
    warn:   "./log/warn.log"
    error:  "./log/error.log"
    fatal:  "./log/fatal.log"
```

### Remote config (optional)

`config_manager.enable_remote: true` causes Viper to pull config from an etcd or Consul path instead of the local file. This allows live config updates without restarting the process.

```yaml
config_manager:
  enable_remote: true
  remote:
    - provider: "etcd3"
      endpoint: "http://10.0.0.1:2379"
      path: "/vectory/config/example"
```

---

## Index Manifests

An index manifest is a YAML or JSON file that describes one logical index — its type, sharding layout, data source, and reload policy.

### Manifest structure

```yaml
meta:
  name: "product_embeddings"
  type: "faiss"           # currently the only working type
  input_dim: 128          # must match the dimension of the compiled FAISS index
  shards: 4               # number of unique shards
  replicas: 2             # replica copies per shard
  c_time: 1711612800      # Unix timestamp (creation)
  m_time: 1711612800      # Unix timestamp (last modification)
  version: "v1"

source:
  type: "local_path"      # only "local_path" is implemented; "s3"/"hdfs"/"ftp"/"sftp" are stubs
  location: "/data/indices/product_embeddings"
  name_fmt: "{index_name}_{shard_id}.faiss"
    # {index_name} and {shard_id} are replaced at load time

reload:
  enable: true
  mode: "active"          # "active" = scheduled auto-reload; "passive" = manual Reload() call only
  schedule:
    type: "interval"      # "cron" | "interval"  ("fixed" is not yet implemented)
    interval: "1h"        # used when type = "interval"; Go duration string
    crontab: ""           # used when type = "cron"; standard cron expression
    random_dwell_time: true   # jitters the reload trigger to avoid burst I/O across all nodes
```

### Local filesystem layout

The `local_path` downloader expects a specific layout under `source.location`:

```
/data/indices/product_embeddings/
├── _revision               ← plain text file containing the current revision integer, e.g. "3"
├── 1/                      ← revision 1
│   ├── product_embeddings_0.faiss
│   └── product_embeddings_1.faiss
├── 2/
│   └── ...
└── 3/                      ← revision 3 (current)
    ├── product_embeddings_0.faiss
    └── product_embeddings_1.faiss
```

On first load, the downloader reads `_revision` to get the current revision integer, then constructs the file path as `<location>/<revision>/<name_fmt resolved>`. On reload, if the revision integer is unchanged the reload is skipped; if higher, the new file is loaded and the old index freed.

### Placing manifests

**Single mode** — drop manifest files (`.yml` or `.json`) into the directory pointed to by `IDX_PATH` (default `./etc/index`). One file per logical index. The server reads this directory at startup.

**Cluster mode** — write manifests into etcd at:

```
/vectory/<cluster_name>/index/<index_name>
```

The server watches this prefix. Creating a key triggers loading across all nodes; deleting a key unloads the index.

---

## Running

### Single-node

```bash
./bin/server/main
```

With `cluster_mode.enabled: false`, the server:

1. Reads all manifest files from `IDX_PATH`
2. Loads all shards of all indices into memory
3. Starts the HTTP server on `PORT` and (if `grpc_enabled`) the gRPC server on `RPC_PORT`
4. Begins serving search requests immediately

### Cluster mode

Each node requires three environment variables:

```
PORT=6060
RPC_PORT=8999
POD_IP=<this node's IP, reachable by other nodes>
```

With `cluster_mode.enabled: true`, startup proceeds as follows:

1. The node connects to etcd and grants a keepalive lease with the configured `ttl`.
2. It registers its metadata (node ID, `POD_IP:RPC_PORT`, status `Init`) under `svc/<cluster_name>/<node_id>` in etcd.
3. It reads all index manifests from etcd and loads only the shards that the consistent hash ring assigns to this node.
4. It waits `grace_period` seconds for other nodes to register and for the hash ring to stabilise.
5. It begins reporting CPU and memory load to etcd periodically.
6. Status transitions to `Healthy` (all assigned shards loaded) or `Unhealthy` (some shards failed).

When a node joins or leaves, all remaining nodes detect the etcd change and each runs `Rebalance()`: shards that now belong to this node are loaded; shards that no longer belong are freed.

### Load balancing modes

| `lb_mode` | Behaviour |
|---|---|
| `none` | Hash ring assigns shards by key hash only, no load awareness |
| `cpu` | Node weights are derived from CPU utilisation: `weight = (1 - cpu_percent) * 1000`. Unhealthy/rebalancing nodes are weighted at 10%; initialising nodes get weight 0 and receive no traffic |

---

## Making Search Requests

### gRPC

Service: `Core` — method: `Search`

```protobuf
message SearchRequest {
  RequestHeader header = 1;
  string index_name = 2;    // must match meta.name in the manifest
  repeated Vector input = 3; // batch: one entry per query vector
  int32 limit = 4;           // top-K results per input vector
}

message Vector {
  string name = 1;           // arbitrary label for this query
  repeated float vector = 2; // must be exactly input_dim floats
}
```

The response contains one `SearchResult` per input vector. Each result holds a list of `Item`:

```protobuf
message Item {
  string id = 1;    // FAISS internal integer ID, returned as a string
  float score = 2;  // distance score (metric depends on how the index was built)
}
```

When the index has multiple shards, results from each shard are gathered and merged before being returned.

### HTTP

`POST /search` — request body is the JSON encoding of `SearchRequest`:

```json
{
  "index_name": "product_embeddings",
  "limit": 10,
  "input": [
    {
      "name": "query-1",
      "vector": [0.1, 0.2, 0.3, "...128 floats total"]
    }
  ]
}
```

The response body is the JSON encoding of `SearchResponse`.

---

## Observability

All surfaces are served on the same HTTP port as the application.

| Endpoint | Config gate | Content |
|---|---|---|
| `GET /metrics` | `admin.enable_metrics: true` | Prometheus metrics (request counts, latencies, Go runtime) |
| `GET /debug/pprof/...` | `admin.enable_pprof: true` | Standard Go pprof profiling |
| `GET <health_check_endpoint>` | always | Current node status |

OpenTelemetry tracing wraps all inbound HTTP requests (via `otelgin` middleware) and all gRPC calls. Configure the exporter via standard OTEL environment variables such as `OTEL_EXPORTER_OTLP_ENDPOINT`.

### Node status lifecycle

```
Start → Init → Healthy
                  ↕
              Unhealthy
                  ↕
             Rebalancing
                  ↓
              Inactive
```

`Unhealthy` means one or more shards are in the pending queue (failed to load or not yet assigned). `Rebalancing` means a topology change is in progress. Both states still serve traffic but at a heavily reduced hash ring weight (10% of normal).

---

## Kubernetes Deployment

The `deploy/` directory contains baseline `Deployment` manifests. The critical environment bindings:

```yaml
env:
  - name: PORT
    value: "6060"
  - name: RPC_PORT
    value: "8999"
  - name: POD_IP
    valueFrom:
      fieldRef:
        fieldPath: status.podIP
```

`POD_IP` must come from `status.podIP`, not a static value. Every node advertises this address to etcd so that peers can open gRPC connections to it for cross-shard routing. A wrong or unreachable IP is the most common cause of shard routing failures in cluster mode.

The `containerPort` in the manifest must match the `PORT` env value. `RPC_PORT` does not need a `containerPort` entry if gRPC is only used for pod-to-pod traffic and not exposed through a Kubernetes Service.

---

## Shard Distribution

The consistent hash ring is built from all registered node IDs. For a given shard key (`<index_name>:<shard_id>:<replica_id>`), the ring deterministically selects the responsible node. This means:

- The same shard always lands on the same node as long as cluster membership does not change.
- Adding or removing one node migrates roughly `1/N` of all shards (where N is the cluster size).
- Each shard has `replicas` copies with different replica IDs, which hash to potentially different nodes, providing redundancy.

**Example — 4-shard index on a 3-node cluster, node A owns shards 0 and 2:**

1. Node A receives a `Search` request for the full index.
2. Shards 0 and 2 are searched directly via FAISS on node A.
3. Shards 1 and 3 are forwarded via `Cluster.SearchShard` gRPC calls to their respective owning nodes.
4. All four shard results are merged on node A and returned to the caller.

---

## Consistency Model

### Index loading

An index becomes searchable the moment its `IndexManifest` is written into `indexManifests` (`sync.Map`). This write happens at the end of `loadIndex`, after all shard load attempts — including partially failed ones. A search against a partially-loaded index in single mode returns errors for the missing shards; in cluster mode those shards are marked `ToRoute` and forwarded to their owning nodes.

Loading at startup is **synchronous and blocking** — the server does not accept traffic until all manifests have been processed.

### Concurrent read safety (`FaissIndex`)

Each `FaissIndex` is protected by an `sync.RWMutex`. All read operations (`Search`, `VectorCount`, `MetricType`, `InputDim`, `Revision`) acquire a read lock; `Delete` and the index-swap inside `Reload` acquire the write lock. Multiple concurrent searches on the same shard are therefore safe and non-blocking with respect to each other.

### Reload (active mode)

Reload follows a revision-based optimistic strategy:

1. The current `revision` is read under `RLock`.
2. The downloader checks `_revision` on disk; if unchanged, the reload is skipped immediately (`ErrIndexRevisionUpToDate`).
3. If a newer revision exists, the new index file is loaded from disk **without holding any lock** — searches continue uninterrupted.
4. The write lock is acquired only for the pointer swap (`f.index = newIndex; f.revision = newRevision`), which is near-instantaneous.

This means search latency is only affected for the brief instant of the pointer swap, not for the entire file-load duration.

The `reloading` atomic flag prevents two concurrent reload triggers (e.g. two cron firings that overlap) from double-loading the same shard.

### Shard eviction during rebalance

When `Rebalance` determines a shard must be evicted from this node, it calls `index.Delete()` (which acquires the write lock) and removes the entry from the `engineStore`. A search that has already retrieved the index pointer from the map before eviction is protected by the read lock — `Delete` will wait for it to finish before freeing the FAISS memory.

### Node status propagation

Each node publishes its own status (`Healthy`, `Unhealthy`, `Rebalancing`) to etcd. Other nodes observe this via the `GetNodeMetaPathPrefix` watch and adjust the hash ring weight accordingly (Unhealthy/Rebalancing nodes are routed to at 10% weight; Init/Inactive nodes receive no traffic). The local in-memory `nodeMetaMap` is the source of truth used for all routing decisions.

### Known limitations

- **Manifest stored on partial shard failure**: `indexManifests` is updated even when some shards fail to load. In single mode this means searches on that index will return errors for the missing shards.
- **No live manifest update**: modifying an existing index manifest in etcd is a no-op at runtime. To update an index, delete the key and re-create it.
- **Startup is blocking**: indices are loaded serially at startup. Large indices or many shards will increase the time before the server starts accepting requests.
