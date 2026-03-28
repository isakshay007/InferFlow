# Overview

InferFlow is a scalable LLM inference routing project. The current repo state provides:

- a Go router exposing `POST /v1/chat/completions`
- health and readiness endpoints
- round-robin backend selection
- a local mock backend for development
- a Triton adapter for AWS GPU-backed inference
- Terraform for AWS infrastructure
- Kubernetes manifests for the router, Triton, and Triton adapter
- GitHub Actions for CI, Terraform, deploy, and destroy workflows

## Architecture

1. Clients send OpenAI-compatible chat completion requests to the router.
2. The router chooses a healthy backend using round-robin.
3. The router forwards to either:
   - the local mock backend
   - the Triton adapter
4. The Triton adapter translates InferFlow backend requests into Triton HTTP inference calls.
5. Triton runs the model and returns generated text.

## Current Scope

Implemented:

- local mock-backed workflow
- AWS Triton deployment assets
- Terraform remote state support
- GitHub Actions pipeline split for CI, Terraform, deploy, and destroy

Planned later:

- streaming SSE responses
- dynamic backend discovery
- least-pending and cost-aware routing
- observability stack
- autoscaling and experiment automation
