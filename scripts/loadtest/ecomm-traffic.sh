#!/usr/bin/env bash
#
# ecomm-traffic.sh — Sustained HTTP traffic to the Online Boutique ecomm app via NGF gateway
#
# Generates varied GET requests across multiple storefront paths to produce
# realistic traffic patterns in the observability dashboards.
#
# Usage:
#   ./ecomm-traffic.sh
#   ELB_HOST=my-elb.example.com DURATION=60 CONCURRENCY=10 ./ecomm-traffic.sh
#

set -euo pipefail

ELB_HOST="${ELB_HOST:-k8s-nginxgat-ecommngi-3106c80616-8d048562cdf95f22.elb.us-east-1.amazonaws.com}"
VIRTUAL_HOST="${VIRTUAL_HOST:-ecomm.llm.local}"
DURATION="${DURATION:-300}"
CONCURRENCY="${CONCURRENCY:-15}"

# Paths to cycle through for varied URL patterns in metrics
PATHS=(
    "/"
    "/"
    "/"
    "/product/OLJCESPC7Z"
    "/product/66VCHSJNUP"
    "/product/1YMWWN1N4O"
    "/cart"
)

# Shared counters via temp files (subshell-safe)
COUNTER_DIR=$(mktemp -d)
echo "0" > "${COUNTER_DIR}/ok"
echo "0" > "${COUNTER_DIR}/err"

PIDS=()
RUNNING=true

get_count() {
    local total=0
    for f in "${COUNTER_DIR}/${1}".*; do
        [ -f "$f" ] && total=$((total + $(cat "$f" 2>/dev/null || echo 0)))
    done
    echo "$total"
}

cleanup() {
    RUNNING=false
    for pid in "${PIDS[@]+"${PIDS[@]}"}"; do
        kill "$pid" 2>/dev/null || true
    done
    wait 2>/dev/null || true
    local ok_total err_total total
    ok_total=$(get_count ok)
    err_total=$(get_count err)
    total=$((ok_total + err_total))
    echo ""
    echo "============================================"
    echo "  Ecomm Traffic — Final Summary"
    echo "============================================"
    echo "  Duration:    ${DURATION}s"
    echo "  Concurrency: ${CONCURRENCY}"
    echo "  Total:       ${total}"
    echo "  Success:     ${ok_total}"
    echo "  Errors:      ${err_total}"
    if [ "$total" -gt 0 ]; then
        local rate
        rate=$(echo "scale=1; $total / $DURATION" | bc 2>/dev/null || echo "N/A")
        echo "  Avg RPS:     ${rate}"
    fi
    echo "============================================"
    rm -rf "${COUNTER_DIR}"
}

trap cleanup EXIT INT TERM

worker() {
    local worker_id=$1
    local ok=0
    local err=0
    local end_time=$(($(date +%s) + DURATION))
    local path_count=${#PATHS[@]}
    local idx=0
    while [ "$(date +%s)" -lt "$end_time" ]; do
        local path="${PATHS[$((idx % path_count))]}"
        idx=$((idx + 1))
        local status
        status=$(curl -s -o /dev/null -w "%{http_code}" \
            -H "Host: ${VIRTUAL_HOST}" \
            --connect-timeout 5 \
            --max-time 10 \
            "http://${ELB_HOST}${path}" 2>/dev/null) || status="000"
        if [ "$status" -ge 200 ] && [ "$status" -lt 400 ]; then
            ok=$((ok + 1))
        else
            err=$((err + 1))
        fi
    done
    echo "$ok" > "${COUNTER_DIR}/ok.${worker_id}"
    echo "$err" > "${COUNTER_DIR}/err.${worker_id}"
}

echo "============================================"
echo "  Ecomm (Online Boutique) Traffic Generator"
echo "============================================"
echo "  ELB:         ${ELB_HOST}"
echo "  Host Header: ${VIRTUAL_HOST}"
echo "  Duration:    ${DURATION}s"
echo "  Concurrency: ${CONCURRENCY} workers"
echo "  Paths:       ${#PATHS[@]} URL patterns"
echo "============================================"
echo ""

# Launch workers
for i in $(seq 1 "$CONCURRENCY"); do
    worker "$i" &
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
    OK_NOW=$(get_count ok)
    ERR_NOW=$(get_count err)
    TOTAL=$((OK_NOW + ERR_NOW))
    if [ "$ELAPSED" -gt 0 ]; then
        RPS=$(echo "scale=1; $TOTAL / $ELAPSED" | bc 2>/dev/null || echo "N/A")
    else
        RPS="0"
    fi
    REMAINING=$((DURATION - ELAPSED))
    echo "[$(date +%H:%M:%S)] ${TOTAL} reqs (${OK_NOW} ok, ${ERR_NOW} err) | ${RPS} req/s | ${REMAINING}s remaining"
done

# Wait for all workers
for pid in "${PIDS[@]+"${PIDS[@]}"}"; do
    wait "$pid" 2>/dev/null || true
done
PIDS=()
