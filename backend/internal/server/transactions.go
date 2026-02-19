package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/matthewtzong/portfolio-tracker/backend/internal/database"
	"github.com/matthewtzong/portfolio-tracker/backend/internal/plaid"
	"github.com/matthewtzong/portfolio-tracker/backend/internal/serverauth"
)

// Registers webhook and transaction API routes.
func registerTransactionsRoutes(mux *http.ServeMux, deps apiDependencies) {
	// Webhook: called by Plaid.
	mux.HandleFunc("/api/webhooks/plaid", func(w http.ResponseWriter, r *http.Request) {
		HandlePlaidWebhook(w, r, deps)
	})
	// Protected transaction routes.
	mux.Handle("/api/transactions", serverauth.JWTAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleListTransactions(w, r, deps)
	})))
	mux.Handle("/api/transactions/summary", serverauth.JWTAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleGetTransactionsSummary(w, r, deps)
	})))
	mux.Handle("/api/transactions/sync", serverauth.JWTAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleSyncTransactions(w, r, deps)
	})))
	mux.Handle("/api/categories", serverauth.JWTAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleListCategories(w, r, deps)
	})))
	mux.Handle("/api/budget", serverauth.JWTAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleBudget(w, r, deps)
	})))
}

// Handles Plaid webhook for transaction sync updates.
func HandlePlaidWebhook(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	// Decodes the request body into a plaidWebhookPayload.
	var payload plaidWebhookPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Returns if not a transaction sync update.
	if payload.WebhookCode != "SYNC_UPDATES_AVAILABLE" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Returns if item_id is missing.
	if payload.ItemID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing item_id")
		return
	}

	// Returns if database is not configured.
	if deps.db == nil {
		writeJSONError(w, http.StatusInternalServerError, "database not configured")
		return
	}

	// Sets new_transactions_pending flag for the item.
	err = deps.db.SetItemNewTransactionsPending(r.Context(), payload.ItemID, true)
	if err != nil {
		log.Printf("webhook: set new_transactions_pending for item %s: %v", payload.ItemID, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to record webhook")
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Runs cursor-based sync and upserts or removes from DB.
func SyncTransactionsForItem(ctx context.Context, db *database.Client, plaidClient *plaid.Client, item *database.PlaidItem) error {
	// Returns if missing dependencies.
	if plaidClient == nil || db == nil {
		return nil
	}
	cursor := ""
	if item.TransactionsCursor != nil {
		cursor = *item.TransactionsCursor
	}

	// Gets categories and rules.
	categories, err := db.ListCategories(ctx)
	if err != nil {
		return err
	}
	rules, err := db.ListCategoryRules(ctx)
	if err != nil {
		return err
	}

	// Maps Plaid primary category names to our category IDs.
	plaidNameToCategoryID := make(map[string]int64)
	var uncategorizedID int64
	for _, category := range categories {
		if category.PlaidName != nil {
			plaidNameToCategoryID[*category.PlaidName] = category.ID
		} else if category.Name == "Uncategorized" {
			uncategorizedID = category.ID
		}
	}

	// Loops until no more transactions.
	for {
		// Gets transactions from Plaid.
		result, err := plaidClient.TransactionsSync(ctx, item.AccessToken, cursor)
		if err != nil {
			return err
		}

		// Converts Plaid transactions to our DB model.
		var toUpsert []database.Transaction
		for _, item := range result.Added {
			transaction := plaidTransactionToDB(item, plaidNameToCategoryID, uncategorizedID, rules)
			toUpsert = append(toUpsert, transaction)
		}
		for _, item := range result.Modified {
			transaction := plaidTransactionToDB(item, plaidNameToCategoryID, uncategorizedID, rules)
			toUpsert = append(toUpsert, transaction)
		}

		// Upserts the transactions.
		if len(toUpsert) > 0 {
			err = db.UpsertTransactions(ctx, toUpsert)
			if err != nil {
				return err
			}
		}

		// Deletes the removed transactions.
		if len(result.Removed) > 0 {
			ids := make([]string, len(result.Removed))
			for i, removed := range result.Removed {
				ids[i] = removed.TransactionID
			}
			err = db.DeleteTransactionsByPlaidIDs(ctx, ids)
			if err != nil {
				return err
			}
		}

		// Updates the cursor and checks if there are more transactions.
		cursor = result.NextCursor
		if !result.HasMore {
			break
		}
	}

	// Updates the cursor and clears the new_transactions_pending flag.
	return db.UpdatePlaidItemCursorAndPending(ctx, item.ItemID, cursor, false)
}

// Converts a Plaid transaction to our DB model.
func plaidTransactionToDB(p plaid.PlaidTransaction, plaidNameToCategoryID map[string]int64, uncategorizedID int64, rules []database.CategoryRule) database.Transaction {
	// Sets up transaction fields.
	amountCents := int64(math.Round(p.Amount * 100))
	date, _ := time.Parse("2006-01-02", p.Date)
	var categoryID *int64
	name := strings.ToLower(p.Name)
	merchantName := ""
	if p.MerchantName != nil {
		merchantName = strings.ToLower(*p.MerchantName)
	}

	// Matches to categories, first by rules then by Plaid primary category.
	for _, rule := range rules {
		match := strings.ToLower(rule.MatchString)
		if strings.Contains(name, match) || (merchantName != "" && strings.Contains(merchantName, match)) {
			categoryID = &rule.CategoryID
			break
		}
	}
	if categoryID == nil {
		if len(p.Category) > 0 {
			primary := p.Category[0]
			if id, ok := plaidNameToCategoryID[primary]; ok {
				categoryID = &id
			} else {
				categoryID = &uncategorizedID
			}
		}
	}

	// Returns the transaction.
	now := time.Now().UTC()
	return database.Transaction{
		PlaidAccountID:     p.AccountID,
		PlaidTransactionID: p.TransactionID,
		Date:               date,
		AmountCents:        amountCents,
		Name:               p.Name,
		MerchantName:       p.MerchantName,
		CategoryID:         categoryID,
		Pending:            p.Pending,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// Returns the summary of transactions for the given month.
func handleGetTransactionsSummary(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if deps.db == nil {
		writeJSONError(w, http.StatusInternalServerError, "database not configured")
		return
	}
	month := r.URL.Query().Get("month")
	if month == "" {
		writeJSONError(w, http.StatusBadRequest, "month required (YYYY-MM)")
		return
	}

	// Gets the transactions and categories.
	list, err := deps.db.ListTransactions(r.Context(), database.ListTransactionsFilter{Month: month})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	categories, err := deps.db.ListCategories(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

    // Calculates the income, expenses, and invested amounts.
	var investmentsID, transferCategoryID int64
	expenseCategoryIDs := make(map[int64]bool)
	for _, category := range categories {
		if category.Name == "Investments" {
			investmentsID = category.ID
		}
		if category.Name == "Transfer" {
			transferCategoryID = category.ID
		}
		if category.Expense {
			expenseCategoryIDs[category.ID] = true
		}
	}

	// Loops through the transactions and calculates the income, expenses, and invested amounts.
	var incomeCents, expensesCents, investedCents int64
	for _, transaction := range list {
		// Skip bank transfers from expense calculations.
		if transaction.CategoryID != nil && *transaction.CategoryID == transferCategoryID {
			continue
		}
		if transaction.AmountCents < 0 {
			incomeCents += -transaction.AmountCents
		} else {
			if transaction.CategoryID != nil {
				if *transaction.CategoryID == investmentsID {
					investedCents += transaction.AmountCents
				} else if expenseCategoryIDs[*transaction.CategoryID] {
					expensesCents += transaction.AmountCents
				}
			} else {
				expensesCents += transaction.AmountCents
			}
		}
	}

	// Encodes the response.
	err = json.NewEncoder(w).Encode(transactionsSummaryResponse{
		IncomeCents:   incomeCents,
		ExpensesCents: expensesCents,
		InvestedCents: investedCents,
	})
	if err != nil {
		log.Printf("get transactions summary encode: %v", err)
	}
}

// Returns transactions for the given month, category, and search.
func handleListTransactions(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if deps.db == nil {
		writeJSONError(w, http.StatusInternalServerError, "database not configured")
		return
	}

	// Parses the query parameters.
	query := r.URL.Query()
	filter := database.ListTransactionsFilter{
		Month:  query.Get("month"),
		Search: query.Get("search"),
	}
	category := query.Get("category")
	if category != "" {
		var id int64
		if _, err := fmt.Sscanf(category, "%d", &id); err == nil {
			filter.CategoryID = &id
		}
	}

	// Gets the transactions.
	list, err := deps.db.ListTransactions(r.Context(), filter)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Maps category IDs to names.
	categories, _ := deps.db.ListCategories(r.Context())
	categoryNameByID := make(map[int64]string)
	for _, category := range categories {
		categoryNameByID[category.ID] = category.Name
	}

	// Converts the transactions to our API model.
	output := make([]transactionJSON, len(list))
	for i, transaction := range list {
		output[i] = transactionJSON{
			ID:           transaction.ID,
			Date:         transaction.Date.Format("2006-01-02"),
			AmountCents:  transaction.AmountCents,
			Name:         transaction.Name,
			MerchantName: transaction.MerchantName,
			CategoryID:   transaction.CategoryID,
			Pending:      transaction.Pending,
		}
		if transaction.CategoryID != nil {
			output[i].CategoryName = categoryNameByID[*transaction.CategoryID]
		}
	}

	// Encodes the response.
	err = json.NewEncoder(w).Encode(transactionsResponse{Transactions: output})
	if err != nil {
		log.Printf("list transactions encode: %v", err)
	}
}

// Syncs transactions for all items with new_transactions_pending.
func handleSyncTransactions(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if deps.db == nil || deps.plaidClient == nil {
		writeJSONError(w, http.StatusInternalServerError, "database or Plaid not configured")
		return
	}

	// Gets the items with pending transactions.
	items, err := deps.db.ListPlaidItemsWithPendingTransactions(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// If no new transactions, return.
	if len(items) == 0 {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"synced":0,"message":"no items with pending transactions"}`))
		return
	}

	// Syncs transactions for each item.
	for _, item := range items {
		err = SyncTransactionsForItem(r.Context(), deps.db, deps.plaidClient, &item)
		if err != nil {
			log.Printf("sync transactions for item %s: %v", item.ItemID, err)
			writeJSONError(w, http.StatusInternalServerError, "sync failed: "+err.Error())
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"synced":` + strconv.Itoa(len(items)) + `}`))
}

// Returns all categories.
func handleListCategories(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if deps.db == nil {
		writeJSONError(w, http.StatusInternalServerError, "database not configured")
		return
	}
	list, err := deps.db.ListCategories(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	output := make([]categoryJSON, len(list))
	for i, category := range list {
		output[i] = categoryJSON{ID: category.ID, Name: category.Name, Expense: category.Expense}
	}
	err = json.NewEncoder(w).Encode(map[string]interface{}{"categories": output})
	if err != nil {
		log.Printf("list categories encode: %v", err)
	}
}

// Returns or updates the current budget.
func handleBudget(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	if deps.db == nil {
		writeJSONError(w, http.StatusInternalServerError, "database not configured")
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleGetBudget(w, r, deps)
	case http.MethodPost, http.MethodPut:
		handleUpdateBudget(w, r, deps)
	default:
		methodNotAllowed(w, http.MethodGet)
	}
}

// Returns the budget allocations and spent-by-category for the requested month.
func handleGetBudget(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	w.Header().Set("Content-Type", "application/json")

	month := r.URL.Query().Get("month")
	if month == "" {
		writeJSONError(w, http.StatusBadRequest, "month required (YYYY-MM)")
		return
	}

	// Load the current budget
	budget, err := deps.db.GetBudget(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Maps the allocations to a map.
	allocations := map[string]int64{}
	if budget != nil && budget.Allocations != nil {
		for k, v := range budget.Allocations {
			allocations[k] = v
		}
	}

	// Computes the monthly spent by category.
	spendingMap, err := calculateMonthlySpentByCategory(r.Context(), deps.db, month)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Returns the response.
	resp := budgetResponse{
		Month:       month,
		Allocations: allocations,
		Spent:       spendingMap,
	}
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		log.Printf("get budget encode: %v", err)
	}
}

// Updates the budget allocations.
func handleUpdateBudget(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	w.Header().Set("Content-Type", "application/json")

	// Decodes the request body into an updateBudgetRequest.
	var req updateBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Sets the allocations to an empty map if nil.
	if req.Allocations == nil {
		req.Allocations = map[string]int64{}
	}

	// Creates the new budget.
	budget := &database.Budget{
		ID:          1,
		Allocations: req.Allocations,
	}

	if err := deps.db.UpsertBudget(r.Context(), budget); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return the updated budget
	err := json.NewEncoder(w).Encode(budget)
	if err != nil {
		log.Printf("update budget encode: %v", err)
	}
}

// Computes the monthly spent by category.
func calculateMonthlySpentByCategory(ctx context.Context, dbClient *database.Client, month string) (map[string]int64, error) {
	monthlySpending := make(map[string]int64)
	if month == "" {
		return monthlySpending, nil
	}
	if dbClient == nil {
		return monthlySpending, nil
	}

	// Fetch transactions for the month.
	transactions, err := dbClient.ListTransactions(ctx, database.ListTransactionsFilter{Month: month})
	if err != nil {
		return nil, err
	}

	// Fetches and filters to expense categories.
	categories, err := dbClient.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	expenseCategoryIDs := make(map[int64]bool)
	for _, category := range categories {
		if category.Expense {
			expenseCategoryIDs[category.ID] = true
		}
	}

    // Loops through the transactions and calculates the monthly spent by category.
	for _, transaction := range transactions {
		categoryID := *transaction.CategoryID
		if !expenseCategoryIDs[categoryID] {
			continue
		}
		categoryName := categories[categoryID].Name
        monthlySpending[categoryName] += transaction.AmountCents
	}
	return monthlySpending, nil
}

// Category for API.
type categoryJSON struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Expense bool   `json:"expense"`
}

// Plaid webhook payload.
type plaidWebhookPayload struct {
	WebhookType string `json:"webhook_type"`
	WebhookCode string `json:"webhook_code"`
	ItemID      string `json:"item_id"`
}

// Transactions response for API.
type transactionsResponse struct {
	Transactions []transactionJSON `json:"transactions"`
}

// Transaction for API.
type transactionJSON struct {
	ID           int64   `json:"id"`
	Date         string  `json:"date"`
	AmountCents  int64   `json:"amountCents"`
	Name         string  `json:"name"`
	MerchantName *string `json:"merchantName,omitempty"`
	CategoryID   *int64  `json:"categoryId,omitempty"`
	CategoryName string  `json:"categoryName,omitempty"`
	Pending      bool    `json:"pending"`
}

// Monthly summary response: income (inflows), expenses (outflows to expense categories), invested (outflows to Investments).
type transactionsSummaryResponse struct {
	IncomeCents   int64 `json:"incomeCents"`
	ExpensesCents int64 `json:"expensesCents"`
	InvestedCents int64 `json:"investedCents"`
}

// Budget API response
type budgetResponse struct {
	Month       string           `json:"month"`
	Allocations map[string]int64 `json:"allocations"`
	Spent       map[string]int64 `json:"spent"`
}

// Budget update request
type updateBudgetRequest struct {
	Allocations map[string]int64 `json:"allocations"`
}
