# Overview

InferFlow is a scalable LLM inference routing project. The current repo state provides:

- a Go router exposing `POST /v1/chat/completions`
- health and readiness endpoints
- a Prometheus-style metrics endpoint
- runtime strategy switching with four active strategies
- a local mock backend for development
- a vLLM adapter for EKS-backed inference
- Kubernetes manifests for the router, Redis, and vLLM workers
- retained Triton code as a deferred backend path
- Terraform for AWS EKS infrastructure
- GitHub Actions for CI and Terraform planning

## Architecture

1. Clients send OpenAI-compatible chat completion requests to the router.
2. The router chooses a healthy backend using one of four strategies:
   - `round_robin`
   - `least_pending`
   - `random`
   - `kv_aware`
3. The router forwards to either:
   - the local mock backend
   - the vLLM adapter
4. The vLLM adapter translates InferFlow backend requests into vLLM completion calls.
5. Successful requests update cache affinity metadata so `kv_aware` can prefer warm backends.

## Current Scope

Implemented:

- local mock-backed workflow
- vLLM-backed EKS deployment assets
- Redis-backed KV-aware routing metadata with safe local fallback
- metrics scraping support for Prometheus
- AWS Terraform environment
- GitHub Actions pipeline for CI and Terraform plan

Planned later:

- streaming SSE responses
- dynamic backend discovery
- richer autoscaling and observability rollout
- Triton reintroduction as a separate future path
