#!/usr/bin/env bash
set -euo pipefail

HEALTH_URL="http://localhost:8080/healthz"
OUT_DIR="results/mock-baseline"
DURATION=30
WARMUP=5

if ! curl -fsS --max-time 5 "${HEALTH_URL}" >/dev/null; then
  echo "Router is not healthy at ${HEALTH_URL}"
  exit 1
fi

mkdir -p "${OUT_DIR}"

strategies=("round_robin" "least_pending" "cost_aware")
concurrency_levels=(1 2 4)

for strategy in "${strategies[@]}"; do
  for concurrency in "${concurrency_levels[@]}"; do
    echo "=== strategy=${strategy} concurrency=${concurrency} ==="
    python3 loadgen/generator.py \
      --strategy "${strategy}" \
      --concurrency "${concurrency}" \
      --duration "${DURATION}" \
      --warmup "${WARMUP}" \
      --output "${OUT_DIR}/${strategy}_c${concurrency}.jsonl"
  done
done

echo "Mock baseline complete. Results in results/mock-baseline/"
