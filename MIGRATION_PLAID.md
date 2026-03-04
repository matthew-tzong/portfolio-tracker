# Migration from Snaptrade to Plaid for Investments

## Background
Snaptrade has stopped support for personal use of certain brokerages, so we have migrated investment tracking from Snaptrade to Plaid.

## Changes Made
- Updated Plaid client (`pkg/plaid/client.go`) to fetch investment holdings and securities.
- Added `cost_basis` support to Plaid structs, database models (`daily_holdings`), and API responses.
- Refactored `handleGetHoldings` and `writeInvestmentSnapshotsForToday` to utilize Plaid.
- Removed all Snaptrade-specific code, initialization, and routes (commented out for reference).
- Updated documentation (README.md) to reflect these changes.

## Impact
Previously connected brokerages via Snaptrade will need to reconnect them via the Plaid Link flow. The application now correctly identifies "investment" type accounts from Plaid and includes them in the portfolio value, holdings, and net worth logic, now featuring `cost_basis` for better tracking.
