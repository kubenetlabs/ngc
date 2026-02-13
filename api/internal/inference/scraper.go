package inference

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	mc "github.com/kubenetlabs/ngc/api/internal/multicluster"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DBConn is the minimal interface for writing to ClickHouse, satisfied by
// clickhouse-go's Conn type. Defined here to avoid an import cycle with
// the clickhouse package.
type DBConn interface {
	Exec(ctx context.Context, query string, args ...any) error
}

const vllmMetricsPort = 8000

// ClickHouse insert queries for scraper-produced data.
const insertMetrics1m = `INSERT INTO ngf_inference_metrics_1m (
	timestamp, cluster_name, pool_name, ttft_ms, tps, total_tokens,
	queue_depth, kv_cache_pct, prefix_cache_hit, gpu_util_pct
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

const insertPodMetrics = `INSERT INTO ngf_pod_metrics (
	timestamp, cluster_name, pool_name, pod_name, node_name, gpu_id, gpu_type,
	queue_depth, kv_cache_util_pct, prefix_cache_state, gpu_util_pct,
	gpu_mem_used_mb, gpu_mem_total_mb, gpu_temperature_c, requests_in_flight
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

var podGVR = schema.GroupVersionResource{Version: "v1", Resource: "pods"}

// podCounters tracks previous counter values for computing deltas.
type podCounters struct {
	prevTokensTotal float64
	prevTTFTSum     float64
	prevTTFTCount   float64
	lastSeen        time.Time
}

// metricsScraper scrapes Prometheus /metrics from vLLM pods and writes to ClickHouse.
type metricsScraper struct {
	pool       *mc.ClientPool
	conn       DBConn
	provider   MetricsProvider
	httpClient *http.Client

	mu       sync.Mutex
	counters map[string]*podCounters // key: "cluster/namespace/podName"
}

// RunMetricsScraper starts a loop that scrapes vLLM pod metrics and writes
// them to ClickHouse at the given interval.
func RunMetricsScraper(ctx context.Context, pool *mc.ClientPool, conn DBConn, provider MetricsProvider, interval time.Duration) {
	s := &metricsScraper{
		pool:       pool,
		conn:       conn,
		provider:   provider,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		counters:   make(map[string]*podCounters),
	}

	slog.Info("metrics scraper starting", "interval", interval)
	s.scrapeAll(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.scrapeAll(ctx)
		case <-ctx.Done():
			slog.Info("metrics scraper stopped")
			return
		}
	}
}

func (s *metricsScraper) scrapeAll(ctx context.Context) {
	pools, err := s.provider.ListPools(ctx)
	if err != nil {
		slog.Warn("scraper: failed to list pools", "error", err)
		return
	}
	if len(pools) == 0 {
		return
	}

	clusters := s.pool.List()
	if len(clusters) == 0 {
		return
	}

	for _, cc := range clusters {
		for i := range pools {
			s.scrapePoolInCluster(ctx, cc, &pools[i])
		}
	}

	s.cleanupStaleCounters(10 * time.Minute)
}

// cleanupStaleCounters removes counter entries for pods not seen recently.
func (s *metricsScraper) cleanupStaleCounters(maxAge time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-maxAge)
	for key, c := range s.counters {
		if c.lastSeen.Before(cutoff) {
			delete(s.counters, key)
		}
	}
}

func (s *metricsScraper) scrapePoolInCluster(ctx context.Context, cc *mc.ClusterClient, pool *PoolStatus) {
	if cc.K8sClient == nil {
		return
	}
	dc := cc.K8sClient.DynamicClient()
	if dc == nil {
		return
	}

	ns := pool.Namespace
	if ns == "" {
		ns = "default"
	}

	listCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Try label selectors to find pods for this pool.
	var podList *unstructured.UnstructuredList
	for _, selector := range []string{
		fmt.Sprintf("app=%s", pool.Name),
		fmt.Sprintf("app.kubernetes.io/instance=%s", pool.Name),
		fmt.Sprintf("app.kubernetes.io/name=%s", pool.Name),
	} {
		list, err := dc.Resource(podGVR).Namespace(ns).List(listCtx, metav1.ListOptions{
			LabelSelector: selector,
		})
		if err != nil {
			continue
		}
		if len(list.Items) > 0 {
			podList = list
			break
		}
	}

	if podList == nil || len(podList.Items) == 0 {
		return
	}

	now := time.Now().UTC()
	var (
		totalTTFT    float64
		totalTPS     float64
		totalTokens  uint64
		totalQueue   float64
		totalKV      float64
		totalGPUUtil float64
		podCount     int
	)

	for i := range podList.Items {
		pod := &podList.Items[i]
		phase, _, _ := unstructured.NestedString(pod.Object, "status", "phase")
		if phase != "Running" {
			continue
		}
		podIP, _, _ := unstructured.NestedString(pod.Object, "status", "podIP")
		if podIP == "" {
			continue
		}
		podName := pod.GetName()
		nodeName, _, _ := unstructured.NestedString(pod.Object, "spec", "nodeName")

		body, err := s.fetchMetrics(ctx, podIP)
		if err != nil {
			slog.Debug("scraper: failed to fetch metrics", "pod", podName, "ip", podIP, "error", err)
			continue
		}

		pm := s.parseVLLMMetrics(body, cc.Name, pool.Name, podName, nodeName)
		podCount++

		// Compute counter deltas.
		key := fmt.Sprintf("%s/%s/%s", cc.Name, ns, podName)
		tokensDelta, computedTPS, ttftAvg := s.updateCounters(key, pm)

		// Prefer computed TPS from token deltas; fall back to Prometheus gauge.
		podTPS := computedTPS
		if podTPS == 0 {
			podTPS = pm.tps
		}

		// Accumulate for pool-level aggregation.
		totalTTFT += ttftAvg
		totalTPS += podTPS
		totalTokens += tokensDelta
		totalQueue += float64(pm.queueDepth)
		totalKV += pm.kvCachePct
		totalGPUUtil += pm.gpuUtilPct

		// Write per-pod snapshot to ngf_pod_metrics.
		if err := s.conn.Exec(ctx, insertPodMetrics,
			now, cc.Name, pool.Name, podName, nodeName,
			uint8(0), pool.GPUType,
			uint16(pm.queueDepth), pm.kvCachePct, uint8(0), pm.gpuUtilPct,
			uint32(0), uint32(0), uint16(0), uint16(pm.requestsInFlight),
		); err != nil {
			slog.Warn("scraper: failed to insert pod metrics", "pod", podName, "error", err)
		}
	}

	if podCount == 0 {
		return
	}

	// Write aggregated pool-level row to ngf_inference_metrics_1m.
	n := float64(podCount)
	if err := s.conn.Exec(ctx, insertMetrics1m,
		now, cc.Name, pool.Name,
		roundTo(totalTTFT/n, 2),
		roundTo(totalTPS/n, 2),
		totalTokens,
		uint32(math.Round(totalQueue/n)),
		roundTo(totalKV/n, 2),
		uint8(0),
		roundTo(totalGPUUtil/n, 2),
	); err != nil {
		slog.Warn("scraper: failed to insert metrics_1m", "pool", pool.Name, "error", err)
	} else {
		slog.Debug("scraper: wrote metrics", "pool", pool.Name, "cluster", cc.Name, "pods", podCount)
	}
}

func (s *metricsScraper) fetchMetrics(ctx context.Context, podIP string) (string, error) {
	url := fmt.Sprintf("http://%s:%d/metrics", podIP, vllmMetricsPort)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// parsedPodMetrics holds raw scraped values for a single pod.
type parsedPodMetrics struct {
	tps              float64
	queueDepth       int
	requestsInFlight int
	kvCachePct       float64
	gpuUtilPct       float64

	// Cumulative counters (need delta computation).
	tokensTotal float64
	ttftSum     float64
	ttftCount   float64
}

func (s *metricsScraper) parseVLLMMetrics(body, clusterName, poolName, podName, nodeName string) parsedPodMetrics {
	pm := parsedPodMetrics{}

	// Gauges — try vllm: prefix (newer) then vllm_ prefix (older).
	pm.tps = firstFound(body,
		"vllm:avg_generation_throughput_toks_per_s",
		"vllm_avg_generation_throughput_toks_per_s",
	)
	pm.queueDepth = int(firstFound(body,
		"vllm:num_requests_waiting",
		"vllm_num_requests_waiting",
	))
	pm.requestsInFlight = int(firstFound(body,
		"vllm:num_requests_running",
		"vllm_num_requests_running",
	))
	pm.kvCachePct = firstFound(body,
		"vllm:kv_cache_usage_perc",
		"vllm:gpu_cache_usage_perc",
		"vllm_gpu_cache_usage_perc",
	) * 100 // 0-1 → 0-100

	// GPU utilization — vLLM doesn't natively expose this; try DCGM/nvidia exporters
	// that may be co-located or exposed on the same /metrics endpoint.
	pm.gpuUtilPct = firstFound(body,
		"DCGM_FI_DEV_GPU_UTIL",
		"nvidia_gpu_duty_cycle",
	)

	// Counter / histogram cumulatives.
	pm.tokensTotal = firstFound(body,
		"vllm:generation_tokens_total",
		"vllm_generation_tokens_total",
	)
	pm.ttftSum = firstFound(body,
		"vllm:time_to_first_token_seconds_sum",
		"vllm_time_to_first_token_seconds_sum",
	)
	pm.ttftCount = firstFound(body,
		"vllm:time_to_first_token_seconds_count",
		"vllm_time_to_first_token_seconds_count",
	)

	return pm
}

// updateCounters computes deltas under the lock and updates stored state.
// Returns (tokensDelta, tps, ttftAvgMs).
func (s *metricsScraper) updateCounters(key string, pm parsedPodMetrics) (uint64, float64, float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	c, ok := s.counters[key]
	if !ok {
		c = &podCounters{}
		s.counters[key] = c
	}
	elapsed := now.Sub(c.lastSeen).Seconds()
	c.lastSeen = now

	// Token counter delta — skip on first observation or counter reset.
	var tokensDelta uint64
	if c.prevTokensTotal > 0 && pm.tokensTotal >= c.prevTokensTotal {
		tokensDelta = uint64(pm.tokensTotal - c.prevTokensTotal)
	}
	c.prevTokensTotal = pm.tokensTotal

	// Compute TPS from token delta / elapsed time.
	var tps float64
	if tokensDelta > 0 && elapsed > 0 {
		tps = float64(tokensDelta) / elapsed
	}

	// TTFT histogram delta — compute average over interval.
	var ttftAvg float64
	ttftCountDelta := pm.ttftCount - c.prevTTFTCount
	ttftSumDelta := pm.ttftSum - c.prevTTFTSum
	if c.prevTTFTCount > 0 && ttftCountDelta > 0 && ttftSumDelta > 0 {
		ttftAvg = (ttftSumDelta / ttftCountDelta) * 1000 // seconds → ms
	}
	c.prevTTFTSum = pm.ttftSum
	c.prevTTFTCount = pm.ttftCount

	return tokensDelta, tps, ttftAvg
}

// firstFound returns the value of the first metric name found in the body.
func firstFound(body string, names ...string) float64 {
	for _, name := range names {
		if v, ok := parsePrometheusValue(body, name); ok {
			return v
		}
	}
	return 0
}

// parsePrometheusValue extracts a numeric value for a metric name from
// Prometheus text exposition format. Handles both bare metrics and those
// with labels (e.g., metric_name{label="val"} 123.45).
// The Prometheus format is: metric_name[{labels}] value [timestamp]
// so the value is always the second field (index 1), not the last.
func parsePrometheusValue(body, name string) (float64, bool) {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}

		// Extract metric name (everything before '{' or first space).
		metricName := line
		if idx := strings.IndexByte(line, '{'); idx > 0 {
			metricName = line[:idx]
		} else if idx := strings.IndexByte(line, ' '); idx > 0 {
			metricName = line[:idx]
		}

		if metricName != name {
			continue
		}

		// Value is the second field (after name+labels, before optional timestamp).
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			v, err := strconv.ParseFloat(fields[1], 64)
			if err == nil {
				return v, true
			}
		}
	}
	return 0, false
}

func roundTo(v float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}
