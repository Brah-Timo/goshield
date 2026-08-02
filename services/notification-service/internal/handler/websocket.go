// Package handler provides the WebSocket HTTP handler for the notification-service.
package handler

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/goshield/pkg/middleware"
	"github.com/goshield/services/notification-service/internal/hub"
)

// WSHandler handles WebSocket upgrade and client lifecycle.
type WSHandler struct {
	hub    *hub.Hub
	logger *zap.Logger
}

// NewWSHandler creates a WSHandler.
func NewWSHandler(h *hub.Hub, logger *zap.Logger) *WSHandler {
	return &WSHandler{hub: h, logger: logger}
}

// RegisterRoutes adds WebSocket and health routes to the Fiber app.
func (h *WSHandler) RegisterRoutes(app *fiber.App, jwtMgr *middleware.JWTManager) {
	app.Get("/health", h.Health)
	app.Get("/readyz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ready"})
	})

	// WebSocket endpoint — requires JWT in ?token= query param (browsers can't set headers for WS)
	app.Use("/ws", func(c *fiber.Ctx) error {
		token := c.Query("token")
		if token == "" {
			token = c.Get("Authorization")
		}
		if token == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "missing token"})
		}
		claims, err := jwtMgr.Validate(token)
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}
		ctx := middleware.InjectClaims(c.UserContext(), claims)
		c.SetUserContext(ctx)
		// Also store in Locals so websocket.Conn can access it after upgrade.
		c.Locals("userCtx", ctx)
		// Forward company_id as a query param so the WS handler has a reliable fallback.
		if cid := middleware.CompanyIDFromContext(ctx); cid != "" {
			c.Request().URI().QueryArgs().Set("company_id", cid)
		}

		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws", websocket.New(h.serveWS))
}

func (h *WSHandler) serveWS(c *websocket.Conn) {
	// The fiber middleware stored the enriched context via c.SetUserContext(ctx) before
	// the WebSocket upgrade.  websocket.Conn.Locals("userContext") is not set; instead,
	// retrieve the context that was forwarded from the Fiber context.
	userCtx := c.Locals("userCtx")
	companyID := ""
	if userCtx != nil {
		if ctx, ok := userCtx.(interface{ Value(any) any }); ok {
			companyID = middleware.CompanyIDFromContext(ctx)
		}
	}
	if companyID == "" {
		// Fallback: company_id forwarded as a query param (set by the upgrade middleware).
		companyID = c.Query("company_id")
	}

	client := &hub.Client{
		ID:        uuid.New().String(),
		CompanyID: companyID,
		Send:      make(chan []byte, 64),
	}

	h.hub.Register(client)
	defer func() {
		h.hub.Unregister(client)
		c.Close()
	}()

	// Write pump
	go func() {
		for msg := range client.Send {
			if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
				h.logger.Debug("ws write error", zap.String("client_id", client.ID), zap.Error(err))
				return
			}
		}
	}()

	// Read pump — keep connection alive, handle pings
	c.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.SetPongHandler(func(string) error {
		c.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			break
		}
	}
}

// Health returns liveness status.
func (h *WSHandler) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "ok",
		"service":   "notification-service",
		"timestamp": time.Now().UTC(),
	})
}
