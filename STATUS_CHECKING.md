# Status Checking and Reconnection

## Overview

This document explains how we detect broken connections (Plaid items and Snaptrade connections), distinguish between reconnection-needed errors vs transient errors, and handle missed data.

## Current Status Checking

### Plaid Items

**Status Values:**
- `"OK"` - Item is working correctly
- `"ITEM_LOGIN_REQUIRED"` - Item needs re-authentication (user must reconnect)

**How Status is Checked:**
- **Automatic check only**: Status is checked automatically in nightly cron jobs (Slice 7)

**Status Check Logic:**
- Calls Plaid's `/item/get` API endpoint with the item's `access_token`
- If API call succeeds: Updates status to whatever Plaid returns
- If API call fails:
  - **Authentication errors** (e.g., `INVALID_ACCESS_TOKEN`, `ITEM_LOGIN_REQUIRED`): Marks as `"ITEM_LOGIN_REQUIRED"` immediately
  - **Other errors**: Does NOT mark as broken to avoid false positives

**Key Point**: We ONLY mark as broken if it's clearly an authentication error. All other errors (transient or unknown) are treated as transient and allow retry. This prevents false positives from temporary network issues, rate limits, or unknown error types.

### Snaptrade Connections

**Status Values:**
- `"OK"` - Connection is working correctly
- `"CONNECTION_ERROR"` - Failed to list connections twice (likely authentication issue)
- `"ACCOUNT_FETCH_ERROR"` - Failed Once (Could be flaky, don't yet mark as authentication issue)

**How Status is Checked:**
- **Automatic check only**: Status is checked automatically in nightly cron jobs (Slice 7)
- **Sync endpoint**: `POST /api/snaptrade/sync-connections` syncs the list of connections but does NOT check status (preserves existing status, cron will update it)

**Status Check Logic** (see 2-strike system below):
- Calls Snaptrade's `ListConnections` API
  - If this fails → Apply 2-strike (e.g. OK → `ACCOUNT_FETCH_ERROR`, `ACCOUNT_FETCH_ERROR` → `CONNECTION_ERROR`)
- If `ListConnections` succeeds, tries `ListAccounts` API
  - If this fails → Apply 2-strike
  - If this succeeds → All connections marked as `"OK"`

**Note**: Snaptrade doesn't provide per-connection status codes, so we do the following:
- If we can't list connections at all → likely user authentication issue
- If we can list connections but can't fetch accounts → connections may be broken

## Integration with Cron Jobs (Slice 7)

The nightly cron job (`POST /api/cron/daily-sync`) is responsible both for syncing data (transactions and portfolio snapshots) and for keeping connection statuses up to date.

### Plaid

1. **Webhook path**: When Plaid sends `SYNC_UPDATES_AVAILABLE`, the webhook handler marks the corresponding item as having `new_transactions_pending = true`.
2. **Cron sync**: The cron handler looks up items with `new_transactions_pending = true` and runs cursor-based `/transactions/sync` for each via `SyncTransactionsForItem`, upserting new/updated transactions, deleting removed ones, updating the cursor, and clearing the pending flag.
3. **Status refresh**: As part of the same cron run, `checkAndUpdatePlaidItemStatuses(ctx, db, plaidClient)` is called to refresh item statuses in the database so the Link Management UI shows up-to-date broken vs OK states.

### Snaptrade

1. **Get data**: Cron calls `ListAccounts` / `ListAccountPositions` to fetch current balances and positions, then writes `daily_holdings` and `daily_snapshots` (and, on month-end, `monthly_snapshots`) into the database.
2. **Status refresh**: The same cron run calls `checkAndUpdateSnaptradeConnectionStatuses(ctx, db, snaptradeClient)` to apply the 2-strike logic and update connection statuses in the DB.

### Summary

- **Plaid**: Webhooks mark items with new transactions; nightly cron runs cursor-based `/transactions/sync` for those items and then calls `checkAndUpdatePlaidItemStatuses` to keep statuses in sync.
- **Snaptrade**: Nightly cron fetches accounts/positions, writes snapshots, and then calls `checkAndUpdateSnaptradeConnectionStatuses` to keep connection statuses up to date using the 2-strike system.
- Status-check helpers in `status_check.go` are invoked from cron (and can be reused on error paths elsewhere if needed); there is no on-demand status check triggered directly from the UI.

## Reconnection Flow

### Plaid Reconnection

1. User sees red alert (status ≠ "OK") in LinkManagement UI
2. User clicks "Reconnect" button
3. Frontend calls `POST /api/plaid/reconnect-link-token` with `itemId`
4. Backend:
   - Fetches existing item from DB
   - Creates Link token in **update mode** using existing `access_token`
   - Returns link token to frontend
5. Frontend opens Plaid Link in update mode
6. User completes re-authentication in Plaid Link
7. Frontend calls `POST /api/plaid/exchange-token` with new `publicToken`
8. Backend:
   - Exchanges public token for new `access_token` and `item_id`
   - **Detects existing item by `institution_id`** (not `item_id`)
   - Updates existing item with new `access_token` and `item_id`
   - Updates accounts to point to new `item_id` (accounts are keyed by `account_id`, so history is preserved)

**Key Point**: Reconnection updates the existing database record, preserving account history and preventing duplicates.

### Snaptrade Reconnection

1. User sees red alert (status = `CONNECTION_ERROR`) in LinkManagement UI
2. User clicks "Reconnect" button
3. Frontend calls `POST /api/snaptrade/connect-url`
4. Backend generates new Connect portal URL
5. Frontend opens Connect portal in a new tab via `openSnaptradeConnect()` (in `frontend/src/lib/snaptrade.ts`)
6. Frontend sets up a **one-time** `visibilitychange` listener (no sync on every page load)
7. User completes reconnection flow in Snaptrade portal
8. User returns to the app tab → listener fires → frontend calls `POST /api/snaptrade/sync-connections`, then runs the callback (e.g. `load()` in LinkManagement so the list refreshes)
9. User sees updated connections without manually refreshing. Listener is removed after running once (or after 5 minutes if they never return).

**Edge case**: If the user returns to the app tab *before* finishing the Snaptrade flow, we sync once (no new connection yet) and remove the listener. If they later complete the flow and return again, we do *not* auto-sync a second time; they can refresh the page or wait for the cron job to update connections.

**Key Point**: Snaptrade reconnection = open Connect portal again. We sync when the user **returns to our tab** (`visibilitychange`), not on every load. Status checks happen in the cron job (only on error path; see Integration with Cron Jobs).

## Handling Missed Data

### Plaid - Webhooks Handle This

**Plaid uses webhooks to tell us which items had new transactions; the nightly cron actually performs the sync:**
- When new transactions are available, Plaid sends a webhook (`SYNC_UPDATES_AVAILABLE`).
- The webhook handler marks that Plaid item as `new_transactions_pending = true`.
- The nightly cron job then runs cursor-based `/transactions/sync` for items with `new_transactions_pending` and advances the cursor. There is no immediate sync from the webhook itself.

**If an item is broken:**
- Webhooks won't fire for that item
- We'll miss data until reconnection
- But this is acceptable because:
  - Webhooks are primary - most data comes through them
  - User will see broken status and reconnect
  - After reconnection, webhooks resume
  - We don't try to backfill missed data (too complex, not worth it)

### Snaptrade - Missing a Day is Acceptable

**Snaptrade has no webhooks:**
- Data is fetched nightly via cron job
- If connection is broken during cron run, we miss that day's snapshot

**Why this is acceptable:**
- Daily snapshots are for trend visualization, not critical financial data
- Missing one day doesn't break the overall portfolio view
- User will see broken status and reconnect
- After reconnection, next cron run will fetch current data
- We don't try to backfill missed days (not worth the complexity)

**Tradeoff**: Simplicity over perfect data completeness. For a personal portfolio tracker, missing a day of snapshots is acceptable.

## Helper Functions

Two helper functions in `backend/pkg/server/status_check.go`:

1. **`checkAndUpdatePlaidItemStatuses()`**
   - Checks all Plaid items via Plaid API
   - Distinguishes auth errors (mark as broken) vs transient errors (allow retry)
   - Updates status in database
   - Can be called from cron jobs or manually

2. **`checkAndUpdateSnaptradeConnectionStatuses()`**
   - Checks all Snaptrade connections via Snaptrade API
   - Updates status in database based on API success/failure
   - Can be called from cron jobs or manually

In the current implementation, the nightly cron job always calls these helpers to keep statuses fresh; they can also be reused on error paths if needed. There is no on-demand status check from the UI.

### Snaptrade Error Handling - 2-Strike System

**Snaptrade doesn't provide explicit error codes**, so we use a **2-strike system** to avoid false positives:

**Status Progression:**
- `"OK"` → First failure → `"ACCOUNT_FETCH_ERROR"` (warning, not broken)
- `"ACCOUNT_FETCH_ERROR"` → Second consecutive failure → `"CONNECTION_ERROR"` (broken, needs reconnect)
- Any error state → Success → `"OK"` (recovery)

**If `ListConnections()` fails:**
- Status was `"OK"` → Mark as `"ACCOUNT_FETCH_ERROR"` (first strike)
- Status was `"ACCOUNT_FETCH_ERROR"` → Mark as `"CONNECTION_ERROR"` (second strike, broken)
- Status was `"CONNECTION_ERROR"` → Keep broken

**If `ListAccounts()` fails:**
- Status was `"OK"` → Mark as `"ACCOUNT_FETCH_ERROR"` (first strike)
- Status was `"ACCOUNT_FETCH_ERROR"` → Mark as `"CONNECTION_ERROR"` (second strike, broken)
- Status was `"CONNECTION_ERROR"` → Keep broken

**If both succeed:**
- Mark all connections as `"OK"` (recovery from any error state)

**Key Point**: Two consecutive days of failure = broken connection. One day failure = warning but not broken. This prevents false positives from transient errors while still detecting real issues quickly.

When the user opens Snaptrade Connect (or Reconnect), we sync connections automatically when they **return to the app tab** (`visibilitychange` listener in `openSnaptradeConnect`). They can also click "Refresh" on the links page to reload the list from the database; the next cron run will update status if a connection is broken.
