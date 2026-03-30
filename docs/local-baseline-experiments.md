# Local Baseline Experiments

## Setup

```bash
# Install analysis deps
pip3 install -r analysis/requirements.txt

# Start 2 real backends
docker compose --profile real up -d --build

# Wait for all backends
bash scripts/wait-for-backends.sh
```

## Run full matrix (all strategies x concurrency levels)

```bash
bash scripts/experiments/run_baseline_matrix.sh
```

Default concurrency levels in the matrix are `1`, `2`, and `4`.

## Summarize results

```bash
# Markdown table only
python3 analysis/summarize_baseline.py

# With charts
python3 analysis/summarize_baseline.py --plot
```

## Run a single strategy manually

```bash
python3 loadgen/generator.py \
  --strategy round_robin \
  --concurrency 4 \
  --duration 30 \
  --warmup 5 \
  --output results/baseline/round_robin_c4.jsonl
```

## Switch strategy mid-run (manual)

```bash
curl -X PUT http://localhost:8080/strategy \
  -H "Content-Type: application/json" \
  -d '{"strategy":"cost_aware"}'
```

## Expected output

The matrix runner produces 9 JSONL files in `results/baseline/` (3 strategies x concurrency `1,2,4`).
`analysis/summarize_baseline.py` writes `results/baseline/summary.md` with a table containing strategy, concurrency, request totals, error rate, latency percentiles, and throughput.
With `--plot`, charts are generated in `results/baseline/plots/`:
- `latency_p95_by_strategy.png`
- `throughput_by_strategy.png`
- `error_rate_by_strategy.png`
On CPU-only laptops using `qwen2.5:0.5b`, p95 latency is commonly in the ~5-30 second range depending on hardware and background load.

## Control experiment: mock vs real backend comparison

```bash
# Step 1 — run mock baseline (uses mock profile)
docker compose --profile mock up -d --build
bash scripts/wait-for-backends.sh
bash scripts/experiments/run_mock_baseline.sh

# Step 2 — run real baseline (uses real profile)
docker compose down
docker compose --profile real up -d --build
bash scripts/wait-for-backends.sh
bash scripts/experiments/run_baseline_matrix.sh

# Step 3 — compare
python3 analysis/summarize_baseline.py \
  --compare results/mock-baseline,results/baseline \
  --plot
```

Mock backend runs should show near-zero latency and consistent throughput.
Real Ollama runs should show much higher latency (often 5-30s per request on CPU) and lower throughput.
Routing strategy differences should be visible in both modes, and usually become more pronounced under real backend load.

## Optional: Locust-based matrix run

Use this when you want standardized load-test style output (CSV history + failures) from Locust.

```bash
# Install locust dependency
pip3 install -r loadgen/requirements.txt

# Ensure real profile is up and healthy
docker compose --profile real up -d --build
curl -sS http://localhost:8080/readyz

# Run locust matrix (strategies x users=1,2,4)
bash scripts/experiments/run_baseline_matrix_locust.sh
```

This writes CSV outputs to `results/baseline-locust/` (including per-run stats, failures, and history files) for reporting.
