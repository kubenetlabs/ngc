package handlers

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
)

// CertificateHandler handles TLS certificate API requests.
type CertificateHandler struct{}

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

// CreateCertificateRequest is the request body for creating a TLS certificate.
type CreateCertificateRequest struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Cert      string `json:"cert"` // PEM-encoded certificate
	Key       string `json:"key"`  // PEM-encoded private key
}

// Create creates a new TLS secret (certificate).
func (h *CertificateHandler) Create(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	var req CreateCertificateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.Name == "" || req.Namespace == "" || req.Cert == "" || req.Key == "" {
		writeError(w, http.StatusBadRequest, "name, namespace, cert, and key are required")
		return
	}

	// Validate PEM certificate
	block, _ := pem.Decode([]byte(req.Cert))
	if block == nil {
		writeError(w, http.StatusBadRequest, "invalid PEM certificate data")
		return
	}
	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid certificate: %v", err))
		return
	}

	// Validate PEM key
	keyBlock, _ := pem.Decode([]byte(req.Key))
	if keyBlock == nil {
		writeError(w, http.StatusBadRequest, "invalid PEM key data")
		return
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": []byte(req.Cert),
			"tls.key": []byte(req.Key),
		},
	}

	if err := k8s.CreateSecret(r.Context(), secret); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("creating TLS secret: %v", err))
		return
	}

	resp, ok := parseTLSSecret(secret)
	if !ok {
		writeJSON(w, http.StatusCreated, map[string]string{"name": req.Name, "namespace": req.Namespace, "status": "created"})
		return
	}
	writeJSON(w, http.StatusCreated, resp)
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
