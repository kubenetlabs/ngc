package handlers

import "net/http"

// PolicyHandler handles policy API requests (rate-limit, auth, retry, etc.).
type PolicyHandler struct{}

// List returns all policies of the given type.
func (h *PolicyHandler) List(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Get returns a single policy by name.
func (h *PolicyHandler) Get(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Create creates a new policy.
func (h *PolicyHandler) Create(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Update modifies an existing policy.
func (h *PolicyHandler) Update(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Delete removes a policy.
func (h *PolicyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Conflicts returns detected policy conflicts.
func (h *PolicyHandler) Conflicts(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}
