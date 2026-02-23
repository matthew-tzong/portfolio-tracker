-- Transactions and expense tracker tables

-- Add cursor and boolean flag for new transactions pending
ALTER TABLE plaid_items
  ADD COLUMN IF NOT EXISTS transactions_cursor TEXT,
  ADD COLUMN IF NOT EXISTS new_transactions_pending BOOLEAN NOT NULL DEFAULT false;

-- Categories for expense tracking
CREATE TABLE IF NOT EXISTS categories (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  plaid_name TEXT,
  expense BOOLEAN NOT NULL DEFAULT true
);

-- Transaction categories (plaid_name = Plaid primary category for auto-mapping; NULL = use category_rules or leave Uncategorized)
INSERT INTO categories (name, plaid_name, expense) VALUES
  ('Uncategorized', NULL, true),
  ('Food and Drink', 'Food and Drink', true),
  ('Shops', 'Shops', true),
  ('Travel', 'Travel', true),
  ('Transfer', 'Transfer', false),
  ('Payment', 'Payment', false),
  ('Recreation', 'Recreation', true),
  ('Service', 'Service', true),
  ('Bank Fees', 'Bank Fees', true),
  ('Rent and Utilities', 'Rent and Utilities', true),
  ('Healthcare', 'Healthcare', true),
  ('Personal', 'Personal', true),
  ('Education', 'Education', true),
  ('Government and Non-Profit', 'Government and Non-Profit', true),
  ('Income', 'Income', false),
  ('Venmo', NULL, true),
  ('Investments', NULL, false)
ON CONFLICT (name) DO NOTHING;

-- Each transaction from Plaid has a name (financial institution) and merchant name 
-- We match first by category rule (if it contains the transaction name/merchant name).
-- If no rule matched, we default to the Plaid category (if it exists) or Uncategorized.

CREATE TABLE IF NOT EXISTS category_rules (
  id BIGSERIAL PRIMARY KEY,
  match_string TEXT NOT NULL,
  category_id BIGINT NOT NULL REFERENCES categories(id)
);

-- Matching is case-insensitive (backend uses LOWER), so one rule per concept is enough.

-- Venmo: Plaid often categorizes as Transfer/Payment; match on name/merchant.
INSERT INTO category_rules (match_string, category_id)
SELECT 'venmo', id FROM categories WHERE name = 'Venmo' LIMIT 1;

-- Investments: ACH to Fidelity often shows as Transfer; match on name/merchant.
INSERT INTO category_rules (match_string, category_id)
SELECT 'fidelity', id FROM categories WHERE name = 'Investments' LIMIT 1;

-- Rent and Utilities: one category for both. This rule matches "rent" in name/merchant (e.g. "Monthly Rent").
--   INSERT INTO category_rules (match_string, category_id)
--   SELECT 'Your Landlord Or Utility Payee Name', id FROM categories WHERE name = 'Rent and Utilities' LIMIT 1;

-- Credit card payments: detect these provideres and map to Transfer category (expense=false).
INSERT INTO category_rules (match_string, category_id)
SELECT 'discover', id FROM categories WHERE name = 'Transfer' LIMIT 1;

-- Plaid transactions
CREATE TABLE IF NOT EXISTS transactions (
  id BIGSERIAL PRIMARY KEY,
  plaid_account_id TEXT NOT NULL,
  plaid_transaction_id TEXT NOT NULL UNIQUE,
  date DATE NOT NULL,
  amount_cents BIGINT NOT NULL,
  name TEXT NOT NULL,
  merchant_name TEXT,
  category_id BIGINT REFERENCES categories(id),
  pending BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);