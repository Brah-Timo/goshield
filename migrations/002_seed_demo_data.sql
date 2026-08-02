-- +goose Up
-- +goose StatementBegin

-- Seed a demo company for development
INSERT INTO companies (id, name, domain, api_key, plan)
VALUES (
  '00000000-0000-0000-0000-000000000001',
  'Demo Insurance Co.',
  'demo.goshield.io',
  'gsh_demo_key_change_in_production',
  'enterprise'
) ON CONFLICT DO NOTHING;

-- Seed admin user (password: Admin@123)
INSERT INTO users (id, company_id, email, password_hash, full_name, role)
VALUES (
  '00000000-0000-0000-0000-000000000002',
  '00000000-0000-0000-0000-000000000001',
  'admin@demo.goshield.io',
  '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewLmj3DSHi9VIuqO', -- Admin@123
  'System Admin',
  'ADMIN'
) ON CONFLICT DO NOTHING;

-- Seed analyst user (password: Analyst@123)
INSERT INTO users (id, company_id, email, password_hash, full_name, role)
VALUES (
  '00000000-0000-0000-0000-000000000003',
  '00000000-0000-0000-0000-000000000001',
  'analyst@demo.goshield.io',
  '$2a$12$R7/LdNFNKhPqPNEMvr5FO.q5aQOxT5fz3kVELZ8hS2UiHV4t.5YAG', -- Analyst@123
  'Jane Analyst',
  'ANALYST'
) ON CONFLICT DO NOTHING;

-- Seed demo rules
INSERT INTO rules (company_id, name, description, condition, action, priority)
VALUES
  (
    '00000000-0000-0000-0000-000000000001',
    'High Amount Auto-Flag',
    'Automatically flag claims over $1,000,000',
    '{"field": "amount", "op": "gt", "value": 1000000}',
    'AUTO_FLAG',
    10
  ),
  (
    '00000000-0000-0000-0000-000000000001',
    'Very High AI Score Alert',
    'Send alert when fraud score > 0.95 and amount > $100,000',
    '{"and": [{"field": "fraud_score", "op": "gt", "value": 0.95}, {"field": "amount", "op": "gt", "value": 100000}]}',
    'NOTIFY',
    20
  )
ON CONFLICT DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM rules WHERE company_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM users WHERE company_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM companies WHERE id = '00000000-0000-0000-0000-000000000001';
-- +goose StatementEnd
