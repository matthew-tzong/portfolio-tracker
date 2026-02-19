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

**Goal:** Dashboard shows all linked accounts and a single "net worth" number (sum of balances; no historical snapshots yet).

- [x] **3.1 Fetch and store balances (Go)**
  - [x] Go: for each Plaid item, call Plaid balances/accounts; upsert `plaid_accounts` in Supabase. For Snaptrade, fetch account balances and store or derive cash per account.
  - [x] Endpoint(s) to trigger sync and/or to return current accounts and balances (e.g. `GET /api/accounts`).
- [x] **3.2 Net worth calculation (Go)**
  - [x] Compute net worth = cash + investment value − liabilities. Put logic in `backend/internal/` or `backend/pkg/`; use from API handler.
- [x] **3.3 Dashboard UI (React)**
  - [x] Page or section: fetch accounts and net worth from Go API; list accounts (name, type, balance, mask); display net worth (single number or breakdown: cash, investments, liabilities).
- [x] **3.4 Red alert and reconnect (Plaid + Snaptrade)**
  - [x] React: when status is broken, show red alert and "Reconnect" button on the connections/dashboard UI.
  - [x] Reconnect: Plaid = Link update or delete item (Go: `POST /api/plaid/remove-item`) + new Link; Snaptrade = open Connect again; Go updates stored connection.

**Done when:** After linking, you see all accounts and one net worth number from the Go API when you load or refresh the dashboard.

---

## Slice 4: Transactions and expense tracker

**Goal:** Plaid transactions use **webhooks + a conditional end‑of‑day cron**. **Webhooks** are primary: when Plaid notifies us (e.g. via `SYNC_UPDATES_AVAILABLE`) that **new transactions were created for an item that day**, we record that fact. At end of day, a single cron job runs **only if at least one item had new transactions that day**, and then uses cursor‑based `/transactions/sync` to fetch and upsert **all transactions created that day**. Each transaction counts as an expense (category) and feeds overall liabilities. You can view/filter by month and category. Slice 5 uses these categories for monthly budget progress.

- [x] **4.1 Plaid webhook marks "new transactions today" (Go)**
  - [x] Go: webhook `POST /api/webhooks/plaid` — on `SYNC_UPDATES_AVAILABLE` mark the corresponding Plaid item as `new_transactions_pending` (flag on `plaid_items`). Signature verification optional per Plaid docs.
  - [x] Each stored transaction is treated as an expense (or credit/refund if positive): contributes to its category and to overall liabilities.
  - [x] **Credit card refunds**: Positive transaction amounts on credit cards reduce liabilities (refunds add back to net worth). Negative amounts increase liabilities (expenses).
- [x] **4.2 End‑of‑day Plaid sync (Go, in Slice 7 cron)**
  - [x] Sync logic: `POST /api/transactions/sync` runs cursor‑based `/transactions/sync` for all items with `new_transactions_pending`; upserts added/modified, deletes removed; clears pending and stores cursor. Nightly cron (Slice 7) will call this endpoint.
- [x] **4.3 Categorization (Go)**
  - [x] Store Plaid category on each transaction; map to `categories` (by Plaid primary category) so Slice 5 can sum spent per category for the month.
- [x] **4.4 Expense tracker UI (React)**
  - [x] Go: `GET /api/transactions?month=...&category=...&search=...`; `GET /api/categories`.
  - [x] React: Expense tracker page at `/expenses` — pick month, filter by category, search; show date, amount, merchant, category; sort by date; "Sync transactions" button.

**Done when:** Plaid webhook + nightly cron (cursor safety check) keep transactions in sync; they feed expense categories and liabilities; React lists/filters them. Slice 5 uses the same categories for monthly budget. Credit card payment/transfer detection is implemented in Slice 7.

---

## Slice 5: Budget tracker

**Goal:** Set a budget by category and see progress (spent vs budget) using expense data. The budget is **global** (same for all months) until manually changed; the month selector only changes the "spent" side.

- [x] **5.1 Budget model and API (Go)**
  - [x] Supabase: table `budgets` with a single global row (`id = 1`, `allocations jsonb`, `updated_at`). Go: `GetBudget` / `UpsertBudget` helpers plus endpoints:
    - [x] `GET /api/budget?month=YYYY-MM` — returns global allocations and "spent per category" for that month, computed from `transactions` (expense categories only, including `Uncategorized`).
    - [x] `PUT /api/budget` — updates the global budget allocations.
- [x] **5.2 Budget UI (React)**
  - [x] Page at `/budget`: pick month; set budget per category (form); fetch budget and spent-from-Go. Shows per-category budget, spent, and remaining, with over-budget categories highlighted.
  - [x] Includes a "Total monthly budget" input; per-category allocations must sum exactly to this total before the budget is saved. If not, the save is rejected with a clear "invalid allocations" message.
  - [x] Every expense category always has an explicit allocation (at least 0); clearing a field sets the allocation to 0 rather than removing it. Footer summary shows total budgeted, total spent for the month, and total remaining across all categories.

**Done when:** You can create/edit the global budget via React, allocations are validated against the total budget, and you can see spent vs budget per category (and overall) for any month using data from the Go API.

---

## Slice 6: Portfolio view (holdings, daily + monthly history)

**Goal:** See **investment portfolio only** (Snaptrade/Fidelity) positions (live) and **daily** portfolio value for the **last 30 days**, plus **monthly per‑account** values for rolled‑off months (end‑of‑month snapshots). Daily data is investments only (no net worth/liabilities); monthly is stored per account and summed for totals.

- [x] **6.1 Current holdings (Go + Snaptrade)**
  - [x] Go: fetch Snaptrade positions per account (symbol, quantity, value) via the official Snaptrade Go SDK; expose an endpoint that returns current holdings with `accountId`, `accountName`, `symbol`, `quantity`, and `valueCents`. The frontend derives account totals and overall total from this response.
- [x] **6.2 Daily and monthly portfolio data (Go + Supabase)**
  - [x] Supabase: `daily_snapshots` (date, `portfolio_value_cents`, **investments only**), `daily_holdings` (date, account_id, symbol, quantity, value_cents) for **last 30 days**; `monthly_snapshots` (month, account_id, portfolio_value_cents) for rolled‑off months (no `monthly_holdings` table).
  - [x] Go: endpoints to return:
    - Daily total portfolio series (last 30 days) and monthly total series (older months) by aggregating `monthly_snapshots` across accounts.
    - Daily holdings history for last 30 days filtered by account or symbol; monthly series **per account only** from `monthly_snapshots` (no monthly per holding).
- [x] **6.3 Portfolio UI (React)**
  - [x] Page at `/portfolio`: shows **today’s total portfolio value**, breakdown **by account**, and positions **within each account** (all clickable).
  - [x] Selecting:
    - **Total** shows last 30 days (daily total) and past 12 months (monthly total).
    - **An account** shows last 30 days (daily for that account) and past 12 months (monthly for that account).
    - **A holding** shows last 30 days (daily for that symbol in that account) and indicates that monthly per holding is not available.
  - [x] Data is updated exclusively by nightly cron; there is no manual “refresh” button on the portfolio page.

**Done when:** React shows current Fidelity/Snaptrade positions and portfolio history: **daily** for last 30 days and **monthly** for older months (per account, summed for total), matching the investments‑only snapshot schema.

---

## Slice 7: Nightly cron — Plaid safety check + Snaptrade + daily snapshots

**Goal:** One **nightly cron** at 11pm that: (1) **Plaid safety check** — use the webhook‑set `new_transactions_pending` flag to run cursor‑based `/transactions/sync` for items that actually had activity, so the cursor is at end of book and we don’t miss transactions. (2) **Snaptrade** — fetch holdings/balances, refresh connection status, and write **daily** portfolio snapshots for today (`daily_snapshots`, `daily_holdings`). (3) At end of month, write that month's per‑account rollup to `monthly_snapshots` before later pruning daily rows (Slice 9). We **do** store daily snapshots for portfolio, but after a month we delete that month's daily rows and keep only the monthly value. So we keep daily for current month + previous month only (e.g. in April: daily for March and April, monthly for January and February).

- [x] **7.1 Snapshot schema (Go + Supabase)**
  - [x] Supabase: portfolio‑only schema is in place via `supabase/migrations/portfolio_snapshots.sql`: `daily_snapshots` (date, `portfolio_value_cents`), `daily_holdings` (date, account_id, symbol, quantity, value_cents), and `monthly_snapshots` (month, account_id, `portfolio_value_cents`). We are **not** storing net worth/cash/liabilities snapshots yet; those will be derived on the fly for graphs.
- [x] **7.2 Cron job — Plaid safety + Snaptrade + daily snapshots (Go)**
  - [x] Go: endpoint `POST /api/cron/daily-sync` protected by `CRON_SECRET` (via `X-Cron-Secret` header or `?secret=`) that:
    - [x] **Plaid:** looks up items with `new_transactions_pending=true` and runs cursor‑based `TransactionsSync` for each via `SyncTransactionsForItem`, upserting new/updated transactions and deleting removed ones, then updates `transactions_cursor` and clears the pending flag.
    - [x] **Snaptrade:** fetches accounts (`ListAccounts`) and positions (`ListAccountPositions`), writes per‑position `daily_holdings` rows and a total `daily_snapshots` row for today (sum of all accounts), and on the last day of the month writes end‑of‑month per‑account `monthly_snapshots`.
    - [x] **Connection status:** calls `checkAndUpdatePlaidItemStatuses` and `checkAndUpdateSnaptradeConnectionStatuses` to keep `plaid_items.status` and `snaptrade_connections.status` up to date for the Link Management UI.
- [x] **7.3 Schedule (use whatever is free)**
  - [x] Use a **free** scheduler: **GitHub Actions** or **Vercel Cron**. Call `POST https://<your-go-api>/api/cron/daily-sync` with `CRON_SECRET` at 11pm.
- [x] **7.4 API for charts**
  - [x] Go: portfolio chart APIs are already in place from Slice 6:
    - `GET /api/portfolio/snapshots` — returns daily total series (from `daily_snapshots`) and monthly total series (aggregated from `monthly_snapshots`) for the portfolio or a single account.
    - `GET /api/portfolio/holdings/history` — returns daily holdings from `daily_holdings` for an account or symbol. Slice 8 will consume these for graphs.
  - [x] Transactions/budget chart data will reuse the existing summary and budget endpoints.

**Done when:** At 11pm, cron runs Plaid cursor‑based sync for items with pending transactions, Snaptrade fetch + daily portfolio snapshots, and at month‑end writes monthly rollups; daily data is kept for current + previous month, then rolled to monthly (Slice 9).

---

## Slice 8: Graphs and visualization

**Goal:** Charts for net worth over time, portfolio value over time, per-holding performance over time, expenses by category (monthly), and budget progress.

- [x] **8.1 Go API for chart data**
  - [x] Endpoints to return: **daily** snapshots/holdings for recent months (last 30 days), **monthly** for older; net worth, portfolio value over time; transactions aggregated by category; budget + spent per category. All auth-protected. (Note: Chart data reuses existing endpoints from Slices 6 and 7; no new endpoints needed.)
- [x] **8.2 Net worth and portfolio over time (React)**
  - [x] Fetch daily snapshot data for last 30 days and monthly for older from Go; chart overall net_worth and portfolio_value over time (line or area). (Note: Portfolio value charts implemented; net worth over time uses current balances only as historical net worth snapshots are not stored.)
- [x] **8.3 Per-stock and per-account performance over time (React)**
  - [x] For chosen symbol or account, fetch daily holdings for past 30 days, show line graph. Implemented: line charts for selected account and selected holding (symbol) showing daily value over last 30 days.
- [x] **8.4 Expenses by category (React)**
  - [x] By month: fetch aggregated-by-category from Go; Pie chart with percentages shown. Implemented: pie chart on Expense Tracker page showing expenses by category for the selected month (client-side aggregation from transactions).
- [x] **8.5 Budget progress (React)**
  - [x] Per category: bar or donut showing spent vs budget for selected month (data from Go). Implemented: bar chart on Budget Tracker page showing budget vs spent per category.
- [x] **8.6 Dashboard integration**
  - [x] Add charts to the appropriate pages in React; keep chart components in `frontend/src/components/`. Charts integrated into Portfolio, Expense Tracker, and Budget Tracker pages.

**Done when:** You can view net worth, portfolio, per-holding, expenses, and budget as charts (React + Go API).

---

## Slice 9: Export and retention

**Goal:** Export each month's data as CSV before deleting it. Apply retention rules so daily data is pruned to monthly, then monthly to yearly, keeping DB small while preserving long-term summaries.

**Retention rules**

- **CC daily transactions:** Delete after **3 months**. Before deleting, keep **monthly overall** (e.g. total spent per category for the month). Export that month's transactions as CSV before delete.
- **CC monthly data:** Keep **1 year**; then delete. Keep **yearly overall** for CC (e.g. total balance or total spent per year). Export month as CSV before deleting.
- **Portfolio daily:** We store **daily** snapshots for the **current month and previous month** only (e.g. in April: daily for March and April). After a month ends, **delete all daily** snapshots/holdings for that month and keep only the **monthly** value (one row in `monthly_snapshots` and rows in `monthly_holdings`). So in April we have daily for March + April, and monthly for January, February. Export that month's daily data as CSV before deleting.
- **Portfolio monthly** (`monthly_snapshots`, `monthly_holdings`): Keep **1 year**; then delete and keep **yearly overall**. Export that month's data as CSV before deleting.
- **Export-before-delete:** For any month being pruned (CC or portfolio), generate and offer download of that month's CSV first, then delete the rows.

- [ ] **9.1 Schema for rollups (Go + Supabase)**
  - [ ] Tables: `monthly_expense_summary` (month, category, total_cents) for CC; `monthly_snapshots` and `monthly_holdings` (Slice 7) for portfolio; `yearly_cc_summary`, `yearly_portfolio_summary` for yearly overall. Populate yearly when pruning monthly after 1 year.
- [ ] **9.2 CSV export (Go + React)**
  - [ ] Go: auth-protected endpoint(s) to export a given month's transactions or snapshot/holdings as CSV (stream or download). Used by retention job and optionally by user ("Export this month").
  - [ ] React: button or page to trigger export (e.g. "Export month" or "Export last 12 months") that calls Go and downloads the file.
- [ ] **9.3 Retention job (Go)**
  - [ ] Go: cron-invoked or scheduled job that: (1) **Portfolio daily:** for any month that is older than "previous month" (e.g. when in April, months before March), ensure that month's daily rows are rolled up: export that month's daily data to CSV, insert/update monthly_snapshots and monthly_holdings for that month, then delete all daily_snapshots and daily_holdings for that month. So we only keep daily for current + previous month. (2) **CC:** prune transactions older than 3 months (after monthly summary); prune CC monthly older than 1 year (keep yearly). (3) **Portfolio monthly:** prune older than 1 year (keep yearly overall). Export before delete in all cases. Run daily or weekly.
- [ ] **9.4 Document rules**
  - [ ] Document retention rules and export-before-delete in README or runbook.

**Done when:** Each month's data is exported as CSV before deletion; portfolio daily kept only for current + previous month (older months rolled to monthly); CC daily → 3 months then monthly (1 year) then yearly; portfolio monthly → 1 year then yearly overall; DB stays within free tier.

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
|-------|-------|
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
