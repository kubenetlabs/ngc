package handlers

import "net/http"

// AuditHandler handles audit log API requests.
type AuditHandler struct{}

// List returns audit log entries.
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Diff returns the diff for a specific audit entry.
func (h *AuditHandler) Diff(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}
