package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/kubenetlabs/ngc/api/internal/inference"
)

// InferenceMetricsHandler handles inference-specific metrics endpoints.
type InferenceMetricsHandler struct {
	Provider inference.MetricsProvider
}

// Summary returns an aggregated inference metrics summary.
func (h *InferenceMetricsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	pool := r.URL.Query().Get("pool")
	summary, err := h.Provider.GetMetricsSummary(r.Context(), pool)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, InferenceMetricsSummaryResponse{
		AvgTTFT:            summary.AvgTTFT,
		P95TTFT:            summary.P95TTFT,
		P99TTFT:            summary.P99TTFT,
		AvgTPS:             summary.AvgTPS,
		TotalTokens:        summary.TotalTokens,
		AvgQueueDepth:      summary.AvgQueueDepth,
		AvgKVCachePct:      summary.AvgKVCachePct,
		PrefixCacheHitRate: summary.PrefixCacheHitRate,
		AvgGPUUtil:         summary.AvgGPUUtil,
	})
}

// ByPool returns inference metrics grouped by pool.
func (h *InferenceMetricsHandler) ByPool(w http.ResponseWriter, r *http.Request) {
	pools, err := h.Provider.ListPools(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make(map[string]InferenceMetricsSummaryResponse, len(pools))
	for _, pool := range pools {
		summary, err := h.Provider.GetMetricsSummary(r.Context(), pool.Name)
		if err != nil {
			slog.Warn("failed to get pool metrics", "pool", pool.Name, "error", err)
			continue
		}
		result[pool.Name] = InferenceMetricsSummaryResponse{
			AvgTTFT:            summary.AvgTTFT,
			P95TTFT:            summary.P95TTFT,
			P99TTFT:            summary.P99TTFT,
			AvgTPS:             summary.AvgTPS,
			TotalTokens:        summary.TotalTokens,
			AvgQueueDepth:      summary.AvgQueueDepth,
			AvgKVCachePct:      summary.AvgKVCachePct,
			PrefixCacheHitRate: summary.PrefixCacheHitRate,
			AvgGPUUtil:         summary.AvgGPUUtil,
		}
	}
	writeJSON(w, http.StatusOK, result)
}

// PodMetrics returns per-pod inference metrics.
func (h *InferenceMetricsHandler) PodMetrics(w http.ResponseWriter, r *http.Request) {
	pool := r.URL.Query().Get("pool")
	if pool == "" {
		writeError(w, http.StatusBadRequest, "pool query parameter is required")
		return
	}
	pods, err := h.Provider.GetPodMetrics(r.Context(), pool)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]PodGPUMetricsResponse, 0, len(pods))
	for _, p := range pods {
		resp = append(resp, PodGPUMetricsResponse{
			PodName:          p.PodName,
			NodeName:         p.NodeName,
			GPUID:            p.GPUID,
			GPUType:          p.GPUType,
			QueueDepth:       p.QueueDepth,
			KVCacheUtilPct:   p.KVCacheUtilPct,
			PrefixCacheState: p.PrefixCacheState,
			GPUUtilPct:       p.GPUUtilPct,
			GPUMemUsedMB:     p.GPUMemUsedMB,
			GPUMemTotalMB:    p.GPUMemTotalMB,
			GPUTemperatureC:  p.GPUTemperatureC,
			RequestsInFlight: p.RequestsInFlight,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

// Cost returns cost estimation for inference workloads.
func (h *InferenceMetricsHandler) Cost(w http.ResponseWriter, r *http.Request) {
	pool := r.URL.Query().Get("pool")
	if pool == "" {
		writeError(w, http.StatusBadRequest, "pool query parameter is required")
		return
	}
	cost, err := h.Provider.GetCostEstimate(r.Context(), pool)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, CostEstimateResponse{
		GPUType:      cost.GPUType,
		ReplicaCount: cost.ReplicaCount,
		HourlyRate:   cost.HourlyRate,
		DailyCost:    cost.DailyCost,
		MonthlyCost:  cost.MonthlyCost,
	})
}

// EPPDecisions returns recent EPP routing decisions for a pool.
func (h *InferenceMetricsHandler) EPPDecisions(w http.ResponseWriter, r *http.Request) {
	pool := r.URL.Query().Get("pool")
	if pool == "" {
		writeError(w, http.StatusBadRequest, "pool query parameter is required")
		return
	}
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	decisions, err := h.Provider.GetRecentEPPDecisions(r.Context(), pool, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]EPPDecisionResponse, 0, len(decisions))
	for _, d := range decisions {
		resp = append(resp, EPPDecisionResponse{
			Timestamp:            formatTime(d.Timestamp),
			RequestID:            d.RequestID,
			SelectedPod:          d.SelectedPod,
			Reason:               d.Reason,
			QueueDepth:           d.QueueDepth,
			KVCachePct:           d.KVCachePct,
			PrefixCacheHit:       d.PrefixCacheHit,
			CandidatesConsidered: d.CandidatesConsidered,
			DecisionLatencyUs:    d.DecisionLatencyUs,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

// TTFTHistogram returns TTFT distribution buckets for a pool.
func (h *InferenceMetricsHandler) TTFTHistogram(w http.ResponseWriter, r *http.Request) {
	pool := chi.URLParam(r, "pool")
	buckets, err := h.Provider.GetTTFTHistogram(r.Context(), pool)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]HistogramBucketResponse, 0, len(buckets))
	for _, b := range buckets {
		resp = append(resp, HistogramBucketResponse{
			RangeStart: b.RangeStart,
			RangeEnd:   b.RangeEnd,
			Count:      b.Count,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

// TPSThroughput returns tokens-per-second timeseries for a pool.
func (h *InferenceMetricsHandler) TPSThroughput(w http.ResponseWriter, r *http.Request) {
	pool := chi.URLParam(r, "pool")
	h.writeTimeseries(w, r, pool, h.Provider.GetTPSThroughput)
}

// QueueDepthSeries returns queue depth timeseries for a pool.
func (h *InferenceMetricsHandler) QueueDepthSeries(w http.ResponseWriter, r *http.Request) {
	pool := chi.URLParam(r, "pool")
	h.writeTimeseries(w, r, pool, h.Provider.GetQueueDepthSeries)
}

// ActiveRequestsSeries returns active in-flight requests timeseries for a pool.
func (h *InferenceMetricsHandler) ActiveRequestsSeries(w http.ResponseWriter, r *http.Request) {
	pool := chi.URLParam(r, "pool")
	h.writeTimeseries(w, r, pool, h.Provider.GetActiveRequestsSeries)
}

// GPUUtilSeries returns GPU utilization timeseries for a pool.
func (h *InferenceMetricsHandler) GPUUtilSeries(w http.ResponseWriter, r *http.Request) {
	pool := chi.URLParam(r, "pool")
	h.writeTimeseries(w, r, pool, h.Provider.GetGPUUtilSeries)
}

// KVCacheSeries returns KV-cache utilization timeseries for a pool.
func (h *InferenceMetricsHandler) KVCacheSeries(w http.ResponseWriter, r *http.Request) {
	pool := chi.URLParam(r, "pool")
	h.writeTimeseries(w, r, pool, h.Provider.GetKVCacheSeries)
}

func (h *InferenceMetricsHandler) writeTimeseries(
	w http.ResponseWriter,
	r *http.Request,
	pool string,
	fn func(ctx context.Context, pool string) ([]inference.TimeseriesPoint, error),
) {
	points, err := fn(r.Context(), pool)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]TimeseriesPointResponse, 0, len(points))
	for _, p := range points {
		resp = append(resp, TimeseriesPointResponse{
			Timestamp: formatTime(p.Timestamp),
			Value:     p.Value,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}
