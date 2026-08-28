#!/usr/bin/env bash
# smoke.sh — hit a running saka HTTP endpoint.
# Usage (after port-forward or ingress):
#   SAKA_BASE_URL=http://127.0.0.1:8080 ./deploy/smoke.sh
set -euo pipefail

BASE="${SAKA_BASE_URL:-http://127.0.0.1:8080}"

curl -sf "${BASE}/health" >/dev/null || { echo "saka down at ${BASE}/health"; exit 1; }
curl -sf "${BASE}/v1/search?q=test&n=3&format=json" | grep -q '"results"' \
  || { echo "search failed at ${BASE}/v1/search"; exit 1; }
echo "ok: stack healthy at ${BASE}"
