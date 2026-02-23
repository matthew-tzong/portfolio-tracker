-- Portfolio Snapshot Tables (investments only — not net worth, cash, or liabilities)

-- Retention policy:
-- - Keep daily snapshots for the last 30 days from today only.
-- - When we delete a daily row that is the last day of a month, we first write that day's
--   snapshot as the monthly snapshot for that month, then delete the daily row.
-- - Cron writes daily; retention job rolls EOM to monthly and deletes old daily.

-- Daily portfolio snapshots (investments / brokerage value only)
CREATE TABLE IF NOT EXISTS daily_snapshots (
  id BIGSERIAL PRIMARY KEY,
  date DATE NOT NULL UNIQUE,
  portfolio_value_cents BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Daily holdings (positions per account per day)
CREATE TABLE IF NOT EXISTS daily_holdings (
  id BIGSERIAL PRIMARY KEY,
  date DATE NOT NULL,
  account_id TEXT NOT NULL,
  symbol TEXT NOT NULL,
  quantity NUMERIC(20, 8) NOT NULL,
  value_cents BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(date, account_id, symbol)
);

-- Monthly portfolio snapshots (end-of-month rollup; investments only).
-- One row per (month, account_id). Total portfolio for a month = SUM(portfolio_value_cents) for that month.
CREATE TABLE IF NOT EXISTS monthly_snapshots (
  id BIGSERIAL PRIMARY KEY,
  month DATE NOT NULL,
  account_id TEXT NOT NULL,
  portfolio_value_cents BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(month, account_id)
);

CREATE TABLE IF NOT EXISTS monthly_net_worth (
  id BIGSERIAL PRIMARY KEY,
  month DATE NOT NULL UNIQUE,
  net_worth_cents BIGINT NOT NULL,
  cash_cents BIGINT NOT NULL,
  investments_cents BIGINT NOT NULL,
  liabilities_cents BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

