package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const testSecret = "test-secret-key-for-unit-tests"

// buildJWT creates an HS256-signed JWT for testing.
func buildJWT(claims jwtClaims, secret string) string {
	header := `{"alg":"HS256","typ":"JWT"}`
	headerEnc := base64.RawURLEncoding.EncodeToString([]byte(header))

	claimsJSON, _ := json.Marshal(claims)
	claimsEnc := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signingInput := headerEnc + "." + claimsEnc
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + sig
}

func TestValidateJWT_ValidToken(t *testing.T) {
	claims := jwtClaims{
		Sub:   "user-123",
		Email: "alice@example.com",
		Role:  "Admin",
		Iss:   "ngf-console",
		Exp:   time.Now().Add(1 * time.Hour).Unix(),
		Iat:   time.Now().Unix(),
	}
	token := buildJWT(claims, testSecret)

	user, err := validateJWT(token, testSecret, "ngf-console")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Subject != "user-123" {
		t.Errorf("Subject = %q, want %q", user.Subject, "user-123")
	}
	if user.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "alice@example.com")
	}
	if user.Role != "Admin" {
		t.Errorf("Role = %q, want %q", user.Role, "Admin")
	}
}

func TestValidateJWT_ExpiredToken(t *testing.T) {
	claims := jwtClaims{
		Sub: "user-123",
		Exp: time.Now().Add(-1 * time.Hour).Unix(), // expired
		Iat: time.Now().Add(-2 * time.Hour).Unix(),
	}
	token := buildJWT(claims, testSecret)

	_, err := validateJWT(token, testSecret, "")
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestValidateJWT_BadSignature(t *testing.T) {
	claims := jwtClaims{
		Sub: "user-123",
		Exp: time.Now().Add(1 * time.Hour).Unix(),
	}
	token := buildJWT(claims, testSecret)

	_, err := validateJWT(token, "wrong-secret", "")
	if err == nil {
		t.Fatal("expected error for bad signature, got nil")
	}
}

func TestValidateJWT_MalformedToken(t *testing.T) {
	_, err := validateJWT("not-a-jwt", testSecret, "")
	if err == nil {
		t.Fatal("expected error for malformed token, got nil")
	}
}

func TestValidateJWT_WrongIssuer(t *testing.T) {
	claims := jwtClaims{
		Sub: "user-123",
		Iss: "other-issuer",
		Exp: time.Now().Add(1 * time.Hour).Unix(),
	}
	token := buildJWT(claims, testSecret)

	_, err := validateJWT(token, testSecret, "ngf-console")
	if err == nil {
		t.Fatal("expected error for wrong issuer, got nil")
	}
}

func TestValidateJWT_DefaultRoleViewer(t *testing.T) {
	claims := jwtClaims{
		Sub: "user-123",
		Exp: time.Now().Add(1 * time.Hour).Unix(),
		// Role is empty — should default to Viewer.
	}
	token := buildJWT(claims, testSecret)

	user, err := validateJWT(token, testSecret, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Role != "Viewer" {
		t.Errorf("Role = %q, want %q", user.Role, "Viewer")
	}
}

func TestAuthMiddleware_DisabledPassesThrough(t *testing.T) {
	cfg := AuthConfig{Enabled: false}
	handler := AuthMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthMiddleware_RejectsMissingToken(t *testing.T) {
	cfg := AuthConfig{Enabled: true, JWTSecret: testSecret}
	handler := AuthMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_AcceptsValidToken(t *testing.T) {
	cfg := AuthConfig{Enabled: true, JWTSecret: testSecret}

	var capturedUser *UserInfo
	handler := AuthMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	claims := jwtClaims{
		Sub:  "user-456",
		Role: "Operator",
		Exp:  time.Now().Add(1 * time.Hour).Unix(),
	}
	token := buildJWT(claims, testSecret)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if capturedUser == nil {
		t.Fatal("expected UserInfo in context")
	}
	if capturedUser.Subject != "user-456" {
		t.Errorf("Subject = %q, want %q", capturedUser.Subject, "user-456")
	}
	if capturedUser.Role != "Operator" {
		t.Errorf("Role = %q, want %q", capturedUser.Role, "Operator")
	}
}

func TestHasRole_Hierarchy(t *testing.T) {
	tests := []struct {
		userRole     string
		requiredRole string
		want         bool
	}{
		{"Admin", "Admin", true},
		{"Admin", "Operator", true},
		{"Admin", "Viewer", true},
		{"Operator", "Admin", false},
		{"Operator", "Operator", true},
		{"Operator", "Viewer", true},
		{"Viewer", "Admin", false},
		{"Viewer", "Operator", false},
		{"Viewer", "Viewer", true},
		{"Unknown", "Viewer", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_needs_%s", tt.userRole, tt.requiredRole), func(t *testing.T) {
			got := hasRole(tt.userRole, tt.requiredRole)
			if got != tt.want {
				t.Errorf("hasRole(%q, %q) = %v, want %v", tt.userRole, tt.requiredRole, got, tt.want)
			}
		})
	}
}

func TestRBACMiddleware_AllowsMatchingRole(t *testing.T) {
	cfg := AuthConfig{Enabled: true, JWTSecret: testSecret}

	inner := RBACMiddleware("Operator")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler := AuthMiddleware(cfg)(inner)

	claims := jwtClaims{
		Sub:  "admin-user",
		Role: "Admin",
		Exp:  time.Now().Add(1 * time.Hour).Unix(),
	}
	token := buildJWT(claims, testSecret)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRBACMiddleware_RejectsInsufficientRole(t *testing.T) {
	cfg := AuthConfig{Enabled: true, JWTSecret: testSecret}

	inner := RBACMiddleware("Admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))
	handler := AuthMiddleware(cfg)(inner)

	claims := jwtClaims{
		Sub:  "viewer-user",
		Role: "Viewer",
		Exp:  time.Now().Add(1 * time.Hour).Unix(),
	}
	token := buildJWT(claims, testSecret)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}
