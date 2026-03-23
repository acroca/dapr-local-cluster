# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A local Kubernetes development environment for testing microservices with [Dapr](https://dapr.io/) (Distributed Application Runtime). Uses Kind for a 3-node cluster, Tilt for live-reload development, and supports Go, Python, and Java apps.

**Requires the Dapr repo cloned as a sibling directory at `../dapr`.**

## Common Commands

```bash
# Cluster lifecycle
mise run cluster-up      # Create Kind cluster + local Docker registry (localhost:5001)
mise run cluster-down    # Tear down cluster

# Development
mise run tilt-up         # Start Tilt dev environment (UI at http://localhost:10350)
mise run tilt-up-e2e     # Run e2e test environment (separate from dev)

# Scaffold a new Go app
mise run gen_go APP_ID=my-app APP_PORT=6007
```

Tool versions are pinned in `mise.toml` (Tilt 0.36, Go 1.26, Kind 0.29, Helm 3.19, Python 3.14).

## Architecture

### Main Tiltfile

The root `Tiltfile` assembles the cluster by:
1. Loading tool functions from `tools/*/Tiltfile` (dapr, redis, kafka, postgres, etc.)
2. Calling those functions to provision infrastructure
3. Using `load_dynamic('apps/*/Tiltfile')` to enable apps (comment/uncomment to toggle)

Infrastructure tools expose composable functions:
- `dapr(version)` — installs Dapr (`"dev"` builds from `../dapr`, else uses Helm release)
- `redis()`, `redis_pubsub_component()`, `redis_state_component()`, `redis_workflowstate_component()`
- `postgres()`, `kafka()`, `pulsar()` — same pattern, alternative backends
- `dapr_config_component(otel_endpoint=..., zipkin_endpoint=...)` — tracing config

### App Structure

Every app follows this layout:
```
apps/<name>/
├── Tiltfile                    # Build + deploy config
├── Dockerfile                  # Container build
├── manifests/deployment.yaml   # K8s Deployment with Dapr annotations
├── main.go or src/main.py      # App code
└── go.mod or requirements.txt  # Dependencies
```

Dapr-enabled apps require these pod annotations:
```yaml
dapr.io/enabled: "true"
dapr.io/app-id: "<name>"
dapr.io/config: "daprconfig"
```

### Tiltfile Patterns

**Go apps** use `docker_build()` for automatic rebuilds on file change.
**Python apps** use `local_resource()` with manual `docker buildx build` + `tilt trigger`.

Apps declare dependencies: `k8s_resource(resource_deps=['dapr', 'redis'])` and are organized with labels (`core`, `apps`, `components`, `bug-repro`).

Interactive testing via `cmd_button()` in Tiltfiles (shows as buttons in Tilt UI).

### Components (tools/component/Tiltfile)

The `component()` helper creates Dapr Component CRDs. All tools use it to register state stores, pub/sub brokers, etc.

## Key Apps

| App | Language | Purpose |
|-----|----------|---------|
| `workflows-go` | Go | Dapr durable workflows |
| `workflows-py` | Python | Async workflows (durabletask-python) |
| `workflows-crossapp` | Java+Go+Python | Cross-language workflow testing |
| `pub` / `sub` | Go / Python | Pub/sub pattern |
| `actors-go` | Go | Stateful actor pattern |
| `nginx-no-dapr` | — | Reproduces dapr/dapr#9379 (no Dapr annotations, custom SA) |

## E2E Testing

The `e2e/` directory has a separate Tiltfile that builds Dapr from source with test flags, provisions Redis + PostgreSQL in a `dapr-tests` namespace, and runs integration tests isolated from the dev cluster.
