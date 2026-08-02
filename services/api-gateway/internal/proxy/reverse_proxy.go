// Package proxy provides a thin reverse-proxy layer for the api-gateway.
package proxy

import (
	"fmt"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"go.uber.org/zap"
)

// UpstreamConfig holds the base URL for each backend service.
type UpstreamConfig struct {
	AuthService          string
	ClaimService         string
	NotificationService  string
	AIServiceGo          string
}

// Router wires Fiber route groups to their upstream services.
type Router struct {
	cfg    UpstreamConfig
	logger *zap.Logger
}

// New creates a Router.
func New(cfg UpstreamConfig, log *zap.Logger) *Router {
	return &Router{cfg: cfg, logger: log}
}

// proxyTo returns a Fiber handler that proxies the request to the given base URL,
// preserving the full path and query string, and forwarding the enriched context
// headers (X-User-ID, X-Company-ID, X-User-Role) injected by the JWT middleware.
func proxyTo(baseURL string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		target, err := url.Parse(baseURL)
		if err != nil {
			return fmt.Errorf("invalid upstream URL %q: %w", baseURL, err)
		}

		// Build the full upstream URL.
		upstream := fmt.Sprintf("%s%s", baseURL, c.OriginalURL())

		// Forward identity headers so downstream services can trust caller identity
		// without re-validating the JWT.
		if uid := c.Locals("userID"); uid != nil {
			c.Request().Header.Set("X-User-ID", fmt.Sprintf("%v", uid))
		}
		if cid := c.Locals("companyID"); cid != nil {
			c.Request().Header.Set("X-Company-ID", fmt.Sprintf("%v", cid))
		}
		if role := c.Locals("role"); role != nil {
			c.Request().Header.Set("X-User-Role", fmt.Sprintf("%v", role))
		}

		_ = target
		return proxy.Do(c, upstream)
	}
}

// proxyWithTimeout wraps proxyTo with a custom timeout using proxy.WithTimeout option.
func proxyWithTimeout(baseURL string, timeout time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		upstream := fmt.Sprintf("%s%s", baseURL, c.OriginalURL())

		if uid := c.Locals("userID"); uid != nil {
			c.Request().Header.Set("X-User-ID", fmt.Sprintf("%v", uid))
		}
		if cid := c.Locals("companyID"); cid != nil {
			c.Request().Header.Set("X-Company-ID", fmt.Sprintf("%v", cid))
		}
		if role := c.Locals("role"); role != nil {
			c.Request().Header.Set("X-User-Role", fmt.Sprintf("%v", role))
		}

		return proxy.Do(c, upstream, &proxy.Config{
			Timeout: timeout,
		})
	}
}

// RegisterRoutes mounts all upstream proxy routes onto the Fiber app.
func (r *Router) RegisterRoutes(app *fiber.App) {
	// ─── Auth Service (/auth/**) ─────────────────────────────────────────────
	app.All("/auth/*", proxyTo(r.cfg.AuthService))

	// ─── Claim Service (/claims/**) ─────────────────────────────────────────
	// File upload may be large → 60 s timeout.
	app.All("/claims/*", proxyWithTimeout(r.cfg.ClaimService, 60*time.Second))
	app.All("/claims", proxyTo(r.cfg.ClaimService))

	// ─── Notification Service (/notifications/**, /ws) ───────────────────────
	// WebSocket upgrade is passed through transparently.
	app.All("/notifications/*", proxyTo(r.cfg.NotificationService))
	app.All("/ws", proxyTo(r.cfg.NotificationService))

	// ─── AI Orchestration (/ai/**) ───────────────────────────────────────────
	// AI inference may take up to 30 s.
	app.All("/ai/*", proxyWithTimeout(r.cfg.AIServiceGo, 30*time.Second))
}
