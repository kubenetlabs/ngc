package handlers

import "net/http"

// CertificateHandler handles TLS certificate API requests.
type CertificateHandler struct{}

// List returns all certificates.
func (h *CertificateHandler) List(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Get returns a single certificate by name.
func (h *CertificateHandler) Get(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Create creates a new certificate.
func (h *CertificateHandler) Create(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Delete removes a certificate.
func (h *CertificateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Expiring returns certificates that are expiring soon.
func (h *CertificateHandler) Expiring(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}
