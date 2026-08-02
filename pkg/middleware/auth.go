// Package middleware provides shared HTTP and gRPC middleware for all GoShield services.
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTConfig holds JWT manager configuration.
type JWTConfig struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

// Claims holds the JWT payload.
type Claims struct {
	UserID    string `json:"user_id"`
	CompanyID string `json:"company_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

// TokenPair holds access and refresh tokens.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// JWTManager handles JWT signing and validation.
type JWTManager struct {
	secret        []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

// NewJWTManager creates a JWT manager from a JWTConfig.
func NewJWTManager(cfg JWTConfig) *JWTManager {
	accessExpiry := cfg.AccessExpiry
	if accessExpiry == 0 {
		accessExpiry = 15 * time.Minute
	}
	refreshExpiry := cfg.RefreshExpiry
	if refreshExpiry == 0 {
		refreshExpiry = 7 * 24 * time.Hour
	}
	return &JWTManager{
		secret:        []byte(cfg.Secret),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

// GenerateTokenPair creates both access and refresh JWTs.
func (m *JWTManager) GenerateTokenPair(userID, companyID, email, role string) (*TokenPair, error) {
	accessToken, expiresAt, err := m.generateToken(userID, companyID, email, role, m.accessExpiry)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, _, err := m.generateToken(userID, companyID, email, role, m.refreshExpiry)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

func (m *JWTManager) generateToken(userID, companyID, email, role string, expiry time.Duration) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(expiry)

	claims := &Claims{
		UserID:    userID,
		CompanyID: companyID,
		Email:     email,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        uuid.New().String(),
			Issuer:    "goshield",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

// Validate parses and validates a JWT token string.
func (m *JWTManager) Validate(tokenStr string) (*Claims, error) {
	tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}

// JWTMiddleware is a Fiber middleware that validates the Authorization bearer token.
func JWTMiddleware(mgr *JWTManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing Authorization header",
				"code":  http.StatusUnauthorized,
			})
		}

		claims, err := mgr.Validate(authHeader)
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid or expired token",
				"code":  http.StatusUnauthorized,
			})
		}

		// Inject claims into both Fiber locals and Go context.
		ctx := InjectClaims(c.UserContext(), claims)
		c.SetUserContext(ctx)

		return c.Next()
	}
}

// contextKey type for context values.
type contextKey string

const (
	ContextKeyUserID    contextKey = "user_id"
	ContextKeyCompanyID contextKey = "company_id"
	ContextKeyRole      contextKey = "role"
	ContextKeyEmail     contextKey = "email"
	ContextKeyClaims    contextKey = "claims"
)

// InjectClaims adds JWT claims to context.
func InjectClaims(ctx context.Context, claims *Claims) context.Context {
	ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID)
	ctx = context.WithValue(ctx, ContextKeyCompanyID, claims.CompanyID)
	ctx = context.WithValue(ctx, ContextKeyRole, claims.Role)
	ctx = context.WithValue(ctx, ContextKeyEmail, claims.Email)
	ctx = context.WithValue(ctx, ContextKeyClaims, claims)
	return ctx
}

// UserIDFromContext extracts user ID from context (returns empty string if missing).
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyUserID).(string)
	return v
}

// CompanyIDFromContext extracts company ID from context.
func CompanyIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyCompanyID).(string)
	return v
}

// RoleFromContext extracts role from context.
func RoleFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyRole).(string)
	return v
}

// EmailFromContext extracts email from context.
func EmailFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyEmail).(string)
	return v
}

// ClaimsFromContext extracts full claims from context.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	v, ok := ctx.Value(ContextKeyClaims).(*Claims)
	return v, ok
}
