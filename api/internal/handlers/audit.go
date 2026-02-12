package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kubenetlabs/ngc/api/internal/database"
)

// AuditHandler handles audit log API requests.
type AuditHandler struct {
	Store database.Store
}

// List returns paginated audit log entries.
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.Store == nil {
		writeError(w, http.StatusServiceUnavailable, "audit store not configured")
		return
	}

	opts := database.AuditListOptions{
		Resource:  r.URL.Query().Get("resource"),
		Action:    r.URL.Query().Get("action"),
		User:      r.URL.Query().Get("user"),
		Namespace: r.URL.Query().Get("namespace"),
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil {
			opts.Limit = v
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil {
			opts.Offset = v
		}
	}
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			opts.Since = &t
		}
	}

	entries, total, err := h.Store.ListAuditEntries(r.Context(), opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if entries == nil {
		entries = []database.AuditEntry{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"entries": entries,
		"total":   total,
	})
}

// Diff returns the before/after diff for a specific audit entry.
func (h *AuditHandler) Diff(w http.ResponseWriter, r *http.Request) {
	if h.Store == nil {
		writeError(w, http.StatusServiceUnavailable, "audit store not configured")
		return
	}

	id := chi.URLParam(r, "id")
	entry, err := h.Store.GetAuditEntry(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if entry == nil {
		writeError(w, http.StatusNotFound, "audit entry not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":         entry.ID,
		"action":     entry.Action,
		"resource":   entry.Resource,
		"name":       entry.Name,
		"namespace":  entry.Namespace,
		"beforeJson": entry.BeforeJSON,
		"afterJson":  entry.AfterJSON,
	})
}
