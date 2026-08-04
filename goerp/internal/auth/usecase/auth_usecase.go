package usecase

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/goerp/goerp/internal/auth/domain"
	"github.com/goerp/goerp/internal/auth/repository"
	"github.com/goerp/goerp/internal/shared/middleware"
	"golang.org/x/crypto/bcrypt"
)

type AuthUsecase struct {
	userRepo   *repository.UserRepository
	tenantRepo *repository.TenantRepository
	jwtSecret  string
	jwtTTL     int
}

func NewAuthUsecase(
	userRepo *repository.UserRepository,
	tenantRepo *repository.TenantRepository,
	jwtSecret string,
	jwtTTL int,
) *AuthUsecase {
	return &AuthUsecase{
		userRepo:   userRepo,
		tenantRepo: tenantRepo,
		jwtSecret:  jwtSecret,
		jwtTTL:     jwtTTL,
	}
}

// Login authenticates a user and returns tokens
func (u *AuthUsecase) Login(req *domain.LoginRequest) (*domain.LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, fmt.Errorf("email and password are required")
	}

	var (
		user   *domain.User
		tenant *domain.Tenant
		err    error
	)

	// Find tenant by slug or use default
	if req.TenantSlug != "" {
		tenant, err = u.tenantRepo.FindBySlug(req.TenantSlug)
		if err != nil {
			return nil, fmt.Errorf("tenant not found")
		}
		user, err = u.userRepo.FindByEmail(tenant.ID, req.Email)
	} else {
		// Try demo tenant
		tenant, err = u.tenantRepo.FindBySlug("demo")
		if err != nil {
			return nil, fmt.Errorf("no default tenant configured")
		}
		user, err = u.userRepo.FindByEmail(tenant.ID, req.Email)
	}

	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Update last login
	_ = u.userRepo.UpdateLastLogin(user.ID)

	// Generate token
	token, err := middleware.GenerateAccessToken(
		user.ID, user.TenantID, user.Email, user.FullName, user.IsSuperAdmin,
		u.jwtSecret, u.jwtTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token")
	}

	// Clear sensitive data
	user.PasswordHash = ""

	return &domain.LoginResponse{
		AccessToken: token,
		User:        user,
		Tenant:      tenant,
		ExpiresIn:   u.jwtTTL * 60,
	}, nil
}

// Register creates a new tenant and admin user
func (u *AuthUsecase) Register(req *domain.RegisterRequest) (*domain.LoginResponse, error) {
	if err := validateRegister(req); err != nil {
		return nil, err
	}

	// Create tenant
	slug := slugify(req.TenantName)
	tenant := &domain.Tenant{
		Name: req.TenantName,
		Slug: slug,
		Plan: "community",
	}
	if err := u.tenantRepo.Create(tenant); err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password")
	}

	// Create admin user
	user := &domain.User{
		TenantID:     tenant.ID,
		Email:        req.Email,
		PasswordHash: string(hash),
		FullName:     req.FullName,
		IsActive:     true,
		IsSuperAdmin: false,
	}
	if err := u.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Auto login
	return u.Login(&domain.LoginRequest{
		Email:      req.Email,
		Password:   req.Password,
		TenantSlug: slug,
	})
}

// GetProfile returns user info
func (u *AuthUsecase) GetProfile(userID string) (*domain.User, error) {
	user, err := u.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = ""
	return user, nil
}

func validateRegister(req *domain.RegisterRequest) error {
	if req.Email == "" || req.Password == "" || req.FullName == "" || req.TenantName == "" {
		return fmt.Errorf("all fields are required")
	}
	if len(req.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	if !strings.Contains(req.Email, "@") {
		return fmt.Errorf("invalid email address")
	}
	return nil
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else if unicode.IsSpace(r) || r == '-' {
			b.WriteRune('-')
		}
	}
	result := b.String()
	result = strings.Trim(result, "-")
	if len(result) > 50 {
		result = result[:50]
	}
	return result
}
