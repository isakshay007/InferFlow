#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${INFERFLOW_BACKENDS:-}" && -f ".env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

if [[ -z "${INFERFLOW_BACKENDS:-}" ]]; then
  echo "INFERFLOW_BACKENDS is not set"
  exit 1
fi

IFS=',' read -r -a backends <<< "${INFERFLOW_BACKENDS}"
deadline=$((SECONDS + 120))

while (( SECONDS < deadline )); do
  ready=()
  waiting=()

  for backend in "${backends[@]}"; do
    backend="$(echo "${backend}" | xargs)"
    if [[ -z "${backend}" ]]; then
      continue
    fi

    health_url="${backend%/}/healthz"
    if curl -fsS --max-time 2 "${health_url}" >/dev/null 2>&1; then
      ready+=("${backend}")
    else
      waiting+=("${backend}")
    fi
  done

  if (( ${#ready[@]} > 0 )); then
    echo "Ready: ${ready[*]}"
  else
    echo "Ready: none"
  fi

  if (( ${#waiting[@]} == 0 )); then
    echo "All backends are ready."
    exit 0
  fi

  echo "Waiting: ${waiting[*]}"
  sleep 3
done

echo "Timeout after 120 seconds waiting for backends."
exit 1
