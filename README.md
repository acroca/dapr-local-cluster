# Local Cluster Development Environment

A local Kubernetes development environment using [Kind](https://kind.sigs.k8s.io/) and [Tilt](https://tilt.dev/) for rapid development and testing of microservices with Dapr.

## Prerequisites

This project uses [mise](https://mise.jdx.dev/) for tool version management. Install mise first, then run:

```bash
mise install
```

This will install the required tools:
- Task (task runner)
- Tilt (development workflow)
- Go (programming language)

You'll also need dapr installed locally at `../dapr`, as tilt will build it automatically from that directory.

**NOTE**: Changes in dapr will not be reflected in the local cluster, you need to trigger the dapr rebuild manually from the Tilt UI. Also remember to rebuild all the apps after a dapr rebuild, as Tilt will not do it automatically so their side will be out of sync.

## Quick Start

1. **Start the cluster and registry:**
   ```bash
   task cluster-up
   ```

2. **Run Tilt for development:**
   ```bash
   task tilt-up
   ```

3. **Access Tilt UI:**
   Open http://localhost:10350 in your browser

## Architecture

The local development environment includes:

- **Kind Cluster**: 3-node Kubernetes cluster (1 control-plane, 2 workers)
- **Local Registry**: Docker registry on `localhost:5001` for fast image pushes
- **Dapr Runtime**: Service mesh for microservice patterns
- **Redis**: Backing store for pub/sub and state management
- **Ingress**: Accessible on ports 8081 (HTTP) and 8443 (HTTPS)

## Available Commands

### Cluster Management

```bash
# Start cluster and registry
task cluster-up

# Stop cluster and registry
task cluster-down

# Rebuild everything (down then up)
task cluster-rebuild

# Run Tilt development environment
task tilt-up
```

## Development Workflow

1. **Initial Setup**: Run `task cluster-up` to create the Kind cluster and start the local registry
2. **Development**: Use `task tilt-up` to start the Tilt development environment
3. **Monitoring**: Monitor services through the Tilt UI at http://localhost:10350
4. **Iteration**: Tilt automatically rebuilds and redeploys when app code changes. Dapr doesn't rebuild automatically, so you need to trigger it manually from the Tilt UI, and rebuild all the apps after a dapr rebuild.
5. **Cleanup**: Use `task cluster-down` when done developing
