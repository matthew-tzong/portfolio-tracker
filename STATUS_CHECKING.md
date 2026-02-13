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

**Note**: Snaptrade doesn't provide per-connection status codes, so we use heuristics:
- If we can't list connections at all → likely user authentication issue
- If we can list connections but can't fetch accounts → connections may be broken

## Integration with Cron Jobs (Slice 7)

When implementing the cron job (`POST /api/cron/daily-sync`), use **get data first, status check only on error** so we don’t do extra API calls when everything works.

### Plaid

1. **Get data**: For each Plaid item, run cursor-based transaction sync (e.g. to end of book).
2. **Only on error**: If sync fails for an item, then run status check for that item (or all items) and update DB.
   - Use `checkAndUpdatePlaidItemStatuses(ctx, db, plaidClient)` when any item’s sync fails, so we mark auth-broken items as `ITEM_LOGIN_REQUIRED`.
   - Alternatively, for only the failed item, call `plaidClient.GetItem` and if it’s a `PlaidConnectionError` with `IsAuthError`, upsert that item’s status.

So: try sync → on success, done; on failure → call status check and persist status.

### Snaptrade

1. **Get data**: Call `ListConnections` and `ListAccounts`, then write daily snapshots (and optionally set all connections to `OK` in DB).
2. **Only on error**: If `ListConnections` or `ListAccounts` fails, call `checkAndUpdateSnaptradeConnectionStatuses(ctx, db, snaptradeClient)` to apply the 2-strike logic and update connection statuses in the DB.

So: try fetch + snapshots → on success, done (and connections stay/are set OK); on failure → run status check and persist status. Note: on the failure path, `checkAndUpdateSnaptradeConnectionStatuses` will call the Snaptrade API again; that’s acceptable so the same function can own the 2-strike logic.

### Summary

- **Plaid**: Sync transactions first; only if sync fails, run `checkAndUpdatePlaidItemStatuses` (or per-item GetItem + status update).
- **Snaptrade**: Fetch connections/accounts and write snapshots first; only if that fails, run `checkAndUpdateSnaptradeConnectionStatuses`.
- Status-check helpers in `status_check.go` are used only on the error path, so normal runs don’t add extra Plaid GetItem or Snaptrade list calls.

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

**Plaid uses webhooks as the primary data source:**
- When new transactions are available, Plaid sends a webhook (`SYNC_UPDATES_AVAILABLE`)
- Webhook triggers transaction sync immediately
- Nightly cron is just a **safety check** to catch anything webhooks missed

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

Two helper functions in `backend/internal/server/status_check.go`:

1. **`checkAndUpdatePlaidItemStatuses()`**
   - Checks all Plaid items via Plaid API
   - Distinguishes auth errors (mark as broken) vs transient errors (allow retry)
   - Updates status in database
   - Can be called from cron jobs or manually

2. **`checkAndUpdateSnaptradeConnectionStatuses()`**
   - Checks all Snaptrade connections via Snaptrade API
   - Updates status in database based on API success/failure
   - Can be called from cron jobs or manually

These functions are used **only on the error path** in the cron job (Slice 7): we try to get data first (Plaid transaction sync, Snaptrade fetch + snapshots); only if that fails do we call these helpers to update connection status. There is no on-demand status check from the UI.

## Error Classification

### Plaid Error Types

**Authentication Errors** (mark as broken immediately):
- `ITEM_LOGIN_REQUIRED`
- `INVALID_ACCESS_TOKEN`
- `ACCESS_TOKEN_EXPIRED`
- `ACCESS_TOKEN_INVALID`
- `ITEM_ERROR`

**Transient Errors** (don't mark as broken, allow retry):
- `RATE_LIMIT_EXCEEDED` (if not auth-related)
- `INSTITUTION_DOWN`
- `TIMEOUT`
- `INTERNAL_SERVER_ERROR`
- Network errors (connection refused, DNS, etc.)

**Unknown Errors**: Treated conservatively as authentication errors

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
