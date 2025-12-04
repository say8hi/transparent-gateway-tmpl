package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// set required environment variables
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("PROXY_TARGET_URL", "http://localhost:9000")
	defer func() {
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("PROXY_TARGET_URL")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.JWT.Secret != "test-secret" {
		t.Errorf("expected JWT secret to be 'test-secret', got '%s'", cfg.JWT.Secret)
	}

	// check that legacy single backend is loaded as "default"
	if len(cfg.Proxy.Targets) != 1 {
		t.Errorf("expected 1 proxy target, got %d", len(cfg.Proxy.Targets))
	}

	defaultTarget, ok := cfg.Proxy.Targets["default"]
	if !ok {
		t.Error("expected 'default' target to exist")
	}

	if defaultTarget.URL != "http://localhost:9000" {
		t.Errorf("expected default target URL to be 'http://localhost:9000', got '%s'", defaultTarget.URL)
	}

	// test default values
	if cfg.Server.Port != 8080 {
		t.Errorf("expected default server port to be 8080, got %d", cfg.Server.Port)
	}
}

func TestLoadMultipleBackends(t *testing.T) {
	// set required environment variables for multiple backends
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("CRM_SERVICE_URL", "http://crm:9001")
	os.Setenv("CBS_SERVICE_URL", "http://cbs:9002")
	defer func() {
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("CRM_SERVICE_URL")
		os.Unsetenv("CBS_SERVICE_URL")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if len(cfg.Proxy.Targets) != 2 {
		t.Errorf("expected 2 proxy targets, got %d", len(cfg.Proxy.Targets))
	}

	crmTarget, ok := cfg.Proxy.Targets["crm"]
	if !ok {
		t.Error("expected 'crm' target to exist")
	}
	if crmTarget.URL != "http://crm:9001" {
		t.Errorf("expected crm target URL to be 'http://crm:9001', got '%s'", crmTarget.URL)
	}

	cbsTarget, ok := cfg.Proxy.Targets["cbs"]
	if !ok {
		t.Error("expected 'cbs' target to exist")
	}
	if cbsTarget.URL != "http://cbs:9002" {
		t.Errorf("expected cbs target URL to be 'http://cbs:9002', got '%s'", cbsTarget.URL)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				JWT: JWTConfig{Secret: "secret"},
				Proxy: ProxyConfig{
					Targets: map[string]TargetConfig{
						"default": {URL: "http://localhost:9000"},
					},
				},
				Server: ServerConfig{Port: 8080},
			},
			wantErr: false,
		},
		{
			name: "valid multi-backend config",
			config: &Config{
				JWT: JWTConfig{Secret: "secret"},
				Proxy: ProxyConfig{
					Targets: map[string]TargetConfig{
						"crm": {URL: "http://crm:9001"},
						"cbs": {URL: "http://cbs:9002"},
					},
				},
				Server: ServerConfig{Port: 8080},
			},
			wantErr: false,
		},
		{
			name: "missing JWT secret",
			config: &Config{
				JWT: JWTConfig{Secret: ""},
				Proxy: ProxyConfig{
					Targets: map[string]TargetConfig{
						"default": {URL: "http://localhost:9000"},
					},
				},
				Server: ServerConfig{Port: 8080},
			},
			wantErr: true,
		},
		{
			name: "no proxy targets",
			config: &Config{
				JWT: JWTConfig{Secret: "secret"},
				Proxy: ProxyConfig{
					Targets: map[string]TargetConfig{},
				},
				Server: ServerConfig{Port: 8080},
			},
			wantErr: true,
		},
		{
			name: "empty target URL",
			config: &Config{
				JWT: JWTConfig{Secret: "secret"},
				Proxy: ProxyConfig{
					Targets: map[string]TargetConfig{
						"crm": {URL: ""},
					},
				},
				Server: ServerConfig{Port: 8080},
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			config: &Config{
				JWT: JWTConfig{Secret: "secret"},
				Proxy: ProxyConfig{
					Targets: map[string]TargetConfig{
						"default": {URL: "http://localhost:9000"},
					},
				},
				Server: ServerConfig{Port: 70000},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetEnvAsDuration(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		fallback time.Duration
		expected time.Duration
	}{
		{
			name:     "valid duration",
			key:      "TEST_DURATION",
			value:    "30s",
			fallback: 10 * time.Second,
			expected: 30 * time.Second,
		},
		{
			name:     "invalid duration",
			key:      "TEST_DURATION",
			value:    "invalid",
			fallback: 10 * time.Second,
			expected: 10 * time.Second,
		},
		{
			name:     "empty value",
			key:      "TEST_DURATION",
			value:    "",
			fallback: 10 * time.Second,
			expected: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvAsDuration(tt.key, tt.fallback)
			if result != tt.expected {
				t.Errorf("getEnvAsDuration() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetEnvAsSlice(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		fallback []string
		expected []string
	}{
		{
			name:     "comma-separated values",
			key:      "TEST_SLICE",
			value:    "a,b,c",
			fallback: []string{"default"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "values with spaces",
			key:      "TEST_SLICE",
			value:    "a, b, c",
			fallback: []string{"default"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "empty value",
			key:      "TEST_SLICE",
			value:    "",
			fallback: []string{"default"},
			expected: []string{"default"},
		},
		{
			name:     "single value",
			key:      "TEST_SLICE",
			value:    "single",
			fallback: []string{"default"},
			expected: []string{"single"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvAsSlice(tt.key, tt.fallback)
			if len(result) != len(tt.expected) {
				t.Errorf("getEnvAsSlice() length = %d, expected %d", len(result), len(tt.expected))
				return
			}

			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("getEnvAsSlice()[%d] = %s, expected %s", i, result[i], tt.expected[i])
				}
			}
		})
	}
}
