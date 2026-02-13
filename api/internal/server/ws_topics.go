package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/kubenetlabs/ngc/api/internal/inference"
)

// RegisterInferenceTopics adds generators for inference WebSocket topics.
// When a non-nil MetricsProvider is given, epp-decisions and gpu-metrics
// are backed by real ClickHouse queries. scaling-events remains synthetic.
func RegisterInferenceTopics(hub *Hub, provider inference.MetricsProvider) {
	registerEPPDecisions(hub, provider)
	registerGPUMetrics(hub, provider)
	registerScalingEvents(hub)
}

// registerEPPDecisions streams the most recent EPP decision every 1s.
func registerEPPDecisions(hub *Hub, provider inference.MetricsProvider) {
	var (
		mu            sync.Mutex
		lastTimestamp time.Time
	)

	hub.AddGenerator("epp-decisions", 1*time.Second, func() (json.RawMessage, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		decisions, err := provider.GetRecentEPPDecisions(ctx, "", 1)
		if err != nil {
			slog.Debug("ws epp-decisions: query failed", "error", err)
			return json.Marshal([]interface{}{})
		}

		if len(decisions) == 0 {
			return json.Marshal([]interface{}{})
		}

		mu.Lock()
		seen := decisions[0].Timestamp.Equal(lastTimestamp)
		if !seen {
			lastTimestamp = decisions[0].Timestamp
		}
		mu.Unlock()

		if seen {
			// No new decision since last broadcast; send empty array.
			return json.Marshal([]interface{}{})
		}

		return json.Marshal(decisions[0])
	})
}

// registerGPUMetrics streams per-pod GPU metrics every 2s.
func registerGPUMetrics(hub *Hub, provider inference.MetricsProvider) {
	hub.AddGenerator("gpu-metrics", 2*time.Second, func() (json.RawMessage, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		pods, err := provider.GetPodMetrics(ctx, "")
		if err != nil {
			slog.Debug("ws gpu-metrics: query failed", "error", err)
			return json.Marshal([]interface{}{})
		}

		if len(pods) == 0 {
			return json.Marshal([]interface{}{})
		}

		return json.Marshal(pods)
	})
}

// registerScalingEvents emits synthetic scaling events every 15s.
// No backing ClickHouse table exists yet; this will be wired to real
// KEDA/HPA events when that data pipeline is built.
func registerScalingEvents(hub *Hub) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var mu sync.Mutex

	hub.AddGenerator("scaling-events", 15*time.Second, func() (json.RawMessage, error) {
		mu.Lock()
		defer mu.Unlock()

		events := []string{"scale-up", "scale-down", "cooldown"}
		event := map[string]any{
			"timestamp":    time.Now().UTC().Format(time.RFC3339),
			"pool":         "llama3-70b-prod",
			"event":        events[rng.Intn(len(events))],
			"fromReplicas": 4 + rng.Intn(4),
			"toReplicas":   4 + rng.Intn(4),
			"trigger":      fmt.Sprintf("gpu_utilization > %d%%", 70+rng.Intn(20)),
		}
		return json.Marshal(event)
	})
}
