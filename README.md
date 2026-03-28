# InferFlow

InferFlow is a Week 1 MVP for a scalable LLM inference router project. This repository bootstraps the local development path, the Go router, a mock inference backend, starter Kubernetes assets, and Terraform scaffolding for AWS EKS so the team can move from proposal to implementation quickly.

## Documentation

Detailed documentation is organized under [docs/README.md](C:/Users/ajinf/Documents/CS%206650/InferFlow/docs/README.md).

Quick links:

- [Overview](C:/Users/ajinf/Documents/CS%206650/InferFlow/docs/overview.md)
- [Local Development](C:/Users/ajinf/Documents/CS%206650/InferFlow/docs/local-development.md)
- [Triton Setup](C:/Users/ajinf/Documents/CS%206650/InferFlow/docs/triton-setup.md)
- [Kubernetes Deployment](C:/Users/ajinf/Documents/CS%206650/InferFlow/docs/kubernetes-deployment.md)
- [Terraform Infrastructure](C:/Users/ajinf/Documents/CS%206650/InferFlow/docs/terraform-infrastructure.md)
- [GitHub Actions](C:/Users/ajinf/Documents/CS%206650/InferFlow/docs/github-actions.md)
- [Destroy Workflow](C:/Users/ajinf/Documents/CS%206650/InferFlow/docs/destroy-workflow.md)

## Current MVP Status

Implemented now:

- Go router with `POST /v1/chat/completions`
- mock-backed local development flow
- Triton adapter plus AWS GPU deployment assets
- Terraform infrastructure with shared remote state support
- GitHub Actions for CI, plan/apply, deploy, and destroy

Planned next:

- Streaming SSE responses
- Kubernetes endpoint discovery
- Least-pending and cost-aware routing
- Observability stack wiring
- Autoscaling and experiment automation

## Local Quick Start

### Option 1: Native processes

Start the mock backend:

```bash
go run ./cmd/mock-backend
```

In another terminal, start the router:

```bash
$env:INFERFLOW_BACKENDS="http://localhost:9000"
go run ./cmd/router
```

Send a sample request:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d "{\"model\":\"mock-llm\",\"messages\":[{\"role\":\"user\",\"content\":\"Hello from InferFlow\"}]}"
```

Run tests:

```bash
go test ./...
```

Generate a sample CSV:

```bash
python loadgen/generator.py --requests 5 --output results/sample.csv
```

### Option 2: Docker Compose

```bash
docker compose up --build
```

The router listens on `http://localhost:8080` and the mock backend is internal to Compose.

## Router API

### `POST /v1/chat/completions`

Accepts a minimal OpenAI-compatible request body:

```json
{
  "model": "mock-llm",
  "messages": [
    { "role": "user", "content": "Hello" }
  ],
  "stream": false
}
```

Returns a minimal OpenAI-compatible response shape containing:

- `id`
- `object`
- `created`
- `model`
- `choices`
- `usage`

### `GET /healthz`

Returns process liveness.

### `GET /readyz`

Returns success only when at least one backend is currently healthy.

## Scripts

- `scripts/local-run.ps1`: starts mock backend and router locally
- `scripts/setup-cluster.sh`: infrastructure and deployment helper notes
- `scripts/teardown-cluster.sh`: destroy helper

Detailed infrastructure, deploy, and destroy docs live under [docs/README.md](C:/Users/ajinf/Documents/CS%206650/InferFlow/docs/README.md).
