# Overview

InferFlow is a scalable LLM inference routing project. The current repo state provides:

- a Go router exposing `POST /v1/chat/completions`
- health and readiness endpoints
- routing strategies: `round_robin`, `least_pending`, and `cost_aware`
- a local mock backend for development
- a Triton adapter for AWS GPU-backed inference
- Terraform for AWS infrastructure
- Kubernetes manifests for the router, Triton, and Triton adapter
- GitHub Actions for CI, Terraform, deploy, and destroy workflows

## Architecture

1. Clients send OpenAI-compatible chat completion requests to the router.
2. The router chooses a healthy backend using the configured routing strategy.
3. The router forwards to either:
   - the local mock backend
   - the Triton adapter
4. The Triton adapter translates InferFlow backend requests into Triton HTTP inference calls.
5. Triton runs the model and returns generated text.

## Current Scope

Implemented:

- local mock-backed workflow
- routing strategies (`round_robin`, `least_pending`, `cost_aware`)
- AWS Triton deployment assets
- Terraform remote state support
- GitHub Actions pipeline split for CI, Terraform, deploy, and destroy
- Local Docker baseline is the current active experiment path. AWS/EKS/Triton is deferred (see docs/triton-eks-deferred.md).

Planned later:

- streaming SSE responses
- dynamic backend discovery
- observability stack
- autoscaling and experiment automation
