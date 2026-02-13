package server

import (
	"encoding/json"
	"math"
	"net/http"

	"github.com/matthewtzong/portfolio-tracker/backend/internal/database"
	"github.com/matthewtzong/portfolio-tracker/backend/internal/serverauth"
	"github.com/matthewtzong/portfolio-tracker/backend/internal/snaptrade"
)

// Account View Model
type AccountJSON struct {
	Provider     string  `json:"provider"`
	PlaidItemID  *string `json:"plaidItemId,omitempty"`
	AccountID    string  `json:"accountId"`
	Name         string  `json:"name"`
	Mask         *string `json:"mask,omitempty"`
	Type         string  `json:"type"`
	Subtype      *string `json:"subtype,omitempty"`
	BalanceCents int64   `json:"balanceCents"`
	IsLiability  bool    `json:"isLiability"`
}

// List of Accounts and Net Worth breakdown.
type AccountsResponse struct {
	Accounts         []AccountJSON `json:"accounts"`
	NetWorthCents    int64         `json:"netWorthCents"`
	CashCents        int64         `json:"cashCents"`
	InvestmentsCents int64         `json:"investmentsCents"`
	LiabilitiesCents int64         `json:"liabilitiesCents"`
}

// Registers the accounts route.
func registerAccountsRoutes(mux *http.ServeMux, deps apiDependencies) {
	// GET /api/accounts returns all accounts and the current net worth breakdown.
	mux.Handle("/api/accounts", serverauth.JWTAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		handleGetAccounts(w, r, deps)
	})))
}

// Fetches the accounts and net worth breakdown.
func handleGetAccounts(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
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

	// Load Plaid accounts from the database.
	plaidAccounts, err := deps.db.ListPlaidAccounts(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list Plaid accounts: "+err.Error())
		return
	}

	var (
		accounts         []AccountJSON
		cashCents        int64
		investmentsCents int64
		liabilitiesCents int64
	)

	// Converts the Plaid accounts to the AccountJSON view model.
	// Plaid accounts only contribute to cash (HYSA, checking, CDs) or liabilities (credit cards).
	for _, a := range plaidAccounts {
		accountJSON, cashDelta, _, liabilityDelta := loadPlaidAccounts(a)
		accounts = append(accounts, accountJSON)
		cashCents += cashDelta
		liabilitiesCents += liabilityDelta
	}

	// Load Snaptrade accounts from the database.
	if deps.snaptradeClient != nil {
		snapUser, err := deps.db.GetSnaptradeUser(r.Context())
		if err == nil && snapUser != nil {
			snapAccounts, err := deps.snaptradeClient.ListAccounts(snapUser.UserID, snapUser.UserSecret)
			if err == nil {
				for _, a := range snapAccounts {
					accountJSON, investDelta := loadSnaptradeAccounts(a)
					accounts = append(accounts, accountJSON)
					investmentsCents += investDelta
				}
			}
		}
	}

	// Net worth = assets (cash + investments) - liabilities.
	netWorthCents := cashCents + investmentsCents - liabilitiesCents

	resp := AccountsResponse{
		Accounts:         accounts,
		NetWorthCents:    netWorthCents,
		CashCents:        cashCents,
		InvestmentsCents: investmentsCents,
		LiabilitiesCents: liabilitiesCents,
	}

	// Return the accounts and net worth breakdown.
	_ = json.NewEncoder(w).Encode(resp)
}

// Load Plaid Accounts from the database
func loadPlaidAccounts(a database.PlaidAccount) (AccountJSON, int64, int64, int64) {
	var (
		subtype *string
		mask    *string
	)
	if a.Subtype != nil {
		subtype = a.Subtype
	}
	if a.Mask != nil {
		mask = a.Mask
	}

	// Convert the floating-point balance to cents.
	rawCents := int64(math.Round(a.CurrentBalance * 100))

	// Checks if the account is a liability to display as negative balance.
	isLiability := isPlaidLiability(a.Type)
	balanceCents := rawCents
	if isLiability {
		balanceCents = -rawCents
	}

	// Checks if the account is cash or liability for net worth calculation.
	var cashDelta, investDelta, liabilityDelta int64
	if isLiability {
		liabilityDelta = rawCents
	} else if isPlaidCash(a.Type, subtype) {
		cashDelta = rawCents
	} else {
		cashDelta = rawCents
	}

	// Converts the Plaid account to the AccountJSON view model.
	account := AccountJSON{
		Provider:     "plaid",
		PlaidItemID:  &a.PlaidItemID,
		AccountID:    a.AccountID,
		Name:         a.Name,
		Mask:         mask,
		Type:         a.Type,
		Subtype:      subtype,
		BalanceCents: balanceCents,
		IsLiability:  isLiability,
	}

	return account, cashDelta, investDelta, liabilityDelta
}

// Returns true if the Plaid account should be treated as a liability.
func isPlaidLiability(accountType string) bool {
	switch accountType {
	case "credit", "loan":
		return true
	default:
		return false
	}
}

// Returns true if the Plaid account should be classified as cash (HYSA, checking, CDs)
func isPlaidCash(accountType string, subtype *string) bool {
	if accountType != "depository" {
		return false
	}
	if subtype == nil {
		return true
	}

	switch *subtype {
	case "checking", "savings", "money market", "cd":
		return true
	default:
		return false
	}
}

// Load Snaptrade Accounts from the database
func loadSnaptradeAccounts(a snaptrade.Account) (AccountJSON, int64) {
	balanceCents := int64(math.Round(a.BalanceAmount * 100))

	// Extract last 4 digits from account number for mask if available.
	var mask *string
	if len(a.Number) >= 4 {
		last4 := a.Number[len(a.Number)-4:]
		mask = &last4
	}

	account := AccountJSON{
		Provider:     "snaptrade",
		PlaidItemID:  nil,
		AccountID:    a.ID,
		Name:         a.Name,
		Mask:         mask,
		Type:         "investment",
		Subtype:      nil,
		BalanceCents: balanceCents,
		IsLiability:  false,
	}

	return account, balanceCents
}
