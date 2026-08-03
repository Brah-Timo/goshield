-- Migration 005: Performance indexes for the claims table
-- Adds composite and partial indexes to speed up the most common query patterns:
--   • listing claims by company with status/risk filters
--   • fraud analytics dashboards (fraud_score, risk_level aggregations)
--   • date-range queries used by the area chart / stats endpoints

-- ── Composite index: list by company + status (most common list pattern) ─────
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_claims_company_status
  ON claims (company_id, status)
  WHERE status IN ('PENDING','PROCESSING','FLAGGED');

-- ── Composite index: company + risk_level (analytics filter) ─────────────────
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_claims_company_risk
  ON claims (company_id, risk_level, created_at DESC)
  WHERE risk_level IS NOT NULL;

-- ── Partial index: FLAGGED claims only — analyst review queue ─────────────────
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_claims_flagged
  ON claims (company_id, created_at DESC)
  WHERE status = 'FLAGGED';

-- ── Index: fraud_score DESC for "highest risk" dashboards ─────────────────────
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_claims_fraud_score
  ON claims (company_id, fraud_score DESC)
  WHERE fraud_score IS NOT NULL;

-- ── Index: created_at for time-series queries (area chart, stats endpoints) ───
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_claims_created_at
  ON claims (created_at DESC);

-- ── Index: updated_at for change-feed queries ─────────────────────────────────
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_claims_updated_at
  ON claims (updated_at DESC);

-- ── Index: policy_number text search prefix ────────────────────────────────────
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_claims_policy_number
  ON claims (policy_number text_pattern_ops);

-- ── BRIN index on created_at for partition pruning (if table is large) ────────
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_claims_created_brin
  ON claims USING BRIN (created_at);

COMMENT ON INDEX idx_claims_company_status   IS 'List claims: company+status filter (pending review queue)';
COMMENT ON INDEX idx_claims_company_risk     IS 'Analytics: company+risk_level+date range';
COMMENT ON INDEX idx_claims_flagged          IS 'Partial: flagged claims only (analyst review)';
COMMENT ON INDEX idx_claims_fraud_score      IS 'Dashboard: highest fraud score ordering';
COMMENT ON INDEX idx_claims_created_at       IS 'Time-series: created_at DESC for charts';
COMMENT ON INDEX idx_claims_updated_at       IS 'Change feed: updated_at DESC';
COMMENT ON INDEX idx_claims_policy_number    IS 'Search: prefix match on policy number';
COMMENT ON INDEX idx_claims_created_brin     IS 'BRIN: efficient range scans on large tables';
