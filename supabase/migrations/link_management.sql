-- Link Management Tables

-- Plaid Items: stores linked Plaid items (banks/credit cards)
CREATE TABLE IF NOT EXISTS plaid_items (
    id BIGSERIAL PRIMARY KEY,
    item_id TEXT NOT NULL UNIQUE,
    access_token TEXT NOT NULL,
    institution_id TEXT,
    institution_name TEXT,
    status TEXT NOT NULL DEFAULT 'OK',
    last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for quick lookups by item_id
CREATE INDEX IF NOT EXISTS idx_plaid_items_item_id ON plaid_items(item_id);

-- Plaid Accounts: stores accounts belonging to each Plaid item
CREATE TABLE IF NOT EXISTS plaid_accounts (
    id BIGSERIAL PRIMARY KEY,
    plaid_item_id TEXT NOT NULL REFERENCES plaid_items(item_id) ON DELETE CASCADE,
    account_id TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    mask TEXT,
    type TEXT NOT NULL,
    subtype TEXT,
    current_balance DECIMAL(15, 2) NOT NULL DEFAULT 0
);

-- Index for quick lookups by item
CREATE INDEX IF NOT EXISTS idx_plaid_accounts_item_id ON plaid_accounts(plaid_item_id);

-- Snaptrade User: stores the single Snaptrade user credentials for this app
-- (single-user app, so only one row expected)
CREATE TABLE IF NOT EXISTS snaptrade_user (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT NOT NULL UNIQUE,
    user_secret TEXT NOT NULL
);

-- Snaptrade Connections: stores brokerage connections from Snaptrade
CREATE TABLE IF NOT EXISTS snaptrade_connections (
    id BIGSERIAL PRIMARY KEY,
    conn_id TEXT NOT NULL UNIQUE,
    brokerage TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'OK',
    last_synced TIMESTAMPTZ
);

-- Index for quick lookups by connection id
CREATE INDEX IF NOT EXISTS idx_snaptrade_connections_conn_id ON snaptrade_connections(conn_id);
