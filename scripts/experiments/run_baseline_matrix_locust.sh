#!/usr/bin/env bash
set -euo pipefail

HEALTH_URL="http://localhost:8080/healthz"
STRATEGY_URL="http://localhost:8080/strategy"
OUT_DIR="results/baseline-locust"
RUN_TIME="30s"

if ! command -v locust >/dev/null 2>&1; then
  echo "locust is not installed. Run: pip3 install -r loadgen/requirements.txt"
  exit 1
fi

if ! curl -fsS --max-time 5 "${HEALTH_URL}" >/dev/null; then
  echo "Router is not healthy at ${HEALTH_URL}"
  exit 1
fi

mkdir -p "${OUT_DIR}"

strategies=("round_robin" "least_pending" "cost_aware")
users_levels=(1 2 4)

for strategy in "${strategies[@]}"; do
  for users in "${users_levels[@]}"; do
    echo "=== locust strategy=${strategy} users=${users} ==="

    curl -fsS -X PUT "${STRATEGY_URL}" \
      -H "Content-Type: application/json" \
      -d "{\"strategy\":\"${strategy}\"}" >/dev/null

    export INFERFLOW_MODEL="${INFERFLOW_MODEL:-local-llm}"
    locust \
      -f loadgen/locustfile.py \
      --headless \
      --host http://localhost:8080 \
      --users "${users}" \
      --spawn-rate "${users}" \
      --run-time "${RUN_TIME}" \
      --csv "${OUT_DIR}/${strategy}_u${users}" \
      --csv-full-history \
      --only-summary
  done
done

echo "Locust matrix complete. Results in ${OUT_DIR}/"
