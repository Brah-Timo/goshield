package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/goerp/goerp/internal/auth/domain"
	"github.com/goerp/goerp/internal/shared/database"
)

// ─── UserRepository ───────────────────────────────────────────────────────────

type UserRepository struct {
	db *database.DB
}

func NewUserRepository(db *database.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByEmail(tenantID, email string) (*domain.User, error) {
	row := r.db.QueryRow(`
		SELECT id, tenant_id, email, password_hash, full_name,
		       COALESCE(avatar,''), is_active, is_superadmin,
		       COALESCE(last_login,''), created_at, updated_at
		FROM users
		WHERE tenant_id = ? AND email = ? AND is_active = 1
	`, tenantID, email)
	return scanUser(row)
}

func (r *UserRepository) FindByEmailGlobal(email string) (*domain.User, error) {
	row := r.db.QueryRow(`
		SELECT id, tenant_id, email, password_hash, full_name,
		       COALESCE(avatar,''), is_active, is_superadmin,
		       COALESCE(last_login,''), created_at, updated_at
		FROM users
		WHERE email = ? AND is_superadmin = 1 AND is_active = 1
		LIMIT 1
	`, email)
	return scanUser(row)
}

func (r *UserRepository) FindByID(id string) (*domain.User, error) {
	row := r.db.QueryRow(`
		SELECT id, tenant_id, email, password_hash, full_name,
		       COALESCE(avatar,''), is_active, is_superadmin,
		       COALESCE(last_login,''), created_at, updated_at
		FROM users
		WHERE id = ?
	`, id)
	return scanUser(row)
}

func (r *UserRepository) Create(u *domain.User) error {
	u.ID = uuid.New().String()
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	_, err := r.db.Exec(`
		INSERT INTO users (id, tenant_id, email, password_hash, full_name, is_active, is_superadmin, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?)
	`, u.ID, u.TenantID, u.Email, u.PasswordHash, u.FullName,
		boolToInt(u.IsActive), boolToInt(u.IsSuperAdmin),
		u.CreatedAt.Format(time.RFC3339), u.UpdatedAt.Format(time.RFC3339))
	return err
}

func (r *UserRepository) UpdateLastLogin(id string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := r.db.Exec(`UPDATE users SET last_login = ? WHERE id = ?`, now, id)
	return err
}

func (r *UserRepository) List(tenantID string) ([]*domain.User, error) {
	rows, err := r.db.Query(`
		SELECT id, tenant_id, email, password_hash, full_name,
		       COALESCE(avatar,''), is_active, is_superadmin,
		       COALESCE(last_login,''), created_at, updated_at
		FROM users
		WHERE tenant_id = ?
		ORDER BY full_name
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			continue
		}
		users = append(users, u)
	}
	if users == nil {
		users = []*domain.User{}
	}
	return users, nil
}

// ─── TenantRepository ─────────────────────────────────────────────────────────

type TenantRepository struct {
	db *database.DB
}

func NewTenantRepository(db *database.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

func (r *TenantRepository) FindBySlug(slug string) (*domain.Tenant, error) {
	t := &domain.Tenant{}
	var isActive int
	var createdAtStr string
	err := r.db.QueryRow(`
		SELECT id, name, slug, plan, is_active, created_at
		FROM tenants WHERE slug = ?
	`, slug).Scan(&t.ID, &t.Name, &t.Slug, &t.Plan, &isActive, &createdAtStr)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tenant not found")
	}
	if err != nil {
		return nil, err
	}
	t.IsActive = isActive == 1
	t.CreatedAt, _ = parseTimeOrZero(createdAtStr)
	return t, nil
}

func (r *TenantRepository) FindByID(id string) (*domain.Tenant, error) {
	t := &domain.Tenant{}
	var isActive int
	var createdAtStr string
	err := r.db.QueryRow(`
		SELECT id, name, slug, plan, is_active, created_at
		FROM tenants WHERE id = ?
	`, id).Scan(&t.ID, &t.Name, &t.Slug, &t.Plan, &isActive, &createdAtStr)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tenant not found")
	}
	if err != nil {
		return nil, err
	}
	t.IsActive = isActive == 1
	t.CreatedAt, _ = parseTimeOrZero(createdAtStr)
	return t, nil
}

func (r *TenantRepository) Create(t *domain.Tenant) error {
	t.ID = uuid.New().String()
	t.CreatedAt = time.Now()
	_, err := r.db.Exec(`
		INSERT INTO tenants (id, name, slug, plan, created_at) VALUES (?,?,?,?,?)
	`, t.ID, t.Name, t.Slug, t.Plan, t.CreatedAt.Format(time.RFC3339))
	return err
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanUser(row scannable) (*domain.User, error) {
	u := &domain.User{}
	var isActive, isSuperAdmin int
	var lastLoginStr, createdAtStr, updatedAtStr string

	err := row.Scan(
		&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.FullName,
		&u.Avatar, &isActive, &isSuperAdmin,
		&lastLoginStr, &createdAtStr, &updatedAtStr,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}

	u.IsActive = isActive == 1
	u.IsSuperAdmin = isSuperAdmin == 1

	if lastLoginStr != "" {
		if t, err := parseTimeOrZero(lastLoginStr); err == nil {
			u.LastLogin = &t
		}
	}
	if t, err := parseTimeOrZero(createdAtStr); err == nil {
		u.CreatedAt = t
	}
	if t, err := parseTimeOrZero(updatedAtStr); err == nil {
		u.UpdatedAt = t
	}

	return u, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func parseTimeOrZero(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %s", s)
}
