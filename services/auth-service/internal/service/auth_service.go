// Package service implements the authentication and authorization business logic.
package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/goshield/pkg/config"
	"github.com/goshield/pkg/middleware"
	"github.com/goshield/services/auth-service/internal/domain"
	"github.com/goshield/services/auth-service/internal/repository"
)

// AuthResponse is returned on successful login/register.
type AuthResponse struct {
	User         *domain.User       `json:"user"`
	AccessToken  string             `json:"access_token"`
	RefreshToken string             `json:"refresh_token"`
	ExpiresAt    time.Time          `json:"expires_at"`
}

// AuthService defines authentication operations.
type AuthService interface {
	Register(ctx context.Context, input domain.RegisterInput) (*AuthResponse, error)
	Login(ctx context.Context, input domain.LoginInput) (*AuthResponse, error)
	RefreshTokens(ctx context.Context, refreshToken string) (*AuthResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, userID string) error
	HandleOAuth(ctx context.Context, input domain.OAuthInput) (*AuthResponse, error)
	ValidateToken(ctx context.Context, token string) (*middleware.Claims, error)
	GetUser(ctx context.Context, userID, companyID string) (*domain.User, error)
	ListUsers(ctx context.Context, filter domain.UserListFilter) ([]*domain.User, int64, error)
	UpdateUser(ctx context.Context, input domain.UpdateUserInput) error
	DeleteUser(ctx context.Context, id, companyID string) error
	CheckPermission(role, resource, action string) bool
}

type authService struct {
	repo     repository.UserRepository
	jwtMgr   *middleware.JWTManager
	enforcer *casbin.Enforcer
	cfg      *config.AppConfig
	logger   *zap.Logger
}

// New creates a fully wired AuthService.
func New(
	repo repository.UserRepository,
	jwtMgr *middleware.JWTManager,
	enforcer *casbin.Enforcer,
	cfg *config.AppConfig,
	logger *zap.Logger,
) AuthService {
	return &authService{
		repo:     repo,
		jwtMgr:   jwtMgr,
		enforcer: enforcer,
		cfg:      cfg,
		logger:   logger,
	}
}

// Register creates a new user account and returns tokens.
func (s *authService) Register(ctx context.Context, input domain.RegisterInput) (*AuthResponse, error) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		ID:           uuid.New().String(),
		CompanyID:    input.CompanyID,
		Email:        input.Email,
		PasswordHash: string(hash),
		FirstName:    input.FirstName,
		LastName:     input.LastName,
		Role:         input.Role,
		Active:       true,
	}
	if user.Role == "" {
		user.Role = domain.RoleAnalyst
	}

	if err := s.repo.Create(ctx, user); err != nil {
		if err == repository.ErrAlreadyExists {
			return nil, fmt.Errorf("email already registered")
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return s.issueTokens(ctx, user)
}

// Login authenticates a user and returns fresh tokens.
func (s *authService) Login(ctx context.Context, input domain.LoginInput) (*AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, fmt.Errorf("invalid credentials")
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		s.logger.Warn("failed login attempt",
			zap.String("email", email),
			zap.String("company_id", user.CompanyID),
		)
		return nil, fmt.Errorf("invalid credentials")
	}

	_ = s.repo.UpdateLastLogin(ctx, user.ID)

	s.logger.Info("user logged in",
		zap.String("user_id", user.ID),
		zap.String("email", user.Email),
	)

	return s.issueTokens(ctx, user)
}

// RefreshTokens validates a refresh token and issues a new token pair (rotation).
func (s *authService) RefreshTokens(ctx context.Context, refreshTokenStr string) (*AuthResponse, error) {
	tokenHash := repository.HashToken(refreshTokenStr)

	rt, err := s.repo.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Revoke the used token (rotation).
	if err := s.repo.RevokeRefreshToken(ctx, tokenHash); err != nil {
		s.logger.Warn("failed to revoke old refresh token", zap.Error(err))
	}

	// rt.UserID is a UUID — use GetByID directly (GetByEmail would always fail here)
	user, err := s.repo.GetByID(ctx, rt.UserID, "")
	if err != nil {
		return nil, fmt.Errorf("user not found for refresh: %w", err)
	}

	return s.issueTokens(ctx, user)
}

// Logout revokes a single refresh token.
func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := repository.HashToken(refreshToken)
	return s.repo.RevokeRefreshToken(ctx, tokenHash)
}

// LogoutAll revokes all tokens for a user.
func (s *authService) LogoutAll(ctx context.Context, userID string) error {
	return s.repo.RevokeAllUserTokens(ctx, userID)
}

// HandleOAuth processes OAuth2 callback, creates user on first visit, returns tokens.
func (s *authService) HandleOAuth(ctx context.Context, input domain.OAuthInput) (*AuthResponse, error) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	// Check if user already exists by OAuth sub.
	user, err := s.repo.GetByOAuthSub(ctx, input.Provider, input.Sub)
	if err == nil {
		// Existing OAuth user — update profile and issue tokens.
		_ = s.repo.UpdateLastLogin(ctx, user.ID)
		return s.issueTokens(ctx, user)
	}

	// Check if email is already registered (link accounts).
	existingByEmail, errEmail := s.repo.GetByEmail(ctx, input.Email)
	if errEmail == nil {
		// Link OAuth sub to existing account.
		_ = s.repo.UpdateLastLogin(ctx, existingByEmail.ID)
		return s.issueTokens(ctx, existingByEmail)
	}

	// New user — register.
	companyID := input.CompanyID
	if companyID == "" {
		// For now use a default demo company; production would resolve by email domain.
		companyID = "00000000-0000-0000-0000-000000000001"
	}

	user = &domain.User{
		ID:            uuid.New().String(),
		CompanyID:     companyID,
		Email:         input.Email,
		FirstName:     input.FirstName,
		LastName:      input.LastName,
		AvatarURL:     input.AvatarURL,
		OAuthProvider: input.Provider,
		OAuthSub:      input.Sub,
		Role:          domain.RoleViewer, // default role for OAuth sign-ups
		Active:        true,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		if err == repository.ErrAlreadyExists {
			// Race condition — fetch and return.
			user, _ = s.repo.GetByEmail(ctx, input.Email)
		} else {
			return nil, fmt.Errorf("create oauth user: %w", err)
		}
	}

	s.logger.Info("new OAuth user registered",
		zap.String("email", user.Email),
		zap.String("provider", input.Provider),
	)

	return s.issueTokens(ctx, user)
}

// ValidateToken validates a JWT access token and returns its claims.
func (s *authService) ValidateToken(_ context.Context, token string) (*middleware.Claims, error) {
	return s.jwtMgr.Validate(token)
}

// GetUser returns a user by ID.
func (s *authService) GetUser(ctx context.Context, userID, companyID string) (*domain.User, error) {
	return s.repo.GetByID(ctx, userID, companyID)
}

// ListUsers returns users in a company.
func (s *authService) ListUsers(ctx context.Context, filter domain.UserListFilter) ([]*domain.User, int64, error) {
	return s.repo.List(ctx, filter)
}

// UpdateUser modifies a user's role or active status.
func (s *authService) UpdateUser(ctx context.Context, input domain.UpdateUserInput) error {
	return s.repo.Update(ctx, input)
}

// DeleteUser soft-deletes a user.
func (s *authService) DeleteUser(ctx context.Context, id, companyID string) error {
	return s.repo.Delete(ctx, id, companyID)
}

// CheckPermission evaluates whether a role has permission for an action on a resource.
func (s *authService) CheckPermission(role, resource, action string) bool {
	if s.enforcer == nil {
		return false
	}
	ok, err := s.enforcer.Enforce(role, resource, action)
	if err != nil {
		s.logger.Warn("casbin enforce error", zap.Error(err))
		return false
	}
	return ok
}

// ─── private helpers ──────────────────────────────────────────────────────────

func (s *authService) issueTokens(ctx context.Context, user *domain.User) (*AuthResponse, error) {
	pair, err := s.jwtMgr.GenerateTokenPair(user.ID, user.CompanyID, user.Email, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("generate token pair: %w", err)
	}

	// Store hashed refresh token for rotation/revocation.
	tokenHash := repository.HashToken(pair.RefreshToken)
	rt := &domain.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(s.cfg.Auth.JWTRefreshExpiry),
	}
	if err := s.repo.StoreRefreshToken(ctx, rt); err != nil {
		s.logger.Warn("failed to store refresh token", zap.Error(err))
	}

	return &AuthResponse{
		User:         user,
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    pair.ExpiresAt,
	}, nil
}

// generateSecureToken creates a cryptographically random token.
func generateSecureToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
