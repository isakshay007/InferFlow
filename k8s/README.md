# AWS Triton Deployment

This folder now supports two separate roles:

- router deployment through [router.yaml](/C:/Users/ajinf/Documents/CS%206650/InferFlow/k8s/router.yaml)
- Triton-backed inference through [triton.yaml](/C:/Users/ajinf/Documents/CS%206650/InferFlow/k8s/triton.yaml) and [triton-adapter.yaml](/C:/Users/ajinf/Documents/CS%206650/InferFlow/k8s/triton-adapter.yaml)

## Prerequisites

- EKS cluster provisioned by Terraform
- GPU node group available
- NVIDIA device plugin installed on the cluster
- Container images published for:
  - `inferflow/router:dev`
  - `inferflow/triton-adapter:dev`
  - `inferflow/triton-qwen3:dev`

## Suggested Deployment Order

```bash
kubectl apply -f k8s/triton.yaml
kubectl apply -f k8s/triton-adapter.yaml
kubectl apply -f k8s/router.yaml
```

## Smoke Test

1. Confirm the Triton pod schedules on a GPU node.
2. Wait for Triton readiness:
   `kubectl get pods -l app=triton`
3. Wait for adapter and router readiness:
   `kubectl get pods -l app=triton-adapter`
   `kubectl get pods -l app=inferflow-router`
4. Port-forward the router service:

```bash
kubectl port-forward svc/inferflow-router 8080:80
```

5. Send one chat completion request:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"qwen3-0.6b","messages":[{"role":"user","content":"Explain InferFlow in one sentence."}]}'
```

## Notes

- The Triton model repository source files are mounted from a ConfigMap, while model weights are downloaded by the Python backend at startup.
- The current setup is optimized for a first real AWS GPU inference milestone, not for fast local Triton development.
