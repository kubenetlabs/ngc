package handlers

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/database"
)

// CertificateHandler handles TLS certificate API requests.
type CertificateHandler struct {
	Store database.Store
}

// CertificateResponse is the API response for a TLS certificate.
type CertificateResponse struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Domains   []string `json:"domains"`
	Issuer    string   `json:"issuer"`
	NotBefore string   `json:"notBefore"`
	NotAfter  string   `json:"notAfter"`
	DaysLeft  int      `json:"daysLeft"`
}

// List returns all TLS certificates (secrets of type kubernetes.io/tls).
func (h *CertificateHandler) List(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	namespace := r.URL.Query().Get("namespace")
	secrets, err := k8s.ListSecrets(r.Context(), namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("listing secrets: %v", err))
		return
	}

	certs := make([]CertificateResponse, 0)
	for i := range secrets {
		if secrets[i].Type != corev1.SecretTypeTLS {
			continue
		}
		if resp, ok := parseTLSSecret(&secrets[i]); ok {
			certs = append(certs, resp)
		}
	}
	writeJSON(w, http.StatusOK, certs)
}

// Get returns a single certificate by name.
func (h *CertificateHandler) Get(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	name := chi.URLParam(r, "name")
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	secret, err := k8s.GetSecret(r.Context(), namespace, name)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("certificate not found: %v", err))
		return
	}

	if secret.Type != corev1.SecretTypeTLS {
		writeError(w, http.StatusNotFound, "not a TLS secret")
		return
	}

	resp, ok := parseTLSSecret(secret)
	if !ok {
		writeError(w, http.StatusInternalServerError, "failed to parse certificate")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Create creates a new TLS secret (certificate).
func (h *CertificateHandler) Create(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Delete removes a certificate (TLS secret).
func (h *CertificateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	name := chi.URLParam(r, "name")
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	if err := k8s.DeleteSecret(r.Context(), namespace, name); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("deleting certificate: %v", err))
		return
	}
	auditLog(h.Store, r.Context(), "delete", "Certificate", name, namespace, map[string]string{"name": name, "namespace": namespace}, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Expiring returns certificates that are expiring within 30 days.
func (h *CertificateHandler) Expiring(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	secrets, err := k8s.ListSecrets(r.Context(), "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("listing secrets: %v", err))
		return
	}

	expiring := make([]CertificateResponse, 0)
	for i := range secrets {
		if secrets[i].Type != corev1.SecretTypeTLS {
			continue
		}
		resp, ok := parseTLSSecret(&secrets[i])
		if !ok {
			continue
		}
		if resp.DaysLeft <= 30 {
			expiring = append(expiring, resp)
		}
	}
	writeJSON(w, http.StatusOK, expiring)
}

func parseTLSSecret(secret *corev1.Secret) (CertificateResponse, bool) {
	resp := CertificateResponse{
		Name:      secret.Name,
		Namespace: secret.Namespace,
		Domains:   []string{},
	}

	certData, ok := secret.Data["tls.crt"]
	if !ok {
		return resp, false
	}

	block, _ := pem.Decode(certData)
	if block == nil {
		// If we can't parse PEM, return basic info without cert details
		return resp, true
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return resp, true
	}

	resp.Issuer = cert.Issuer.CommonName
	resp.NotBefore = cert.NotBefore.UTC().Format("2006-01-02T15:04:05Z")
	resp.NotAfter = cert.NotAfter.UTC().Format("2006-01-02T15:04:05Z")
	resp.DaysLeft = int(math.Max(0, cert.NotAfter.Sub(time.Now()).Hours()/24))

	// Collect domains from CN and SANs
	domains := make(map[string]bool)
	if cert.Subject.CommonName != "" {
		domains[cert.Subject.CommonName] = true
	}
	for _, dns := range cert.DNSNames {
		domains[dns] = true
	}
	for d := range domains {
		resp.Domains = append(resp.Domains, d)
	}

	return resp, true
}
