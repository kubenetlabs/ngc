#!/usr/bin/env bash
#
# run-all.sh — Orchestrate the full observability load test
#
# Runs both traffic generators concurrently, then verifies metrics are populated.
#
# Usage:
#   ./run-all.sh                    # Full 5-minute test with defaults
#   DURATION=60 ./run-all.sh        # Quick 1-minute test
#   SKIP_PORT_FORWARD=1 ./run-all.sh  # Skip port-forward setup
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ELB_HOST="${ELB_HOST:-k8s-nginxgat-llmgatew-bda66d5efd-8abb027b3d9f5051.elb.us-east-1.amazonaws.com}"
DURATION="${DURATION:-300}"
SKIP_PORT_FORWARD="${SKIP_PORT_FORWARD:-0}"
API_URL="${API_URL:-http://localhost:8080}"
PF_PIDS=()
CLEANUP_DONE=false

cleanup() {
    if [ "$CLEANUP_DONE" = true ]; then
        return
    fi
    CLEANUP_DONE=true
    echo ""
    echo "Cleaning up..."
    # Kill port-forwards we started
    for pid in "${PF_PIDS[@]+"${PF_PIDS[@]}"}"; do
        kill "$pid" 2>/dev/null || true
    done
    # Kill any child processes
    jobs -p 2>/dev/null | while read -r pid; do
        kill "$pid" 2>/dev/null || true
    done
    wait 2>/dev/null || true
    echo "Done."
}

trap cleanup EXIT INT TERM

echo "============================================================"
echo "  NGF Console — Observability Load Test"
echo "============================================================"
echo "  ELB:      ${ELB_HOST}"
echo "  Duration: ${DURATION}s"
echo "  Scripts:  ${SCRIPT_DIR}"
echo "============================================================"
echo ""

# ---- Step 1: Port Forwards ----
if [ "$SKIP_PORT_FORWARD" = "0" ]; then
    echo "[Step 1/5] Setting up port-forwards..."

    # Check if API port is already in use
    if curl -s -o /dev/null --connect-timeout 2 "${API_URL}/api/v1/health" 2>/dev/null; then
        echo "  API already reachable at ${API_URL} — skipping port-forward"
    else
        echo "  Starting API port-forward (8080)..."
        kubectl port-forward -n ngf-console svc/ngf-console-api 8080:8080 >/dev/null 2>&1 &
        PF_PIDS+=($!)
        sleep 2
    fi

    # Check if frontend port is already in use
    if curl -s -o /dev/null --connect-timeout 2 "http://localhost:3000" 2>/dev/null; then
        echo "  Frontend already reachable at localhost:3000 — skipping port-forward"
    else
        echo "  Starting frontend port-forward (3000)..."
        kubectl port-forward -n ngf-console svc/ngf-console-frontend 3000:80 >/dev/null 2>&1 &
        PF_PIDS+=($!)
        sleep 2
    fi
    echo ""
else
    echo "[Step 1/5] Skipping port-forward setup (SKIP_PORT_FORWARD=1)"
    echo ""
fi

# ---- Step 2: Connectivity Check ----
echo "[Step 2/5] Verifying gateway reachability..."

echo -n "  openwebui-route (chat.llm.local): "
OW_STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "Host: chat.llm.local" \
    --connect-timeout 10 \
    --max-time 15 \
    "http://${ELB_HOST}/" 2>/dev/null) || OW_STATUS="000"
if [ "$OW_STATUS" -ge 200 ] && [ "$OW_STATUS" -lt 400 ]; then
    echo "OK (${OW_STATUS})"
else
    echo "WARNING (${OW_STATUS}) — traffic may still generate metrics"
fi

echo -n "  inference-route (inference.llm.local): "
INF_STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "Host: inference.llm.local" \
    -H "Content-Type: application/json" \
    --connect-timeout 10 \
    --max-time 60 \
    -d '{"model":"TinyLlama/TinyLlama-1.1B-Chat-v1.0","messages":[{"role":"user","content":"Hi"}],"max_tokens":5,"stream":false}' \
    "http://${ELB_HOST}/v1/chat/completions" 2>/dev/null) || INF_STATUS="000"
if [ "$INF_STATUS" -ge 200 ] && [ "$INF_STATUS" -lt 400 ]; then
    echo "OK (${INF_STATUS})"
else
    echo "WARNING (${INF_STATUS}) — inference requests may fail"
fi
echo ""

# ---- Step 3: Launch OpenWebUI Traffic (background) ----
echo "[Step 3/5] Starting openwebui-route traffic (background)..."
ELB_HOST="$ELB_HOST" DURATION="$DURATION" "${SCRIPT_DIR}/openwebui-traffic.sh" &
OPENWEBUI_PID=$!
echo "  PID: ${OPENWEBUI_PID}"
echo ""

# Small delay so output doesn't interleave
sleep 1

# ---- Step 4: Launch Inference Traffic (foreground) ----
echo "[Step 4/5] Starting inference pool traffic (foreground)..."
echo ""
python3 "${SCRIPT_DIR}/inference-traffic.py" \
    --elb-host "$ELB_HOST" \
    --duration "$DURATION"

# Wait for openwebui traffic to finish
echo ""
echo "  Waiting for openwebui traffic to complete..."
wait "$OPENWEBUI_PID" 2>/dev/null || true
echo ""

# ---- Step 5: Verify Metrics ----
echo "[Step 5/5] Verifying observability metrics..."
echo ""
API_URL="$API_URL" "${SCRIPT_DIR}/verify-metrics.sh"

echo ""
echo "============================================================"
echo "  Load test complete!"
echo ""
echo "  View dashboards:"
echo "    Frontend:  http://localhost:3000"
echo "    API Health: ${API_URL}/api/v1/health"
echo "    Metrics:    ${API_URL}/api/v1/metrics/summary"
echo "============================================================"
