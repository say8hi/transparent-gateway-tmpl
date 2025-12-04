package auth

import (
	"errors"
	"net/http"
	"strings"
)

// ContextKey is the type for context keys
type ContextKey string

const (
	// ClaimsContextKey is the context key for JWT claims
	ClaimsContextKey ContextKey = "jwt_claims"
	// UserIDContextKey is the context key for user ID
	UserIDContextKey ContextKey = "user_id"
)

// AuthError represents an authentication error with HTTP status code
type AuthError struct {
	Code    int
	Message string
	Err     error
}

// Error implements the error interface
func (e *AuthError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// ExtractBearerToken extracts the bearer token from the Authorization header
func ExtractBearerToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", &AuthError{
			Code:    http.StatusUnauthorized,
			Message: "missing authorization header",
			Err:     nil,
		}
	}

	// check if the header has the Bearer scheme
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return "", &AuthError{
			Code:    http.StatusUnauthorized,
			Message: "invalid authorization header format",
			Err:     nil,
		}
	}

	scheme := strings.ToLower(parts[0])
	if scheme != "bearer" {
		return "", &AuthError{
			Code:    http.StatusUnauthorized,
			Message: "invalid authorization scheme (expected Bearer)",
			Err:     nil,
		}
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", &AuthError{
			Code:    http.StatusUnauthorized,
			Message: "empty bearer token",
			Err:     nil,
		}
	}

	return token, nil
}

// ValidateRequest validates the JWT token from the request and returns claims
func (m *Manager) ValidateRequest(authHeader string) (*Claims, error) {
	token, err := ExtractBearerToken(authHeader)
	if err != nil {
		return nil, err
	}

	claims, err := m.ValidateToken(token)
	if err != nil {
		statusCode := http.StatusUnauthorized
		message := "invalid or expired token"

		if errors.Is(err, ErrExpiredToken) {
			message = "token has expired"
		} else if errors.Is(err, ErrInvalidSigningMethod) {
			message = "invalid token signing method"
		} else if errors.Is(err, ErrInvalidClaims) {
			message = "invalid token claims"
		}

		return nil, &AuthError{
			Code:    statusCode,
			Message: message,
			Err:     err,
		}
	}

	return claims, nil
}

// RequireRole checks if the claims contain the required role
func RequireRole(claims *Claims, role string) error {
	if claims == nil {
		return &AuthError{
			Code:    http.StatusForbidden,
			Message: "no claims provided",
			Err:     nil,
		}
	}

	for _, r := range claims.Roles {
		if r == role {
			return nil
		}
	}

	return &AuthError{
		Code:    http.StatusForbidden,
		Message: "insufficient permissions",
		Err:     nil,
	}
}

// RequireAnyRole checks if the claims contain any of the required roles
func RequireAnyRole(claims *Claims, roles ...string) error {
	if claims == nil {
		return &AuthError{
			Code:    http.StatusForbidden,
			Message: "no claims provided",
			Err:     nil,
		}
	}

	if len(roles) == 0 {
		return nil
	}

	roleSet := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		roleSet[role] = struct{}{}
	}

	for _, userRole := range claims.Roles {
		if _, ok := roleSet[userRole]; ok {
			return nil
		}
	}

	return &AuthError{
		Code:    http.StatusForbidden,
		Message: "insufficient permissions",
		Err:     nil,
	}
}

// RequireAllRoles checks if the claims contain all of the required roles
func RequireAllRoles(claims *Claims, roles ...string) error {
	if claims == nil {
		return &AuthError{
			Code:    http.StatusForbidden,
			Message: "no claims provided",
			Err:     nil,
		}
	}

	if len(roles) == 0 {
		return nil
	}

	userRoleSet := make(map[string]struct{}, len(claims.Roles))
	for _, role := range claims.Roles {
		userRoleSet[role] = struct{}{}
	}

	for _, requiredRole := range roles {
		if _, ok := userRoleSet[requiredRole]; !ok {
			return &AuthError{
				Code:    http.StatusForbidden,
				Message: "insufficient permissions",
				Err:     nil,
			}
		}
	}

	return nil
}
