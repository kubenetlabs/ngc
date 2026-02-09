package handlers

import "net/http"

// InferenceDiagHandler handles inference diagnostics endpoints.
type InferenceDiagHandler struct{}

// SlowInference returns information about slow inference requests.
func (h *InferenceDiagHandler) SlowInference(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Replay replays a recorded inference request for debugging.
func (h *InferenceDiagHandler) Replay(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Benchmark runs a benchmark against an inference pool.
func (h *InferenceDiagHandler) Benchmark(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}
