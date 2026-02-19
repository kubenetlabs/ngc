package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/inference"
)

// RegisterInferenceTopics adds generators for inference WebSocket topics.
// When a non-nil MetricsProvider is given, epp-decisions and gpu-metrics
// are backed by real ClickHouse queries. When a non-nil cluster.Provider
// is given, scaling-events are sourced from real HPA status; otherwise
// they fall back to synthetic data.
func RegisterInferenceTopics(hub *Hub, provider inference.MetricsProvider, clusters cluster.Provider) {
	registerEPPDecisions(hub, provider)
	registerGPUMetrics(hub, provider)
	registerScalingEvents(hub, clusters)
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
			return nil, nil
		}

		if len(decisions) == 0 {
			return nil, nil
		}

		mu.Lock()
		seen := decisions[0].Timestamp.Equal(lastTimestamp)
		if !seen {
			lastTimestamp = decisions[0].Timestamp
		}
		mu.Unlock()

		if seen {
			// No new decision since last broadcast; skip.
			return nil, nil
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

// hpaState tracks the last-seen replica counts for an HPA so we can detect changes.
type hpaState struct {
	currentReplicas int32
	desiredReplicas int32
}

var hpaGVR = schema.GroupVersionResource{
	Group:    "autoscaling",
	Version:  "v2",
	Resource: "horizontalpodautoscalers",
}

// registerScalingEvents polls HPAs every 10s and emits scaling events when
// replica counts change. Falls back to synthetic events when clusters is nil.
func registerScalingEvents(hub *Hub, clusters cluster.Provider) {
	if clusters == nil {
		registerSyntheticScalingEvents(hub)
		return
	}

	var (
		mu       sync.Mutex
		previous = make(map[string]hpaState)
	)

	hub.AddGenerator("scaling-events", 10*time.Second, func() (json.RawMessage, error) {
		kc, err := clusters.Default()
		if err != nil {
			slog.Debug("ws scaling-events: no default cluster", "error", err)
			return nil, nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		list, err := kc.DynamicClient().Resource(hpaGVR).Namespace("").List(ctx, metav1.ListOptions{})
		if err != nil {
			slog.Debug("ws scaling-events: HPA list failed", "error", err)
			return nil, nil
		}

		mu.Lock()
		defer mu.Unlock()

		var events []map[string]any

		for _, item := range list.Items {
			name := item.GetNamespace() + "/" + item.GetName()

			status, ok := item.Object["status"].(map[string]interface{})
			if !ok {
				continue
			}

			currentReplicas := int32(0)
			desiredReplicas := int32(0)
			if v, ok := status["currentReplicas"].(int64); ok {
				currentReplicas = int32(v)
			}
			if v, ok := status["desiredReplicas"].(int64); ok {
				desiredReplicas = int32(v)
			}

			prev, seen := previous[name]
			previous[name] = hpaState{
				currentReplicas: currentReplicas,
				desiredReplicas: desiredReplicas,
			}

			if !seen {
				continue // First observation — no delta to report.
			}

			if prev.currentReplicas == currentReplicas && prev.desiredReplicas == desiredReplicas {
				continue // No change.
			}

			eventType := "scale-up"
			if desiredReplicas < prev.desiredReplicas {
				eventType = "scale-down"
			} else if desiredReplicas == currentReplicas && prev.desiredReplicas != prev.currentReplicas {
				eventType = "cooldown"
			}

			events = append(events, map[string]any{
				"timestamp":    time.Now().UTC().Format(time.RFC3339),
				"pool":         name,
				"event":        eventType,
				"fromReplicas": prev.currentReplicas,
				"toReplicas":   desiredReplicas,
				"trigger":      "hpa",
				"source":       "live",
			})
		}

		if len(events) == 0 {
			return nil, nil
		}

		// Return the first event (or batch them if needed later).
		return json.Marshal(events[0])
	})
}

// registerSyntheticScalingEvents emits random scaling events for dev/demo mode.
func registerSyntheticScalingEvents(hub *Hub) {
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
			"source":       "synthetic",
		}
		return json.Marshal(event)
	})
}
