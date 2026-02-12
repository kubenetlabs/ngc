package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	ch "github.com/kubenetlabs/ngc/api/internal/clickhouse"
)

// LogHandler handles log query API requests.
type LogHandler struct {
	CH *ch.Client
}

// AccessLogEntry is the API response for a single access log row.
type AccessLogEntry struct {
	Timestamp       string  `json:"timestamp"`
	Method          string  `json:"method"`
	Path            string  `json:"path"`
	StatusCode      int     `json:"statusCode"`
	Latency         float64 `json:"latency"`
	UpstreamService string  `json:"upstreamService"`
	Namespace       string  `json:"namespace"`
	Hostname        string  `json:"hostname"`
}

// TopNLogEntry is the API response for a top-N aggregation.
type TopNLogEntry struct {
	Key        string  `json:"key"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

// LogQueryRequest is the request body for log queries.
type LogQueryRequest struct {
	Namespace string `json:"namespace,omitempty"`
	Hostname  string `json:"hostname,omitempty"`
	Search    string `json:"search,omitempty"`
	Limit     int    `json:"limit"`
}

// Query executes a log search query.
func (h *LogHandler) Query(w http.ResponseWriter, r *http.Request) {
	if h.CH == nil {
		writeError(w, http.StatusServiceUnavailable, "clickhouse not configured")
		return
	}

	var req LogQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Limit <= 0 || req.Limit > 500 {
		req.Limit = 50
	}

	query := `SELECT timestamp, method, path, status, latency_ms, upstream_name, namespace, route FROM ngf_access_logs WHERE 1=1`
	args := []interface{}{}

	if req.Namespace != "" {
		query += ` AND namespace = ?`
		args = append(args, req.Namespace)
	}
	if req.Hostname != "" {
		query += ` AND route LIKE ?`
		args = append(args, "%"+req.Hostname+"%")
	}
	if req.Search != "" {
		query += ` AND path LIKE ?`
		args = append(args, "%"+req.Search+"%")
	}

	query += ` ORDER BY timestamp DESC LIMIT ?`
	args = append(args, req.Limit)

	rows, err := h.CH.Conn().Query(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed: "+err.Error())
		return
	}
	defer rows.Close()

	entries := make([]AccessLogEntry, 0)
	for rows.Next() {
		var e AccessLogEntry
		var status uint16
		var latency float64
		if err := rows.Scan(&e.Timestamp, &e.Method, &e.Path, &status, &latency, &e.UpstreamService, &e.Namespace, &e.Hostname); err != nil {
			writeError(w, http.StatusInternalServerError, "scan failed: "+err.Error())
			return
		}
		e.StatusCode = int(status)
		e.Latency = latency
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "rows error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, entries)
}

// TopN returns the top N log entries by frequency.
func (h *LogHandler) TopN(w http.ResponseWriter, r *http.Request) {
	if h.CH == nil {
		writeError(w, http.StatusServiceUnavailable, "clickhouse not configured")
		return
	}

	field := r.URL.Query().Get("field")
	if field == "" {
		field = "path"
	}
	// Whitelist allowed fields to prevent SQL injection
	allowedFields := map[string]bool{"path": true, "method": true, "upstream_name": true, "namespace": true, "route": true}
	if !allowedFields[field] {
		writeError(w, http.StatusBadRequest, "invalid field, must be one of: path, method, upstream_name, namespace, route")
		return
	}

	n := 10
	if nStr := r.URL.Query().Get("n"); nStr != "" {
		if v, err := strconv.Atoi(nStr); err == nil && v > 0 && v <= 100 {
			n = v
		}
	}

	query := `SELECT ` + field + ` AS key, count() AS cnt, cnt * 100.0 / sum(cnt) OVER () AS pct FROM ngf_access_logs WHERE timestamp >= now() - INTERVAL 1 HOUR GROUP BY key ORDER BY cnt DESC LIMIT ?`

	rows, err := h.CH.Conn().Query(r.Context(), query, n)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed: "+err.Error())
		return
	}
	defer rows.Close()

	entries := make([]TopNLogEntry, 0)
	for rows.Next() {
		var e TopNLogEntry
		if err := rows.Scan(&e.Key, &e.Count, &e.Percentage); err != nil {
			writeError(w, http.StatusInternalServerError, "scan failed: "+err.Error())
			return
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "rows error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, entries)
}
