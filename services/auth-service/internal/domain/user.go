// Package domain defines the user entity and authentication value objects.
package domain

import "time"

// UserRole defines the access level of a user within a company.
type UserRole string

const (
	RoleAdmin   UserRole = "ADMIN"
	RoleAnalyst UserRole = "ANALYST"
	RoleViewer  UserRole = "VIEWER"
)

// User is the core entity for authentication and authorization.
type User struct {
	ID           string    `json:"id"`
	CompanyID    string    `json:"company_id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // never serialised
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Role         UserRole  `json:"role"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	OAuthProvider string   `json:"oauth_provider,omitempty"` // google | ""
	OAuthSub     string    `json:"-"`                        // provider subject
	Active       bool      `json:"active"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RefreshToken tracks issued refresh tokens for rotation and revocation.
type RefreshToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TokenHash string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// RegisterInput carries validated input for new user registration.
type RegisterInput struct {
	CompanyID string
	Email     string
	Password  string
	FirstName string
	LastName  string
	Role      UserRole
}

// LoginInput carries credentials for password-based login.
type LoginInput struct {
	Email    string
	Password string
}

// OAuthInput carries data returned by the OAuth provider after callback.
type OAuthInput struct {
	Provider  string
	Sub       string
	Email     string
	FirstName string
	LastName  string
	AvatarURL string
	CompanyID string // resolved from invite or domain mapping
}

// UpdateUserInput for admin to change another user's role/status.
type UpdateUserInput struct {
	UserID    string
	CompanyID string
	Role      *UserRole
	Active    *bool
	FirstName *string
	LastName  *string
}

// UserListFilter for listing users in a company.
type UserListFilter struct {
	CompanyID string
	Role      UserRole
	Active    *bool
	Page      int
	PageSize  int
}
