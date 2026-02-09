package handlers

import "net/http"

// LogHandler handles log query API requests.
type LogHandler struct{}

// Query executes a log search query.
func (h *LogHandler) Query(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// TopN returns the top N log entries by frequency.
func (h *LogHandler) TopN(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}
