package server

import (
	"context"
	"encoding/csv"
	"io"
	"log"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/matthewtzong/portfolio-tracker/backend/pkg/database"
	"github.com/matthewtzong/portfolio-tracker/backend/pkg/serverauth"
)

const (
	FidelityManualItemID    = "fidelity_manual_item"
	FidelityManualAccountID = "fidelity_manual_account"
	FidelityInstitutionName = "Fidelity"
)

// Registers Fidelity routes.
func registerFidelityRoutes(mux *http.ServeMux, deps apiDependencies) {
	// Upload monthly statement CSV.
	mux.Handle("/api/fidelity/upload-statement", serverauth.JWTAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		handleFidelityMonthlyUpload(w, r, deps)
	})))

	// Upload current holdings CSV.
	mux.Handle("/api/fidelity/upload-holdings", serverauth.JWTAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		handleFidelityHoldingsUpload(w, r, deps)
	})))
}

// Handles the monthly statement CSV upload.
func handleFidelityMonthlyUpload(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	// Gets file.
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to get file from request")
		return
	}
	defer file.Close()

	// Validates filename: Statement{Month}{Date}{Year}.csv
	re := regexp.MustCompile(`^Statement(\d{1,2})(\d{2})(\d{4})\.csv$`)
	matches := re.FindStringSubmatch(header.Filename)
	if len(matches) != 4 {
		writeJSONError(w, http.StatusBadRequest, "invalid filename format; expected StatementMMDDYYYY.csv")
		return
	}

	// Parses filename.
	month, _ := strconv.Atoi(matches[1])
	day, _ := strconv.Atoi(matches[2])
	year, _ := strconv.Atoi(matches[3])

	// Creates statement date.
	statementDate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, GetLocalLocation())

	// Ensures the Fidelity account exists.
	err = ensureFidelityAccountExists(r.Context(), deps)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to ensure Fidelity account exists: "+err.Error())
		return
	}

	// Parses CSV.
	holdings, err := parseFidelityMonthlyCSV(file)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to parse CSV: "+err.Error())
		return
	}

	// Clear existing manual holdings for this date to support re-uploads.
	err = deps.db.DeleteDailyHoldingsByAccountAndDate(r.Context(), FidelityManualAccountID, statementDate)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to clear old holdings: "+err.Error())
		return
	}

	// Upsert holdings for that date.
	var totalValueCents int64
	for _, h := range holdings {
		h.Date = database.DateOnly{Time: statementDate}
		h.AccountID = FidelityManualAccountID
		err := deps.db.UpsertDailyHolding(r.Context(), &h)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to save holding: "+err.Error())
			return
		}
		totalValueCents += h.ValueCents
	}

	// Update snapshots (daily and monthly) for that date.
	err = updatePortfolioSnapshots(r, deps, statementDate)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to update snapshots: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"message":"Successfully uploaded monthly statement"}`))
}

// Handles the current holdings CSV upload.
func handleFidelityHoldingsUpload(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	// Gets file.
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to get file from request")
		return
	}
	defer file.Close()

	// Validates filename: Portfolio_Positions_Mar-03-2026.csv
	// Format: Portfolio_Positions_Mon-DD-YYYY.csv
	re := regexp.MustCompile(`^Portfolio_Positions_([A-Z][a-z]{2})-(\d{2})-(\d{4})\.csv$`)
	matches := re.FindStringSubmatch(header.Filename)
	if len(matches) != 4 {
		writeJSONError(w, http.StatusBadRequest, "invalid filename format; expected Portfolio_Positions_Mon-DD-YYYY.csv")
		return
	}

	// Parses filename.
	monthStr := matches[1]
	day, _ := strconv.Atoi(matches[2])
	year, _ := strconv.Atoi(matches[3])

	// Creates a map of month strings to month enums.
	monthMap := map[string]time.Month{
		"Jan": time.January, "Feb": time.February, "Mar": time.March, "Apr": time.April,
		"May": time.May, "Jun": time.June, "Jul": time.July, "Aug": time.August,
		"Sep": time.September, "Oct": time.October, "Nov": time.November, "Dec": time.December,
	}
	month, ok := monthMap[monthStr]
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "invalid month in filename")
		return
	}

	// Creates today's date.
	today := time.Date(year, month, day, 0, 0, 0, 0, GetLocalLocation())

	// Ensure the Fidelity account exists.
	err = ensureFidelityAccountExists(r.Context(), deps)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to ensure Fidelity account exists: "+err.Error())
		return
	}

	// Parses CSV.
	holdings, err := parseFidelityHoldingsCSV(file)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to parse CSV: "+err.Error())
		return
	}

	// Clear existing manual holdings for today.
	err = deps.db.DeleteDailyHoldingsByAccountAndDate(r.Context(), FidelityManualAccountID, today)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to clear old holdings: "+err.Error())
		return
	}

	// Upsert holdings.
	for _, h := range holdings {
		h.Date = database.DateOnly{Time: today}
		h.AccountID = FidelityManualAccountID
		err = deps.db.UpsertDailyHolding(r.Context(), &h)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to save holding: "+err.Error())
			return
		}
	}

	// Update snapshots (daily and monthly) for that date.
	err = updatePortfolioSnapshots(r, deps, today)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to update snapshots: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"message":"Successfully uploaded current holdings"}`))
}

// Parses the Fidelity monthly holdings CSV.
func parseFidelityMonthlyCSV(r io.Reader) ([]database.DailyHolding, error) {
	reader := csv.NewReader(r)
	// Allow variable field counts.
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var holdings []database.DailyHolding
	parsingHoldings := false

	// Skip header rows.
	for _, row := range rows {
		if len(row) < 7 {
			continue
		}

		// Look for the "Symbol/CUSIP" header to start parsing individual holdings.
		if strings.Contains(row[0], "Symbol/CUSIP") {
			parsingHoldings = true
			continue
		}

		// Skip if we're not parsing holdings.
		if !parsingHoldings {
			continue
		}

		// Skip subtotal or empty lines.
		if strings.HasPrefix(row[0], "Subtotal") || row[0] == "" || row[0] == " " {
			continue
		}

		symbol := strings.TrimSpace(row[0])
		if symbol == "" {
			continue
		}

		quantity, _ := strconv.ParseFloat(strings.ReplaceAll(row[2], ",", ""), 64)
		valueCents := parseCents(row[5])

		var costBasis *int64
		if row[6] != "not applicable" && row[6] != "unavailable" && row[6] != "" {
			cb := parseCents(row[6])
			costBasis = &cb
		}

		holdings = append(holdings, database.DailyHolding{
			Symbol:         symbol,
			Quantity:       quantity,
			ValueCents:     valueCents,
			CostBasisCents: costBasis,
		})
	}

	return holdings, nil
}

func parseFidelityHoldingsCSV(r io.Reader) ([]database.DailyHolding, error) {
	reader := csv.NewReader(r)
	// Allow variable field counts.
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var holdings []database.DailyHolding
	if len(rows) < 2 {
		return holdings, nil
	}

	// Header is row 0.
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 15 {
			continue
		}

		symbol := strings.TrimSpace(row[2])
		// Remove "**" suffix often found in SPAXX
		symbol = strings.TrimSuffix(symbol, "**")

		if symbol == "" || strings.Contains(strings.ToLower(symbol), "pending activity") {
			continue
		}

		quantity, _ := strconv.ParseFloat(strings.ReplaceAll(row[4], ",", ""), 64)
		valueCents := parseCents(row[7])

		// Add logic to handle SPAXX and FCASH (Cash).
		if quantity == 0 && valueCents > 0 && (symbol == "SPAXX" || symbol == "FCASH") {
			quantity = float64(valueCents) / 100.0
		}

		var costBasis *int64
		if strings.TrimSpace(row[13]) != "" && strings.TrimSpace(row[13]) != "n/a" {
			cb := parseCents(row[13])
			costBasis = &cb
		}

		holdings = append(holdings, database.DailyHolding{
			Symbol:         symbol,
			Quantity:       quantity,
			ValueCents:     valueCents,
			CostBasisCents: costBasis,
		})
	}

	return holdings, nil
}

func parseCents(s string) int64 {
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	if s == "" || s == "--" {
		return 0
	}
	val, _ := strconv.ParseFloat(s, 64)
	return int64(math.Round(val * 100))
}

// Ensures the Fidelity account exists.
func ensureFidelityAccountExists(ctx context.Context, deps apiDependencies) error {
	// Check if item exists.
	item, err := deps.db.GetPlaidItemByItemID(ctx, FidelityManualItemID)
	if err != nil {
		return err
	}

	// If item doesn't exist, create it.
	if item == nil {
		item = &database.PlaidItem{
			ItemID:          FidelityManualItemID,
			AccessToken:     "manual",
			Status:          "OK",
			InstitutionName: stringPtr(FidelityInstitutionName),
			LastUpdated:     GetLocalNow(),
		}
		if err := deps.db.UpsertPlaidItem(ctx, item); err != nil {
			return err
		}
	}

	// Check if account exists.
	plaidAccounts, err := deps.db.ListPlaidAccounts(ctx)
	if err != nil {
		return err
	}

	found := false
	for _, a := range plaidAccounts {
		if a.AccountID == FidelityManualAccountID {
			found = true
			break
		}
	}

	if !found {
		account := database.PlaidAccount{
			PlaidItemID:    FidelityManualItemID,
			AccountID:      FidelityManualAccountID,
			Name:           "Fidelity Brokerage",
			Type:           "investment",
			CurrentBalance: 0, // Will be updated by holdings
		}
		if err := deps.db.UpsertPlaidAccounts(ctx, []database.PlaidAccount{account}); err != nil {
			return err
		}
	}

	return nil
}

func stringPtr(s string) *string { return &s }

// Updates both the daily and monthly snapshots for a given date.
func updatePortfolioSnapshots(r *http.Request, deps apiDependencies, date time.Time) error {
	// Fetch all daily holdings for this date.
	holdings, err := deps.db.ListDailyHoldings(r.Context(), date, date)
	if err != nil {
		return err
	}

	// Calculate total value for this date across ALL accounts (Plaid + Manual).
	var totalValueCents int64
	accountTotals := make(map[string]int64)
	for _, holding := range holdings {
		totalValueCents += holding.ValueCents
		accountTotals[holding.AccountID] += holding.ValueCents
	}

	// If we failed to get daily holdings for an account, add the total account balance to the total.
	allAccounts, err := deps.db.ListPlaidAccounts(r.Context())
	if err == nil {
		for _, account := range allAccounts {
			if isPlaidInvestment(account.Type) {
				_, exists := accountTotals[account.AccountID]
				if !exists {
					cents := int64(math.Round(account.CurrentBalance * 100))
					totalValueCents += cents
					accountTotals[account.AccountID] = cents
				}
			}
		}
	} else {
		log.Printf("fidelity: failed to list plaid accounts for gap-filling: %v", err)
	}

	// Upsert daily snapshot.
	snapshot := &database.DailySnapshot{
		Date:                database.DateOnly{Time: date},
		PortfolioValueCents: totalValueCents,
	}
	err = deps.db.UpsertDailySnapshot(r.Context(), snapshot)
	if err != nil {
		return err
	}

	// Update PlaidAccount current_balance for the Fidelity account.
	fidelityTotal, ok := accountTotals[FidelityManualAccountID]
	if ok {
		manualItem := &database.PlaidItem{
			ItemID:          FidelityManualItemID,
			InstitutionName: stringPtr(FidelityInstitutionName),
			AccessToken:     "manual",
			Status:          "OK",
			LastUpdated:     date,
		}
		_ = deps.db.UpsertPlaidItem(r.Context(), manualItem)

		account := database.PlaidAccount{
			PlaidItemID:    FidelityManualItemID,
			AccountID:      FidelityManualAccountID,
			Name:           "Fidelity Brokerage",
			Type:           "investment",
			CurrentBalance: float64(fidelityTotal) / 100.0,
		}
		_ = deps.db.UpsertPlaidAccounts(r.Context(), []database.PlaidAccount{account})
	}

	// Update monthly snapshots if it's month-end.
	year, month, day := date.Date()
	lastDayOfMonthDate := time.Date(year, month+1, 0, 0, 0, 0, 0, GetLocalLocation())
	lastDayOfMonth := lastDayOfMonthDate.Day()

	if day == lastDayOfMonth {
		// Record monthly snapshots if the account existed on or before this date.
		for accountID, total := range accountTotals {
			var accountCreatedAt *time.Time
			for _, acc := range allAccounts {
				if acc.AccountID == accountID {
					accountCreatedAt = acc.CreatedAt
					break
				}
			}

			// Skip any accounts created after this month.
			if accountCreatedAt != nil && accountCreatedAt.After(lastDayOfMonthDate) {
				continue
			}

			monthlySnapshot := &database.MonthlySnapshot{
				Month:               database.DateOnly{Time: lastDayOfMonthDate},
				AccountID:           accountID,
				PortfolioValueCents: total,
			}
			err := deps.db.UpsertMonthlySnapshot(r.Context(), monthlySnapshot)
			if err != nil {
				log.Printf("fidelity: failed to upsert monthly snapshot for %s: %v", accountID, err)
			}
		}
		// Update monthly net worth.
		_ = maybeWriteMonthlyNetWorth(r, deps, date, totalValueCents)
	}

	return nil
}
