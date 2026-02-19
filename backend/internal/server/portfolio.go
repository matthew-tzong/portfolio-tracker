package server

import (
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/matthewtzong/portfolio-tracker/backend/internal/database"
	"github.com/matthewtzong/portfolio-tracker/backend/internal/serverauth"
)

// Go's reference layout for YYYY-MM-DD.
const dateLayout = "2006-01-02"

// Current holding from Snaptrade in JSON format.
type HoldingJSON struct {
	AccountID   string  `json:"accountId"`
	AccountName string  `json:"accountName"`
	Symbol      string  `json:"symbol"`
	Quantity    float64 `json:"quantity"`
	ValueCents  int64   `json:"valueCents"`
}

// Current holdings response.
type HoldingsResponse struct {
	Holdings []HoldingJSON `json:"holdings"`
}

// Snapshot data point for charts in JSON format.
type SnapshotDataPoint struct {
	Date                string `json:"date"`
	PortfolioValueCents int64  `json:"portfolioValueCents"`
}

// Holding data point for charts in JSON format.
type HoldingDataPoint struct {
	Date        string  `json:"date"`
	AccountID   string  `json:"accountId"`
	AccountName string  `json:"accountName,omitempty"`
	Symbol      string  `json:"symbol"`
	Quantity    float64 `json:"quantity,omitempty"`
	ValueCents  int64   `json:"valueCents"`
}

// Portfolio snapshots response.
type SnapshotsResponse struct {
	Daily   []SnapshotDataPoint `json:"daily"`
	Monthly []SnapshotDataPoint `json:"monthly"`
}

// Portfolio holdings history in JSON format.
type HoldingsHistoryResponse struct {
	Daily []HoldingDataPoint `json:"daily"`
}

// Registers the portfolio routes.
func registerPortfolioRoutes(mux *http.ServeMux, deps apiDependencies) {
	// GET /api/portfolio/holdings returns current positions from Snaptrade.
	mux.Handle("/api/portfolio/holdings", serverauth.JWTAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		handleGetHoldings(w, r, deps)
	})))

	// GET /api/portfolio/snapshots returns daily and monthly snapshot data.
	mux.Handle("/api/portfolio/snapshots", serverauth.JWTAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		handleGetSnapshots(w, r, deps)
	})))

	// GET /api/portfolio/holdings/history returns daily and monthly holdings data.
	mux.Handle("/api/portfolio/holdings/history", serverauth.JWTAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		handleGetHoldingsHistory(w, r, deps)
	})))
}

// Fetches current holdings from Snaptrade.
func handleGetHoldings(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	w.Header().Set("Content-Type", "application/json")

	if deps.db == nil || deps.snaptradeClient == nil {
		writeJSONError(w, http.StatusInternalServerError, "database or snaptrade client not configured")
		return
	}

	// Validate the authenticated user exists
	if userID, ok := serverauth.UserIDFromContext(r.Context()); !ok || userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "missing authenticated user")
		return
	}

	// Get Snaptrade user
	snapUser, err := deps.db.GetSnaptradeUser(r.Context())
	if err != nil || snapUser == nil {
		writeJSONError(w, http.StatusNotFound, "snaptrade user not found")
		return
	}

	// List all accounts
	accounts, err := deps.snaptradeClient.ListAccounts(snapUser.UserID, snapUser.UserSecret)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list accounts: "+err.Error())
		return
	}

	// Fetch positions for each account
	var holdings []HoldingJSON
	accountMap := make(map[string]string)
	for _, account := range accounts {
		accountMap[account.ID] = account.Name

		// List positions for the account.
		positions, err := deps.snaptradeClient.ListAccountPositions(snapUser.UserID, snapUser.UserSecret, account.ID)
		if err != nil {
			continue
		}

		// Add the positions to the holdings.
		for _, pos := range positions {
			holdings = append(holdings, HoldingJSON{
				AccountID:   account.ID,
				AccountName: account.Name,
				Symbol:      pos.Symbol,
				Quantity:    pos.Quantity,
				ValueCents:  pos.ValueCents,
			})
		}
	}

	resp := HoldingsResponse{
		Holdings: holdings,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// Fetches daily and monthly snapshot data.
func handleGetSnapshots(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	w.Header().Set("Content-Type", "application/json")

	if deps.db == nil {
		writeJSONError(w, http.StatusInternalServerError, "database is not configured")
		return
	}

	// Validate the authenticated user exists
	if userID, ok := serverauth.UserIDFromContext(r.Context()); !ok || userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "missing authenticated user")
		return
	}

	// Get necessary dates and parameters.
	accountID := r.URL.Query().Get("accountId")
	now := time.Now().UTC()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	dailyStart := dayStart.AddDate(0, 0, -30)

	// List the daily snapshots.
	dailySnapshots, err := deps.db.ListDailySnapshots(r.Context(), dailyStart, dayStart)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list daily snapshots: "+err.Error())
		return
	}

	// List the monthly snapshots
	monthlyStart := time.Date(now.Year()-2, 1, 1, 0, 0, 0, 0, time.UTC)
	var monthlyPoints []SnapshotDataPoint
	if accountID != "" {
		// List the monthly snapshots for a given account.
		byAccount, err := deps.db.ListMonthlySnapshotsByAccount(r.Context(), monthlyStart, dayStart, accountID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to list monthly snapshots by account: "+err.Error())
			return
		}
		// Convert the monthly snapshots to the data points.
		monthlyPoints = make([]SnapshotDataPoint, 0, len(byAccount))
		for _, snapshot := range byAccount {
			monthlyPoints = append(monthlyPoints, SnapshotDataPoint{
				Date:                snapshot.Month.Format(dateLayout),
				PortfolioValueCents: snapshot.PortfolioValueCents,
			})
		}
	} else {
		// List the total monthly snapshot across all accounts.
		allMonthly, err := deps.db.ListMonthlySnapshots(r.Context(), monthlyStart, dayStart)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to list monthly snapshots: "+err.Error())
			return
		}
		sumByMonth := make(map[string]int64)
		for _, snapshot := range allMonthly {
			month := snapshot.Month.Format(dateLayout)
			sumByMonth[month] += snapshot.PortfolioValueCents
		}
		for month, sum := range sumByMonth {
			monthlyPoints = append(monthlyPoints, SnapshotDataPoint{Date: month, PortfolioValueCents: sum})
		}
		sortSnapshotDataPoints(monthlyPoints)
	}

	// Convert the daily snapshots to the data points.
	dailyPoints := make([]SnapshotDataPoint, 0, len(dailySnapshots))
	for _, snapshot := range dailySnapshots {
		dailyPoints = append(dailyPoints, SnapshotDataPoint{
			Date:                snapshot.Date.Format(dateLayout),
			PortfolioValueCents: snapshot.PortfolioValueCents,
		})
	}

	resp := SnapshotsResponse{
		Daily:   dailyPoints,
		Monthly: monthlyPoints,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// Sorts the snapshot data points by date.
func sortSnapshotDataPoints(points []SnapshotDataPoint) {
	sort.Slice(points, func(i, j int) bool { return points[i].Date < points[j].Date })
}

// Fetches daily holdings history.
func handleGetHoldingsHistory(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	w.Header().Set("Content-Type", "application/json")

	if deps.db == nil {
		writeJSONError(w, http.StatusInternalServerError, "database is not configured")
		return
	}

	// Validate the authenticated user exists
	if userID, ok := serverauth.UserIDFromContext(r.Context()); !ok || userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "missing authenticated user")
		return
	}

	// Get the account ID and symbol from the query parameters.
	accountID := r.URL.Query().Get("accountId")
	symbol := r.URL.Query().Get("symbol")

	now := time.Now().UTC()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	dailyStart := dayStart.AddDate(0, 0, -30)

	var dailyHoldings []database.DailyHolding
	var err error

	if accountID != "" {
		dailyHoldings, err = deps.db.ListDailyHoldingsByAccount(r.Context(), accountID, dailyStart, dayStart)
	} else if symbol != "" {
		dailyHoldings, err = deps.db.ListDailyHoldingsBySymbol(r.Context(), symbol, dailyStart, dayStart)
	} else {
		dailyHoldings, err = deps.db.ListDailyHoldings(r.Context(), dailyStart, dayStart)
	}

	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list daily holdings: "+err.Error())
		return
	}

	accountMap := make(map[string]string)

	// Convert the daily holdings to the data points.
	dailyPoints := make([]HoldingDataPoint, 0, len(dailyHoldings))
	for _, holding := range dailyHoldings {
		dailyPoints = append(dailyPoints, HoldingDataPoint{
			Date:        holding.Date.Format(dateLayout),
			AccountID:   holding.AccountID,
			AccountName: accountMap[holding.AccountID],
			Symbol:      holding.Symbol,
			Quantity:    holding.Quantity,
			ValueCents:  holding.ValueCents,
		})
	}

	resp := HoldingsHistoryResponse{
		Daily: dailyPoints,
	}

	_ = json.NewEncoder(w).Encode(resp)
}
