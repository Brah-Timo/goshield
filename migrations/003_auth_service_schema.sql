-- +goose Up
-- +goose StatementBegin

-- Add auth-service specific columns to users table
-- (migration 001 used full_name, oauth_id, is_active — we add the new names as aliases)

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS first_name  VARCHAR(100),
    ADD COLUMN IF NOT EXISTS last_name   VARCHAR(100),
    ADD COLUMN IF NOT EXISTS oauth_sub   VARCHAR(255),
    ADD COLUMN IF NOT EXISTS active      BOOLEAN DEFAULT TRUE;

-- Populate from existing columns
UPDATE users SET
    first_name = SPLIT_PART(full_name, ' ', 1),
    last_name  = CASE WHEN full_name LIKE '% %'
                      THEN SUBSTRING(full_name FROM POSITION(' ' IN full_name) + 1)
                      ELSE '' END,
    oauth_sub  = oauth_id,
    active     = is_active
WHERE first_name IS NULL;

-- Create refresh_tokens table for token rotation/revocation
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user    ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash    ON refresh_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires ON refresh_tokens(expires_at)
    WHERE revoked_at IS NULL;

-- Index for OAuth sub lookups
CREATE INDEX IF NOT EXISTS idx_users_oauth_sub ON users(oauth_provider, oauth_sub)
    WHERE oauth_sub IS NOT NULL;

-- Add password_hash column if auth service uses it directly
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS password_hash_v2 VARCHAR(255);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS refresh_tokens;

ALTER TABLE users
    DROP COLUMN IF EXISTS first_name,
    DROP COLUMN IF EXISTS last_name,
    DROP COLUMN IF EXISTS oauth_sub,
    DROP COLUMN IF EXISTS active,
    DROP COLUMN IF EXISTS password_hash_v2;

-- +goose StatementEnd
