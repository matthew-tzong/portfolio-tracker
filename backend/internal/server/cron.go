package server

import (
	"encoding/json"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/matthewtzong/portfolio-tracker/backend/internal/database"
)

// Cron Response.
type cronSyncResponse struct {
	PlaidSyncedItems        int  `json:"plaidSyncedItems"`
	DailySnapshotWritten    bool `json:"dailySnapshotWritten"`
	MonthlySnapshotsWritten int  `json:"monthlySnapshotsWritten"`
}

// Registers cron routes.
func registerCronRoutes(mux *http.ServeMux, deps apiDependencies) {
	mux.HandleFunc("/api/cron/daily-sync", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		handleDailySync(w, r, deps)
	})
}

// Handles the nightly cron job: (Plaid/Snaptrade Syncs + Status Checks).
func handleDailySync(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	w.Header().Set("Content-Type", "application/json")

	// Gets the cron secret.
	cronSecret := os.Getenv("CRON_SECRET")

	// Checks the cron secret.
	headerSecret := r.Header.Get("X-Cron-Secret")
	if headerSecret != cronSecret {
		writeJSONError(w, http.StatusUnauthorized, "invalid cron secret")
		return
	}

	// Update connection statuses for Plaid and Snaptrade.
	_ = checkAndUpdatePlaidItemStatuses(r.Context(), deps.db, deps.plaidClient)
	_ = checkAndUpdateSnaptradeConnectionStatuses(r.Context(), deps.db, deps.snaptradeClient)

	// Sync Plaid items with pending transactions.
	plaidSynced, err := runPlaidSafetySync(r, deps)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Plaid sync failed: "+err.Error())
		return
	}

	// Fetch Snaptrade holdings/balances and write today's snapshots.
	dailyWritten, monthlyWritten, err := writeSnaptradeSnapshotsForToday(r, deps)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Snaptrade snapshot failed: "+err.Error())
		return
	}

	// Returns the response.
	resp := cronSyncResponse{
		PlaidSyncedItems:        plaidSynced,
		DailySnapshotWritten:    dailyWritten,
		MonthlySnapshotsWritten: monthlyWritten,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// Syncs Plaid items with pending transactions using cursor-based sync.
func runPlaidSafetySync(r *http.Request, deps apiDependencies) (int, error) {
	if deps.db == nil || deps.plaidClient == nil {
		return 0, nil
	}

	// Lists the items with pending transactions.
	items, err := deps.db.ListPlaidItemsWithPendingTransactions(r.Context())
	if err != nil {
		return 0, err
	}

	// Syncs the transactions for each item.
	for _, item := range items {
		err = SyncTransactionsForItem(r.Context(), deps.db, deps.plaidClient, &item)
		if err != nil {
			return 0, err
		}
	}

	return len(items), nil
}

// Adds today's Snaptrade daily holdings/snapshots along with end of month monthly snapshots.
func writeSnaptradeSnapshotsForToday(r *http.Request, deps apiDependencies) (bool, int, error) {
	if deps.db == nil || deps.snaptradeClient == nil {
		return false, 0, nil
	}

	// Gets the Snaptrade user.
	user, err := deps.db.GetSnaptradeUser(r.Context())
	if err != nil || user == nil {
		return false, 0, err
	}

	// List all Snaptrade accounts with balances.
	accounts, err := deps.snaptradeClient.ListAccounts(user.UserID, user.UserSecret)
	if err != nil {
		return false, 0, err
	}
	if len(accounts) == 0 {
		return false, 0, nil
	}

	// Sets the current time as the snapshot date.
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	var (
		totalPortfolioCents int64
		accountTotals       = make(map[string]int64)
	)

	// Loops through the accounts and computes the total portfolio value and per-account totals.
	for _, account := range accounts {
		accountTotalCents := int64(math.Round(account.BalanceAmount * 100))
		totalPortfolioCents += accountTotalCents
		accountTotals[account.ID] = accountTotalCents

		// Lists the positions for the account.
		positions, err := deps.snaptradeClient.ListAccountPositions(user.UserID, user.UserSecret, account.ID)
		if err != nil {
			continue
		}

		// Loops through the positions and adds them to the daily holdings.
		for _, pos := range positions {
			holding := &database.DailyHolding{
				Date:       today,
				AccountID:  account.ID,
				Symbol:     pos.Symbol,
				Quantity:   pos.Quantity,
				ValueCents: pos.ValueCents,
			}
			_ = deps.db.UpsertDailyHolding(r.Context(), holding)
		}
	}

	// Writes today's total portfolio snapshot.
	snapshot := &database.DailySnapshot{
		Date:                today,
		PortfolioValueCents: totalPortfolioCents,
	}
	err = deps.db.UpsertDailySnapshot(r.Context(), snapshot)
	if err != nil {
		return false, 0, err
	}

	monthlyWritten, err := maybeWriteMonthlySnapshots(r, deps, today, accountTotals)
	if err != nil {
		return true, 0, err
	}

	return true, monthlyWritten, nil
}

// If today is the end of the month, write per-account monthly snapshots using today's values.
func maybeWriteMonthlySnapshots(r *http.Request, deps apiDependencies, today time.Time, accountTotals map[string]int64) (int, error) {
	// Checks if today is the last calendar day of the month.
	tomorrow := today.AddDate(0, 0, 1)
	if tomorrow.Month() == today.Month() {
		return 0, nil
	}

	if deps.db == nil {
		return 0, nil
	}

	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	written := 0

	// Loops through the accounts and writes the monthly snapshots.
	for accountID, total := range accountTotals {
		snapshot := &database.MonthlySnapshot{
			Month:               monthStart,
			AccountID:           accountID,
			PortfolioValueCents: total,
		}
		err := deps.db.UpsertMonthlySnapshot(r.Context(), snapshot)
		if err != nil {
			return written, err
		}
		written++
	}

	return written, nil
}
