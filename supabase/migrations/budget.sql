-- Budget tracker tables

-- Global budget for expense categories.
CREATE TABLE IF NOT EXISTS budgets (
  id BIGINT PRIMARY KEY,
  allocations JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Ensure a default empty budget row exists so reads always succeed.
INSERT INTO budgets (id, allocations)
VALUES (1, '{}'::jsonb)
ON CONFLICT (id) DO NOTHING;

