package handlers

import "net/http"

// XCHandler handles cross-cluster (XC) API requests.
type XCHandler struct{}

// Status returns the cross-cluster connectivity status.
func (h *XCHandler) Status(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Publish creates a new cross-cluster service publication.
func (h *XCHandler) Publish(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// GetPublish returns a specific publication by ID.
func (h *XCHandler) GetPublish(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// DeletePublish removes a cross-cluster service publication.
func (h *XCHandler) DeletePublish(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Metrics returns cross-cluster traffic metrics.
func (h *XCHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}
