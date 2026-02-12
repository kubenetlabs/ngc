package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// AuthConfig controls JWT authentication behavior.
type AuthConfig struct {
	Enabled   bool
	JWTSecret string
	Issuer    string
}

// UserInfo holds the identity extracted from a validated JWT.
type UserInfo struct {
	Subject string
	Email   string
	Role    string // Admin, Operator, Viewer
	Issuer  string
}

type authContextKey int

const userInfoKey authContextKey = iota

// UserFromContext retrieves the authenticated user from the request context.
func UserFromContext(ctx context.Context) *UserInfo {
	u, _ := ctx.Value(userInfoKey).(*UserInfo)
	return u
}

// AuthMiddleware returns middleware that validates JWT Bearer tokens.
// If cfg.Enabled is false the middleware is a no-op pass-through.
func AuthMiddleware(cfg AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeMiddlewareError(w, http.StatusUnauthorized, "missing Authorization header")
				return
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeMiddlewareError(w, http.StatusUnauthorized, "invalid Authorization header format")
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			user, err := validateJWT(token, cfg.JWTSecret, cfg.Issuer)
			if err != nil {
				slog.Warn("jwt validation failed", "error", err.Error())
				writeMiddlewareError(w, http.StatusUnauthorized, "invalid token: "+err.Error())
				return
			}

			ctx := context.WithValue(r.Context(), userInfoKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RBACMiddleware enforces role-based access control. It requires
// AuthMiddleware to have run first. Roles are hierarchical:
// Admin > Operator > Viewer.
func RBACMiddleware(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user == nil {
				writeMiddlewareError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			if !hasRole(user.Role, requiredRole) {
				writeMiddlewareError(w, http.StatusForbidden, fmt.Sprintf("role %q required, you have %q", requiredRole, user.Role))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// hasRole checks if the user's role meets the required role level.
// Hierarchy: Admin > Operator > Viewer.
func hasRole(userRole, requiredRole string) bool {
	roleLevel := map[string]int{
		"Viewer":   1,
		"Operator": 2,
		"Admin":    3,
	}

	userLevel, ok := roleLevel[userRole]
	if !ok {
		return false
	}
	requiredLevel, ok := roleLevel[requiredRole]
	if !ok {
		return false
	}
	return userLevel >= requiredLevel
}

// jwtHeader is the decoded JOSE header of a JWT.
type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// jwtClaims is the decoded payload of a JWT.
type jwtClaims struct {
	Sub    string `json:"sub"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Iss    string `json:"iss"`
	Exp    int64  `json:"exp"`
	Iat    int64  `json:"iat"`
}

// validateJWT verifies an HS256 JWT and returns the extracted user info.
func validateJWT(token, secret, expectedIssuer string) (*UserInfo, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed token: expected 3 parts, got %d", len(parts))
	}

	// Decode header
	headerBytes, err := base64URLDecode(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid header encoding: %w", err)
	}

	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("invalid header JSON: %w", err)
	}

	if header.Alg != "HS256" {
		return nil, fmt.Errorf("unsupported algorithm: %s", header.Alg)
	}

	// Verify signature
	signingInput := parts[0] + "." + parts[1]
	expectedSig, err := base64URLDecode(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding: %w", err)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	computedSig := mac.Sum(nil)

	if !hmac.Equal(computedSig, expectedSig) {
		return nil, fmt.Errorf("signature verification failed")
	}

	// Decode claims
	claimsBytes, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid claims encoding: %w", err)
	}

	var claims jwtClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, fmt.Errorf("invalid claims JSON: %w", err)
	}

	// Check expiration
	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return nil, fmt.Errorf("token expired")
	}

	// Check issuer
	if expectedIssuer != "" && claims.Iss != expectedIssuer {
		return nil, fmt.Errorf("invalid issuer: expected %q, got %q", expectedIssuer, claims.Iss)
	}

	// Default role to Viewer if not specified
	role := claims.Role
	if role == "" {
		role = "Viewer"
	}

	return &UserInfo{
		Subject: claims.Sub,
		Email:   claims.Email,
		Role:    role,
		Issuer:  claims.Iss,
	}, nil
}

// base64URLDecode decodes a base64url-encoded string (no padding).
func base64URLDecode(s string) ([]byte, error) {
	// Add padding if necessary
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}
