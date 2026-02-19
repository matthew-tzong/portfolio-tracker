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

CREATE INDEX IF NOT EXISTS idx_daily_snapshots_date ON daily_snapshots(date DESC);

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

CREATE INDEX IF NOT EXISTS idx_daily_holdings_date ON daily_holdings(date DESC);
CREATE INDEX IF NOT EXISTS idx_daily_holdings_account_id ON daily_holdings(account_id);
CREATE INDEX IF NOT EXISTS idx_daily_holdings_symbol ON daily_holdings(symbol);
CREATE INDEX IF NOT EXISTS idx_daily_holdings_date_account ON daily_holdings(date, account_id);

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

CREATE INDEX IF NOT EXISTS idx_monthly_snapshots_month ON monthly_snapshots(month DESC);
CREATE INDEX IF NOT EXISTS idx_monthly_snapshots_account_id ON monthly_snapshots(account_id);
CREATE INDEX IF NOT EXISTS idx_monthly_snapshots_month_account ON monthly_snapshots(month, account_id);
