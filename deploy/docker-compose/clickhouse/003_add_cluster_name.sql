-- Migration 003: Add cluster_name column to all tables for multi-cluster support.
-- This column is populated by the OTel Collector's resource processor.

-- Add cluster_name to raw log tables.
ALTER TABLE ngf_access_logs ADD COLUMN IF NOT EXISTS cluster_name LowCardinality(String) DEFAULT '' AFTER timestamp;
ALTER TABLE ngf_inference_logs ADD COLUMN IF NOT EXISTS cluster_name LowCardinality(String) DEFAULT '' AFTER timestamp;

-- Add cluster_name to metadata tables (seed data).
ALTER TABLE ngf_inference_pools ADD COLUMN IF NOT EXISTS cluster_name LowCardinality(String) DEFAULT '' AFTER name;
ALTER TABLE ngf_epp_decisions ADD COLUMN IF NOT EXISTS cluster_name LowCardinality(String) DEFAULT '' AFTER pool_name;
ALTER TABLE ngf_pod_metrics ADD COLUMN IF NOT EXISTS cluster_name LowCardinality(String) DEFAULT '' AFTER pool_name;
ALTER TABLE ngf_inference_metrics_1m ADD COLUMN IF NOT EXISTS cluster_name LowCardinality(String) DEFAULT '' AFTER pool_name;

-- Drop and recreate materialized views to include cluster_name.
-- Uses AggregatingMergeTree with -State combinators for correct avg/quantile merging.
DROP VIEW IF EXISTS ngf_metrics_1m;
DROP VIEW IF EXISTS ngf_inference_metrics_1m;

CREATE MATERIALIZED VIEW ngf_metrics_1m
ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMMDD(window_start)
ORDER BY (cluster_name, gateway, route, status_class, window_start)
TTL window_start + INTERVAL 90 DAY
AS SELECT
    toStartOfMinute(timestamp) AS window_start,
    cluster_name,
    gateway, route,
    multiIf(status < 200, '1xx', status < 300, '2xx', status < 400, '3xx',
            status < 500, '4xx', '5xx') AS status_class,
    countState() AS request_count,
    avgState(latency_ms) AS avg_latency,
    quantileState(0.95)(latency_ms) AS p95_latency,
    quantileState(0.99)(latency_ms) AS p99_latency,
    sumState(request_size) AS total_request_bytes,
    sumState(response_size) AS total_response_bytes
FROM ngf_access_logs
GROUP BY window_start, cluster_name, gateway, route, status_class;

CREATE MATERIALIZED VIEW ngf_inference_metrics_1m
ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMMDD(window_start)
ORDER BY (cluster_name, inference_pool, model_name, window_start)
TTL window_start + INTERVAL 90 DAY
AS SELECT
    toStartOfMinute(timestamp) AS window_start,
    cluster_name,
    inference_pool,
    model_name,
    countState() AS request_count,
    avgState(time_to_first_token_ms) AS avg_ttft,
    quantileState(0.50)(time_to_first_token_ms) AS p50_ttft,
    quantileState(0.95)(time_to_first_token_ms) AS p95_ttft,
    quantileState(0.99)(time_to_first_token_ms) AS p99_ttft,
    avgState(tokens_per_second) AS avg_tps,
    sumState(tokens_generated) AS total_tokens,
    avgState(queue_depth_at_selection) AS avg_queue_depth,
    avgState(kv_cache_pct_at_selection) AS avg_kv_cache_pct,
    avgState(gpu_utilization_pct) AS avg_gpu_util,
    avgState(gpu_memory_used_mb) AS avg_gpu_mem_used,
    maxState(gpu_memory_used_mb) AS max_gpu_mem_used,
    avgState(epp_decision_latency_us) AS avg_epp_latency
FROM ngf_inference_logs
GROUP BY window_start, cluster_name, inference_pool, model_name;
