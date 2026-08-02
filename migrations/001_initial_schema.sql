-- +goose Up
-- +goose StatementBegin

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- for full-text search

-- ── Enums ────────────────────────────────────────────────────────────────────

CREATE TYPE user_role AS ENUM ('ADMIN', 'ANALYST', 'VIEWER');

CREATE TYPE claim_status AS ENUM (
  'PENDING',
  'PROCESSING',
  'FLAGGED',
  'APPROVED',
  'REJECTED',
  'MORE_INFO'
);

CREATE TYPE claim_type AS ENUM (
  'HEALTH',
  'CAR',
  'PROPERTY',
  'LIFE',
  'TRAVEL',
  'OTHER'
);

-- ── Companies (Multi-tenancy) ─────────────────────────────────────────────────

CREATE TABLE companies (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name        VARCHAR(255) NOT NULL,
  domain      VARCHAR(255) UNIQUE,
  api_key     VARCHAR(100) UNIQUE,
  plan        VARCHAR(50) DEFAULT 'standard',
  is_active   BOOLEAN DEFAULT TRUE,
  settings    JSONB DEFAULT '{}',
  created_at  TIMESTAMPTZ DEFAULT NOW(),
  updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_companies_api_key ON companies(api_key) WHERE api_key IS NOT NULL;

-- ── Users ────────────────────────────────────────────────────────────────────

CREATE TABLE users (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id   UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  email        VARCHAR(255) NOT NULL,
  password_hash VARCHAR(255),                -- NULL for OAuth-only users
  full_name    VARCHAR(255),
  avatar_url   TEXT,
  role         user_role NOT NULL DEFAULT 'ANALYST',
  oauth_provider VARCHAR(50),               -- google | microsoft | NULL
  oauth_id     VARCHAR(255),
  refresh_token_hash VARCHAR(255),
  is_active    BOOLEAN DEFAULT TRUE,
  last_login_at TIMESTAMPTZ,
  created_at   TIMESTAMPTZ DEFAULT NOW(),
  updated_at   TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(company_id, email)
);

CREATE INDEX idx_users_company ON users(company_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_oauth ON users(oauth_provider, oauth_id) WHERE oauth_id IS NOT NULL;

-- ── Claims (Partitioned by month) ────────────────────────────────────────────

CREATE TABLE claims (
  id              UUID NOT NULL DEFAULT gen_random_uuid(),
  user_id         UUID NOT NULL,
  company_id      UUID NOT NULL,
  policy_number   VARCHAR(100) NOT NULL,
  claim_type      claim_type NOT NULL DEFAULT 'OTHER',
  amount          NUMERIC(15,2) NOT NULL CHECK (amount > 0),
  incident_date   DATE,
  doc_url         TEXT,
  doc_key         TEXT,                      -- storage object key
  description     TEXT,

  -- AI Analysis Results
  fraud_score     NUMERIC(4,3) DEFAULT 0.0 CHECK (fraud_score >= 0 AND fraud_score <= 1),
  fraud_reasons   JSONB DEFAULT '[]',
  risk_factors    JSONB DEFAULT '[]',
  ai_analyzed_at  TIMESTAMPTZ,

  -- Workflow
  status          claim_status NOT NULL DEFAULT 'PENDING',
  analyst_id      UUID,
  analyst_notes   TEXT,
  decided_at      TIMESTAMPTZ,

  -- Audit
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create partitions for the next 12 months
CREATE TABLE claims_2024_01 PARTITION OF claims
  FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE claims_2024_q2 PARTITION OF claims
  FOR VALUES FROM ('2024-04-01') TO ('2024-07-01');
CREATE TABLE claims_2024_h2 PARTITION OF claims
  FOR VALUES FROM ('2024-07-01') TO ('2025-01-01');
CREATE TABLE claims_2025_h1 PARTITION OF claims
  FOR VALUES FROM ('2025-01-01') TO ('2025-07-01');
CREATE TABLE claims_2025_h2 PARTITION OF claims
  FOR VALUES FROM ('2025-07-01') TO ('2026-01-01');
CREATE TABLE claims_default PARTITION OF claims DEFAULT;

-- Indexes (created on parent, propagated to partitions)
CREATE INDEX idx_claims_status    ON claims(status);
CREATE INDEX idx_claims_fraud     ON claims(fraud_score DESC);
CREATE INDEX idx_claims_company   ON claims(company_id);
CREATE INDEX idx_claims_user      ON claims(user_id);
CREATE INDEX idx_claims_created   ON claims(created_at DESC);
CREATE INDEX idx_claims_policy    ON claims(policy_number);
CREATE INDEX idx_claims_trgm      ON claims USING GIN (description gin_trgm_ops);

-- ── Audit Log ────────────────────────────────────────────────────────────────

CREATE TABLE audit_logs (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id  UUID NOT NULL,
  user_id     UUID,
  action      VARCHAR(100) NOT NULL,     -- CLAIM_APPROVED, USER_CREATED, etc.
  resource    VARCHAR(100) NOT NULL,
  resource_id UUID,
  old_value   JSONB,
  new_value   JSONB,
  ip_address  INET,
  user_agent  TEXT,
  created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_audit_company ON audit_logs(company_id, created_at DESC);
CREATE INDEX idx_audit_resource ON audit_logs(resource, resource_id);

-- ── API Keys ─────────────────────────────────────────────────────────────────

CREATE TABLE api_keys (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id  UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  name        VARCHAR(100) NOT NULL,
  key_hash    VARCHAR(255) NOT NULL UNIQUE,
  key_prefix  VARCHAR(20) NOT NULL,      -- shown to user: "gsh_abc12..."
  scopes      TEXT[] DEFAULT '{}',
  is_active   BOOLEAN DEFAULT TRUE,
  last_used_at TIMESTAMPTZ,
  expires_at  TIMESTAMPTZ,
  created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_api_keys_company ON api_keys(company_id);
CREATE INDEX idx_api_keys_hash    ON api_keys(key_hash);

-- ── Rule Engine ───────────────────────────────────────────────────────────────

CREATE TABLE rules (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id  UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  name        VARCHAR(255) NOT NULL,
  description TEXT,
  condition   JSONB NOT NULL,            -- {"field":"amount","op":"gt","value":1000000}
  action      VARCHAR(50) NOT NULL,      -- AUTO_REJECT | AUTO_FLAG | NOTIFY
  priority    INT DEFAULT 100,
  is_active   BOOLEAN DEFAULT TRUE,
  created_at  TIMESTAMPTZ DEFAULT NOW(),
  updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_rules_company ON rules(company_id) WHERE is_active = TRUE;

-- ── Triggers: updated_at ─────────────────────────────────────────────────────

CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$;

CREATE TRIGGER trg_companies_updated_at BEFORE UPDATE ON companies
  FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_users_updated_at BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_rules_updated_at BEFORE UPDATE ON rules
  FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ── Materialized View: Daily Fraud Stats ──────────────────────────────────────

CREATE MATERIALIZED VIEW daily_fraud_stats AS
SELECT
  date_trunc('day', created_at) AS day,
  company_id,
  COUNT(*) AS total_claims,
  COUNT(*) FILTER (WHERE fraud_score >= 0.8) AS flagged_claims,
  COUNT(*) FILTER (WHERE status = 'APPROVED') AS approved_claims,
  COUNT(*) FILTER (WHERE status = 'REJECTED') AS rejected_claims,
  ROUND(AVG(fraud_score)::NUMERIC, 3) AS avg_fraud_score,
  SUM(amount) AS total_amount,
  SUM(amount) FILTER (WHERE fraud_score >= 0.8) AS flagged_amount
FROM claims
GROUP BY 1, 2
WITH DATA;

CREATE UNIQUE INDEX ON daily_fraud_stats(day, company_id);

-- +goose StatementEnd

