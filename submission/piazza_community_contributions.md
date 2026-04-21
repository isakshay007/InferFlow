# Final Project: InferFlow — LLM Inference Router with Pluggable Routing Strategies

**Team:** Ajin Frank Justin, Akshay Keerthi Adhikasavan Suresh

InferFlow is a Go-based LLM inference router that sits in front of multiple `llama.cpp` backends and exposes an OpenAI-compatible `/v1/chat/completions` API. It implements four pluggable routing strategies — `round_robin`, `random`, `least_pending`, and `kv_aware` (Redis-backed prompt-affinity) — switchable at runtime via `PUT /strategy`. It runs on AWS EKS with 3x c5.xlarge nodes each serving Qwen2.5-0.5B-Instruct (CPU-only), fronted by an AWS ALB. Our experiments quantify the classic tradeoff between routing overhead and load-balance quality, and isolate the real-world KV-cache reuse benefit of prompt-affinity routing.

**Video:** <sharepoint link>
**Code:** https://github.com/isakshay007/InferFlow
**Experiments Report:** <attached PDF>
**Project Management:** <attached Excel export>

---

## Three related projects and what we learned from them

### 1. Scalable LLM-backed Q&A System with RAG — Yalin Sun
Piazza: https://piazza.com/class/mk3hftotl6e229/post/1443
Repo: https://github.com/ylsneu0028/Scalable-LLM-backed-Q-A-System-with-RAG

**Similarities:** Both projects are LLM-serving systems deployed on cloud infrastructure, and both instrument per-stage timings (their `timings_ms = {embed, retrieve, llm, total}` mirrors our `X-Inferflow-*` response headers and CSV captures). Both use a small open model (`llama3.2:1b` via Ollama for them; `Qwen2.5-0.5B-Instruct` via llama.cpp for us) and both run experiments varying concurrency.

**Differences:** Their system is a full RAG pipeline with four decomposed services (api/embed/vector/llm) plus Qdrant as a vector DB — the interesting scaling axes are index size and horizontal service replicas. InferFlow keeps a single control-plane (the Go router) and varies the *routing policy* over a fixed backend fleet. Their work targets end-to-end Q&A latency; ours targets the tail-latency effect of how requests are *steered*.

**Takeaway:** Their micro-service decomposition is a cleaner way to pinpoint bottlenecks than what we did — we had to infer per-stage costs from logs. If we extended InferFlow with retrieval/embedding, splitting those into separate pods would make profiling far easier.

### 2. Owl — Real-Time Distributed Video Inference — Kasaraneni, Sainani, Patil
Piazza: https://piazza.com/class/mk3hftotl6e229/post/1444
Repo: https://github.com/kasaranenikarthik/owl/

**Similarities:** Both are distributed inference systems that explore saturation behavior and worker scaling under load. Both use Redis for caching in front of the model layer (they cache similar frames; we cache prompt-to-backend affinity). Both report where the system *stops scaling* — Owl saturates at ~4.5–5 FPS; InferFlow's `kv_aware` strategy saturates one backend at 46% of traffic before tail latency blows up. Both projects also found that naive batching/queueing hurt rather than helped under their workloads.

**Differences:** Owl is vision-first (YOLOv8 on Triton with GPUs) with Kafka as a decoupling buffer between ingestion and inference. InferFlow is text-only, CPU-only, and synchronous (no queue — the router is a direct reverse-proxy with strategy selection). Their concerns are frame-drop under backpressure; ours are tail latency and backend hotspotting.

**Takeaway:** Owl's use of Kafka to decouple offered-load from served-load is something we should have considered for the `kv_aware` hotspot problem — a bounded queue per backend would let a "hot" backend fall behind without starving the others.

### 3. ChoreMate — Household Chore Management Platform — Shrijan Shetty
Piazza: https://piazza.com/class/mk3hftotl6e229/post/1434
Repo: https://github.com/IamShrijan/choremate

**Similarities:** Both are production-deployed AWS applications provisioned from scratch with Terraform. Both use Redis (ElastiCache for them, an in-cluster pod for us) and both sit behind an Application Load Balancer. Both integrate an LLM (Gemini API for their chatbot; llama.cpp for our actual inference tier). Both teams ran formal load tests on the deployed AWS environment, not just local benchmarks.

**Differences:** ChoreMate is an application-layer project (ECS Fargate + RDS + SQS) where the LLM is a *consumer* of a hosted model (Gemini). InferFlow is an infrastructure-layer project that *hosts* the LLM and measures how the router decides which replica serves each request. Their scaling challenges are database connection pools and job queues; ours are prompt affinity and in-flight request counts.

**Takeaway:** ChoreMate's approach to Terraform-from-scratch and clean separation of stateful (RDS) vs. stateless tiers was a model for how our `terraform/environments/aws/` could be organized more cleanly. We kept Redis in-cluster for simplicity; a managed ElastiCache instance would have been closer to production-grade.

---

## Cross-cutting observations

All four projects, including ours, hit the same wall in different forms: **caching helps on the happy path but creates new failure modes under concurrency.** Yalin's vector search benefits from HNSW but index size dominates p95; Owl's similarity cache helps frame-rate but pushes contention to Redis; ChoreMate caches chore schedules in Redis but has to invalidate on swap-request state changes; InferFlow's `kv_aware` cache gives a real 22% speedup on repeated prompts but concentrates 46% of traffic on one backend. The recurring lesson across all four is that cache-aware routing/placement needs a load-based escape hatch — none of us implemented one for our final submission, but all four reports flagged it as the obvious next step.
