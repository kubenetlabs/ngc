package server

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RegisterInferenceTopics adds mock generators for inference WebSocket topics.
func RegisterInferenceTopics(hub *Hub) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// EPP Decisions: emitted every 1s
	hub.AddGenerator("epp-decisions", 1*time.Second, func() (json.RawMessage, error) {
		strategies := []string{"least_queue", "kv_cache", "prefix_affinity", "composite"}
		pools := []string{"llama3-70b-prod", "mixtral-8x7b-staging", "codellama-34b-prod"}
		pool := pools[rng.Intn(len(pools))]
		decision := map[string]any{
			"timestamp":            time.Now().UTC().Format(time.RFC3339),
			"requestId":            fmt.Sprintf("req-%04d", rng.Intn(10000)),
			"selectedPod":          fmt.Sprintf("%s-pod-%d", pool, rng.Intn(6)),
			"pool":                 pool,
			"reason":               strategies[rng.Intn(len(strategies))],
			"queueDepth":           rng.Intn(12),
			"kvCachePct":           math.Round((40+rng.Float64()*40)*100) / 100,
			"prefixCacheHit":       rng.Float64() > 0.6,
			"candidatesConsidered": 3 + rng.Intn(4),
			"decisionLatencyUs":    80 + rng.Intn(200),
		}
		return json.Marshal(decision)
	})

	// GPU Metrics: emitted every 2s
	hub.AddGenerator("gpu-metrics", 2*time.Second, func() (json.RawMessage, error) {
		pods := make([]map[string]any, 6)
		for i := range pods {
			pods[i] = map[string]any{
				"podName":          fmt.Sprintf("llama3-70b-prod-pod-%d", i),
				"gpuUtilPct":       math.Round((40+rng.Float64()*55)*100) / 100,
				"gpuMemUsedMb":     40000 + rng.Intn(30000),
				"gpuMemTotalMb":    81920,
				"kvCacheUtilPct":   math.Round((30+rng.Float64()*50)*100) / 100,
				"queueDepth":       rng.Intn(15),
				"requestsInFlight": rng.Intn(8),
				"gpuTemperatureC":  55 + rng.Intn(20),
			}
		}
		return json.Marshal(pods)
	})

	// Scaling Events: emitted every 15s
	hub.AddGenerator("scaling-events", 15*time.Second, func() (json.RawMessage, error) {
		events := []string{"scale-up", "scale-down", "cooldown"}
		event := map[string]any{
			"timestamp":    time.Now().UTC().Format(time.RFC3339),
			"pool":         "llama3-70b-prod",
			"event":        events[rng.Intn(len(events))],
			"fromReplicas": 4 + rng.Intn(4),
			"toReplicas":   4 + rng.Intn(4),
			"trigger":      "gpu_utilization > 85%",
		}
		return json.Marshal(event)
	})
}
