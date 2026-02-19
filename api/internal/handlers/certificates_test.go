package handlers

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

// generateTestCert generates a self-signed PEM certificate for testing.
func generateTestCert(t *testing.T, dnsName string, notAfter time.Time) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: dnsName},
		DNSNames:     []string{dnsName},
		NotBefore:    time.Now().Add(-24 * time.Hour),
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
}

func certContextMiddleware(secrets []corev1.Secret) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scheme := setupScheme(nil) // reuse from gateways_test.go
			if scheme == nil {
				// Fallback: create minimal scheme
				scheme = runtime.NewScheme()
				_ = corev1.AddToScheme(scheme)
			}
			builder := fake.NewClientBuilder().WithScheme(scheme)
			for i := range secrets {
				builder = builder.WithObjects(&secrets[i])
			}
			fakeClient := builder.Build()
			k8s := kubernetes.NewForTest(fakeClient)
			ctx := cluster.WithClient(r.Context(), k8s)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func TestCertificateHandler_List_HappyPath(t *testing.T) {
	certPEM := generateTestCert(t, "example.com", time.Now().Add(90*24*time.Hour))

	secrets := []corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "tls-cert-1", Namespace: "default"},
			Type:       corev1.SecretTypeTLS,
			Data:       map[string][]byte{"tls.crt": certPEM, "tls.key": []byte("fake-key")},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "tls-cert-2", Namespace: "default"},
			Type:       corev1.SecretTypeTLS,
			Data:       map[string][]byte{"tls.crt": certPEM, "tls.key": []byte("fake-key")},
		},
		{
			// Opaque secret — should be filtered out
			ObjectMeta: metav1.ObjectMeta{Name: "opaque-secret", Namespace: "default"},
			Type:       corev1.SecretTypeOpaque,
			Data:       map[string][]byte{"data": []byte("not-a-cert")},
		},
	}

	handler := &CertificateHandler{}
	r := chi.NewRouter()
	r.Use(certContextMiddleware(secrets))
	r.Get("/certificates", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/certificates", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []CertificateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp) != 2 {
		t.Fatalf("expected 2 TLS certs (not opaque), got %d", len(resp))
	}
}

func TestCertificateHandler_List_NoClusterContext(t *testing.T) {
	handler := &CertificateHandler{}

	r := chi.NewRouter()
	r.Get("/certificates", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/certificates", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCertificateHandler_Get_HappyPath(t *testing.T) {
	certPEM := generateTestCert(t, "example.com", time.Now().Add(90*24*time.Hour))

	secrets := []corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "tls-cert-1", Namespace: "default"},
			Type:       corev1.SecretTypeTLS,
			Data:       map[string][]byte{"tls.crt": certPEM, "tls.key": []byte("fake-key")},
		},
	}

	handler := &CertificateHandler{}
	r := chi.NewRouter()
	r.Use(certContextMiddleware(secrets))
	r.Get("/certificates/{name}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/certificates/tls-cert-1?namespace=default", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp CertificateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	domainFound := false
	for _, d := range resp.Domains {
		if d == "example.com" {
			domainFound = true
		}
	}
	if !domainFound {
		t.Errorf("expected domains to contain 'example.com', got %v", resp.Domains)
	}
}

func TestCertificateHandler_Delete_HappyPath(t *testing.T) {
	certPEM := generateTestCert(t, "example.com", time.Now().Add(90*24*time.Hour))

	secrets := []corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "tls-cert-1", Namespace: "default"},
			Type:       corev1.SecretTypeTLS,
			Data:       map[string][]byte{"tls.crt": certPEM, "tls.key": []byte("fake-key")},
		},
	}

	handler := &CertificateHandler{}
	r := chi.NewRouter()
	r.Use(certContextMiddleware(secrets))
	r.Delete("/certificates/{name}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/certificates/tls-cert-1?namespace=default", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCertificateHandler_Expiring_HappyPath(t *testing.T) {
	// One cert expiring in 5 days (should be returned)
	expiringCert := generateTestCert(t, "expiring.example.com", time.Now().Add(5*24*time.Hour))
	// One cert valid for 90 days (should not be returned)
	validCert := generateTestCert(t, "valid.example.com", time.Now().Add(90*24*time.Hour))

	secrets := []corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "expiring-cert", Namespace: "default"},
			Type:       corev1.SecretTypeTLS,
			Data:       map[string][]byte{"tls.crt": expiringCert, "tls.key": []byte("fake-key")},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "valid-cert", Namespace: "default"},
			Type:       corev1.SecretTypeTLS,
			Data:       map[string][]byte{"tls.crt": validCert, "tls.key": []byte("fake-key")},
		},
	}

	handler := &CertificateHandler{}
	r := chi.NewRouter()
	r.Use(certContextMiddleware(secrets))
	r.Get("/certificates/expiring", handler.Expiring)

	req := httptest.NewRequest(http.MethodGet, "/certificates/expiring", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []CertificateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp) != 1 {
		t.Fatalf("expected 1 expiring cert, got %d", len(resp))
	}
	if resp[0].Name != "expiring-cert" {
		t.Errorf("expected expiring cert name, got %s", resp[0].Name)
	}
}

func TestCertificateHandler_Create_Returns501(t *testing.T) {
	handler := &CertificateHandler{}

	r := chi.NewRouter()
	r.Post("/certificates", handler.Create)

	req := httptest.NewRequest(http.MethodPost, "/certificates", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d: %s", w.Code, w.Body.String())
	}
}
