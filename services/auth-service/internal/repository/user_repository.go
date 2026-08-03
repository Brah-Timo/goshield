// Package repository provides PostgreSQL data access for the auth-service.
package repository

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/goshield/pkg/database"
	"github.com/goshield/services/auth-service/internal/domain"
)

// Sentinel errors.
var (
	ErrNotFound       = errors.New("user not found")
	ErrAlreadyExists  = errors.New("user already exists")
	ErrTokenNotFound  = errors.New("refresh token not found")
	ErrTokenRevoked   = errors.New("refresh token revoked")
	ErrTokenExpired   = errors.New("refresh token expired")
)

// UserRepository defines data access for users.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id, companyID string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByOAuthSub(ctx context.Context, provider, sub string) (*domain.User, error)
	List(ctx context.Context, filter domain.UserListFilter) ([]*domain.User, int64, error)
	Update(ctx context.Context, input domain.UpdateUserInput) error
	UpdateLastLogin(ctx context.Context, id string) error
	Delete(ctx context.Context, id, companyID string) error

	// Refresh token management
	StoreRefreshToken(ctx context.Context, token *domain.RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllUserTokens(ctx context.Context, userID string) error
}

type userRepo struct {
	db     *database.DB
	logger *zap.Logger
}

// New creates a UserRepository backed by PostgreSQL.
func New(db *database.DB, logger *zap.Logger) UserRepository {
	return &userRepo{db: db, logger: logger}
}

// Create inserts a new user, returns ErrAlreadyExists if email taken.
func (r *userRepo) Create(ctx context.Context, user *domain.User) error {
	const q = `
		INSERT INTO users (
			id, company_id, email, password_hash, first_name, last_name,
			role, avatar_url, oauth_provider, oauth_sub, active, is_active,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (company_id, email) DO NOTHING`

	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	tag, err := r.db.Pool.Exec(ctx, q,
		user.ID, user.CompanyID, user.Email, user.PasswordHash,
		user.FirstName, user.LastName, string(user.Role),
		user.AvatarURL, user.OAuthProvider, user.OAuthSub,
		user.Active, user.Active, // $11=active, $12=is_active (kept in sync)
		now, now,
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAlreadyExists
	}
	return nil
}

// GetByID fetches a user scoped to a company.
func (r *userRepo) GetByID(ctx context.Context, id, companyID string) (*domain.User, error) {
	q := `
		SELECT id, company_id, email, password_hash,
		       COALESCE(first_name, SPLIT_PART(COALESCE(full_name,''), ' ', 1)) AS first_name,
		       COALESCE(last_name, '') AS last_name,
		       role, COALESCE(avatar_url,''), COALESCE(oauth_provider,''),
		       COALESCE(oauth_sub, COALESCE(oauth_id,'')),
		       COALESCE(active, is_active, true),
		       last_login_at, created_at, updated_at
		FROM users
		WHERE id = $1`

	args := []any{id}
	if companyID != "" {
		q += " AND company_id = $2"
		args = append(args, companyID)
	}
	q += " AND COALESCE(is_active, active, true) = true"

	return r.scanUser(r.db.Pool.QueryRow(ctx, q, args...))
}

// GetByEmail fetches a user by email (cross-tenant for login).
func (r *userRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	const q = `
		SELECT id, company_id, email, password_hash,
		       COALESCE(first_name, SPLIT_PART(COALESCE(full_name,''), ' ', 1)) AS first_name,
		       COALESCE(last_name, '') AS last_name,
		       role, COALESCE(avatar_url,''), COALESCE(oauth_provider,''),
		       COALESCE(oauth_sub, COALESCE(oauth_id,'')),
		       COALESCE(active, is_active, true),
		       last_login_at, created_at, updated_at
		FROM users
		WHERE email = $1 AND COALESCE(is_active, active, true) = true`

	return r.scanUser(r.db.Pool.QueryRow(ctx, q, email))
}

// GetByOAuthSub finds a user by their OAuth provider subject ID.
func (r *userRepo) GetByOAuthSub(ctx context.Context, provider, sub string) (*domain.User, error) {
	const q = `
		SELECT id, company_id, email, password_hash,
		       COALESCE(first_name, SPLIT_PART(COALESCE(full_name,''), ' ', 1)) AS first_name,
		       COALESCE(last_name, '') AS last_name,
		       role, COALESCE(avatar_url,''), COALESCE(oauth_provider,''),
		       COALESCE(oauth_sub, COALESCE(oauth_id,'')),
		       COALESCE(active, is_active, true),
		       last_login_at, created_at, updated_at
		FROM users
		WHERE oauth_provider = $1
		  AND COALESCE(oauth_sub, oauth_id) = $2
		  AND COALESCE(is_active, active, true) = true`

	return r.scanUser(r.db.Pool.QueryRow(ctx, q, provider, sub))
}

// List returns users in a company with optional role/status filter.
func (r *userRepo) List(ctx context.Context, filter domain.UserListFilter) ([]*domain.User, int64, error) {
	var args []any
	conds := "WHERE company_id = $1"
	args = append(args, filter.CompanyID)
	idx := 2

	if filter.Role != "" {
		conds += fmt.Sprintf(" AND role = $%d", idx)
		args = append(args, string(filter.Role))
		idx++
	}
	if filter.Active != nil {
		conds += fmt.Sprintf(" AND active = $%d", idx)
		args = append(args, *filter.Active)
		idx++
	}

	var total int64
	if err := r.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users "+conds, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}

	dataArgs := append(args, pageSize, (page-1)*pageSize)
	q := fmt.Sprintf(`
		SELECT id, company_id, email, password_hash,
		       COALESCE(first_name, SPLIT_PART(COALESCE(full_name,''), ' ', 1)) AS first_name,
		       COALESCE(last_name, '') AS last_name,
		       role, COALESCE(avatar_url,''), COALESCE(oauth_provider,''),
		       COALESCE(oauth_sub, COALESCE(oauth_id,'')),
		       COALESCE(active, is_active, true),
		       last_login_at, created_at, updated_at
		FROM users %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, conds, idx, idx+1)

	rows, err := r.db.Pool.Query(ctx, q, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u, err := r.scanUserRow(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

// Update modifies mutable user fields.
func (r *userRepo) Update(ctx context.Context, input domain.UpdateUserInput) error {
	if input.Role != nil {
		const q = `UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2 AND company_id = $3`
		if _, err := r.db.Pool.Exec(ctx, q, string(*input.Role), input.UserID, input.CompanyID); err != nil {
			return fmt.Errorf("update role: %w", err)
		}
	}
	if input.Active != nil {
		const q = `UPDATE users SET active = $1, updated_at = NOW() WHERE id = $2 AND company_id = $3`
		if _, err := r.db.Pool.Exec(ctx, q, *input.Active, input.UserID, input.CompanyID); err != nil {
			return fmt.Errorf("update active: %w", err)
		}
	}
	if input.FirstName != nil || input.LastName != nil {
		const q = `UPDATE users SET first_name = COALESCE($1, first_name), last_name = COALESCE($2, last_name), updated_at = NOW() WHERE id = $3 AND company_id = $4`
		if _, err := r.db.Pool.Exec(ctx, q, input.FirstName, input.LastName, input.UserID, input.CompanyID); err != nil {
			return fmt.Errorf("update names: %w", err)
		}
	}
	return nil
}

// UpdateLastLogin records login timestamp.
func (r *userRepo) UpdateLastLogin(ctx context.Context, id string) error {
	const q = `UPDATE users SET last_login_at = NOW() WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, q, id)
	return err
}

// Delete deactivates a user (soft delete).
func (r *userRepo) Delete(ctx context.Context, id, companyID string) error {
	const q = `UPDATE users SET active = false, is_active = false, updated_at = NOW() WHERE id = $1 AND company_id = $2`
	tag, err := r.db.Pool.Exec(ctx, q, id, companyID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ── Refresh token methods ─────────────────────────────────────────────────────

// StoreRefreshToken saves a hashed refresh token.
func (r *userRepo) StoreRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	const q = `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	now := time.Now().UTC()
	token.CreatedAt = now
	_, err := r.db.Pool.Exec(ctx, q, token.ID, token.UserID, token.TokenHash, token.ExpiresAt, now)
	return err
}

// GetRefreshToken retrieves an un-revoked refresh token by its hash.
func (r *userRepo) GetRefreshToken(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	const q = `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1`

	rt := &domain.RefreshToken{}
	err := r.db.Pool.QueryRow(ctx, q, tokenHash).Scan(
		&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.CreatedAt, &rt.RevokedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	if rt.RevokedAt != nil {
		return nil, ErrTokenRevoked
	}
	if rt.ExpiresAt.Before(time.Now()) {
		return nil, ErrTokenExpired
	}
	return rt, nil
}

// RevokeRefreshToken marks a single token as revoked.
func (r *userRepo) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	const q = `UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1`
	_, err := r.db.Pool.Exec(ctx, q, tokenHash)
	return err
}

// RevokeAllUserTokens revokes all tokens for a user (logout-all / password change).
func (r *userRepo) RevokeAllUserTokens(ctx context.Context, userID string) error {
	const q = `UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := r.db.Pool.Exec(ctx, q, userID)
	return err
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (r *userRepo) scanUser(row pgx.Row) (*domain.User, error) {
	u := &domain.User{}
	var roleStr, oauthProvider, oauthSub string
	err := row.Scan(
		&u.ID, &u.CompanyID, &u.Email, &u.PasswordHash,
		&u.FirstName, &u.LastName, &roleStr,
		&u.AvatarURL, &oauthProvider, &oauthSub,
		&u.Active, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	u.Role = domain.UserRole(roleStr)
	u.OAuthProvider = oauthProvider
	u.OAuthSub = oauthSub
	return u, nil
}

func (r *userRepo) scanUserRow(rows pgx.Rows) (*domain.User, error) {
	u := &domain.User{}
	var roleStr, oauthProvider, oauthSub string
	err := rows.Scan(
		&u.ID, &u.CompanyID, &u.Email, &u.PasswordHash,
		&u.FirstName, &u.LastName, &roleStr,
		&u.AvatarURL, &oauthProvider, &oauthSub,
		&u.Active, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan user row: %w", err)
	}
	u.Role = domain.UserRole(roleStr)
	u.OAuthProvider = oauthProvider
	u.OAuthSub = oauthSub
	return u, nil
}

// HashToken returns the SHA-256 hex digest of a token string.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}
