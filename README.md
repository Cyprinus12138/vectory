# Vectory

A distributed vector search engine built on [FAISS](https://github.com/facebookresearch/faiss), with automatic sharding, replication, and etcd-based cluster coordination.

## Features

- **FAISS-backed indexing** with thread-safe concurrent search
- **Sharding & replication** across cluster nodes via consistent hashing
- **Hot reload** of indices with cron, interval, or passive scheduling
- **etcd cluster coordination** with keepalive leases and automatic rebalancing
- **Dual API** — gRPC (inter-node + client) and HTTP (Gin)
- **Observability** — OpenTelemetry tracing, Prometheus metrics

## Prerequisites

- Go 1.20+
- FAISS C library (`libfaiss_c.so`) — see [FAISS Setup](#faiss-setup)
- protoc + plugins (for regenerating proto files only)
- etcd (for cluster mode only)

Set up the Go environment for private modules:

```bash
export GOPRIVATE="github.com/Cyprinus12138/*"
```

## FAISS Setup

FAISS is a C++ library. Vectory uses the C API binding (`libfaiss_c.so`) via [go-faiss](https://github.com/DataIntelligenceCrew/go-faiss).

### Option 1: Install script (Linux)

```bash
sudo ./scripts/setup_faiss.sh
```

This clones, builds, and installs FAISS to `/usr/local`. Requires `cmake`, `g++`, `git`, and `make`.

To build in a specific directory (useful for caching):

```bash
sudo ./scripts/setup_faiss.sh /opt/faiss-build
```

### Option 2: Docker (no local install needed)

Build and run entirely in Docker — the multi-stage `Dockerfile` handles FAISS compilation:

```bash
make build_image
```

### Verifying the install

```bash
ldconfig -p | grep libfaiss_c
```

If the library is installed but not found, add its path to the linker:

```bash
echo "/usr/local/lib" | sudo tee /etc/ld.so.conf.d/faiss.conf
sudo ldconfig
```

## Build

```bash
make build        # Binary output: bin/server/main
make build_image  # Docker image
make gen          # Regenerate protobuf (requires protoc)
```

## Test

```bash
go test ./...                                          # All tests
go test ./internal/engine/... -run TestFaissIndex      # FAISS-specific tests
go test ./internal/engine/... -run TestManagerSearch    # Single test
go test ./... -v                                       # Verbose
```

Note: FAISS unit tests use a mock interface and do **not** require `libfaiss_c.so`. Building the binary or running integration tests does.

## Configuration

Vectory loads configuration from `./etc/config.yml`. Key sections:

| Section | Purpose |
|---------|---------|
| `cluster.cluster_name` | Cluster identifier |
| `cluster.cluster_mode` | Enable/disable cluster, etcd endpoints, TTL, grace period, LB mode |
| `env` | Environment variables (port, pod IP, index path, etcd root) |
| `logger` | Log level, output paths |

Index manifests are JSON/YAML files describing each index (type, dimensions, shards, replicas, source, reload schedule). In single mode they're read from the local filesystem; in cluster mode from etcd.

## Project Structure

```
cmd/server/          Entry point
internal/
  core/              Orchestrator (HTTP + gRPC servers, lifecycle)
  engine/            Index management, FAISS wrapper, downloaders, reload scheduling
  cluster/           etcd-based node registration, hash ring, routing
  processor/         Search business logic (multi-shard aggregation)
  grpc_handler/      gRPC service implementations
  grpc_client/       Client-side gRPC with custom shard resolver
  config/            Configuration structs and path helpers
  utils/             Logger, metrics, priority queue, error codes
proto/               Protobuf definitions (core.proto, cluster.proto)
pkg/                 Global node status
mocks/               Mock etcd server for testing
scripts/             Build, setup, and codegen scripts
```

## License

See [LICENSE](LICENSE) for details.
