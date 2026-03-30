#!/usr/bin/env bash
set -euo pipefail

bash scripts/experiments/run_mock_baseline.sh

echo
echo "Next steps for real baseline:"
echo "1) docker compose down"
echo "2) docker compose --profile real up -d --build"
echo "3) bash scripts/wait-for-backends.sh"
echo "4) bash scripts/experiments/run_baseline_matrix.sh"
echo

python3 analysis/summarize_baseline.py \
  --input-dir results/mock-baseline \
  --output results/mock-baseline/summary.md

python3 analysis/summarize_baseline.py \
  --input-dir results/baseline \
  --output results/baseline/summary.md

if ! command -v column >/dev/null 2>&1; then
  echo "column command not found; cannot print side-by-side tables"
  exit 1
fi

tmp_left="$(mktemp)"
tmp_right="$(mktemp)"
trap 'rm -f "${tmp_left}" "${tmp_right}"' EXIT

column -t -s '|' results/mock-baseline/summary.md > "${tmp_left}"
column -t -s '|' results/baseline/summary.md > "${tmp_right}"

echo "mock-baseline summary (left) vs baseline summary (right):"
paste "${tmp_left}" "${tmp_right}"

echo "Compare mock-baseline/summary.md vs baseline/summary.md for full analysis"
