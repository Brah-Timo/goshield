// Package middleware contains Fiber middleware for the api-gateway.
package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/goshield/services/api-gateway/internal/ratelimit"
)

// RateLimitMiddleware creates a Fiber handler that enforces per-IP sliding-window
// rate limits using the provided Redis-backed Limiter.
func RateLimitMiddleware(limiter *ratelimit.Limiter, skipPaths []string, log *zap.Logger) fiber.Handler {
	skipSet := make(map[string]struct{}, len(skipPaths))
	for _, p := range skipPaths {
		skipSet[p] = struct{}{}
	}

	return func(c *fiber.Ctx) error {
		path := c.Path()
		if _, skip := skipSet[path]; skip {
			return c.Next()
		}

		ip := c.IP()
		key := ratelimit.KeyForRequest(ip, path)

		result, err := limiter.Allow(c.UserContext(), key)
		if err != nil {
			log.Warn("rate limiter redis error — failing open", zap.Error(err))
			return c.Next()
		}

		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", 60))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", result.ResetAt.Unix()))

		if !result.Allowed {
			log.Warn("rate limit exceeded",
				zap.String("ip", ip),
				zap.String("path", path),
			)
			return c.Status(http.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "rate limit exceeded",
				"retry_after": time.Until(result.ResetAt).Seconds(),
			})
		}

		return c.Next()
	}
}
