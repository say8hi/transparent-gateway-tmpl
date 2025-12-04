package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type Response struct {
	Service   string            `json:"service"`
	Message   string            `json:"message"`
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Headers   map[string]string `json:"headers,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func main() {
	serviceName := getEnv("SERVICE_NAME", "mock-backend")
	port := getEnv("PORT", "9000")

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "healthy",
			"service": serviceName,
		})
	})

	// Echo endpoint - returns request information
	mux.HandleFunc("/api/echo", func(w http.ResponseWriter, r *http.Request) {
		headers := make(map[string]string)
		for name, values := range r.Header {
			if len(values) > 0 {
				headers[name] = values[0]
			}
		}

		response := Response{
			Service:   serviceName,
			Message:   "echo response",
			Method:    r.Method,
			Path:      r.URL.Path,
			Headers:   headers,
			Timestamp: time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	// Users endpoint - simulates user data
	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			users := []map[string]interface{}{
				{"id": 1, "name": "John Doe", "email": "john@example.com"},
				{"id": 2, "name": "Jane Smith", "email": "jane@example.com"},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(users)

		case http.MethodPost:
			response := Response{
				Service:   serviceName,
				Message:   "user created",
				Method:    r.Method,
				Path:      r.URL.Path,
				Timestamp: time.Now(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(response)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error:   "method_not_allowed",
				Message: fmt.Sprintf("method %s not allowed", r.Method),
			})
		}
	})

	// Protected endpoint - checks for authorization header
	mux.HandleFunc("/api/protected", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		userID := r.Header.Get("X-User-Id")

		if authHeader == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error:   "unauthorized",
				Message: "authorization header required",
			})
			return
		}

		response := Response{
			Service: serviceName,
			Message: fmt.Sprintf("protected resource accessed by user %s", userID),
			Method:  r.Method,
			Path:    r.URL.Path,
			Headers: map[string]string{
				"Authorization": authHeader,
				"X-User-Id":     userID,
			},
			Timestamp: time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	// Error endpoint - returns error status
	mux.HandleFunc("/api/error", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "internal_error",
			Message: "simulated backend error",
		})
	})

	// Catch-all handler
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response := Response{
			Service:   serviceName,
			Message:   "catch-all handler",
			Method:    r.Method,
			Path:      r.URL.Path,
			Timestamp: time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	addr := ":" + port
	log.Printf("Mock backend '%s' starting on %s", serviceName, addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
