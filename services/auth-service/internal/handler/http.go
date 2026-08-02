// Package handler provides HTTP handlers for the auth-service.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/goshield/pkg/config"
	"github.com/goshield/pkg/middleware"
	"github.com/goshield/services/auth-service/internal/domain"
	"github.com/goshield/services/auth-service/internal/repository"
	"github.com/goshield/services/auth-service/internal/service"
)

// Handler groups Fiber HTTP handlers for the auth-service.
type Handler struct {
	svc          service.AuthService
	validate     *validator.Validate
	oauthCfg     *oauth2.Config
	oauthEnabled bool
	logger       *zap.Logger
}

// New creates a new Handler with all required dependencies.
func New(svc service.AuthService, cfg *config.AppConfig, logger *zap.Logger) *Handler {
	var oauthCfg *oauth2.Config
	oauthEnabled := cfg.Auth.OAuthGoogleClientID != "" && cfg.Auth.OAuthGoogleSecret != ""
	if oauthEnabled {
		oauthCfg = &oauth2.Config{
			ClientID:     cfg.Auth.OAuthGoogleClientID,
			ClientSecret: cfg.Auth.OAuthGoogleSecret,
			RedirectURL:  cfg.Auth.OAuthGoogleCallback,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		}
	}
	return &Handler{
		svc:          svc,
		validate:     validator.New(),
		oauthCfg:     oauthCfg,
		oauthEnabled: oauthEnabled,
		logger:       logger,
	}
}

// RegisterRoutes wires all auth routes onto the Fiber router group.
// The group already carries the versioned prefix (e.g. /auth/v1 or /api/v1),
// so paths here should NOT repeat that prefix.
func (h *Handler) RegisterRoutes(r fiber.Router, jwtMgr *middleware.JWTManager) {
	// Public routes
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/refresh", h.RefreshTokens)
	r.Post("/logout", h.Logout)

	// OAuth2 routes
	r.Get("/oauth/google", h.GoogleOAuthRedirect)
	r.Get("/oauth/google/callback", h.GoogleOAuthCallback)

	// Protected user management routes
	protected := r.Group("/users", middleware.JWTMiddleware(jwtMgr))
	protected.Get("/me", h.GetCurrentUser)
	protected.Get("/", h.ListUsers)
	protected.Get("/:id", h.GetUser)
	protected.Patch("/:id", h.UpdateUser)
	protected.Delete("/:id", h.DeleteUser)
	protected.Post("/logout-all", h.LogoutAll)
}

// ─── DTOs ─────────────────────────────────────────────────────────────────────

type registerRequest struct {
	Email     string `json:"email"      validate:"required,email"`
	Password  string `json:"password"   validate:"required,min=8,max=72"`
	FirstName string `json:"first_name" validate:"required,min=1,max=50"`
	LastName  string `json:"last_name"  validate:"required,min=1,max=50"`
	CompanyID string `json:"company_id" validate:"required,uuid4"`
	Role      string `json:"role"       validate:"omitempty,oneof=ADMIN ANALYST VIEWER"`
}

type loginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type updateUserRequest struct {
	Role      *string `json:"role"      validate:"omitempty,oneof=ADMIN ANALYST VIEWER"`
	Active    *bool   `json:"active"`
	FirstName *string `json:"first_name" validate:"omitempty,min=1,max=50"`
	LastName  *string `json:"last_name"  validate:"omitempty,min=1,max=50"`
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

// Register godoc
// @Summary  Register a new user
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body registerRequest true "Registration data"
// @Success  201 {object} service.AuthResponse
// @Router   /auth/register [post]
func (h *Handler) Register(c *fiber.Ctx) error {
	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "invalid request body", nil)
	}
	if err := h.validate.Struct(req); err != nil {
		return respondError(c, http.StatusUnprocessableEntity, "validation failed", formatValidationErrors(err))
	}

	role := domain.UserRole(req.Role)
	if role == "" {
		role = domain.RoleAnalyst
	}

	resp, err := h.svc.Register(c.Context(), domain.RegisterInput{
		CompanyID: req.CompanyID,
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      role,
	})
	if err != nil {
		if strings.Contains(err.Error(), "already registered") {
			return respondError(c, http.StatusConflict, err.Error(), nil)
		}
		h.logger.Error("register failed", zap.Error(err))
		return respondError(c, http.StatusInternalServerError, "registration failed", nil)
	}

	return c.Status(http.StatusCreated).JSON(normaliseAuthResponse(resp))
}

// Login godoc
// @Summary  Login with email and password
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body loginRequest true "Login credentials"
// @Success  200 {object} service.AuthResponse
// @Router   /auth/login [post]
func (h *Handler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "invalid request body", nil)
	}
	if err := h.validate.Struct(req); err != nil {
		return respondError(c, http.StatusUnprocessableEntity, "validation failed", formatValidationErrors(err))
	}

	resp, err := h.svc.Login(c.Context(), domain.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if strings.Contains(err.Error(), "invalid credentials") {
			return respondError(c, http.StatusUnauthorized, "invalid email or password", nil)
		}
		h.logger.Error("login failed", zap.Error(err))
		return respondError(c, http.StatusInternalServerError, "login failed", nil)
	}

	// Set refresh token in HttpOnly cookie for web clients.
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    resp.RefreshToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HTTPOnly: true,
		SameSite: "Strict",
		Secure:   true,
	})

	return c.JSON(normaliseAuthResponse(resp))
}

// RefreshTokens godoc
// @Summary  Refresh access token using refresh token
// @Tags     auth
// @Accept   json
// @Produce  json
// @Router   /auth/refresh [post]
func (h *Handler) RefreshTokens(c *fiber.Ctx) error {
	// Accept from body OR cookie.
	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		var req refreshRequest
		if err := c.BodyParser(&req); err == nil && req.RefreshToken != "" {
			refreshToken = req.RefreshToken
		}
	}
	if refreshToken == "" {
		return respondError(c, http.StatusBadRequest, "refresh token required", nil)
	}

	resp, err := h.svc.RefreshTokens(c.Context(), refreshToken)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, "invalid or expired refresh token", nil)
	}

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    resp.RefreshToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HTTPOnly: true,
		SameSite: "Strict",
		Secure:   true,
	})

	return c.JSON(normaliseAuthResponse(resp))
}

// Logout godoc
// @Summary  Logout and revoke refresh token
// @Tags     auth
// @Accept   json
// @Produce  json
// @Router   /auth/logout [post]
func (h *Handler) Logout(c *fiber.Ctx) error {
	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		var req logoutRequest
		if err := c.BodyParser(&req); err == nil {
			refreshToken = req.RefreshToken
		}
	}

	if refreshToken != "" {
		_ = h.svc.Logout(c.Context(), refreshToken)
	}

	// Clear cookie.
	c.Cookie(&fiber.Cookie{
		Name:    "refresh_token",
		Value:   "",
		Expires: time.Unix(0, 0),
		MaxAge:  -1,
	})

	return c.JSON(fiber.Map{"message": "logged out successfully"})
}

// LogoutAll godoc
// @Summary  Revoke all refresh tokens for the current user
// @Tags     auth
// @Router   /users/logout-all [post]
func (h *Handler) LogoutAll(c *fiber.Ctx) error {
	userID := middleware.UserIDFromContext(c.UserContext())
	if userID == "" {
		return respondError(c, http.StatusUnauthorized, "unauthorized", nil)
	}

	if err := h.svc.LogoutAll(c.Context(), userID); err != nil {
		h.logger.Error("logout-all failed", zap.String("user_id", userID), zap.Error(err))
		return respondError(c, http.StatusInternalServerError, "failed to revoke tokens", nil)
	}

	return c.JSON(fiber.Map{"message": "all sessions revoked"})
}

// GoogleOAuthRedirect godoc
// @Summary  Redirect to Google OAuth2 consent page
// @Tags     auth
// @Router   /auth/oauth/google [get]
func (h *Handler) GoogleOAuthRedirect(c *fiber.Ctx) error {
	if !h.oauthEnabled {
		return respondError(c, http.StatusServiceUnavailable, "Google OAuth is not configured", nil)
	}

	// Use a secure random state in production (store in Redis for CSRF validation).
	state := fmt.Sprintf("gs_%d", time.Now().UnixNano())
	url := h.oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return c.Redirect(url, http.StatusTemporaryRedirect)
}

// GoogleOAuthCallback godoc
// @Summary  Handle Google OAuth2 callback
// @Tags     auth
// @Router   /auth/oauth/google/callback [get]
func (h *Handler) GoogleOAuthCallback(c *fiber.Ctx) error {
	if !h.oauthEnabled {
		return respondError(c, http.StatusServiceUnavailable, "Google OAuth is not configured", nil)
	}

	code := c.Query("code")
	if code == "" {
		return respondError(c, http.StatusBadRequest, "missing authorization code", nil)
	}

	token, err := h.oauthCfg.Exchange(context.Background(), code)
	if err != nil {
		h.logger.Error("OAuth token exchange failed", zap.Error(err))
		return respondError(c, http.StatusUnauthorized, "OAuth token exchange failed", nil)
	}

	// Fetch user info from Google.
	userInfo, err := fetchGoogleUserInfo(token.AccessToken)
	if err != nil {
		h.logger.Error("failed to fetch Google user info", zap.Error(err))
		return respondError(c, http.StatusInternalServerError, "failed to get user info", nil)
	}

	resp, err := h.svc.HandleOAuth(c.Context(), domain.OAuthInput{
		Provider:  "google",
		Sub:       userInfo["sub"].(string),
		Email:     userInfo["email"].(string),
		FirstName: stringOrEmpty(userInfo["given_name"]),
		LastName:  stringOrEmpty(userInfo["family_name"]),
		AvatarURL: stringOrEmpty(userInfo["picture"]),
	})
	if err != nil {
		h.logger.Error("OAuth user handling failed", zap.Error(err))
		return respondError(c, http.StatusInternalServerError, "OAuth login failed", nil)
	}

	// Set cookie and redirect to frontend dashboard.
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    resp.RefreshToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HTTPOnly: true,
		SameSite: "Strict",
		Secure:   true,
	})

	// Redirect to frontend with access token in URL fragment.
	frontendURL := fmt.Sprintf("/dashboard?token=%s", resp.AccessToken)
	return c.Redirect(frontendURL, http.StatusTemporaryRedirect)
}

// GetCurrentUser returns the authenticated user's profile.
func (h *Handler) GetCurrentUser(c *fiber.Ctx) error {
	userID := middleware.UserIDFromContext(c.UserContext())
	companyID := middleware.CompanyIDFromContext(c.UserContext())

	user, err := h.svc.GetUser(c.Context(), userID, companyID)
	if err != nil {
		if err == repository.ErrNotFound {
			return respondError(c, http.StatusNotFound, "user not found", nil)
		}
		return respondError(c, http.StatusInternalServerError, "failed to get user", nil)
	}

	return c.JSON(fiber.Map{"data": user})
}

// GetUser returns a specific user (admin only).
func (h *Handler) GetUser(c *fiber.Ctx) error {
	role := middleware.RoleFromContext(c.UserContext())
	companyID := middleware.CompanyIDFromContext(c.UserContext())

	if role != "ADMIN" {
		return respondError(c, http.StatusForbidden, "admin role required", nil)
	}

	user, err := h.svc.GetUser(c.Context(), c.Params("id"), companyID)
	if err != nil {
		if err == repository.ErrNotFound {
			return respondError(c, http.StatusNotFound, "user not found", nil)
		}
		return respondError(c, http.StatusInternalServerError, "failed to get user", nil)
	}

	return c.JSON(fiber.Map{"data": user})
}

// ListUsers returns all users in the company (admin only).
func (h *Handler) ListUsers(c *fiber.Ctx) error {
	role := middleware.RoleFromContext(c.UserContext())
	companyID := middleware.CompanyIDFromContext(c.UserContext())

	if role != "ADMIN" {
		return respondError(c, http.StatusForbidden, "admin role required", nil)
	}

	filter := domain.UserListFilter{
		CompanyID: companyID,
		Role:      domain.UserRole(c.Query("role")),
		Page:      c.QueryInt("page", 1),
		PageSize:  c.QueryInt("page_size", 20),
	}

	users, total, err := h.svc.ListUsers(c.Context(), filter)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "failed to list users", nil)
	}

	return c.JSON(fiber.Map{
		"data":  users,
		"total": total,
	})
}

// UpdateUser modifies a user's role or status.
func (h *Handler) UpdateUser(c *fiber.Ctx) error {
	callerRole := middleware.RoleFromContext(c.UserContext())
	companyID := middleware.CompanyIDFromContext(c.UserContext())
	callerID := middleware.UserIDFromContext(c.UserContext())
	targetID := c.Params("id")

	// Non-admins can only update themselves.
	if callerRole != "ADMIN" && callerID != targetID {
		return respondError(c, http.StatusForbidden, "insufficient permissions", nil)
	}

	var req updateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "invalid request body", nil)
	}

	input := domain.UpdateUserInput{
		UserID:    targetID,
		CompanyID: companyID,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Active:    req.Active,
	}
	if req.Role != nil {
		r := domain.UserRole(*req.Role)
		input.Role = &r
	}

	// Only admins can change roles.
	if input.Role != nil && callerRole != "ADMIN" {
		input.Role = nil
	}

	if err := h.svc.UpdateUser(c.Context(), input); err != nil {
		return respondError(c, http.StatusInternalServerError, "failed to update user", nil)
	}

	return c.JSON(fiber.Map{"message": "user updated"})
}

// DeleteUser deactivates a user (admin only).
func (h *Handler) DeleteUser(c *fiber.Ctx) error {
	callerRole := middleware.RoleFromContext(c.UserContext())
	companyID := middleware.CompanyIDFromContext(c.UserContext())

	if callerRole != "ADMIN" {
		return respondError(c, http.StatusForbidden, "admin role required", nil)
	}

	if err := h.svc.DeleteUser(c.Context(), c.Params("id"), companyID); err != nil {
		if err == repository.ErrNotFound {
			return respondError(c, http.StatusNotFound, "user not found", nil)
		}
		return respondError(c, http.StatusInternalServerError, "failed to delete user", nil)
	}

	return c.SendStatus(http.StatusNoContent)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func respondError(c *fiber.Ctx, status int, msg string, details any) error {
	return c.Status(status).JSON(fiber.Map{
		"error":   msg,
		"code":    status,
		"details": details,
	})
}

func formatValidationErrors(err error) map[string]string {
	errs := make(map[string]string)
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range ve {
			errs[fe.Field()] = fe.Tag()
		}
	}
	return errs
}

func fetchGoogleUserInfo(accessToken string) (map[string]any, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v3/userinfo?access_token=" + accessToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var info map[string]any
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	return info, nil
}

func stringOrEmpty(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// userDTO is a camelCase representation of domain.User for the frontend.
type userDTO struct {
	ID          string     `json:"id"`
	CompanyID   string     `json:"companyId"`
	Email       string     `json:"email"`
	FirstName   string     `json:"firstName"`
	LastName    string     `json:"lastName"`
	Role        string     `json:"role"`
	AvatarURL   string     `json:"avatarUrl,omitempty"`
	Active      bool       `json:"active"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// normaliseAuthResponse converts the backend AuthResponse to the shape the
// frontend expects: { user (camelCase), tokens: { accessToken, refreshToken } }.
func normaliseAuthResponse(r *service.AuthResponse) fiber.Map {
	u := r.User
	dto := userDTO{
		ID:          u.ID,
		CompanyID:   u.CompanyID,
		Email:       u.Email,
		FirstName:   u.FirstName,
		LastName:    u.LastName,
		Role:        string(u.Role),
		AvatarURL:   u.AvatarURL,
		Active:      u.Active,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
	return fiber.Map{
		"user": dto,
		"tokens": fiber.Map{
			"accessToken":  r.AccessToken,
			"refreshToken": r.RefreshToken,
			"expiresAt":    r.ExpiresAt,
		},
	}
}

// Health returns a simple liveness check.
func Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "ok",
		"service":   "auth-service",
		"timestamp": time.Now().UTC(),
	})
}
