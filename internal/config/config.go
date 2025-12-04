package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	Server ServerConfig
	CORS   CORSConfig
	JWT    JWTConfig
	Proxy  ProxyConfig
	Log    LogConfig
}

// ServerConfig holds server-specific configuration.
type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// CORSConfig holds CORS-specific configuration.
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// JWTConfig holds JWT-specific configuration.
type JWTConfig struct {
	Secret     string
	Issuer     string
	Audience   string
	Expiration time.Duration
}

// ProxyConfig holds proxy-specific configuration.
type ProxyConfig struct {
	Targets map[string]TargetConfig
	Timeout time.Duration
}

// TargetConfig holds configuration for a single proxy target.
type TargetConfig struct {
	URL string
}

// LogConfig holds logging-specific configuration.
type LogConfig struct {
	Level         string
	ComponentName string
}

// Load loads configuration from environment variables.
// It attempts to load from .env file first, then falls back to system environment.
func Load() (*Config, error) {
	// try to load .env file, ignore error if it doesn't exist
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         getEnvAsInt("SERVER_PORT", 8080),
			ReadTimeout:  getEnvAsDuration("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getEnvAsDuration("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getEnvAsDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		CORS: CORSConfig{
			AllowedOrigins:   getEnvAsSlice("CORS_ALLOWED_ORIGINS", []string{"*"}),
			AllowedMethods:   getEnvAsSlice("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}),
			AllowedHeaders:   getEnvAsSlice("CORS_ALLOWED_HEADERS", []string{"Content-Type", "Authorization"}),
			AllowCredentials: getEnvAsBool("CORS_ALLOW_CREDENTIALS", true),
			MaxAge:           getEnvAsInt("CORS_MAX_AGE", 3600),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", ""),
			Issuer:     getEnv("JWT_ISSUER", "api-gateway"),
			Audience:   getEnv("JWT_AUDIENCE", "api-gateway"),
			Expiration: getEnvAsDuration("JWT_EXPIRATION", 24*time.Hour),
		},
		Proxy: ProxyConfig{
			Targets: loadProxyTargets(),
			Timeout: getEnvAsDuration("PROXY_TIMEOUT", 30*time.Second),
		},
		Log: LogConfig{
			Level:         getEnv("LOG_LEVEL", "info"),
			ComponentName: getEnv("LOG_COMPONENT_NAME", "api-gateway"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	if len(c.Proxy.Targets) == 0 {
		return fmt.Errorf("at least one proxy target is required")
	}

	for name, target := range c.Proxy.Targets {
		if target.URL == "" {
			return fmt.Errorf("proxy target %q URL is required", name)
		}
	}

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("SERVER_PORT must be between 1 and 65535")
	}

	return nil
}

// getEnv retrieves the value of the environment variable named by the key.
// If the variable is not present, it returns the fallback value.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvAsInt retrieves the value of the environment variable as an integer.
// If the variable is not present or cannot be parsed, it returns the fallback value.
func getEnvAsInt(key string, fallback int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return fallback
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return fallback
	}
	return value
}

// getEnvAsBool retrieves the value of the environment variable as a boolean.
// If the variable is not present or cannot be parsed, it returns the fallback value.
func getEnvAsBool(key string, fallback bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return fallback
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return fallback
	}
	return value
}

// getEnvAsDuration retrieves the value of the environment variable as a duration.
// If the variable is not present or cannot be parsed, it returns the fallback value.
func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return fallback
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return fallback
	}
	return value
}

// getEnvAsSlice retrieves the value of the environment variable as a string slice.
// The value is expected to be comma-separated.
// If the variable is not present, it returns the fallback value.
func getEnvAsSlice(key string, fallback []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return fallback
	}
	parts := strings.Split(valueStr, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}

// loadProxyTargets loads proxy targets from environment variables.
// Supports two formats:
// 1. Legacy: PROXY_TARGET_URL (single backend)
// 2. Multi-backend: SERVICE_NAME_URL (e.g., CRM_SERVICE_URL, CBS_SERVICE_URL)
func loadProxyTargets() map[string]TargetConfig {
	targets := make(map[string]TargetConfig)

	// check for legacy single target format
	if legacyURL := os.Getenv("PROXY_TARGET_URL"); legacyURL != "" {
		targets["default"] = TargetConfig{URL: legacyURL}
		return targets
	}

	// load multiple targets
	// common service names to check
	serviceNames := []string{"CRM", "CBS", "BILLING", "AUTH", "NOTIFICATION", "PAYMENT"}

	for _, name := range serviceNames {
		envKey := name + "_SERVICE_URL"
		if url := os.Getenv(envKey); url != "" {
			targets[strings.ToLower(name)] = TargetConfig{URL: url}
		}
	}

	return targets
}
