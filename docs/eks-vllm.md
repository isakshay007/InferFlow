# EKS vLLM Deployment

The primary cloud deployment path for InferFlow is Amazon Elastic Kubernetes Service (EKS) with vLLM workers serving `Qwen/Qwen2.5-0.5B-Instruct`.

## Components

- Go router deployment with `GET /strategy`, `GET /metrics`, and `POST /v1/chat/completions`
- Redis for KV-aware routing metadata
- vLLM worker StatefulSet with a colocated adapter sidecar
- KEDA and Prometheus/Grafana scaffolding for scaling and observability
- AWS load balancer in front of the router service (provisioned automatically by EKS)
- Terraform-managed EKS cluster, node groups, VPC, and ECR

## Strategy Set

Active runtime strategies:

- `round_robin`
- `least_pending`
- `random`
- `kv_aware`

Deferred:

- `cost_aware`
- `session_affinity`
- Triton as the primary runtime

## Rollout Notes

- Start with one vLLM worker replica until the router-to-vLLM path is stable.
- Add a second worker by scaling the StatefulSet and updating `INFERFLOW_BACKENDS` to include the second pod DNS name.
- `kv_aware` stores prefix affinity in Redis, but automatically falls back when Redis is empty or unavailable.
- The router and Redis should stay on the `inferflow/node-pool=system` pool.
- The vLLM worker pods should land on the `inferflow/node-pool=worker` pool.

## Terraform

```bash
cd terraform/environments/aws
terraform init -backend=false
terraform plan
terraform apply
```

After apply, configure kubectl:

```bash
aws eks update-kubeconfig --region us-east-1 --name inferflow-eks
```

## Verify

```bash
kubectl get pods -l app=inferflow-router
kubectl get pods -l app=vllm-worker
kubectl port-forward svc/inferflow-router 8080:80
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"Qwen/Qwen2.5-0.5B-Instruct","messages":[{"role":"user","content":"Summarize InferFlow in one sentence."}]}'
```
