#!/usr/bin/env bash
#
# verify-metrics.sh — Check that observability dashboards are populated with metrics
#
# Usage:
#   ./verify-metrics.sh
#   API_URL=http://localhost:8080 ./verify-metrics.sh
#

set -euo pipefail

API_URL="${API_URL:-http://localhost:8080}"
PASS=0
FAIL=0

check() {
    local name="$1"
    local condition="$2"
    local detail="$3"

    if [ "$condition" = "true" ]; then
        echo "  [PASS] ${name}"
        [ -n "$detail" ] && echo "         ${detail}"
        PASS=$((PASS + 1))
    else
        echo "  [FAIL] ${name}"
        [ -n "$detail" ] && echo "         ${detail}"
        FAIL=$((FAIL + 1))
    fi
}

echo "============================================"
echo "  Observability Metrics Verification"
echo "============================================"
echo "  API: ${API_URL}"
echo "============================================"
echo ""

# --- Check 1: Metrics Summary ---
echo "  Checking /api/v1/metrics/summary..."
SUMMARY=$(curl -s "${API_URL}/api/v1/metrics/summary" 2>/dev/null || echo "{}")
RPS=$(echo "$SUMMARY" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('requestsPerSec',0))" 2>/dev/null || echo "0")
TOTAL_REQ=$(echo "$SUMMARY" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('totalRequests',0))" 2>/dev/null || echo "0")
ACTIVE=$(echo "$SUMMARY" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('activeConnections',0))" 2>/dev/null || echo "0")
P95=$(echo "$SUMMARY" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('p95LatencyMs',0))" 2>/dev/null || echo "0")

RPS_OK=$(python3 -c "print('true' if float('${RPS}') > 0 else 'false')" 2>/dev/null || echo "false")
check "Requests/sec > 0" "$RPS_OK" "requestsPerSec=${RPS}"

TOTAL_OK=$(python3 -c "print('true' if float('${TOTAL_REQ}') > 0 else 'false')" 2>/dev/null || echo "false")
check "Total requests > 0" "$TOTAL_OK" "totalRequests=${TOTAL_REQ}"

ACTIVE_OK=$(python3 -c "print('true' if float('${ACTIVE}') > 0 else 'false')" 2>/dev/null || echo "false")
check "Active connections > 0" "$ACTIVE_OK" "activeConnections=${ACTIVE}"

# P95 latency — note: only available with rich NGF metrics (nginx_gateway_fabric_*)
# Basic nginx metrics don't expose latency histograms, so 0 is acceptable
P95_OK=$(python3 -c "print('true' if float('${P95}') > 0 else 'false')" 2>/dev/null || echo "false")
if [ "$P95_OK" = "true" ]; then
    check "P95 Latency available" "$P95_OK" "p95LatencyMs=${P95}"
else
    echo "  [SKIP] P95 Latency (not available with basic nginx metrics)"
fi

echo ""

# --- Check 2: Metrics by Route ---
echo "  Checking /api/v1/metrics/by-route..."
BY_ROUTE=$(curl -s "${API_URL}/api/v1/metrics/by-route" 2>/dev/null || echo "[]")
ROUTE_COUNT=$(echo "$BY_ROUTE" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d) if isinstance(d,list) else 0)" 2>/dev/null || echo "0")

ROUTES_OK=$(python3 -c "print('true' if int('${ROUTE_COUNT}') > 0 else 'false')" 2>/dev/null || echo "false")
check "Routes/pods with metrics" "$ROUTES_OK" "${ROUTE_COUNT} route(s)/pod(s) reporting metrics"

# Show actual route names for visibility
echo "$BY_ROUTE" | python3 -c "
import sys,json
d=json.load(sys.stdin)
if isinstance(d,list):
    for r in d:
        ns = r.get('namespace','?')
        name = r.get('name','?')
        host = r.get('hostname','?')
        rps = r.get('requestsPerSec',0)
        print(f'         -> {ns}/{name} (host={host}, rps={rps:.2f})')
" 2>/dev/null || true

echo ""

# --- Check 3: Metrics by Gateway ---
echo "  Checking /api/v1/metrics/by-gateway..."
BY_GW=$(curl -s "${API_URL}/api/v1/metrics/by-gateway" 2>/dev/null || echo "[]")
GW_COUNT=$(echo "$BY_GW" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d) if isinstance(d,list) else 0)" 2>/dev/null || echo "0")

GW_OK=$(python3 -c "print('true' if int('${GW_COUNT}') > 0 else 'false')" 2>/dev/null || echo "false")
check "Gateway metrics present" "$GW_OK" "${GW_COUNT} gateway(s) reporting metrics"

# Show actual gateway names
echo "$BY_GW" | python3 -c "
import sys,json
d=json.load(sys.stdin)
if isinstance(d,list):
    for g in d:
        ns = g.get('namespace','?')
        name = g.get('name','?')
        rps = g.get('requestsPerSec',0)
        active = g.get('activeConnections',0)
        print(f'         -> {ns}/{name} (rps={rps:.2f}, connections={active})')
" 2>/dev/null || true

echo ""

# --- Check 4: Inference Metrics Summary ---
echo "  Checking /api/v1/inference/metrics/summary..."
INF_SUMMARY=$(curl -s "${API_URL}/api/v1/inference/metrics/summary" 2>/dev/null || echo "{}")
INF_STATUS=$?

INF_OK="false"
if [ "$INF_STATUS" -eq 0 ] && [ "$INF_SUMMARY" != "{}" ] && [ -n "$INF_SUMMARY" ]; then
    INF_OK=$(echo "$INF_SUMMARY" | python3 -c "
import sys,json
try:
    d=json.load(sys.stdin)
    # Check if response has any non-empty data
    print('true' if d and not all(v==0 or v=='' or v is None for v in d.values() if not isinstance(v,(dict,list))) else 'false')
except:
    print('false')
" 2>/dev/null || echo "false")
fi
check "Inference metrics available" "$INF_OK" ""

echo ""

# --- Check 5: Inference Pod Metrics ---
echo "  Checking /api/v1/inference/metrics/pods..."
POD_METRICS=$(curl -s "${API_URL}/api/v1/inference/metrics/pods?pool=vllm-llama3-8b-instruct" 2>/dev/null || echo "[]")
POD_COUNT=$(echo "$POD_METRICS" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d) if isinstance(d,list) else 0)" 2>/dev/null || echo "0")

POD_OK=$(python3 -c "print('true' if int('${POD_COUNT}') > 0 else 'false')" 2>/dev/null || echo "false")
check "Inference pod metrics" "$POD_OK" "${POD_COUNT} pod(s) reporting"

echo ""

# --- Summary ---
TOTAL=$((PASS + FAIL))
echo "============================================"
echo "  Results: ${PASS}/${TOTAL} checks passed"
if [ "$FAIL" -eq 0 ]; then
    echo "  Status: ALL CHECKS PASSED"
else
    echo "  Status: ${FAIL} CHECK(S) FAILED"
fi
echo "============================================"

exit "$FAIL"
