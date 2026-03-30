# InferFlow (CS 6650) — Milestone 1 Report

Repository: `https://github.com/isakshay007/InferFlow`  
Branch used for Milestone 1: `demo`  
Primary test command: `go test ./...`

## 1) Problem, Team, and Project Overview

### Problem Statement
LLM-backed applications need predictable latency and reliability, but model backends are slow, bursty, and expensive. A fixed single-backend design fails under load and gives no way to evaluate routing behavior before cloud deployment.

InferFlow addresses this by providing an OpenAI-compatible router (`POST /v1/chat/completions`) that can switch routing strategy at runtime and route across multiple backends.

### Team and Ownership (Milestone 1 snapshot)
This report is formatted for individual submission while reflecting team workstreams.

| Owner | Workstream | Milestone 1 Delivery |
|---|---|---|
| Akshay | Integration + local infra | Docker profiles, end-to-end wiring, strategy switching |
| Teammate A | Load/experiments | Matrix runs, harness scripts, control runs |
| Teammate B | Analysis/reporting | Summaries, plots, report narrative |
| Teammate C | Cloud path | Terraform/EKS/Triton scaffolding and deferred plan |

### What the System Does
Client request flow:
1. Client calls router at `POST /v1/chat/completions`.
2. Router selects backend using `round_robin`, `least_pending`, or `cost_aware`.
3. Router sends backend contract to adapter at `POST /infer`.
4. Adapter converts message list to prompt and calls Ollama `POST /api/generate`.
5. Router returns OpenAI-like response (`choices`, `usage`, etc.).

## 2) Repo Overview (Codebase Structure and Components)

### Top-Level Structure
- `cmd/router/`: router entrypoint.
- `cmd/mock-backend/`: deterministic local mock backend.
- `cmd/runtime-adapter/`: adapter for local Ollama.
- `internal/server/`: router HTTP handlers, strategy endpoint, readiness/health logic.
- `internal/router/`: strategy implementations (`round_robin`, `least_pending`, `cost_aware`).
- `internal/proxy/`: backend request client (`/healthz`, `/infer`).
- `internal/runtime/ollama/`: Ollama client used by runtime adapter.
- `loadgen/`: experiment load generation (`generator.py`, optional `locustfile.py`).
- `scripts/experiments/`: experiment matrix runners and comparison scripts.
- `analysis/`: summary + chart generation scripts.
- `docs/`: operational docs and experiment instructions.
- `docker-compose.yml`: local orchestration (`mock`, `real`, `three-backends`, `mixed`).
- `terraform/`, `k8s/`, `.github/workflows/`: preserved cloud/deploy scaffolding.

### Test Coverage in Repo
- Go unit/integration tests for router, strategies, adapter, and triton client compatibility.
- Command used: `go test ./...`

## 3) Current Status (Initial Phase), Blockers, and Next Steps

### As of March 30, 2026
Completed in local environment:
- End-to-end local path: router -> adapter -> Ollama.
- Multi-backend local profiles:
  - `mock` (control)
  - `real` (2 real adapters/backends)
  - `three-backends` (3 real adapters/backends)
  - `mixed` (2 real + mock fallback)
- Runtime strategy switching via `PUT /strategy`.
- Experiment harness and analysis pipeline with per-run JSONL and markdown/PNG summaries.

Blocked / deferred:
- AWS GPU request is pending; stable GPU-backed EKS runtime is not available yet.
- Because of this, production-like Triton/EKS validation is deferred.
- Cloud assets remain in repo and are ready to resume once GPU access is approved.

Next steps:
1. Complete stable local baseline and mock-vs-real comparison.
2. Add stronger per-request observability for backend timeout/failure attribution.
3. Re-enable Triton/EKS path when GPU access becomes reliable.
4. Compare local findings against cloud runs once available.

## 4) Project Plan and Recent Progress

### Timeline (Milestone 1)
- Week 1:
  - Router + mock backend + OpenAI-compatible API surface.
- Week 2:
  - Strategy implementations and runtime strategy switching endpoint.
- Week 3:
  - Local runtime adapter for Ollama and multi-backend Docker profiles.
- Week 4 (current milestone close):
  - Experiment matrix scripts, summary/plot scripts, and documentation cleanup.

### What Was Done Recently
- Added `real`, `three-backends`, and `mixed` compose workflows.
- Added matrix scripts for real and mock control runs.
- Added summary script with p50/p95/p99, throughput, error-rate outputs.
- Added compare mode for mock-vs-real reporting.

### AI Usage (Cost/Benefit)
AI-assisted work (code + docs) was used for:
- rapid script scaffolding,
- documentation restructuring,
- consistency checks across experiment workflow.

Observed benefit:
- faster implementation of repetitive glue code and report structure.

Observed cost/risk:
- generated scripts still required manual validation to avoid misleading metrics under overload.
- final interpretation remains human-reviewed to prevent wrong conclusions.

## 5) Objectives (Short-Term, Long-Term, Observability)

### Short-Term Objectives (Milestone 2)
- Produce reproducible local baselines with strategy comparison under non-overloaded settings.
- Improve experiment hygiene:
  - separate quality runs (low concurrency) vs stress runs (high concurrency),
  - preserve clear labels in report outputs.
- Add request-level traces/log fields for backend selection and failure cause.

### Long-Term Objectives (Semester End)
- Validate routing behavior on GPU-backed infrastructure.
- Add stronger observability stack (router metrics + backend health + tracing).
- Establish repeatable decision framework for selecting routing policy under workload classes.

### Observability Plan
- Current:
  - `/healthz`, `/readyz`, strategy endpoint, response usage fields.
  - experiment summaries and error-rate plots.
- Planned:
  - explicit counters by error class (`502`, `503`, timeout, unhealthy backend),
  - latency histograms by strategy and backend,
  - traces linking request -> selected backend -> upstream latency.

## 6) Related Work

### Course Readings Applied
1. Tail latency and variability in large systems (used to justify routing + overload testing).
2. SRE-style reliability metrics (used for error-rate and readiness interpretation).
3. Distributed tracing/observability principles (used in observability roadmap).

### Related Piazza Project Directions
1. LLM gateway/router project (backend abstraction focus) — add team/thread link.
2. Inference load/performance benchmarking project — add team/thread link.
3. Cloud deployment + observability project — add team/thread link.

## 7) Methodology (Experiments, Tradeoffs, and Worst-Case Workload)

### Experiment Method
- For each strategy: run fixed-duration load tests and collect per-request JSONL.
- Aggregate to table + plots:
  - error rate,
  - p50/p95/p99 latency,
  - throughput.
- Use control mode (`mock`) versus real mode (`Ollama`) to separate system overhead from model compute constraints.

### Worst-Case Workload Considered
- Concurrent client load where CPU-bound model generation exceeds service capacity.
- Expected outcomes:
  - rising latency,
  - backend timeouts (`502`),
  - temporary no-healthy-backend responses (`503`).

### Tradeoffs
- Local CPU testing is cheap and reproducible, but not equivalent to GPU production behavior.
- High concurrency can create misleading throughput if failures dominate (fast fail responses).
- Therefore, low-concurrency runs are treated as quality baselines; higher concurrency runs are stress characterization.

## 8) Preliminary Results and Interpretation

Data currently collected:
- Full strategy matrix outputs in `results/baseline/`.
- Summary table in `results/baseline/summary.md`.
- Plots in `results/baseline/plots/`.

Key observations from current baseline summary:
- At concurrency `1`, all strategies return useful latency signals (multi-second CPU-bound model generation).
- At concurrency `2` and `4`, failure rates rise sharply (mostly `503`, plus `502` timeouts), indicating local CPU saturation and health-flap behavior under stress.
- This behavior is consistent with current hardware and timeout limits; it is not a routing path break.

What remains before final report:
- Repeat stable runs for stronger confidence intervals.
- Generate control (`mock`) vs real (`Ollama`) comparison artifacts for cleaner attribution.
- Re-run cloud-side once GPU/EKS path is unblocked.

## 9) Impact and Reproducibility

Why this matters:
- InferFlow provides a practical way to evaluate routing strategy behavior before costly production deployment.
- Even in milestone phase, the system demonstrates a full working inference-routing pipeline and measurable behavior under load.

Can classmates run it now?
- Yes, on CPU-only laptops.
- Quick path:
  1. `docker compose --profile real up -d --build`
  2. `curl http://localhost:8080/readyz`
  3. `bash scripts/experiments/run_baseline_matrix.sh`
  4. `python3 analysis/summarize_baseline.py --plot`

---

## Milestone 1 Figures (Embedded)

### Figure 1 — P95 Latency by Strategy (Local CPU Baseline)
![Milestone 1 P95 Latency](docs/assets/milestone1_latency_p95_by_strategy.png)

### Figure 2 — Error Rate by Strategy (Local CPU Baseline)
![Milestone 1 Error Rate](docs/assets/milestone1_error_rate_by_strategy.png)
