# Data Retention and Export Rules

This document describes the data retention policies and export-before-delete rules for the portfolio tracker application.

## Overview

To keep the database within free tier limits while preserving long-term summaries, we implement retention rules that:
1. Export data as CSV before deletion
2. Create aggregated summaries (monthly/yearly) before deleting detailed data
3. Maintain a rolling window of recent detailed data

## Retention Rules

### Expenses (Transactions)

- **Retention**: Delete transactions older than **1 year**
- **Before deletion**: 
  - Export that month's transactions as CSV
  - Create monthly expense summaries (`monthly_expense_summary`) aggregated by category
- **Yearly summaries**: 
  - At end of year, aggregate monthly summaries into yearly expense summaries (`yearly_expense_summary`)
  - Keep yearly summaries indefinitely
  - Yearly summaries show total spent per category per year

**Example**: 
- In January 2026, transactions from January 2025 and earlier are exported and deleted
- Monthly summaries for 2025 are created before deletion
- At end of 2025, yearly summaries for 2024 are created from monthly summaries, then 2024 monthly summaries are deleted

### Portfolio Daily Snapshots

- **Retention**: Keep daily snapshots (`daily_snapshots`) and holdings (`daily_holdings`) for the **last 30 days only**
- **Before deletion**:
  - If deleting a day that is the last day of a month, ensure monthly snapshot exists for that month
  - Export daily snapshots/holdings as CSV before deletion
- **Monthly rollup**: 
  - End-of-month values are written to `monthly_snapshots` (per account)
  - Daily data older than 30 days is deleted

**Example**:
- On February 1, 2026, daily snapshots from January 1, 2026 and earlier are deleted
- If January 31, 2026 is being deleted, ensure monthly snapshot for January 2026 exists first

### Portfolio Monthly Snapshots

- **Retention**: Keep monthly snapshots (`monthly_snapshots`) for the **entire year** until reaching end of the following year
- **Before deletion**:
  - Export that year's monthly snapshots as CSV
  - Create yearly portfolio summaries (`yearly_portfolio_summary`) aggregated by account
- **Yearly summaries**: 
  - Store end-of-year portfolio value per account
  - Keep yearly summaries indefinitely

**Example**:
- Store all of 2024 monthly data until the end of December 2025
- At end of December 2025, export 2024 monthly data as CSV, create yearly summaries for 2024, then delete 2024 monthly data
- Keep yearly summaries showing end-of-year portfolio value per account

## Export-Before-Delete

All data being pruned is exported as CSV before deletion. The retention job emails the CSV to the user (using **Resend**, free tier) before deleting:

1. **Transactions**: On the 1st of each month, the month that is exactly 1 year old is exported as CSV, emailed to `ALLOWED_USER_EMAIL`, then that month's transactions are deleted.
2. **Daily snapshots**: No email (daily data rolls into monthly); when the day being deleted is month-end, a monthly snapshot is written first, then daily data older than 30 days is deleted.
3. **Monthly snapshots**: On Dec 31, the previous year's monthly snapshots are exported as CSV, emailed, then that year's monthly rows are deleted.

Requires `RESEND_API_KEY` (and optionally `RESEND_FROM`). If `RESEND_API_KEY` is not set, retention still runs but no email is sent.

## Manual Export

Users can export data on demand via the UI:

- **Expense Tracker**: Export transactions for any month as CSV
- **Portfolio**: Export snapshots or holdings for any month as CSV

## Implementation

### Cron Job

The nightly cron job (`POST /api/cron/daily-sync`) runs retention logic:

1. **Transaction retention**: 
   - Find months older than 1 year
   - Create monthly expense summaries for those months
   - Delete transactions older than 1 year

2. **Daily snapshot retention**:
   - Delete daily snapshots/holdings older than 30 days
   - Before deletion, ensure monthly snapshots exist for month-end dates

3. **Monthly snapshot retention** (on December 31):
   - Create yearly summaries from monthly snapshots for the previous year
   - Export and delete monthly snapshots for the previous year

### Database Tables

- `monthly_expense_summary`: Month, category, total_cents, transaction_count
- `yearly_expense_summary`: Year, category, total_cents, transaction_count
- `yearly_portfolio_summary`: Year, account_id, portfolio_value_cents

### Export Endpoints

- `GET /api/export/transactions?month=YYYY-MM`: Export transactions for a month
- `GET /api/export/portfolio/snapshots?month=YYYY-MM`: Export portfolio snapshots for a month
- `GET /api/export/portfolio/holdings?month=YYYY-MM`: Export portfolio holdings for a month

All export endpoints require authentication and return CSV files.

## Notes

- Retention jobs run automatically via nightly cron
- Manual exports are available via UI buttons
- Yearly summaries provide long-term trends without storing detailed data
- CSV exports preserve historical data before deletion
