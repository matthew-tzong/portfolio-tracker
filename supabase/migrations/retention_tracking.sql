-- Retention tracking tables

-- Monthly expense summary (aggregated by month and category)
CREATE TABLE IF NOT EXISTS monthly_expense_summary (
  id BIGSERIAL PRIMARY KEY,
  month DATE NOT NULL,
  category_id BIGINT NOT NULL REFERENCES categories(id),
  total_cents BIGINT NOT NULL DEFAULT 0,
  transaction_count INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(month, category_id)
);

-- Yearly expense summary (aggregated by year and category)
CREATE TABLE IF NOT EXISTS yearly_expense_summary (
  id BIGSERIAL PRIMARY KEY,
  year INTEGER NOT NULL,
  category_id BIGINT NOT NULL REFERENCES categories(id),
  total_cents BIGINT NOT NULL DEFAULT 0,
  transaction_count INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(year, category_id)
);

-- Yearly portfolio summary (aggregated by year and account)
CREATE TABLE IF NOT EXISTS yearly_portfolio_summary (
  id BIGSERIAL PRIMARY KEY,
  year INTEGER NOT NULL,
  account_id TEXT NOT NULL,
  portfolio_value_cents BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(year, account_id)
);
