package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims holds the JWT payload
type JWTClaims struct {
	UserID    string `json:"user_id"`
	TenantID  string `json:"tenant_id"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	IsSuperAdmin bool `json:"is_superadmin"`
	jwt.RegisteredClaims
}

// AuthMiddleware validates JWT tokens
func AuthMiddleware(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			// Try cookie
			authHeader = "Bearer " + c.Cookies("access_token")
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing or invalid authorization header",
			})
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// Parse and validate
		token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid or expired token",
			})
		}

		claims, ok := token.Claims.(*JWTClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid token claims",
			})
		}

		// Store claims in context
		c.Locals("claims", claims)
		c.Locals("user_id", claims.UserID)
		c.Locals("tenant_id", claims.TenantID)
		return c.Next()
	}
}

// GenerateAccessToken creates a signed JWT
func GenerateAccessToken(userID, tenantID, email, fullName string, isSuper bool, secret string, ttlMinutes int) (string, error) {
	claims := &JWTClaims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
		FullName: fullName,
		IsSuperAdmin: isSuper,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(ttlMinutes) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// LoggerMiddleware logs request/response
func LoggerMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)
		
		statusCode := c.Response().StatusCode()
		method := c.Method()
		path := c.Path()
		
		_ = statusCode
		_ = method
		_ = path
		_ = duration
		// Production logger would go here
		return err
	}
}

// CORSMiddleware adds CORS headers
func CORSMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if c.Method() == "OPTIONS" {
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.Next()
	}
}

// TenantMiddleware extracts tenant from JWT and sets DB context
func TenantMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals("claims").(*JWTClaims)
		if !ok || claims == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "tenant context missing",
			})
		}
		c.Locals("tenant_id", claims.TenantID)
		return c.Next()
	}
}

// GetClaims is a helper to extract claims from context
func GetClaims(c *fiber.Ctx) *JWTClaims {
	claims, _ := c.Locals("claims").(*JWTClaims)
	return claims
}

func GetTenantID(c *fiber.Ctx) string {
	if tid, ok := c.Locals("tenant_id").(string); ok {
		return tid
	}
	return ""
}

func GetUserID(c *fiber.Ctx) string {
	if uid, ok := c.Locals("user_id").(string); ok {
		return uid
	}
	return ""
}
