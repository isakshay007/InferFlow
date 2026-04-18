# InferFlow

InferFlow is a scalable LLM inference router project centered on a Go control-plane router and experimentable backend routing strategies. The repo now supports a mock-backed local loop and a primary cloud path based on Amazon Elastic Kubernetes Service (EKS) with vLLM, while keeping Triton code in the repo as a deferred backend path.

## Documentation

Detailed documentation is organized under [docs/README.md](C:/Users/ajinf/Documents/CS%206650/InferFlow/docs/README.md).

Quick links:

- [Overview](C:/Users/ajinf/Documents/CS%206650/InferFlow/docs/overview.md)
- [Local Development](C:/Users/ajinf/Documents/CS%206650/InferFlow/docs/local-development.md)
- [EKS vLLM Deployment](C:/Users/ajinf/Documents/CS%206650/InferFlow/docs/eks-vllm.md)
- [Triton Setup](C:/Users/ajinf/Documents/CS%206650/InferFlow/docs/triton-setup.md)
- [Kubernetes Deployment](C:/Users/ajinf/Documents/CS%206650/InferFlow/docs/kubernetes-deployment.md)
- [Terraform Infrastructure](C:/Users/ajinf/Documents/CS%206650/InferFlow/docs/terraform-infrastructure.md)
- [GitHub Actions](C:/Users/ajinf/Documents/CS%206650/InferFlow/docs/github-actions.md)
- [Destroy Workflow](C:/Users/ajinf/Documents/CS%206650/InferFlow/docs/destroy-workflow.md)

## Current MVP Status

Implemented now:

- Go router with `POST /v1/chat/completions`
- runtime routing strategies: `round_robin`, `least_pending`, `random`, `kv_aware`
- strategy switching through `GET/PUT /strategy`
- metrics endpoint at `GET /metrics`
- mock-backed local development flow
- vLLM adapter plus EKS deployment assets
- retained Triton code as a deferred backend path

Planned next:

- Streaming SSE responses
- Kubernetes endpoint discovery
- richer Prometheus/Grafana dashboards
- KEDA autoscaling rollout
- AWS Terraform automation

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

Run all active strategies locally:

```bash
python loadgen/generator.py --requests 5 --strategies round_robin,least_pending,random,kv_aware --output results/strategies.csv
```

### Option 2: Docker Compose

```bash
docker compose up --build
```

The router listens on `http://localhost:8080` and the mock backend is internal to Compose.

## EKS Terraform Quick Start

```bash
cd terraform/environments/aws
terraform init -backend=false
terraform plan
terraform apply
```

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

### `GET /metrics`

Returns Prometheus-style router metrics.

### `GET /strategy` and `PUT /strategy`

Supported runtime strategies:

- `round_robin`
- `least_pending`
- `random`
- `kv_aware`

## Infrastructure Note

The active infrastructure path is AWS EKS plus vLLM.

## Scripts

- `scripts/local-run.ps1`: starts mock backend and router locally
- `scripts/setup-cluster.sh`: infrastructure and deployment helper notes
- `scripts/teardown-cluster.sh`: destroy helper

Detailed infrastructure, deploy, and destroy docs live under [docs/README.md](C:/Users/ajinf/Documents/CS%206650/InferFlow/docs/README.md).
