// Package handler provides HTTP health endpoints for the ai-service-go.
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/goshield/services/ai-service-go/internal/bridge"
)

// HealthHandler provides liveness and readiness probes.
type HealthHandler struct {
	aiClient *bridge.PythonAIClient
}

// NewHealthHandler creates a HealthHandler.
func NewHealthHandler(aiClient *bridge.PythonAIClient) *HealthHandler {
	return &HealthHandler{aiClient: aiClient}
}

// Live returns 200 if the Go service is running.
func (h *HealthHandler) Live(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "ok",
		"service":   "ai-service-go",
		"timestamp": time.Now().UTC(),
	})
}

// Ready returns 200 if the Python AI service is reachable.
func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
	defer cancel()

	if err := h.aiClient.HealthCheck(ctx); err != nil {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{
			"status": "not_ready",
			"reason": err.Error(),
		})
	}
	return c.JSON(fiber.Map{"status": "ready"})
}
