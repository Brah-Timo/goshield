// Package proxy provides a thin reverse-proxy layer for the api-gateway.
package proxy

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"go.uber.org/zap"
)

// UpstreamConfig holds the base URL for each backend service.
type UpstreamConfig struct {
	AuthService         string
	ClaimService        string
	NotificationService string
	AIServiceGo         string
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

// injectIdentityHeaders forwards the enriched context headers injected by the
// JWT middleware so downstream services can trust caller identity without
// re-validating the JWT.
func injectIdentityHeaders(c *fiber.Ctx) {
	if uid := c.Locals("userID"); uid != nil {
		c.Request().Header.Set("X-User-ID", fmt.Sprintf("%v", uid))
	}
	if cid := c.Locals("companyID"); cid != nil {
		c.Request().Header.Set("X-Company-ID", fmt.Sprintf("%v", cid))
	}
	if role := c.Locals("role"); role != nil {
		c.Request().Header.Set("X-User-Role", fmt.Sprintf("%v", role))
	}
}

// proxyTo returns a Fiber handler that proxies the request to the given base URL,
// preserving the full path and query string.
// Uses the default fasthttp client (no explicit timeout override).
func proxyTo(baseURL string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		injectIdentityHeaders(c)
		upstream := fmt.Sprintf("%s%s", baseURL, c.OriginalURL())
		return proxy.Do(c, upstream)
	}
}

// proxyWithTimeout proxies the request with a custom deadline.
// proxy.DoTimeout(c, addr, timeout, clients...) is the correct Fiber v2 API —
// it sets read/write deadlines on the underlying fasthttp request without
// requiring a *fasthttp.Client or *proxy.Config argument.
func proxyWithTimeout(baseURL string, timeout time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		injectIdentityHeaders(c)
		upstream := fmt.Sprintf("%s%s", baseURL, c.OriginalURL())
		return proxy.DoTimeout(c, upstream, timeout)
	}
}

// RegisterRoutes mounts all upstream proxy routes onto the Fiber app.
func (r *Router) RegisterRoutes(app *fiber.App) {
	// ─── Auth Service (/auth/**) ─────────────────────────────────────────────
	app.All("/auth/*", proxyTo(r.cfg.AuthService))

	// ─── Claim Service (/claims/**) ──────────────────────────────────────────
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
