#!/bin/sh
# smoke.sh — run after: docker compose up -d && sleep 15
curl -sf http://localhost:8080/health || { echo "saka down"; exit 1; }
curl -sf "http://localhost:8080/v1/search?q=test" | grep -q '"results"' || { echo "search failed"; exit 1; }
echo "✓ stack healthy"
