//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	gatewayURL = "http://localhost:8080"
	jwtSecret  = "test-secret-key-for-integration-tests-only"
)

type BackendResponse struct {
	Service   string            `json:"service"`
	Message   string            `json:"message"`
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Headers   map[string]string `json:"headers,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

func TestMain(m *testing.M) {
	// Wait for services to be ready
	if !waitForService(gatewayURL+"/health", 30*time.Second) {
		fmt.Println("Gateway not ready, skipping integration tests")
		os.Exit(1)
	}
	fmt.Println("Gateway is ready, running integration tests...")

	// Run tests
	code := m.Run()
	os.Exit(code)
}

func TestGatewayHealth(t *testing.T) {
	resp, err := http.Get(gatewayURL + "/health")
	if err != nil {
		t.Fatalf("failed to call health endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "OK" {
		t.Errorf("expected body 'OK', got '%s'", string(body))
	}
}

func TestProxyToCRMService(t *testing.T) {
	resp, err := http.Get(gatewayURL + "/crm/api/users")
	if err != nil {
		t.Fatalf("failed to call CRM service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}

	var users []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestProxyToCBSService(t *testing.T) {
	resp, err := http.Get(gatewayURL + "/cbs/api/echo")
	if err != nil {
		t.Fatalf("failed to call CBS service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var backendResp BackendResponse
	if err := json.NewDecoder(resp.Body).Decode(&backendResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if backendResp.Service != "cbs-service" {
		t.Errorf("expected service 'cbs-service', got '%s'", backendResp.Service)
	}
}

func TestProxyToBillingService(t *testing.T) {
	resp, err := http.Get(gatewayURL + "/billing/api/echo")
	if err != nil {
		t.Fatalf("failed to call Billing service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var backendResp BackendResponse
	if err := json.NewDecoder(resp.Body).Decode(&backendResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if backendResp.Service != "billing-service" {
		t.Errorf("expected service 'billing-service', got '%s'", backendResp.Service)
	}
}

func TestJWTAuthenticationRequired(t *testing.T) {
	// Request without token should fail
	resp, err := http.Get(gatewayURL + "/crm/api/protected")
	if err != nil {
		t.Fatalf("failed to call protected endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestJWTAuthenticationSuccess(t *testing.T) {
	// Generate valid token
	token := generateTestToken("user123", "test@example.com")

	// Request with valid token
	req, _ := http.NewRequest("GET", gatewayURL+"/crm/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to call protected endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}

	var backendResp BackendResponse
	if err := json.NewDecoder(resp.Body).Decode(&backendResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Check that X-User-Id header was added by auth middleware
	if backendResp.Headers["X-User-Id"] != "user123" {
		t.Errorf("expected X-User-Id 'user123', got '%s'", backendResp.Headers["X-User-Id"])
	}
}

func TestJWTAuthenticationInvalidToken(t *testing.T) {
	req, _ := http.NewRequest("GET", gatewayURL+"/crm/api/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to call protected endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestCORSHeaders(t *testing.T) {
	req, _ := http.NewRequest("OPTIONS", gatewayURL+"/crm/api/users", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to send OPTIONS request: %v", err)
	}
	defer resp.Body.Close()

	// Check CORS headers
	if origin := resp.Header.Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got '%s'", origin)
	}

	if methods := resp.Header.Get("Access-Control-Allow-Methods"); methods == "" {
		t.Error("Access-Control-Allow-Methods header not set")
	}
}

func TestBackendError(t *testing.T) {
	token := generateTestToken("user123", "test@example.com")

	req, _ := http.NewRequest("GET", gatewayURL+"/crm/api/error", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to call error endpoint: %v", err)
	}
	defer resp.Body.Close()

	// Gateway should proxy the error response from backend
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}

func TestMultipleServicesRouting(t *testing.T) {
	token := generateTestToken("user123", "test@example.com")

	services := []struct {
		name string
		path string
	}{
		{"crm-service", "/crm/api/echo"},
		{"cbs-service", "/cbs/api/echo"},
		{"billing-service", "/billing/api/echo"},
	}

	for _, svc := range services {
		t.Run(svc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", gatewayURL+svc.path, nil)
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("failed to call %s: %v", svc.name, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status 200 for %s, got %d", svc.name, resp.StatusCode)
			}

			var backendResp BackendResponse
			if err := json.NewDecoder(resp.Body).Decode(&backendResp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if backendResp.Service != svc.name {
				t.Errorf("expected service '%s', got '%s'", svc.name, backendResp.Service)
			}
		})
	}
}

// Helper function to generate JWT token for testing
func generateTestToken(userID, email string) string {
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"iss":   "api-gateway-test",
		"aud":   "test-audience",
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(jwtSecret))
	return tokenString
}
