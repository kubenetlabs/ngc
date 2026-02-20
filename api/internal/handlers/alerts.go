package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kubenetlabs/ngc/api/internal/alerting"
	"github.com/kubenetlabs/ngc/api/internal/database"
)

// AlertHandler handles alert rule API requests.
type AlertHandler struct {
	Store     database.Store
	Evaluator *alerting.Evaluator
}

// List returns all alert rules.
func (h *AlertHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.Store == nil {
		writeError(w, http.StatusServiceUnavailable, "alert store not configured")
		return
	}

	rules, err := h.Store.ListAlertRules(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if rules == nil {
		rules = []database.AlertRule{}
	}

	writeJSON(w, http.StatusOK, rules)
}

// Get returns a single alert rule by ID.
func (h *AlertHandler) Get(w http.ResponseWriter, r *http.Request) {
	if h.Store == nil {
		writeError(w, http.StatusServiceUnavailable, "alert store not configured")
		return
	}

	id := chi.URLParam(r, "id")
	rule, err := h.Store.GetAlertRule(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if rule == nil {
		writeError(w, http.StatusNotFound, "alert rule not found")
		return
	}

	writeJSON(w, http.StatusOK, rule)
}

// createAlertRuleRequest is the JSON body for creating an alert rule.
type createAlertRuleRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Resource    string   `json:"resource"`
	Metric      string   `json:"metric"`
	Operator    string   `json:"operator"`
	Threshold   *float64 `json:"threshold"`
	Severity    string   `json:"severity"`
	Enabled     *bool    `json:"enabled"`
}

// Create creates a new alert rule.
func (h *AlertHandler) Create(w http.ResponseWriter, r *http.Request) {
	if h.Store == nil {
		writeError(w, http.StatusServiceUnavailable, "alert store not configured")
		return
	}

	var req createAlertRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Validate required fields
	if req.Name == "" || req.Resource == "" || req.Metric == "" || req.Operator == "" || req.Threshold == nil || req.Severity == "" {
		writeError(w, http.StatusBadRequest, "name, resource, metric, operator, threshold, and severity are required")
		return
	}

	// Validate operator
	if req.Operator != "gt" && req.Operator != "lt" && req.Operator != "eq" {
		writeError(w, http.StatusBadRequest, "operator must be one of: gt, lt, eq")
		return
	}

	// Validate severity
	if req.Severity != "critical" && req.Severity != "warning" && req.Severity != "info" {
		writeError(w, http.StatusBadRequest, "severity must be one of: critical, warning, info")
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	now := time.Now().UTC()
	rule := database.AlertRule{
		ID:          uuid.NewString(),
		Name:        req.Name,
		Description: req.Description,
		Resource:    req.Resource,
		Metric:      req.Metric,
		Operator:    req.Operator,
		Threshold:   *req.Threshold,
		Severity:    req.Severity,
		Enabled:     enabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.Store.CreateAlertRule(r.Context(), rule); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	auditLog(h.Store, r.Context(), "create", "AlertRule", rule.Name, "", nil, rule)
	writeJSON(w, http.StatusCreated, rule)
}

// updateAlertRuleRequest is the JSON body for updating an alert rule.
type updateAlertRuleRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Resource    string   `json:"resource"`
	Metric      string   `json:"metric"`
	Operator    string   `json:"operator"`
	Threshold   *float64 `json:"threshold"`
	Severity    string   `json:"severity"`
	Enabled     *bool    `json:"enabled"`
}

// Update modifies an existing alert rule.
func (h *AlertHandler) Update(w http.ResponseWriter, r *http.Request) {
	if h.Store == nil {
		writeError(w, http.StatusServiceUnavailable, "alert store not configured")
		return
	}

	id := chi.URLParam(r, "id")
	existing, err := h.Store.GetAlertRule(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "alert rule not found")
		return
	}

	var req updateAlertRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Validate operator if provided
	if req.Operator != "" && req.Operator != "gt" && req.Operator != "lt" && req.Operator != "eq" {
		writeError(w, http.StatusBadRequest, "operator must be one of: gt, lt, eq")
		return
	}

	// Validate severity if provided
	if req.Severity != "" && req.Severity != "critical" && req.Severity != "warning" && req.Severity != "info" {
		writeError(w, http.StatusBadRequest, "severity must be one of: critical, warning, info")
		return
	}

	// Apply updates
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Resource != "" {
		existing.Resource = req.Resource
	}
	if req.Metric != "" {
		existing.Metric = req.Metric
	}
	if req.Operator != "" {
		existing.Operator = req.Operator
	}
	if req.Threshold != nil {
		existing.Threshold = *req.Threshold
	}
	if req.Severity != "" {
		existing.Severity = req.Severity
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}

	existing.UpdatedAt = time.Now().UTC()

	if err := h.Store.UpdateAlertRule(r.Context(), *existing); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	auditLog(h.Store, r.Context(), "update", "AlertRule", existing.Name, "", nil, existing)
	writeJSON(w, http.StatusOK, existing)
}

// Delete removes an alert rule.
func (h *AlertHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if h.Store == nil {
		writeError(w, http.StatusServiceUnavailable, "alert store not configured")
		return
	}

	id := chi.URLParam(r, "id")
	existing, err := h.Store.GetAlertRule(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "alert rule not found")
		return
	}

	if err := h.Store.DeleteAlertRule(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	auditLog(h.Store, r.Context(), "delete", "AlertRule", existing.Name, "", existing, nil)
	writeJSON(w, http.StatusOK, map[string]string{"message": "alert rule deleted", "id": id})
}

// Toggle flips the enabled state of an alert rule.
func (h *AlertHandler) Toggle(w http.ResponseWriter, r *http.Request) {
	if h.Store == nil {
		writeError(w, http.StatusServiceUnavailable, "alert store not configured")
		return
	}

	id := chi.URLParam(r, "id")
	existing, err := h.Store.GetAlertRule(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "alert rule not found")
		return
	}

	existing.Enabled = !existing.Enabled
	existing.UpdatedAt = time.Now().UTC()

	if err := h.Store.UpdateAlertRule(r.Context(), *existing); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

// Firing returns all currently firing alerts from the evaluator.
func (h *AlertHandler) Firing(w http.ResponseWriter, r *http.Request) {
	if h.Evaluator == nil {
		writeJSON(w, http.StatusOK, []alerting.FiringAlert{})
		return
	}

	firing := h.Evaluator.GetFiring()
	if firing == nil {
		firing = []alerting.FiringAlert{}
	}

	writeJSON(w, http.StatusOK, firing)
}
