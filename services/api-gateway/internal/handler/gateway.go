// Package handler exposes gateway-level HTTP handlers (health, metrics redirect).
package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// GatewayHandler holds gateway-level handlers.
type GatewayHandler struct {
	rdb    *redis.Client
	logger *zap.Logger
}

// New creates a GatewayHandler.
func New(rdb *redis.Client, log *zap.Logger) *GatewayHandler {
	return &GatewayHandler{rdb: rdb, logger: log}
}

// RegisterRoutes adds gateway-internal routes.
func (g *GatewayHandler) RegisterRoutes(app *fiber.App) {
	app.Get("/health", g.Health)
	app.Get("/readyz", g.Readyz)
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
}

// Health returns a simple liveness check.
func (g *GatewayHandler) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "ok",
		"service":   "api-gateway",
		"timestamp": time.Now().UTC(),
	})
}

// Readyz checks Redis reachability.
func (g *GatewayHandler) Readyz(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if err := g.rdb.Ping(ctx).Err(); err != nil {
		g.logger.Warn("readyz: redis unhealthy", zap.Error(err))
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status": "unhealthy",
			"redis":  err.Error(),
		})
	}
	return c.JSON(fiber.Map{"status": "ready"})
}
