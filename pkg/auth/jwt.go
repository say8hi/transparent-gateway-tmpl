package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken is returned when token validation fails
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken is returned when token has expired
	ErrExpiredToken = errors.New("token has expired")
	// ErrInvalidSigningMethod is returned when token signing method is unexpected
	ErrInvalidSigningMethod = errors.New("invalid signing method")
	// ErrInvalidClaims is returned when token claims are invalid
	ErrInvalidClaims = errors.New("invalid token claims")
)

// Config holds JWT configuration
type Config struct {
	Secret     string        // secret key for signing tokens
	Issuer     string        // issuer claim
	Audience   string        // audience claim
	Expiration time.Duration // token expiration duration
}

// Claims represents JWT claims structure
type Claims struct {
	UserID   string                 `json:"sub"`
	Username string                 `json:"username,omitempty"`
	Email    string                 `json:"email,omitempty"`
	Roles    []string               `json:"roles,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	jwt.RegisteredClaims
}

// Manager handles JWT operations
type Manager struct {
	config *Config
}

// NewManager creates a new JWT manager
func NewManager(config *Config) (*Manager, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}
	if config.Secret == "" {
		return nil, errors.New("secret cannot be empty")
	}
	if config.Expiration <= 0 {
		config.Expiration = 24 * time.Hour // default 24 hours
	}
	if config.Issuer == "" {
		config.Issuer = "api-gateway"
	}
	if config.Audience == "" {
		config.Audience = "api-gateway"
	}

	return &Manager{
		config: config,
	}, nil
}

// GenerateToken generates a new JWT token with the given claims
func (m *Manager) GenerateToken(userID string, metadata map[string]interface{}) (string, error) {
	if userID == "" {
		return "", errors.New("user id cannot be empty")
	}

	now := time.Now()
	claims := &Claims{
		UserID:   userID,
		Metadata: metadata,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.config.Issuer,
			Audience:  jwt.ClaimStrings{m.config.Audience},
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(m.config.Expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.Secret))
}

// GenerateTokenWithClaims generates a new JWT token with custom claims
func (m *Manager) GenerateTokenWithClaims(claims *Claims) (string, error) {
	if claims == nil {
		return "", errors.New("claims cannot be nil")
	}
	if claims.UserID == "" {
		return "", errors.New("user id cannot be empty")
	}

	now := time.Now()
	if claims.Issuer == "" {
		claims.Issuer = m.config.Issuer
	}
	if len(claims.Audience) == 0 {
		claims.Audience = jwt.ClaimStrings{m.config.Audience}
	}
	if claims.Subject == "" {
		claims.Subject = claims.UserID
	}
	if claims.ExpiresAt == nil {
		claims.ExpiresAt = jwt.NewNumericDate(now.Add(m.config.Expiration))
	}
	if claims.IssuedAt == nil {
		claims.IssuedAt = jwt.NewNumericDate(now)
	}
	if claims.NotBefore == nil {
		claims.NotBefore = jwt.NewNumericDate(now)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.Secret))
}

// ValidateToken validates and parses a JWT token
func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrInvalidToken
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: %v", ErrInvalidSigningMethod, token.Header["alg"])
		}
		return []byte(m.config.Secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidClaims
	}

	// validate issuer
	if claims.Issuer != m.config.Issuer {
		return nil, fmt.Errorf("%w: invalid issuer", ErrInvalidClaims)
	}

	// validate audience
	validAudience := false
	for _, aud := range claims.Audience {
		if aud == m.config.Audience {
			validAudience = true
			break
		}
	}
	if !validAudience {
		return nil, fmt.Errorf("%w: invalid audience", ErrInvalidClaims)
	}

	return claims, nil
}

// RefreshToken generates a new token with the same claims but updated expiration
func (m *Manager) RefreshToken(tokenString string) (string, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		// allow refresh even if token is expired
		if !errors.Is(err, ErrExpiredToken) {
			return "", err
		}

		// try to parse expired token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(m.config.Secret), nil
		}, jwt.WithoutClaimsValidation())

		if err != nil {
			return "", fmt.Errorf("failed to parse expired token: %w", err)
		}

		var ok bool
		claims, ok = token.Claims.(*Claims)
		if !ok {
			return "", ErrInvalidClaims
		}
	}

	// generate new token with same claims
	return m.GenerateTokenWithClaims(claims)
}

// ExtractUserID extracts user ID from token without full validation
// useful for logging purposes
func (m *Manager) ExtractUserID(tokenString string) string {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(m.config.Secret), nil
	}, jwt.WithoutClaimsValidation())

	if err != nil {
		return ""
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return ""
	}

	return claims.UserID
}
