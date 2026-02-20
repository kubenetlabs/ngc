#!/usr/bin/env bash
#
# openwebui-traffic.sh — Sustained HTTP GET traffic through openwebui-route via NGF gateway
#
# Usage:
#   ./openwebui-traffic.sh
#   ELB_HOST=my-elb.example.com DURATION=60 CONCURRENCY=10 ./openwebui-traffic.sh
#

set -euo pipefail

ELB_HOST="${ELB_HOST:-k8s-nginxgat-llmgatew-bda66d5efd-8abb027b3d9f5051.elb.us-east-1.amazonaws.com}"
VIRTUAL_HOST="${VIRTUAL_HOST:-chat.llm.local}"
DURATION="${DURATION:-300}"
CONCURRENCY="${CONCURRENCY:-20}"

OK_COUNT=0
ERR_COUNT=0
PIDS=()
RUNNING=true

cleanup() {
    RUNNING=false
    for pid in "${PIDS[@]+"${PIDS[@]}"}"; do
        kill "$pid" 2>/dev/null || true
    done
    wait 2>/dev/null || true
    local total=$((OK_COUNT + ERR_COUNT))
    echo ""
    echo "============================================"
    echo "  OpenWebUI Traffic — Final Summary"
    echo "============================================"
    echo "  Duration:    ${DURATION}s"
    echo "  Concurrency: ${CONCURRENCY}"
    echo "  Total:       ${total}"
    echo "  Success:     ${OK_COUNT}"
    echo "  Errors:      ${ERR_COUNT}"
    if [ "$total" -gt 0 ]; then
        local rate
        rate=$(echo "scale=1; $total / $DURATION" | bc 2>/dev/null || echo "N/A")
        echo "  Avg RPS:     ${rate}"
    fi
    echo "============================================"
}

trap cleanup EXIT INT TERM

worker() {
    local end_time=$(($(date +%s) + DURATION))
    while [ "$(date +%s)" -lt "$end_time" ] && [ "$RUNNING" = true ]; do
        local status
        status=$(curl -s -o /dev/null -w "%{http_code}" \
            -H "Host: ${VIRTUAL_HOST}" \
            --connect-timeout 5 \
            --max-time 10 \
            "http://${ELB_HOST}/" 2>/dev/null) || status="000"
        if [ "$status" -ge 200 ] && [ "$status" -lt 400 ]; then
            OK_COUNT=$((OK_COUNT + 1))
        else
            ERR_COUNT=$((ERR_COUNT + 1))
        fi
    done
}

echo "============================================"
echo "  OpenWebUI Traffic Generator"
echo "============================================"
echo "  ELB:         ${ELB_HOST}"
echo "  Host Header: ${VIRTUAL_HOST}"
echo "  Duration:    ${DURATION}s"
echo "  Concurrency: ${CONCURRENCY} workers"
echo "============================================"
echo ""

# Launch workers
for i in $(seq 1 "$CONCURRENCY"); do
    worker &
    PIDS+=($!)
done

echo "[$(date +%H:%M:%S)] Started ${CONCURRENCY} workers..."

# Progress reporter
START_TIME=$(date +%s)
while true; do
    sleep 10
    ELAPSED=$(( $(date +%s) - START_TIME ))
    if [ "$ELAPSED" -ge "$DURATION" ]; then
        break
    fi
    TOTAL=$((OK_COUNT + ERR_COUNT))
    if [ "$ELAPSED" -gt 0 ]; then
        RPS=$(echo "scale=1; $TOTAL / $ELAPSED" | bc 2>/dev/null || echo "N/A")
    else
        RPS="0"
    fi
    REMAINING=$((DURATION - ELAPSED))
    echo "[$(date +%H:%M:%S)] ${TOTAL} reqs (${OK_COUNT} ok, ${ERR_COUNT} err) | ${RPS} req/s | ${REMAINING}s remaining"
done

# Wait for all workers
for pid in "${PIDS[@]+"${PIDS[@]}"}"; do
    wait "$pid" 2>/dev/null || true
done
PIDS=()
