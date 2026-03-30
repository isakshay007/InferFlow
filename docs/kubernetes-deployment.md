# Kubernetes Deployment

The Kubernetes manifests deploy the Triton-backed stack on EKS.

## Manifests

- [router.yaml](../k8s/router.yaml)
- [triton-adapter.yaml](../k8s/triton-adapter.yaml)
- [triton.yaml](../k8s/triton.yaml)

## Prerequisites

- EKS cluster provisioned
- GPU node group available
- NVIDIA device plugin installed
- container images available to the cluster

## Deployment Order

```bash
kubectl apply -f k8s/triton.yaml
kubectl apply -f k8s/triton-adapter.yaml
kubectl apply -f k8s/router.yaml
```

## Smoke Test

Check pods:

```bash
kubectl get pods -l app=triton
kubectl get pods -l app=triton-adapter
kubectl get pods -l app=inferflow-router
```

Port-forward:

```bash
kubectl port-forward svc/inferflow-router 8080:80
```

Test request:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"qwen3-0.6b","messages":[{"role":"user","content":"Explain InferFlow in one sentence."}]}'
```
