package middleware

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"

	pkgmw "github.com/goshield/pkg/middleware"
)

// JWTAuthMiddleware validates the Bearer token from the Authorization header and
// injects user claims into the request context.  It skips validation for paths
// that are in the public allowlist.
func JWTAuthMiddleware(mgr *pkgmw.JWTManager, publicPaths []string) fiber.Handler {
	publicSet := make(map[string]struct{}, len(publicPaths))
	for _, p := range publicPaths {
		publicSet[p] = struct{}{}
	}

	return func(c *fiber.Ctx) error {
		path := c.Path()
		if _, ok := publicSet[path]; ok {
			return c.Next()
		}
		// Also allow sub-paths of public prefixes (e.g. /auth/v1/oauth/google),
		// but require the next character to be "/" so that /auth/v1/oauth does
		// NOT grant access to /auth/v1/oauth-admin or /auth/v1/oauthEvil.
		for _, p := range publicPaths {
			if strings.HasPrefix(path, p+"/") {
				return c.Next()
			}
		}

		raw := c.Get("Authorization")
		if raw == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing Authorization header",
			})
		}
		token := strings.TrimPrefix(raw, "Bearer ")
		if token == raw {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authorization header must use Bearer scheme",
			})
		}

		claims, err := mgr.Validate(token)
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid or expired token",
			})
		}

		ctx := pkgmw.InjectClaims(c.UserContext(), claims)
		c.SetUserContext(ctx)
		return c.Next()
	}
}
