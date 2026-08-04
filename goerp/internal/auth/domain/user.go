package domain

import "time"

// User domain model
type User struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	FullName     string    `json:"full_name"`
	Avatar       string    `json:"avatar,omitempty"`
	IsActive     bool      `json:"is_active"`
	IsSuperAdmin bool      `json:"is_superadmin"`
	LastLogin    *time.Time `json:"last_login,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Tenant domain model
type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Plan      string    `json:"plan"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// Role domain model
type Role struct {
	ID          string      `json:"id"`
	TenantID    string      `json:"tenant_id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Permissions []Permission `json:"permissions"`
	CreatedAt   time.Time   `json:"created_at"`
}

// Permission domain model
type Permission struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Scope    string `json:"scope"`
}

// LoginRequest payload
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	TenantSlug string `json:"tenant_slug,omitempty"`
}

// LoginResponse payload
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	User         *User  `json:"user"`
	Tenant       *Tenant `json:"tenant"`
	ExpiresIn    int    `json:"expires_in"`
}

// RegisterRequest payload
type RegisterRequest struct {
	TenantName string `json:"tenant_name"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	FullName   string `json:"full_name"`
}
