# Portfolio Tracker — TODO (Vertical Slices)

**Single-user only.** This app is for one owner. No multi-user or multi-tenant features.

**Stack:** Go backend (`backend/`) + React frontend (`frontend/`), Supabase (DB + Auth), Vercel (host React and/or cron that calls Go). Same repo; Go serves React for one deploy.

Each slice is a **vertical slice**: a complete end-to-end piece of value you can run and verify, from UI to API to data. Order is chosen so early slices give a usable app quickly.

---

## Slice 1: Repo, app shell, and auth

**Goal:** One repo with Go backend and React frontend; you can sign in (Supabase) and only you can access the app (single Supabase user, no public sign-up).

- [x] **1.1 Repo and stack**
  - [x] Monorepo: `backend/` (Go module), `frontend/` (React — Vite or CRA, TypeScript).
  - [x] Backend: Go HTTP server (e.g. Chi or Echo), CORS for frontend origin. Frontend: ESLint + Prettier.
  - [x] `.gitignore`: `node_modules`, `.env`, `.env.local`, `backend/.env`, binaries, logs. No `.env*` with secrets.
- [x] **1.2 Supabase**
  - [x] Create Supabase project; get URL, anon key, and JWT secret (for Go to verify tokens).
  - [x] Enable Auth (email/password provider); disable public sign-ups.
  - [x] Create a single Supabase user (you) and record their email for backend locking.
  - [x] Frontend: add `@supabase/supabase-js`; create client in `frontend/src/lib/supabase.ts` (or similar).
- [x] **1.3 Auth UI and protection (React)**
  - [x] Sign-in page (no public sign-up); wire to Supabase Auth.
  - [x] Protected layout: redirect unauthenticated users to sign-in; after sign-in, show minimal dashboard placeholder.
  - [x] React app sends Supabase JWT (e.g. in `Authorization: Bearer <access_token>`) to Go API for protected routes.
- [x] **1.4 Go API: JWT validation**
  - [x] Middleware or helper in Go: verify Supabase JWT (ES256) via JWKS discovery from `SUPABASE_URL`, extract `sub` (user id), and enforce `ALLOWED_USER_EMAIL` for single-user access. Reject requests without valid token. Use for all protected endpoints.

**Done when:** You can run backend and frontend locally, sign in in the React app, and see a protected dashboard; unauthenticated users cannot access it; Go API rejects requests without a valid Supabase JWT.

---

## Slice 2: Link management (Plaid + Snaptrade)

**Goal:** You can add and list connections (Plaid and Snaptrade), see their status, and have a path to reconnect when broken.

- [x] **2.1 Plaid Link — connect**
  - [x] **Go**: endpoint to create Plaid link token (`GET /api/plaid/link-token`); endpoint to exchange public token and store item (`POST /api/plaid/exchange-token`). Uses a minimal custom Plaid HTTP client (`internal/plaid/client.go`) and stores data in `plaid_items` and `plaid_accounts` in Supabase (via Go). Never logs `access_token`.
  - [x] **React**: embed Plaid Link via the browser script; get link token from Go API; on success send public token (plus institution metadata) to Go; show success banner on the dashboard.
- [x] **2.2 Snaptrade Connect**
  - [x] **Go**: endpoints to create/ensure a Snaptrade user and generate a connection portal URL (`POST /api/snaptrade/connect-url`); endpoint to sync connections into `snaptrade_connections` (`POST /api/snaptrade/sync-connections`) using the official SnapTrade Go SDK wrapper.
  - [x] **React**: `SnaptradeConnectSection` opens the Snaptrade Connect portal (URL from Go) in a new tab; after opening, a dashboard callback triggers a sync and shows a success banner.
- [x] **2.3 Link management page (React)**
  - [x] Page listing all Plaid items and Snaptrade connections at `/links`. Data from Go API (`GET /api/links`), which reads from Supabase only.
  - [x] **Go**: endpoint to list items/connections and their basic status (currently a simple `"OK"` string for Snaptrade; more detailed reconnect states can be added later).

**Done when:** You can link a bank/CC (Plaid) and a brokerage via Snaptrade in React, see them on a link management page (data from Go API), and remove/sync connections.

---

## Slice 3: Accounts and net worth (current balances)

**Goal:** Dashboard shows all linked accounts and a single “net worth” number (sum of balances; no historical snapshots yet).

- [ ] **3.1 Fetch and store balances (Go)**
  - [ ] Go: for each Plaid item, call Plaid balances/accounts; upsert `plaid_accounts` in Supabase. For Snaptrade, fetch account balances and store or derive cash per account.
  - [ ] Endpoint(s) to trigger sync and/or to return current accounts and balances (e.g. `GET /api/accounts`).
- [ ] **3.2 Net worth calculation (Go)**
  - [ ] Compute net worth = cash + investment value − liabilities. Put logic in `backend/internal/` or `backend/pkg/`; use from API handler.
- [ ] **3.3 Dashboard UI (React)**
  - [ ] Page or section: fetch accounts and net worth from Go API; list accounts (name, type, balance, mask); display net worth (single number or breakdown: cash, investments, liabilities).
- [ ] **3.4 Red alert and reconnect (Plaid + Snaptrade)**
  - [ ] React: when status is broken, show red alert and “Reconnect” button on the connections/dashboard UI.
  - [ ] Reconnect: Plaid = Link update or delete item (Go: `POST /api/plaid/remove-item`) + new Link; Snaptrade = open Connect again; Go updates stored connection.
- [ ] **3.5 Plaid item rotation (optional)**
  - [ ] Go: endpoint to remove item (`/item/remove`); React “Re-link” flow: call remove then open Link for the same institution; Go saves new item. Keep transactions/accounts keyed by institution/account so re-link preserves history.

**Done when:** After linking, you see all accounts and one net worth number from the Go API when you load or refresh the dashboard.

---

## Slice 4: Transactions and expense tracker

**Goal:** Plaid transactions use **webhooks + a single nightly cron**. **Webhooks** are primary: on `SYNC_UPDATES_AVAILABLE`, call `/transactions/sync` and upsert. **Nightly cron** is a safety check: run cursor-based sync to ensure the cursor is at the very end of the “book” (no missed transactions). Each transaction counts as an expense (category) and feeds overall liabilities. You can view/filter by month and category. Slice 5 uses these categories for monthly budget progress.

- [ ] **4.1 Plaid webhook ingests transactions (Go)**
  - [ ] Go: webhook `POST /api/webhooks/plaid` — verify signature; on `SYNC_UPDATES_AVAILABLE` call Plaid `/transactions/sync` (cursor-based) for that item and upsert into `transactions`. Deploy Go so Plaid can reach the URL.
  - [ ] Each stored transaction is treated as an expense (or credit/refund if positive): contributes to its category and to overall liabilities.
- [ ] **4.2 Nightly Plaid safety check (Go, in Slice 7 cron)**
  - [ ] In the same nightly cron: for each Plaid item, run cursor-based `/transactions/sync` and advance to the very end (ensure cursor is at end of book; catch anything webhooks might have missed). Persist any new/updated transactions.
- [ ] **4.3 Categorization (Go)**
  - [ ] Store Plaid category on each transaction; map to `categories` or category_resolved so Slice 5 can sum spent per category for the month.
- [ ] **4.4 Expense tracker UI (React)**
  - [ ] Go: endpoint to list transactions (e.g. `GET /api/transactions?month=...&category=...`).
  - [ ] React: page to pick month, filter by category, search; show date, amount, merchant, category; sort by date.

**Done when:** Plaid webhook + nightly cron (cursor safety check) keep transactions in sync; they feed expense categories and liabilities; React lists/filters them. Slice 5 uses the same categories for monthly budget.

---

## Slice 5: Budget tracker

**Goal:** Set a monthly budget by category and see progress (spent vs budget) using expense data.

- [ ] **5.1 Budget model and API (Go)**
  - [ ] Supabase: table `budgets` (month, allocations jsonb). Go: endpoints to create/update and read budget for a month; helper to compute “spent per category” from `transactions` for that month.
- [ ] **5.2 Budget UI (React)**
  - [ ] Page: pick month; set budget per category (form); fetch budget and spent-from-Go. Show progress (bar or %) per category; highlight over-budget.

**Done when:** You can create/edit a monthly budget via React and see spent vs budget per category (data from Go API).

---

## Slice 6: Portfolio view (holdings, daily + monthly history)

**Goal:** See Fidelity (Snaptrade) positions (live) and **daily** portfolio snapshots for recent time (current month + previous month), and **monthly** values for older months. Example: in April we store all daily snapshots for March and April, and only monthly data for January and February. After a month ends, we delete that month’s daily rows and keep only its monthly value (Slice 9).

- [ ] **6.1 Current holdings (Go + Snaptrade)**
  - [ ] Go: fetch Snaptrade positions per account (symbol, quantity, value); endpoint to return current holdings and account totals for the portfolio UI.
- [ ] **6.2 Daily and monthly portfolio data (Go + Supabase)**
  - [ ] Supabase: `daily_snapshots` (date, net_worth_cents, portfolio_value_cents, cash_cents, liabilities_cents) and `daily_holdings` (date, account_id, symbol, quantity, value_cents) for recent months; `monthly_snapshots` and `monthly_holdings` for older months (after daily for that month is deleted and rolled up).
  - [ ] Go: endpoints to return (1) daily series for date ranges where we have daily data (current + previous month), (2) monthly series for older months; net worth, portfolio value, per-account, per-symbol.
- [ ] **6.3 Portfolio UI (React)**
  - [ ] Page: current positions from Go (live Snaptrade); charts show **daily** portfolio/net worth for recent months and **monthly** for older months (per stock, per account, overall).

**Done when:** React shows current Fidelity positions and portfolio history: daily for recent two months, monthly for older months.

---

## Slice 7: Nightly cron — Plaid safety check + Snaptrade + daily snapshots

**Goal:** One **nightly cron** at 11pm that: (1) **Plaid safety check** — for each Plaid item, run cursor-based `/transactions/sync` to the very end so the cursor is at end of book (catch anything webhooks missed). (2) **Snaptrade** — fetch holdings/balances, refresh connection status, and write **daily** portfolio snapshots for today (`daily_snapshots`, `daily_holdings`). (3) At end of month (or first run of new month), write that month’s rollup to `monthly_snapshots` and `monthly_holdings`. We **do** store daily snapshots for portfolio, but after a month we delete that month’s daily rows and keep only the monthly value (Slice 9). So we keep daily for current month + previous month only (e.g. in April: daily for March and April, monthly for January and February).

- [ ] **7.1 Snapshot schema (Go + Supabase)**
  - [ ] Supabase: `daily_snapshots` (date, net_worth_cents, portfolio_value_cents, cash_cents, liabilities_cents); `daily_holdings` (date, account_id, symbol, quantity, value_cents); `monthly_snapshots` (month, …); `monthly_holdings` (month, account_id, symbol, value_cents).
- [ ] **7.2 Cron job — Plaid safety + Snaptrade + daily snapshots (Go)**
  - [ ] Go: endpoint `POST /api/cron/daily-sync` that (1) **Plaid:** for each item, run cursor-based `/transactions/sync` to the end and upsert any new/updated transactions; (2) **Snaptrade:** fetch holdings/balances, refresh connection status, compute today’s net worth and per-holding values, insert into `daily_snapshots` and `daily_holdings`; (3) if today is last day of month or first run of new month, compute that month’s rollup and insert into `monthly_snapshots` and `monthly_holdings`. Protect with `CRON_SECRET`.
- [ ] **7.3 Schedule (use whatever is free)**
  - [ ] Use a **free** scheduler: **GitHub Actions** or **Vercel Cron**. Call `POST https://<your-go-api>/api/cron/daily-sync` with `CRON_SECRET` at 11pm.
- [ ] **7.4 API for charts**
  - [ ] Go: endpoint(s) to read daily snapshots/holdings for recent date ranges and monthly for older (for Slice 8).

**Done when:** At 11pm, cron runs Plaid cursor safety check + Snaptrade fetch + daily portfolio snapshots; at month-end, monthly rollup is written; daily data is kept for current + previous month, then rolled to monthly (Slice 9).

---

## Slice 8: Graphs and visualization

**Goal:** Charts for net worth over time, portfolio value over time, per-holding performance over time, expenses by category (monthly), and budget progress.

- [ ] **8.1 Go API for chart data**
  - [ ] Endpoints to return: **daily** snapshots/holdings for recent months (current + previous), **monthly** for older; net worth, portfolio value, per-symbol, per-account over time; transactions aggregated by category; budget + spent per category. All auth-protected.
- [ ] **8.2 Net worth and portfolio over time (React)**
  - [ ] Fetch daily snapshot data for recent range and monthly for older from Go; chart net_worth and portfolio_value (and optionally cash/liabilities) over time (line or area). Use chart lib (e.g. Recharts, Chart.js).
- [ ] **8.3 Per-stock and per-account performance over time (React)**
  - [ ] For chosen symbol or account, fetch daily holdings for recent months and monthly for older from Go; chart value over time.
- [ ] **8.4 Expenses by category (React)**
  - [ ] By month: fetch aggregated-by-category from Go; bar or donut chart.
- [ ] **8.5 Budget progress (React)**
  - [ ] Per category: bar or donut showing spent vs budget for selected month (data from Go).
- [ ] **8.6 Dashboard integration**
  - [ ] Add charts to dashboard or “Insights” / “Graphs” page in React; keep chart components in `frontend/src/components/`.

**Done when:** You can view net worth, portfolio, per-holding, expenses, and budget as charts (React + Go API).

---

## Slice 9: Export and retention

**Goal:** Export each month’s data as CSV before deleting it. Apply retention rules so daily data is pruned to monthly, then monthly to yearly, keeping DB small while preserving long-term summaries.

**Retention rules**

- **CC daily transactions:** Delete after **3 months**. Before deleting, keep **monthly overall** (e.g. total spent per category for the month). Export that month’s transactions as CSV before delete.
- **CC monthly data:** Keep **1 year**; then delete. Keep **yearly overall** for CC (e.g. total balance or total spent per year). Export month as CSV before deleting.
- **Portfolio daily:** We store **daily** snapshots for the **current month and previous month** only (e.g. in April: daily for March and April). After a month ends, **delete all daily** snapshots/holdings for that month and keep only the **monthly** value (one row in `monthly_snapshots` and rows in `monthly_holdings`). So in April we have daily for March + April, and monthly for January, February. Export that month’s daily data as CSV before deleting.
- **Portfolio monthly** (`monthly_snapshots`, `monthly_holdings`): Keep **1 year**; then delete and keep **yearly overall**. Export that month’s data as CSV before deleting.
- **Export-before-delete:** For any month being pruned (CC or portfolio), generate and offer download of that month’s CSV first, then delete the rows.

- [ ] **9.1 Schema for rollups (Go + Supabase)**
  - [ ] Tables: `monthly_expense_summary` (month, category, total_cents) for CC; `monthly_snapshots` and `monthly_holdings` (Slice 7) for portfolio; `yearly_cc_summary`, `yearly_portfolio_summary` for yearly overall. Populate yearly when pruning monthly after 1 year.
- [ ] **9.2 CSV export (Go + React)**
  - [ ] Go: auth-protected endpoint(s) to export a given month’s transactions or snapshot/holdings as CSV (stream or download). Used by retention job and optionally by user (“Export this month”).
  - [ ] React: button or page to trigger export (e.g. “Export month” or “Export last 12 months”) that calls Go and downloads the file.
- [ ] **9.3 Retention job (Go)**
  - [ ] Go: cron-invoked or scheduled job that: (1) **Portfolio daily:** for any month that is older than “previous month” (e.g. when in April, months before March), ensure that month’s daily rows are rolled up: export that month’s daily data to CSV, insert/update monthly_snapshots and monthly_holdings for that month, then delete all daily_snapshots and daily_holdings for that month. So we only keep daily for current + previous month. (2) **CC:** prune transactions older than 3 months (after monthly summary); prune CC monthly older than 1 year (keep yearly). (3) **Portfolio monthly:** prune older than 1 year (keep yearly overall). Export before delete in all cases. Run daily or weekly.
- [ ] **9.4 Document rules**
  - [ ] Document retention rules and export-before-delete in README or runbook.

**Done when:** Each month’s data is exported as CSV before deletion; portfolio daily kept only for current + previous month (older months rolled to monthly); CC daily → 3 months then monthly (1 year) then yearly; portfolio monthly → 1 year then yearly overall; DB stays within free tier.

---

## Slice 10: CI/CD and deploy

**Goal:** Lint and test both Go and React on push; deploy React to Vercel and Go to a host; cron and webhook working.

- [ ] **10.1 Tests**
  - [ ] **Go**: unit tests for categorization, net-worth calculation, snapshot aggregation (mocked Plaid/Snaptrade). `go test ./...`
  - [ ] **React**: optional unit tests for critical UI or helpers; e.g. Jest or Vitest in `frontend/`.
- [ ] **10.2 CI (GitHub Actions)**
  - [ ] On push/PR: checkout repo; run Go lint and tests (`backend/`); run React install, lint, typecheck, tests (`frontend/`). No secrets in workflow; optional Vercel deploy for frontend on main.
- [ ] **10.3 Deploy**
  - [ ] **React**: connect `frontend/` (or repo with root dir) to Vercel; set env vars (e.g. Supabase URL, anon key, API base URL for Go — use `VITE_*` if using Vite). Build command and output dir for your React setup.
  - [ ] **Go**: deploy backend to Fly.io, Railway, or similar. Set env: Supabase URL/key, JWT secret, Plaid, Snaptrade, CRON_SECRET, webhook secret. Ensure Go API URL is HTTPS so Plaid webhook and Vercel cron can call it.
  - [ ] **Plaid**: webhook URL `https://<your-go-api>/api/webhooks/plaid`; nightly cron runs Plaid cursor-based sync as safety check (cursor to end of book). **Cron**: single nightly job calls `https://<your-go-api>/api/cron/daily-sync` with CRON_SECRET at 11pm (Plaid safety + Snaptrade + daily portfolio snapshots).
- [ ] **10.4 CORS**
  - [ ] Go backend allows frontend origin (Vercel preview and prod URLs) in CORS so React can call the API.

**Done when:** Push runs CI for Go and React; React is live on Vercel and Go is live on Fly/Railway; Plaid webhook + nightly cron (cursor safety) keep transactions in sync; 11pm cron runs Plaid safety + Snaptrade + daily portfolio snapshots; app is usable end-to-end.

---

## Summary

| Slice | Focus |
|-------|--------|
| 1 | Repo, app shell, auth |
| 2 | Link management (Plaid + Snaptrade, status, reconnect) |
| 3 | Accounts and net worth (current) |
| 4 | Transactions and expense tracker |
| 5 | Budget tracker |
| 6 | Portfolio view (daily for recent 2 months + monthly for older) |
| 7 | Nightly cron: Plaid cursor safety + Snaptrade + daily portfolio snapshots |
| 8 | Graphs |
| 9 | Export and retention |
| 10 | CI/CD and deploy |

Complete slices in order for a steady, shippable progression. Each slice should leave the codebase clean and maintainable per `.cursorrules`.

**Stack reminder:** Go backend (`backend/`), React frontend (`frontend/`), Supabase (DB + Auth), Vercel (React hosting + cron that calls Go). Go API must validate Supabase JWT and expose endpoints for frontend and for webhook/cron.
