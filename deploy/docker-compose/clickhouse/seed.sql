-- Inference observability seed data for demos.
-- Run after init.sql to populate tables with ~1000 rows of realistic data.

-- Pool metadata
CREATE TABLE IF NOT EXISTS ngf_inference_pools (
    name String,
    namespace String,
    model_name String,
    model_version String,
    serving_backend String,
    gpu_type String,
    gpu_count UInt32,
    replicas UInt32,
    ready_replicas UInt32,
    min_replicas UInt32,
    max_replicas UInt32,
    status String,
    created_at DateTime
) ENGINE = MergeTree() ORDER BY name;

INSERT INTO ngf_inference_pools VALUES
    ('llama3-70b-prod', 'inference', 'meta-llama/Llama-3-70B-Instruct', 'v1.2', 'vllm', 'H100', 4, 6, 6, 2, 12, 'Ready', now() - INTERVAL 72 HOUR),
    ('mixtral-8x7b-staging', 'inference', 'mistralai/Mixtral-8x7B-Instruct-v0.1', 'v0.1', 'vllm', 'A100', 2, 3, 3, 1, 8, 'Ready', now() - INTERVAL 48 HOUR),
    ('phi3-mini-dev', 'dev', 'microsoft/Phi-3-mini-4k-instruct', '', 'triton', 'L40S', 1, 2, 1, 1, 4, 'Degraded', now() - INTERVAL 24 HOUR),
    ('codellama-34b-prod', 'inference', 'codellama/CodeLlama-34b-Instruct-hf', 'v2.0', 'tgi', 'A100', 2, 4, 4, 2, 8, 'Ready', now() - INTERVAL 120 HOUR);

-- EPP decisions (500 rows, last 10 minutes)
CREATE TABLE IF NOT EXISTS ngf_epp_decisions (
    timestamp DateTime,
    pool_name String,
    request_id String,
    selected_pod String,
    reason String,
    queue_depth UInt32,
    kv_cache_pct Float64,
    prefix_cache_hit UInt8,
    candidates_considered UInt32,
    decision_latency_us UInt32
) ENGINE = MergeTree() ORDER BY (pool_name, timestamp);

INSERT INTO ngf_epp_decisions
SELECT
    now() - INTERVAL number SECOND AS timestamp,
    arrayElement(['llama3-70b-prod', 'mixtral-8x7b-staging', 'codellama-34b-prod'], 1 + number % 3) AS pool_name,
    concat('req-', toString(number)) AS request_id,
    concat(arrayElement(['llama3-70b-prod', 'mixtral-8x7b-staging', 'codellama-34b-prod'], 1 + number % 3), '-pod-', toString(number % 6)) AS selected_pod,
    arrayElement(['least_queue', 'kv_cache', 'prefix_affinity', 'composite'], 1 + number % 4) AS reason,
    rand() % 12 AS queue_depth,
    40 + (rand() % 40) AS kv_cache_pct,
    rand() % 2 AS prefix_cache_hit,
    3 + rand() % 4 AS candidates_considered,
    80 + rand() % 200 AS decision_latency_us
FROM numbers(500);

-- Pod metrics (latest snapshot)
CREATE TABLE IF NOT EXISTS ngf_pod_metrics (
    pool_name String,
    pod_name String,
    node_name String,
    gpu_id UInt32,
    gpu_type String,
    queue_depth UInt32,
    kv_cache_util_pct Float64,
    prefix_cache_state UInt8,
    gpu_util_pct Float64,
    gpu_mem_used_mb UInt32,
    gpu_mem_total_mb UInt32,
    gpu_temperature_c UInt32,
    requests_in_flight UInt32
) ENGINE = MergeTree() ORDER BY (pool_name, pod_name);

INSERT INTO ngf_pod_metrics
SELECT
    'llama3-70b-prod' AS pool_name,
    concat('llama3-70b-prod-pod-', toString(number)) AS pod_name,
    concat('gpu-node-', toString(number % 3)) AS node_name,
    number % 4 AS gpu_id,
    'H100' AS gpu_type,
    rand() % 10 AS queue_depth,
    40 + (rand() % 40) AS kv_cache_util_pct,
    rand() % 2 AS prefix_cache_state,
    50 + (rand() % 40) AS gpu_util_pct,
    50000 + (rand() % 25000) AS gpu_mem_used_mb,
    81920 AS gpu_mem_total_mb,
    55 + (rand() % 20) AS gpu_temperature_c,
    rand() % 6 AS requests_in_flight
FROM numbers(6);

-- 1-minute aggregated metrics (last 60 minutes, 4 pools)
CREATE TABLE IF NOT EXISTS ngf_inference_metrics_1m (
    timestamp DateTime,
    pool_name String,
    ttft_ms Float64,
    tps Float64,
    total_tokens UInt64,
    queue_depth Float64,
    kv_cache_pct Float64,
    prefix_cache_hit UInt8,
    gpu_util_pct Float64
) ENGINE = MergeTree() ORDER BY (pool_name, timestamp);

INSERT INTO ngf_inference_metrics_1m
SELECT
    now() - INTERVAL (number % 60) MINUTE AS timestamp,
    arrayElement(['llama3-70b-prod', 'mixtral-8x7b-staging', 'codellama-34b-prod', 'phi3-mini-dev'], 1 + intDiv(number, 60)) AS pool_name,
    80 + (rand() % 120) AS ttft_ms,
    60 + (rand() % 60) AS tps,
    1000 + (rand() % 5000) AS total_tokens,
    2 + (rand() % 10) AS queue_depth,
    40 + (rand() % 40) AS kv_cache_pct,
    rand() % 2 AS prefix_cache_hit,
    50 + (rand() % 40) AS gpu_util_pct
FROM numbers(240);
