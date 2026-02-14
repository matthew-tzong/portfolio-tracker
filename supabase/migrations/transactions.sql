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

CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_name ON categories(name);
CREATE INDEX IF NOT EXISTS idx_categories_plaid_name ON categories(plaid_name);

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

-- =============================================================================
-- CATEGORY RULES: how we identify Rent, Fidelity, Venmo, etc.
-- =============================================================================
--
-- Each transaction from Plaid has (same for bank and credit card):
--   - name: the transaction description from the financial institution (lightly cleaned by Plaid)
--   - merchant_name: Plaid's enriched/standardized merchant name (often null for ACH, transfers)
--
-- MATCHING FLOW (for each transaction, in order):
--
--   Step 1 — Category rules
--   We take the transaction's name and merchant_name (lowercased) and check each
--   rule in id order. If name OR merchant_name CONTAINS the rule's match_string
--   (case-insensitive), we assign that rule's category and stop.
--
--   Examples:
--     • name = "VENMO PAYMENT TO JOHN"     → contains "venmo" → category Venmo
--     • name = "ACH TRANSFER FIDELITY INV"  → contains "fidelity" → category Investments
--     • name = "MONTHLY RENT PAYMENT"       → contains "rent" → category Rent and Utilities
--
--   Step 2 — Plaid's category (if no rule matched)
--   We use Plaid's primary category (e.g. "Food and Drink", "Transfer") and map
--   it to our categories via categories.plaid_name.
--
--   Step 3 — Uncategorized (if still no match)
--   We assign Uncategorized.
--
-- So: we identify Venmo/Fidelity/Rent by looking for those words (or your custom
-- match_string) inside the transaction name or merchant name. Plaid often labels
-- ACH transfers as "Transfer", so without rules they'd all look the same; rules
-- reclassify using the actual payee text.
--
-- Rules are added only by editing the DB (no API to create rules). Example:
--   INSERT INTO category_rules (match_string, category_id) SELECT 'Landlord Name', id FROM categories WHERE name = 'Rent and Utilities' LIMIT 1;
-- =============================================================================
CREATE TABLE IF NOT EXISTS category_rules (
  id BIGSERIAL PRIMARY KEY,
  match_string TEXT NOT NULL,
  category_id BIGINT NOT NULL REFERENCES categories(id)
);
CREATE INDEX IF NOT EXISTS idx_category_rules_match ON category_rules(LOWER(match_string));

-- Matching is case-insensitive (backend uses LOWER), so one rule per concept is enough.
-- Venmo: Plaid often categorizes as Transfer/Payment; match on name/merchant.
INSERT INTO category_rules (match_string, category_id)
SELECT 'venmo', id FROM categories WHERE name = 'Venmo' LIMIT 1;
-- Investments: ACH to Fidelity often shows as Transfer; match on name/merchant.
INSERT INTO category_rules (match_string, category_id)
SELECT 'fidelity', id FROM categories WHERE name = 'Investments' LIMIT 1;
-- Rent and Utilities: one category for both. This rule matches "rent" in name/merchant (e.g. "Monthly Rent").
-- Keep this rule. If your bank shows only the landlord or utility company name (no "rent" in the text),
-- ADD another rule—do not replace this one. Example (add a row, keep the 'rent' row):
--   INSERT INTO category_rules (match_string, category_id)
--   SELECT 'Your Landlord Or Utility Payee Name', id FROM categories WHERE name = 'Rent and Utilities' LIMIT 1;
-- INSERT INTO category_rules (match_string, category_id)
-- SELECT 'rent', id FROM categories WHERE name = 'Rent and Utilities' LIMIT 1;

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

CREATE INDEX IF NOT EXISTS idx_transactions_plaid_account_id ON transactions(plaid_account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(date);
CREATE INDEX IF NOT EXISTS idx_transactions_category_id ON transactions(category_id);
CREATE INDEX IF NOT EXISTS idx_transactions_date_category ON transactions(date, category_id);
