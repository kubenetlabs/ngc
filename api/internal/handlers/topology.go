package handlers

import "net/http"

// TopologyHandler handles topology graph API requests.
type TopologyHandler struct{}

// Full returns the full cluster topology graph.
func (h *TopologyHandler) Full(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// ByGateway returns the topology graph scoped to a specific gateway.
func (h *TopologyHandler) ByGateway(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}
