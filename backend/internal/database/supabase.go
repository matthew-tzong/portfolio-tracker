package database

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Supabase client for database operations.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

// Creates a Supabase client from environment variables.
func NewClientFromEnv() (*Client, error) {
	url := strings.TrimSuffix(os.Getenv("SUPABASE_URL"), "/")
	apiKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")

	if url == "" || apiKey == "" {
		return nil, errors.New("SUPABASE_URL and SUPABASE_SERVICE_ROLE_KEY must be set")
	}

	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    url,
		apiKey:     apiKey,
	}, nil
}

// Returns the PostgREST URL for a table.
func (c *Client) restURL(table string) string {
	return c.baseURL + "/rest/v1/" + table
}

// Executes an HTTP request with Supabase auth headers.
func (c *Client) doRequest(ctx context.Context, method, url string, body interface{}) (*http.Response, error) {
	// If there is a body, marshal it into a JSON reader.
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	// Creates a new request with the context, method, URL, and body reader.
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	// Sets the headers for the request.
	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")

	return c.httpClient.Do(req)
}

// Returns the JSON-safe representation of a PlaidItem.
func (p *PlaidItem) ToJSON() PlaidItemJSON {
	name := ""
	if p.InstitutionName != nil {
		name = *p.InstitutionName
	}

	return PlaidItemJSON{
		ItemID:          p.ItemID,
		InstitutionName: name,
		Status:          p.Status,
		LastUpdated:     p.LastUpdated,
	}
}

// Inserts or updates a Plaid item.
func (c *Client) UpsertPlaidItem(ctx context.Context, item *PlaidItem) error {
	url := c.restURL("plaid_items") + "?on_conflict=item_id"

	resp, err := c.doRequest(ctx, http.MethodPost, url, item)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase upsert plaid_items failed: %s", string(body))
	}
	return nil
}

// Returns all Plaid items.
func (c *Client) ListPlaidItems(ctx context.Context) ([]PlaidItem, error) {
	url := c.restURL("plaid_items") + "?order=last_updated.desc"

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase list plaid_items failed: %s", string(body))
	}

	// Decodes the response body into a slice of Plaid items.
	var items []PlaidItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}
	return items, nil
}

// Returns a Plaid item by its Plaid item_id.
func (c *Client) GetPlaidItemByItemID(ctx context.Context, itemID string) (*PlaidItem, error) {
	url := c.restURL("plaid_items") + "?item_id=eq." + itemID

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase get plaid_item failed: %s", string(body))
	}

	// Decodes the response body into a Plaid item.
	var items []PlaidItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}

// Returns a Plaid item by institution_id (for detecting existing connections).
func (c *Client) GetPlaidItemByInstitutionID(ctx context.Context, institutionID string) (*PlaidItem, error) {
	url := c.restURL("plaid_items") + "?institution_id=eq." + institutionID + "&limit=1"

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase get plaid_item by institution_id failed: %s", string(body))
	}

	// Decodes the response body into a Plaid item
	var items []PlaidItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}

// Deletes a Plaid item by its Plaid item_id.
func (c *Client) DeletePlaidItem(ctx context.Context, itemID string) error {
	url := c.restURL("plaid_items") + "?item_id=eq." + itemID

	resp, err := c.doRequest(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase delete plaid_item failed: %s", string(body))
	}
	return nil
}

// Inserts or updates multiple Plaid accounts.
func (c *Client) UpsertPlaidAccounts(ctx context.Context, accounts []PlaidAccount) error {
	if len(accounts) == 0 {
		return nil
	}

	url := c.restURL("plaid_accounts") + "?on_conflict=account_id"

	resp, err := c.doRequest(ctx, http.MethodPost, url, accounts)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase upsert plaid_accounts failed: %s", string(body))
	}
	return nil
}

// Deletes all accounts for a Plaid item.
func (c *Client) DeletePlaidAccountsByItemID(ctx context.Context, itemID string) error {
	url := c.restURL("plaid_accounts") + "?plaid_item_id=eq." + itemID

	resp, err := c.doRequest(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase delete plaid_accounts failed: %s", string(body))
	}
	return nil
}

// Returns all Plaid accounts.
func (c *Client) ListPlaidAccounts(ctx context.Context) ([]PlaidAccount, error) {
	url := c.restURL("plaid_accounts")

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase list plaid_accounts failed: %s", string(body))
	}

	// Decodes the response body into a slice of Plaid accounts.
	var accounts []PlaidAccount
	err = json.NewDecoder(resp.Body).Decode(&accounts)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

// Converts a SnaptradeConnection to its JSON-safe representation.
func (s *SnaptradeConnection) ToJSON() SnaptradeConnectionJSON {
	return SnaptradeConnectionJSON{
		ID:         s.ConnID,
		Brokerage:  s.Brokerage,
		Status:     s.Status,
		LastSynced: s.LastSynced,
	}
}

// Returns the Snaptrade user.
func (c *Client) GetSnaptradeUser(ctx context.Context) (*SnaptradeUser, error) {
	url := c.restURL("snaptrade_user") + "?limit=1"
	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil
	}

	// Decodes the response body into a Snaptrade user.
	var users []SnaptradeUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil || len(users) == 0 {
		return nil, nil
	}
	return &users[0], nil
}

// Inserts a new Snaptrade user into the database.
func (c *Client) CreateSnaptradeUser(ctx context.Context, userID, userSecret string) (*SnaptradeUser, error) {
	user := &SnaptradeUser{
		UserID:     userID,
		UserSecret: userSecret,
	}

	url := c.restURL("snaptrade_user")
	resp, err := c.doRequest(ctx, http.MethodPost, url, user)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase insert snaptrade_user failed: %s", string(body))
	}

	return user, nil
}

// Upserts or Delete Snaptrade connections
func (c *Client) UpsertSnaptradeConnections(ctx context.Context, conns []SnaptradeConnection) error {
	// If no connections from API, delete all.
	if len(conns) == 0 {
		deletionURL := c.restURL("snaptrade_connections")
		deletionResponse, err := c.doRequest(ctx, http.MethodDelete, deletionURL+"?id=gt.0", nil)
		if err != nil {
			return err
		}
		deletionResponse.Body.Close()
		return nil
	}

	// Upsert by conn_id (update existing, insert new).
	url := c.restURL("snaptrade_connections") + "?on_conflict=conn_id"
	resp, err := c.doRequest(ctx, http.MethodPost, url, conns)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase upsert snaptrade_connections failed: %s", string(body))
	}

	// Delete connections no longer returned by the API.
	existingIDs := make([]string, 0, len(conns))
	for _, conn := range conns {
		existingIDs = append(existingIDs, conn.ConnID)
	}
	return c.deleteSnaptradeConnectionsNotIn(ctx, existingIDs)
}

// Deletes Snaptrade connections not in the list of existing IDs.
func (c *Client) deleteSnaptradeConnectionsNotIn(ctx context.Context, existingIDs []string) error {
	if len(existingIDs) == 0 {
		return nil
	}

	// PostgREST: conn_id=not.in.(existingIDs)
	var b strings.Builder
	b.WriteString("not.in.(")
	for i, id := range existingIDs {
		if i > 0 {
			b.WriteString(",")
		}
		escaped := strings.ReplaceAll(id, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		b.WriteString("\"" + escaped + "\"")
	}
	b.WriteString(")")

	// Delete the connections
	deletionURL := c.restURL("snaptrade_connections") + "?conn_id=" + url.QueryEscape(b.String())
	deletionResponse, err := c.doRequest(ctx, http.MethodDelete, deletionURL, nil)
	if err != nil {
		return err
	}
	defer deletionResponse.Body.Close()
	if deletionResponse.StatusCode < 200 || deletionResponse.StatusCode >= 300 {
		body, _ := io.ReadAll(deletionResponse.Body)
		return fmt.Errorf("supabase delete snaptrade_connections not in failed: %s", string(body))
	}
	return nil
}

// Updates the status and last_synced for existing connections.
func (c *Client) UpdateSnaptradeConnectionStatuses(ctx context.Context, conns []SnaptradeConnection) error {
	for _, conn := range conns {
		payload := map[string]interface{}{
			"status":      conn.Status,
			"last_synced": conn.LastSynced,
		}

		// Patch the connection by conn_id.
		patchURL := c.restURL("snaptrade_connections") + "?conn_id=eq." + url.QueryEscape(conn.ConnID)
		resp, err := c.doRequest(ctx, http.MethodPatch, patchURL, payload)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("supabase patch snaptrade_connection %s failed: %s", conn.ConnID, string(body))
		}
	}
	return nil
}

// Returns all Snaptrade connections.
func (c *Client) ListSnaptradeConnections(ctx context.Context) ([]SnaptradeConnection, error) {
	url := c.restURL("snaptrade_connections") + "?order=last_synced.desc"

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase list snaptrade_connections failed: %s", string(body))
	}

	// Decodes the response body into a slice of Snaptrade connections.
	var conns []SnaptradeConnection
	if err := json.NewDecoder(resp.Body).Decode(&conns); err != nil {
		return nil, err
	}
	return conns, nil
}

// Deletes a Snaptrade connection by its connection ID.
func (c *Client) DeleteSnaptradeConnection(ctx context.Context, connID string) error {
	url := c.restURL("snaptrade_connections") + "?conn_id=eq." + connID

	resp, err := c.doRequest(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase delete snaptrade_connection failed: %s", string(body))
	}
	return nil
}

// Returns all name-based category rules.
func (c *Client) ListCategoryRules(ctx context.Context) ([]CategoryRule, error) {
	url := c.restURL("category_rules") + "?order=id.asc"
	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase list category_rules failed: %s", string(body))
	}

	// Decodes the response body into a slice of category rules.
	var categoryRules []CategoryRule
	if err := json.NewDecoder(resp.Body).Decode(&categoryRules); err != nil {
		return nil, err
	}
	return categoryRules, nil
}

// Updates transactions_cursor and new_transactions flag for a Plaid item.
func (c *Client) UpdatePlaidItemCursorAndPending(ctx context.Context, itemID, cursor string, pending bool) error {
	payload := map[string]interface{}{
		"transactions_cursor":      cursor,
		"new_transactions_pending": pending,
	}

	// Updates the Plaid item by item_id.
	url := c.restURL("plaid_items") + "?item_id=eq." + url.QueryEscape(itemID)
	resp, err := c.doRequest(ctx, http.MethodPatch, url, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase update plaid_item cursor/pending failed: %s", string(body))
	}
	return nil
}

// Sets new_transactions flag for a Plaid item (used by webhook).
func (c *Client) SetItemNewTransactionsPending(ctx context.Context, itemID string, pending bool) error {
	payload := map[string]interface{}{"new_transactions_pending": pending}
	url := c.restURL("plaid_items") + "?item_id=eq." + url.QueryEscape(itemID)
	resp, err := c.doRequest(ctx, http.MethodPatch, url, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase set new_transactions_pending failed: %s", string(body))
	}
	return nil
}

// Returns all Plaid items that received new transactions.
func (c *Client) ListPlaidItemsWithPendingTransactions(ctx context.Context) ([]PlaidItem, error) {
	url := c.restURL("plaid_items") + "?new_transactions_pending=eq.true"
	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase list plaid_items pending failed: %s", string(body))
	}
	var items []PlaidItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}
	return items, nil
}

// Returns all categories.
func (c *Client) ListCategories(ctx context.Context) ([]Category, error) {
	url := c.restURL("categories")
	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase list categories failed: %s", string(body))
	}
	var list []Category
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}
	return list, nil
}

// Upserts transactions by their Plaid_transaction_id.
func (c *Client) UpsertTransactions(ctx context.Context, txns []Transaction) error {
	if len(txns) == 0 {
		return nil
	}
	url := c.restURL("transactions") + "?on_conflict=plaid_transaction_id"
	resp, err := c.doRequest(ctx, http.MethodPost, url, txns)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase upsert transactions failed: %s", string(body))
	}
	return nil
}

// Deletes transactions by their Plaid transaction_id.
func (c *Client) DeleteTransactionsByPlaidIDs(ctx context.Context, plaidIDs []string) error {
	if len(plaidIDs) == 0 {
		return nil
	}
	// Constructs the query
	var b strings.Builder
	b.WriteString("in.(")
	for i, id := range plaidIDs {
		if i > 0 {
			b.WriteString(",")
		}
		escaped := strings.ReplaceAll(id, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		b.WriteString("\"" + escaped + "\"")
	}
	b.WriteString(")")

	// Deletes the transactions
	url := c.restURL("transactions") + "?plaid_transaction_id=" + url.QueryEscape(b.String())
	resp, err := c.doRequest(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase delete transactions failed: %s", string(body))
	}
	return nil
}

// Returns transactions for the given month, optionally filtered by category and search.
func (c *Client) ListTransactions(ctx context.Context, f ListTransactionsFilter) ([]Transaction, error) {
	// Build query: order by date desc
	reqURL := c.restURL("transactions") + "?order=date.desc"
	if f.Month != "" {
		start := f.Month + "-01"
		reqURL += "&date=gte." + start + "&date=lte." + endOfMonth(f.Month)
	}
	if f.CategoryID != nil {
		reqURL += "&category_id=eq." + fmt.Sprintf("%d", *f.CategoryID)
	}
	if f.Search != "" {
		pattern := "%" + f.Search + "%"
		reqURL += "&or=(name.ilike." + url.QueryEscape(pattern) + ",merchant_name.ilike." + url.QueryEscape(pattern) + ")"
	}

	// Gets the transactions
	resp, err := c.doRequest(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase list transactions failed: %s", string(body))
	}

	// Decodes the response body into a slice of transactions.
	var list []Transaction
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}
	return list, nil
}

// Returns the last day of the month.
func endOfMonth(month string) string {
	if len(month) != 7 {
		return month + "-31"
	}
	parts := strings.Split(month, "-")
	if len(parts) != 2 {
		return month + "-31"
	}
	var y, m int
	if _, err := fmt.Sscanf(parts[0], "%d", &y); err != nil {
		return month + "-31"
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &m); err != nil {
		return month + "-31"
	}
	// Adds one month and subtracts one day to get the last day of the month.
	t := time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.UTC)
	t = t.AddDate(0, 1, -1)
	return t.Format("2006-01-02")
}

// Returns the current budget.
func (c *Client) GetBudget(ctx context.Context) (*Budget, error) {
	url := c.restURL("budgets") + "?id=eq.1&limit=1"

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase get budget failed: %s", string(body))
	}

	// Decodes the response body into a Budget slice
	var budgets []Budget
	if err := json.NewDecoder(resp.Body).Decode(&budgets); err != nil {
		return nil, err
	}
	if len(budgets) == 0 {
		return nil, nil
	}
	return &budgets[0], nil
}

// Inserts or updates the current budget.
func (c *Client) UpsertBudget(ctx context.Context, budget *Budget) error {
	if budget == nil {
		return errors.New("budget is nil")
	}

	url := c.restURL("budgets") + "?on_conflict=id"
	resp, err := c.doRequest(ctx, http.MethodPost, url, budget)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase upsert budget failed: %s", string(body))
	}
	return nil
}

// Inserts or updates a daily snapshot.
func (c *Client) UpsertDailySnapshot(ctx context.Context, snapshot *DailySnapshot) error {
	if snapshot == nil {
		return errors.New("snapshot is nil")
	}

	url := c.restURL("daily_snapshots") + "?on_conflict=date"
	resp, err := c.doRequest(ctx, http.MethodPost, url, snapshot)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase upsert daily_snapshots failed: %s", string(body))
	}
	return nil
}

// Lists daily snapshots within a date range (should be for last 30 days).
func (c *Client) ListDailySnapshots(ctx context.Context, startDate, endDate time.Time) ([]DailySnapshot, error) {
	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")
	url := c.restURL("daily_snapshots") + fmt.Sprintf("?date=gte.%s&date=lte.%s&order=date.asc", startStr, endStr)

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase list daily_snapshots failed: %s", string(body))
	}

	// Decodes the response body into a slice of daily snapshots.
	var snapshots []DailySnapshot
	err = json.NewDecoder(resp.Body).Decode(&snapshots)
	if err != nil {
		return nil, err
	}
	return snapshots, nil
}

// Inserts or updates a daily holding.
func (c *Client) UpsertDailyHolding(ctx context.Context, holding *DailyHolding) error {
	if holding == nil {
		return errors.New("holding is nil")
	}

	url := c.restURL("daily_holdings") + "?on_conflict=date,account_id,symbol"
	resp, err := c.doRequest(ctx, http.MethodPost, url, holding)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase upsert daily_holdings failed: %s", string(body))
	}
	return nil
}

// Lists daily holdings within a date range (should be for last 30 days).
func (c *Client) ListDailyHoldings(ctx context.Context, startDate, endDate time.Time) ([]DailyHolding, error) {
	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")
	url := c.restURL("daily_holdings") + fmt.Sprintf("?date=gte.%s&date=lte.%s&order=date.asc", startStr, endStr)

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase list daily_holdings failed: %s", string(body))
	}

	// Decodes the response body into a slice of daily holdings.
	var holdings []DailyHolding
	if err := json.NewDecoder(resp.Body).Decode(&holdings); err != nil {
		return nil, err
	}
	return holdings, nil
}

// Lists daily holdings for a specific account (should be for last 30 days).
func (c *Client) ListDailyHoldingsByAccount(ctx context.Context, accountID string, startDate, endDate time.Time) ([]DailyHolding, error) {
	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")
	url := c.restURL("daily_holdings") + fmt.Sprintf("?account_id=eq.%s&date=gte.%s&date=lte.%s&order=date.asc", accountID, startStr, endStr)

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase list daily_holdings by account failed: %s", string(body))
	}

	// Decodes the response body into a slice of daily holdings.
	var holdings []DailyHolding
	err = json.NewDecoder(resp.Body).Decode(&holdings)
	if err != nil {
		return nil, err
	}
	return holdings, nil
}

// Lists daily holdings for a specific symbol (should be for last 30 days).
func (c *Client) ListDailyHoldingsBySymbol(ctx context.Context, symbol string, startDate, endDate time.Time) ([]DailyHolding, error) {
	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")
	url := c.restURL("daily_holdings") + fmt.Sprintf("?symbol=eq.%s&date=gte.%s&date=lte.%s&order=date.asc", symbol, startStr, endStr)

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase list daily_holdings by symbol failed: %s", string(body))
	}

	var holdings []DailyHolding
	if err := json.NewDecoder(resp.Body).Decode(&holdings); err != nil {
		return nil, err
	}
	return holdings, nil
}

// Inserts or updates a monthly snapshot.
func (c *Client) UpsertMonthlySnapshot(ctx context.Context, snapshot *MonthlySnapshot) error {
	if snapshot == nil {
		return errors.New("snapshot is nil")
	}

	url := c.restURL("monthly_snapshots") + "?on_conflict=month,account_id"
	resp, err := c.doRequest(ctx, http.MethodPost, url, snapshot)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase upsert monthly_snapshots failed: %s", string(body))
	}
	return nil
}

// Lists monthly snapshots within a month range.
func (c *Client) ListMonthlySnapshots(ctx context.Context, startMonth, endMonth time.Time) ([]MonthlySnapshot, error) {
	startStr := startMonth.Format("2006-01-02")
	endStr := endMonth.Format("2006-01-02")
	url := c.restURL("monthly_snapshots") + fmt.Sprintf("?month=gte.%s&month=lte.%s&order=month.asc", startStr, endStr)

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase list monthly_snapshots failed: %s", string(body))
	}

	var snapshots []MonthlySnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snapshots); err != nil {
		return nil, err
	}
	return snapshots, nil
}

// Lists monthly snapshots for a single account within a month range.
func (c *Client) ListMonthlySnapshotsByAccount(ctx context.Context, startMonth, endMonth time.Time, accountID string) ([]MonthlySnapshot, error) {
	startStr := startMonth.Format("2006-01-02")
	endStr := endMonth.Format("2006-01-02")
	url := c.restURL("monthly_snapshots") + fmt.Sprintf("?account_id=eq.%s&month=gte.%s&month=lte.%s&order=month.asc",
		url.QueryEscape(accountID), startStr, endStr)

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase list monthly_snapshots by account failed: %s", string(body))
	}

	var snapshots []MonthlySnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snapshots); err != nil {
		return nil, err
	}
	return snapshots, nil
}

// Represents a row in the plaid_items table.
type PlaidItem struct {
	ID                     int64     `json:"id,omitempty"`
	ItemID                 string    `json:"item_id"`
	AccessToken            string    `json:"access_token"`
	InstitutionID          *string   `json:"institution_id,omitempty"`
	InstitutionName        *string   `json:"institution_name,omitempty"`
	Status                 string    `json:"status"`
	LastUpdated            time.Time `json:"last_updated"`
	CreatedAt              time.Time `json:"created_at,omitempty"`
	TransactionsCursor     *string   `json:"transactions_cursor,omitempty"`
	NewTransactionsPending bool      `json:"new_transactions_pending"`
}

// The JSON-safe representation for API responses (hides access_token).
type PlaidItemJSON struct {
	ItemID          string    `json:"itemId"`
	InstitutionName string    `json:"institutionName,omitempty"`
	Status          string    `json:"status"`
	LastUpdated     time.Time `json:"lastUpdated"`
}

// Represents a row in the plaid_accounts table.
type PlaidAccount struct {
	ID             int64   `json:"id,omitempty"`
	PlaidItemID    string  `json:"plaid_item_id"`
	AccountID      string  `json:"account_id"`
	Name           string  `json:"name"`
	Mask           *string `json:"mask,omitempty"`
	Type           string  `json:"type"`
	Subtype        *string `json:"subtype,omitempty"`
	CurrentBalance float64 `json:"current_balance"`
}

// Represents a row in the snaptrade_user table.
type SnaptradeUser struct {
	ID         int64  `json:"id,omitempty"`
	UserID     string `json:"user_id"`
	UserSecret string `json:"user_secret"`
}

// Represents a row in the snaptrade_connections table.
type SnaptradeConnection struct {
	ID         int64      `json:"id,omitempty"`
	ConnID     string     `json:"conn_id"`
	Brokerage  string     `json:"brokerage"`
	Status     string     `json:"status"`
	LastSynced *time.Time `json:"last_synced,omitempty"`
}

// The JSON-safe representation for API responses (hides conn_id).
type SnaptradeConnectionJSON struct {
	ID         string     `json:"id"`
	Brokerage  string     `json:"brokerage"`
	Status     string     `json:"status"`
	LastSynced *time.Time `json:"lastSynced,omitempty"`
}

// Represents a row in the categories table.
type Category struct {
	ID        int64   `json:"id,omitempty"`
	Name      string  `json:"name"`
	PlaidName *string `json:"plaid_name,omitempty"`
	Expense   bool    `json:"expense"`
}

// Category rule maps transaction name/merchant substring to a category.
type CategoryRule struct {
	ID          int64  `json:"id,omitempty"`
	MatchString string `json:"match_string"`
	CategoryID  int64  `json:"category_id"`
}

// Represents a row in the transactions table.
type Transaction struct {
	ID                 int64     `json:"id,omitempty"`
	PlaidAccountID     string    `json:"plaid_account_id"`
	PlaidTransactionID string    `json:"plaid_transaction_id"`
	Date               time.Time `json:"date"`
	AmountCents        int64     `json:"amount_cents"`
	Name               string    `json:"name"`
	MerchantName       *string   `json:"merchant_name,omitempty"`
	CategoryID         *int64    `json:"category_id,omitempty"`
	Pending            bool      `json:"pending"`
	CreatedAt          time.Time `json:"created_at,omitempty"`
	UpdatedAt          time.Time `json:"updated_at,omitempty"`
}

// Holds optional filters for listing transactions.
type ListTransactionsFilter struct {
	Month      string
	CategoryID *int64
	Search     string
}

// Represents a row in the budgets table.
type Budget struct {
	ID          int64            `json:"id,omitempty"`
	Allocations map[string]int64 `json:"allocations"`
	UpdatedAt   time.Time        `json:"updated_at,omitempty"`
}

// Represents a row in the daily_snapshots table
type DailySnapshot struct {
	ID                  int64     `json:"id,omitempty"`
	Date                time.Time `json:"date"`
	PortfolioValueCents int64     `json:"portfolio_value_cents"`
	CreatedAt           time.Time `json:"created_at,omitempty"`
}

// Represents a row in the daily_holdings table.
type DailyHolding struct {
	ID         int64     `json:"id,omitempty"`
	Date       time.Time `json:"date"`
	AccountID  string    `json:"account_id"`
	Symbol     string    `json:"symbol"`
	Quantity   float64   `json:"quantity"`
	ValueCents int64     `json:"value_cents"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
}

// Represents a row in the monthly_snapshots table.
type MonthlySnapshot struct {
	ID                  int64     `json:"id,omitempty"`
	Month               time.Time `json:"month"`
	AccountID           string    `json:"account_id"`
	PortfolioValueCents int64     `json:"portfolio_value_cents"`
	CreatedAt           time.Time `json:"created_at,omitempty"`
}
