package clickhouse

// SQL queries for inference metrics.
// These target ClickHouse MergeTree tables populated by the OTel Collector.
// All queries include an optional cluster_name filter: pass '' to match all clusters.

const queryListPools = `
SELECT
    name, namespace, model_name, model_version, serving_backend,
    gpu_type, gpu_count, replicas, ready_replicas, min_replicas, max_replicas,
    status, created_at
FROM ngf_inference_pools
WHERE (? = '' OR cluster_name = ?)
ORDER BY name
`

const queryGetPool = `
SELECT
    name, namespace, model_name, model_version, serving_backend,
    gpu_type, gpu_count, replicas, ready_replicas, min_replicas, max_replicas,
    status, created_at
FROM ngf_inference_pools
WHERE name = ?
  AND (? = '' OR cluster_name = ?)
LIMIT 1
`

const queryMetricsSummary = `
SELECT
    avg(ttft_ms) AS avg_ttft,
    quantile(0.95)(ttft_ms) AS p95_ttft,
    quantile(0.99)(ttft_ms) AS p99_ttft,
    avg(tps) AS avg_tps,
    sum(total_tokens) AS total_tokens,
    avg(queue_depth) AS avg_queue_depth,
    avg(kv_cache_pct) AS avg_kv_cache_pct,
    countIf(prefix_cache_hit = 1) / count() AS prefix_cache_hit_rate,
    avg(gpu_util_pct) AS avg_gpu_util
FROM ngf_inference_metrics_1m
WHERE timestamp >= now() - INTERVAL 60 MINUTE
  AND (? = '' OR pool_name = ?)
  AND (? = '' OR cluster_name = ?)
`

const queryPodMetrics = `
SELECT
    pod_name, node_name, gpu_id, gpu_type, queue_depth,
    kv_cache_util_pct, prefix_cache_state, gpu_util_pct,
    gpu_mem_used_mb, gpu_mem_total_mb, gpu_temperature_c,
    requests_in_flight
FROM ngf_pod_metrics
WHERE pool_name = ?
  AND (? = '' OR cluster_name = ?)
ORDER BY pod_name
`

const queryRecentEPPDecisions = `
SELECT
    timestamp, request_id, selected_pod, reason, queue_depth,
    kv_cache_pct, prefix_cache_hit, candidates_considered,
    decision_latency_us
FROM ngf_epp_decisions
WHERE pool_name = ?
  AND (? = '' OR cluster_name = ?)
ORDER BY timestamp DESC
LIMIT ?
`

const queryTTFTHistogram = `
SELECT
    floor(ttft_ms / 50) * 50 AS range_start,
    floor(ttft_ms / 50) * 50 + 50 AS range_end,
    count() AS cnt
FROM ngf_inference_metrics_1m
WHERE pool_name = ?
  AND (? = '' OR cluster_name = ?)
  AND timestamp >= now() - INTERVAL 60 MINUTE
GROUP BY range_start, range_end
ORDER BY range_start
`

const queryTPSThroughput = `
SELECT
    toStartOfMinute(timestamp) AS ts,
    avg(tps) AS value
FROM ngf_inference_metrics_1m
WHERE pool_name = ?
  AND (? = '' OR cluster_name = ?)
  AND timestamp >= now() - INTERVAL 60 MINUTE
GROUP BY ts
ORDER BY ts
`

const queryQueueDepthSeries = `
SELECT
    toStartOfMinute(timestamp) AS ts,
    avg(queue_depth) AS value
FROM ngf_inference_metrics_1m
WHERE pool_name = ?
  AND (? = '' OR cluster_name = ?)
  AND timestamp >= now() - INTERVAL 60 MINUTE
GROUP BY ts
ORDER BY ts
`

const queryGPUUtilSeries = `
SELECT
    toStartOfMinute(timestamp) AS ts,
    avg(gpu_util_pct) AS value
FROM ngf_inference_metrics_1m
WHERE pool_name = ?
  AND (? = '' OR cluster_name = ?)
  AND timestamp >= now() - INTERVAL 60 MINUTE
GROUP BY ts
ORDER BY ts
`

const queryKVCacheSeries = `
SELECT
    toStartOfMinute(timestamp) AS ts,
    avg(kv_cache_pct) AS value
FROM ngf_inference_metrics_1m
WHERE pool_name = ?
  AND (? = '' OR cluster_name = ?)
  AND timestamp >= now() - INTERVAL 60 MINUTE
GROUP BY ts
ORDER BY ts
`
