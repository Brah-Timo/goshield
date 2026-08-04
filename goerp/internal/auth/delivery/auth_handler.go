package delivery

import (
	"github.com/goerp/goerp/internal/auth/domain"
	"github.com/goerp/goerp/internal/auth/usecase"
	"github.com/goerp/goerp/internal/shared/middleware"
	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	uc *usecase.AuthUsecase
}

func NewAuthHandler(uc *usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{uc: uc}
}

// RegisterRoutes mounts auth routes on the Fiber app
func (h *AuthHandler) RegisterRoutes(app *fiber.App, jwtSecret string) {
	auth := app.Group("/api/v1/auth")
	auth.Post("/login", h.Login)
	auth.Post("/register", h.Register)
	auth.Get("/me", middleware.AuthMiddleware(jwtSecret), h.Me)
	auth.Post("/logout", h.Logout)
}

// Login godoc POST /api/v1/auth/login
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req domain.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	resp, err := h.uc.Login(&req)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	}

	// Set cookie too (for web UI)
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    resp.AccessToken,
		HTTPOnly: true,
		SameSite: "lax",
	})

	return c.JSON(resp)
}

// Register godoc POST /api/v1/auth/register
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req domain.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	resp, err := h.uc.Register(&req)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(resp)
}

// Me godoc GET /api/v1/auth/me
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	user, err := h.uc.GetProfile(userID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "user not found"})
	}
	return c.JSON(user)
}

// Logout godoc POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    "",
		MaxAge:   -1,
	})
	return c.JSON(fiber.Map{"message": "logged out"})
}
