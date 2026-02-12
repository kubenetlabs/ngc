CREATE TABLE IF NOT EXISTS ngf_access_logs (
    timestamp DateTime64(3),
    cluster_name LowCardinality(String) DEFAULT '',
    gateway String,
    listener String,
    route String,
    namespace String,
    method LowCardinality(String),
    path String,
    status UInt16,
    latency_ms Float64,
    upstream_latency_ms Float64,
    request_size UInt64,
    response_size UInt64,
    upstream_name String,
    upstream_addr String,
    client_ip String,
    user_agent String,
    request_id String,
    trace_id String,
    tls_version LowCardinality(String),
    tls_cipher LowCardinality(String),
    waf_action LowCardinality(String),
    waf_violation_rating Float32,
    waf_signatures Array(String),
    bot_classification LowCardinality(String),
    xc_edge_latency_ms Float64
) ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (cluster_name, gateway, route, timestamp)
TTL timestamp + INTERVAL 7 DAY;

CREATE TABLE IF NOT EXISTS ngf_inference_logs (
    timestamp DateTime64(3),
    cluster_name LowCardinality(String) DEFAULT '',
    inference_pool String,
    model_name String,
    model_version String,
    pod_name String,
    node_name String,
    gpu_id UInt8,
    gpu_type LowCardinality(String),
    request_id String,
    trace_id String,
    time_to_first_token_ms Float64,
    total_inference_time_ms Float64,
    tokens_generated UInt32,
    input_tokens UInt32,
    output_tokens UInt32,
    tokens_per_second Float32,
    epp_selected_reason LowCardinality(String),
    epp_decision_latency_us Float32,
    queue_depth_at_selection UInt16,
    kv_cache_pct_at_selection Float32,
    prefix_cache_hit Boolean,
    candidate_pods_considered UInt8,
    gpu_utilization_pct Float32,
    gpu_memory_used_mb UInt32,
    gpu_memory_total_mb UInt32,
    gpu_temperature_c UInt16,
    pool_replica_count UInt16,
    pool_target_replica_count UInt16,
    status UInt16,
    client_ip String,
    path String,
    method LowCardinality(String),
    request_size UInt64,
    response_size UInt64,
    xc_edge_latency_ms Float64,
    xc_waf_action LowCardinality(String),
    xc_bot_classification LowCardinality(String)
) ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (cluster_name, inference_pool, model_name, timestamp)
TTL timestamp + INTERVAL 14 DAY;

CREATE MATERIALIZED VIEW IF NOT EXISTS ngf_metrics_1m
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

CREATE MATERIALIZED VIEW IF NOT EXISTS ngf_inference_metrics_1m
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
