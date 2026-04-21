# Triton Setup

Triton remains in the repo as a deferred backend path. It is no longer the primary deployment target for the current MVP, which now centers on EKS + vLLM.

## Model

Historical first model target:

- `Qwen/Qwen3-0.6B`

Model repository files:

- [config.pbtxt](../triton/qwen3_0_6b/config.pbtxt)
- [model.py](../triton/qwen3_0_6b/1/model.py)

## Components

- router: public API entrypoint
- Triton adapter: internal `/healthz` and `/infer` service
- Triton: GPU model runtime

## Container Images

- [Dockerfile.triton](../Dockerfile.triton)
- [Dockerfile.triton-adapter](../Dockerfile.triton-adapter)

Build manually if needed:

```bash
docker build -f Dockerfile.triton -t inferflow/triton-qwen3:dev .
docker build -f Dockerfile.triton-adapter -t inferflow/triton-adapter:dev .
docker build -f Dockerfile.router -t inferflow/router:dev .
```

## Runtime Assumptions

- Triton runs as a separate container or Kubernetes deployment.
- The Triton Python backend downloads the Hugging Face model at startup using `MODEL_ID`.
- The adapter talks to Triton over HTTP, not via a Go SDK.

## Important Note

Keep these assets for future Triton work, but treat the current production-like path as:

- Go router
- vLLM adapter
- EKS
- `Qwen/Qwen2.5-0.5B-Instruct`
