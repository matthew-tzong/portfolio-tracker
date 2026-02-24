package server

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/matthewtzong/portfolio-tracker/backend/internal/database"
	"github.com/matthewtzong/portfolio-tracker/backend/internal/email"
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

	// Run retention job to prune old data.
	_ = runRetentionJob(r.Context(), deps)

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
				Date:       database.DateOnly{Time: today},
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
		Date:                database.DateOnly{Time: today},
		PortfolioValueCents: totalPortfolioCents,
	}
	err = deps.db.UpsertDailySnapshot(r.Context(), snapshot)
	if err != nil {
		return false, 0, err
	}

	// Writes the monthly snapshots.
	monthlyWritten, err := maybeWriteMonthlySnapshots(r, deps, today, accountTotals)
	if err != nil {
		return true, 0, err
	}

	// Writes the monthly net worth snapshot.
	err = maybeWriteMonthlyNetWorth(r, deps, today, totalPortfolioCents)
	if err != nil {
		return true, monthlyWritten, err
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
			Month:               database.DateOnly{Time: monthStart},
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

// If today is the end of the month, write a single overall monthly net worth snapshot.
func maybeWriteMonthlyNetWorth(r *http.Request, deps apiDependencies, today time.Time, investmentsCents int64) error {
	// Checks if today is the last calendar day of the month.
	tomorrow := today.AddDate(0, 0, 1)
	if tomorrow.Month() == today.Month() {
		return nil
	}
	if deps.db == nil {
		return nil
	}

	// Gets the cash and liabilities from the Plaid accounts.
	var cashCents, liabilitiesCents int64
	plaidAccounts, err := deps.db.ListPlaidAccounts(r.Context())
	if err == nil {
		for _, account := range plaidAccounts {
			_, cashDelta, _, liabilityDelta := loadPlaidAccounts(account)
			cashCents += cashDelta
			liabilitiesCents += liabilityDelta
		}
	}

	netWorthCents := cashCents + investmentsCents - liabilitiesCents
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)

	snapshot := &database.MonthlyNetWorth{
		Month:            database.DateOnly{Time: monthStart},
		NetWorthCents:    netWorthCents,
		CashCents:        cashCents,
		InvestmentsCents: investmentsCents,
		LiabilitiesCents: liabilitiesCents,
	}
	return deps.db.UpsertMonthlyNetWorth(r.Context(), snapshot)
}

// Runs retention job to prune old data according to retention rules.
func runRetentionJob(ctx context.Context, deps apiDependencies) error {
	if deps.db == nil {
		return nil
	}

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	userEmail := os.Getenv("ALLOWED_USER_EMAIL")

	// Prunes 1 year old transactions on the first day of each month.
	if today.Day() == 1 {
		monthToPrune := time.Date(today.Year()-1, today.Month(), 1, 0, 0, 0, 0, time.UTC)
		transactions, err := deps.db.ListTransactionsForMonth(ctx, monthToPrune)
		if err != nil {
			log.Printf("retention: list transactions for month %s: %v", monthToPrune.Format("2006-01"), err)
			return err
		}
		if len(transactions) == 0 {
			return nil
		}
		// Builds CSV, emails, creates summary, then deletes the data from the database.
		csvBytes, err := BuildTransactionsCSV(ctx, deps.db, monthToPrune)
		if err != nil {
			log.Printf("retention: build transactions CSV for %s: %v", monthToPrune.Format("2006-01"), err)
			return err
		}
		monthStr := monthToPrune.Format("2006-01")
		err = email.SendCSV(ctx, userEmail, "Portfolio Tracker: transactions export for "+monthStr, "transactions-"+monthStr+".csv", csvBytes)
		if err != nil {
			log.Printf("retention: send transactions email: %v", err)
			return err
		}
		_ = createMonthlyExpenseSummary(ctx, deps, monthToPrune)
		err = deps.db.DeleteTransactionsInMonth(ctx, monthToPrune)
		if err != nil {
			log.Printf("retention: delete transactions for month %s: %v", monthToPrune.Format("2006-01"), err)
		}
	}

	// Prunes daily snapshots and holdings older than 30 days.
	thirtyDaysAgo := today.AddDate(0, 0, -30)
	dayBeingDeleted := thirtyDaysAgo
	nextDay := dayBeingDeleted.AddDate(0, 0, 1)
	if nextDay.Month() != dayBeingDeleted.Month() {
		// Writes the monthly snapshot for the day being deleted.
		monthStart := time.Date(dayBeingDeleted.Year(), dayBeingDeleted.Month(), 1, 0, 0, 0, 0, time.UTC)
		holdings, _ := deps.db.ListDailyHoldings(ctx, dayBeingDeleted, dayBeingDeleted)
		if len(holdings) > 0 {
			accountTotals := make(map[string]int64)
			for _, holding := range holdings {
				accountTotals[holding.AccountID] += holding.ValueCents
			}
			for accountID, total := range accountTotals {
				monthlySnapshot := &database.MonthlySnapshot{
					Month:               database.DateOnly{Time: monthStart},
					AccountID:           accountID,
					PortfolioValueCents: total,
				}
				_ = deps.db.UpsertMonthlySnapshot(ctx, monthlySnapshot)
			}
		}
	}
	// Deletes the daily snapshots and holdings older than 30 days.
	_ = deps.db.DeleteDailySnapshotsOlderThan(ctx, thirtyDaysAgo)
	_ = deps.db.DeleteDailyHoldingsOlderThan(ctx, thirtyDaysAgo)

	// Prunes yearly monthly snapshots on December 31.
	if now.Month() != 12 || now.Day() != 31 {
		return nil
	}

	lastYear := now.Year() - 1
	snapshots, err := deps.db.ListMonthlySnapshotsForYear(ctx, lastYear)
	// Creates yearly summaries and deletes the monthly snapshots.
	if err != nil || len(snapshots) == 0 {
		_ = createYearlyExpenseSummaries(ctx, deps, lastYear)
		_ = deps.db.DeleteMonthlySnapshotsForYear(ctx, lastYear)
		return nil
	}

	// Builds the CSV for the yearly monthly snapshots.
	csvBytes, err := BuildPortfolioSnapshotsCSV(snapshots)
	if err != nil {
		log.Printf("retention: build portfolio CSV for year %d: %v", lastYear, err)
		return err
	}
	yearStr := strconv.Itoa(lastYear)
	err = email.SendCSV(ctx, userEmail, "Portfolio Tracker: portfolio snapshots export for "+yearStr, "portfolio-snapshots-"+yearStr+".csv", csvBytes)
	if err != nil {
		log.Printf("retention: send portfolio email: %v", err)
		return err
	}
	_ = createYearlyExpenseSummaries(ctx, deps, lastYear)
	_ = deps.db.DeleteMonthlySnapshotsForYear(ctx, lastYear)
	return nil
}

// Creates monthly expense summary for a given month from transactions.
func createMonthlyExpenseSummary(ctx context.Context, deps apiDependencies, month time.Time) error {
	// Get transactions for the month.
	transactions, err := deps.db.ListTransactionsForMonth(ctx, month)
	if err != nil {
		return err
	}

	// Gets the categories.
	categories, err := deps.db.ListCategories(ctx)
	if err != nil {
		return err
	}
	categoriesByID := make(map[int64]database.Category, len(categories))
	for _, category := range categories {
		categoriesByID[category.ID] = category
	}

	categoryTotals := make(map[int64]int64)
	categoryCounts := make(map[int64]int)

	// Loops through the transactions and aggregates the spend by category.
	for _, transaction := range transactions {
		if transaction.CategoryID == nil {
			continue
		}
		category, ok := categoriesByID[*transaction.CategoryID]
		if !ok || !category.Expense {
			continue
		}
		delta := -transaction.AmountCents
		if delta == 0 {
			continue
		}
		categoryTotals[category.ID] += delta
		categoryCounts[category.ID]++
	}

	// Upsert monthly expense summaries.
	for categoryID, total := range categoryTotals {
		summary := &database.MonthlyExpenseSummary{
			Month:            database.DateOnly{Time: month},
			CategoryID:       categoryID,
			TotalCents:       total,
			TransactionCount: categoryCounts[categoryID],
		}
		_ = deps.db.UpsertMonthlyExpenseSummary(ctx, summary)
	}

	return nil
}

// Creates yearly summaries from monthly data for a given year.
func createYearlyExpenseSummaries(ctx context.Context, deps apiDependencies, year int) error {
	yearStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC)

	// Aggregate monthly expense summaries by category.
	monthlySummaries, err := deps.db.ListMonthlyExpenseSummaries(ctx, yearStart, yearEnd)
	if err == nil {
		categoryTotals := make(map[int64]int64)
		categoryCounts := make(map[int64]int)

		for _, summary := range monthlySummaries {
			categoryTotals[summary.CategoryID] += summary.TotalCents
			categoryCounts[summary.CategoryID] += summary.TransactionCount
		}

		for categoryID, total := range categoryTotals {
			yearlyExpenseSummary := &database.YearlyExpenseSummary{
				Year:             year,
				CategoryID:       categoryID,
				TotalCents:       total,
				TransactionCount: categoryCounts[categoryID],
			}
			_ = deps.db.UpsertYearlyExpenseSummary(ctx, yearlyExpenseSummary)
		}
	}

	// Aggregate monthly portfolio snapshots by account.
	monthlySnapshots, err := deps.db.ListMonthlySnapshotsForYear(ctx, year)
	if err == nil {
		accountTotals := make(map[string]int64)

		for _, snapshot := range monthlySnapshots {
			accountTotals[snapshot.AccountID] = snapshot.PortfolioValueCents
		}

		for accountID, total := range accountTotals {
			yearlyPortfolio := &database.YearlyPortfolioSummary{
				Year:                year,
				AccountID:           accountID,
				PortfolioValueCents: total,
			}
			_ = deps.db.UpsertYearlyPortfolioSummary(ctx, yearlyPortfolio)
		}
	}

	return nil
}
