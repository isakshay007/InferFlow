# EKS vLLM Deployment

This folder now supports two separate roles:

- router deployment through [router.yaml](/C:/Users/ajinf/Documents/CS%206650/InferFlow/k8s/router.yaml)
- vLLM-backed inference through [vllm-worker.yaml](/C:/Users/ajinf/Documents/CS%206650/InferFlow/k8s/vllm-worker.yaml) and [redis.yaml](/C:/Users/ajinf/Documents/CS%206650/InferFlow/k8s/redis.yaml)
- retained Triton manifests for future use

## Prerequisites

- EKS cluster provisioned
- Worker node group available
- Container images published for:
  - `inferflow/router:dev`
  - `inferflow/vllm-adapter:dev`
- manifests assume Terraform-created node labels:
  - `inferflow/node-pool=system`
  - `inferflow/node-pool=worker`

## Suggested Deployment Order

```bash
kubectl apply -f k8s/redis.yaml
kubectl apply -f k8s/vllm-worker.yaml
kubectl apply -f k8s/router.yaml
kubectl apply -f k8s/keda-vllm.yaml
```

## Smoke Test

1. Confirm the vLLM worker pod schedules on a worker node.
2. Wait for Redis, worker, and router readiness:
   `kubectl get pods -l app=redis`
   `kubectl get pods -l app=vllm-worker`
   `kubectl get pods -l app=inferflow-router`
3. Port-forward the router service:

```bash
kubectl port-forward svc/inferflow-router 8080:80
```

4. Send one chat completion request:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"Qwen/Qwen2.5-0.5B-Instruct","messages":[{"role":"user","content":"Explain InferFlow in one sentence."}]}'
```

## Notes

- The router points at explicit worker pod DNS names so each strategy can make its own backend choice.
- Scale the worker StatefulSet and update `INFERFLOW_BACKENDS` when you add more worker replicas.
- Triton manifests remain in this folder as deferred assets.
