-- +goose Up
-- +goose StatementBegin

-- NEW-E fix: rename fraud_reasons (JSONB) -> fraud_reason (TEXT) to match Go domain.
-- The Go code (claim_repository.go, domain/claim.go) uses fraud_reason as a TEXT field.
-- The original schema had fraud_reasons (JSONB), causing all INSERT/SELECT/UPDATE to fail.

ALTER TABLE claims
  RENAME COLUMN fraud_reasons TO fraud_reason;

-- Change type from JSONB to TEXT (the Go domain stores a single reason string).
ALTER TABLE claims
  ALTER COLUMN fraud_reason TYPE TEXT USING (
    CASE
      WHEN fraud_reason IS NULL OR fraud_reason::text = '[]' THEN NULL
      ELSE fraud_reason->>0
    END
  );

-- Set a sensible default (NULL is fine for a reason string).
ALTER TABLE claims
  ALTER COLUMN fraud_reason DROP DEFAULT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE claims
  ALTER COLUMN fraud_reason TYPE JSONB USING (
    CASE
      WHEN fraud_reason IS NULL OR fraud_reason = '' THEN '[]'::jsonb
      ELSE jsonb_build_array(fraud_reason)
    END
  );

ALTER TABLE claims
  ALTER COLUMN fraud_reason SET DEFAULT '[]';

ALTER TABLE claims
  RENAME COLUMN fraud_reason TO fraud_reasons;

-- +goose StatementEnd
