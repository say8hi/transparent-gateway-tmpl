package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gateway/template/internal/config"
	"github.com/gateway/template/pkg/auth"
	"github.com/gateway/template/pkg/logger"
)

// ContextKey is the type for context keys
type ContextKey string

const (
	// UserIDContextKey is the context key for user ID
	UserIDContextKey ContextKey = "user_id"
	// ClaimsContextKey is the context key for JWT claims
	ClaimsContextKey ContextKey = "claims"
)

// Logging returns a chi middleware for logging requests
func Logging(log logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// create response writer wrapper to capture status code
			ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// process request
			next.ServeHTTP(ww, r)

			// log after request
			latency := time.Since(start)

			// extract user ID from context if available
			userID := ""
			if uid := r.Context().Value(UserIDContextKey); uid != nil {
				if uidStr, ok := uid.(string); ok {
					userID = uidStr
				}
			}

			log.Info("http request processed",
				"client_ip", getClientIP(r),
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.statusCode,
				"latency_ms", latency.Milliseconds(),
				"user_agent", r.UserAgent(),
				"user_id", userID,
			)
		})
	}
}

// CORS returns a chi middleware for CORS
func CORS(cfg *config.CORSConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// check if origin is allowed
			if isOriginAllowed(origin, cfg.AllowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)

				if cfg.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

				w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))

				if cfg.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", string(rune(cfg.MaxAge)))
				}
			}

			// handle preflight request
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Auth returns a chi middleware for JWT authentication
//
// ⚠️ WARNING: This is a LOCAL IMPLEMENTATION for development/testing only!
//
// Before deploying to production, you MUST replace this with your corporate
// authentication middleware from your common package.
func Auth(cfg *config.JWTConfig, log logger.Logger) func(next http.Handler) http.Handler {
	// create JWT manager
	authManager, err := auth.NewManager(&auth.Config{
		Secret:     cfg.Secret,
		Issuer:     cfg.Issuer,
		Audience:   cfg.Audience,
		Expiration: cfg.Expiration,
	})
	if err != nil {
		log.Error("failed to create auth manager", "error", err)
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				respondJSON(w, http.StatusInternalServerError, map[string]string{
					"error": "internal server error",
				})
			})
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			// validate request and extract claims
			claims, err := authManager.ValidateRequest(authHeader)
			if err != nil {
				var authErr *auth.AuthError
				statusCode := http.StatusUnauthorized
				message := "unauthorized"

				if errors.As(err, &authErr) {
					statusCode = authErr.Code
					message = authErr.Message
				}

				log.Warn("authentication failed",
					"path", r.URL.Path,
					"method", r.Method,
					"error", err.Error(),
				)

				respondJSON(w, statusCode, map[string]string{
					"error": message,
				})
				return
			}

			// set claims and user ID in context
			ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
			ctx = context.WithValue(ctx, UserIDContextKey, claims.UserID)

			log.Debug("authenticated request",
				"path", r.URL.Path,
				"method", r.Method,
				"user_id", claims.UserID,
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserIDFromContext extracts the user ID from request context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID := ctx.Value(UserIDContextKey)
	if userID == nil {
		return "", false
	}
	userIDStr, ok := userID.(string)
	return userIDStr, ok
}

// GetClaimsFromContext extracts the JWT claims from request context
func GetClaimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims := ctx.Value(ClaimsContextKey)
	if claims == nil {
		return nil, false
	}
	authClaims, ok := claims.(*auth.Claims)
	return authClaims, ok
}

// responseWriter is a wrapper for http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// getClientIP extracts the real client IP from the request
func getClientIP(r *http.Request) string {
	// check X-Forwarded-For header first
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// fallback to RemoteAddr
	ip := r.RemoteAddr
	// remove port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// isOriginAllowed checks if the origin is in the allowed origins list
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// simple JSON encoding for error responses
	if m, ok := data.(map[string]string); ok {
		w.Write([]byte(`{"error":"` + m["error"] + `"}`))
	}
}
