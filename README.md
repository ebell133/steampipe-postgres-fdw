# Steampipe Postgres FDW (Direct gRPC Fork)

Fork of [turbot/steampipe-postgres-fdw](https://github.com/turbot/steampipe-postgres-fdw) with a **direct gRPC hub mode** that moves plugin execution out of the PostgreSQL process into a standalone gRPC server.

## What's Different From Upstream

The upstream FDW loads Steampipe plugins **in-process** inside PostgreSQL (via the Steampipe CLI or embedded plugin binaries). This fork adds an alternative hub implementation (`HubDirect`) that delegates all plugin operations over **gRPC to an external server**, making the FDW a thin client.

### Why

- **Memory** -- PostgreSQL stays at 80Mi instead of carrying the full weight of every plugin (240Mi+ per provider)
- **Scaling** -- The gRPC server (cloudquery) scales horizontally as a separate Kubernetes Deployment; the FDW inside PostgreSQL does not need to
- **Isolation** -- A plugin panic or OOM in the gRPC server doesn't crash PostgreSQL
- **Workload separation** -- Live dashboard queries (via FDW) and batch cache refreshes (via a separate HTTP service) run in independent processes with independent resource limits

### Changed Files

| File | Change |
|------|--------|
| `hub/hub_direct.go` | New `HubDirect` implementation -- gRPC client that calls `GetSchema` and `Execute` on a remote `WrapperPlugin` server |
| `hub/scan_iterator_direct.go` | New `scanIteratorDirect` -- calls `Execute` over gRPC, returns a `WrapperPlugin_ExecuteClient` stream that implements `row_stream.Receiver` |
| `hub/hub_create.go` | Branching logic -- if `STEAMPIPE_GRPC_ADDRESS` env var is set, creates `HubDirect`; otherwise falls back to the upstream `RemoteHub` |

All upstream functionality is preserved. When `STEAMPIPE_GRPC_ADDRESS` is not set, behavior is identical to upstream.

### How It Works

```
PostgreSQL                          gRPC Server (cloudquery)
+-----------------------+           +---------------------------+
| steampipe_postgres_fdw|           | WrapperPlugin service     |
|   HubDirect           |--gRPC-->  |   PluginRouter            |
|   scanIteratorDirect  |  :50051   |     aws PluginServer      |
+-----------------------+           |     gcp PluginServer      |
                                    |     k8s PluginServer      |
                                    +---------------------------+
```

1. PostgreSQL loads the FDW extension
2. `CreateHub()` checks for `STEAMPIPE_GRPC_ADDRESS` and creates a `HubDirect` with a gRPC connection
3. `IMPORT FOREIGN SCHEMA` calls `GetSchema()` over gRPC
4. `SELECT * FROM foreign_table` calls `Execute()` over gRPC, streaming rows back

### Configuration

Single env var on the PostgreSQL pod:

```
STEAMPIPE_GRPC_ADDRESS=cloudquery.steampipe.svc.cluster.local:50051
```

## Building

Prerequisites:
- PostgreSQL 15 server dev headers
- Go 1.26+
- gcc (Linux)

```bash
make clean && make
```

Produces `build-Linux/steampipe_postgres_fdw.so` (or `build-Darwin/` on macOS).

## Releases

Tagged releases trigger the **Build Draft Release** workflow (`.github/workflows/buildimage.yml`), which builds the `.so` for Linux amd64/arm64 and attaches it as a release asset. The CNPG PostgreSQL image Dockerfile downloads the `.so` from the release URL at build time.

## Upstream

Based on [turbot/steampipe-postgres-fdw](https://github.com/turbot/steampipe-postgres-fdw) (`develop` branch). To pull upstream changes:

```bash
git fetch upstream
git merge upstream/develop
```

## License

This repository is published under the [Apache 2.0](https://www.apache.org/licenses/LICENSE-2.0) license, same as upstream.

