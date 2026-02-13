package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/matthewtzong/portfolio-tracker/backend/internal/database"
	"github.com/matthewtzong/portfolio-tracker/backend/internal/plaid"
	"github.com/matthewtzong/portfolio-tracker/backend/internal/serverauth"
	"github.com/matthewtzong/portfolio-tracker/backend/internal/snaptrade"
)

// Client dependencies for the link management routes.
type apiDependencies struct {
	db              *database.Client
	plaidClient     *plaid.Client
	snaptradeClient *snaptrade.Client
}

// Exchange Plaid public token request format.
type exchangePlaidPublicTokenRequest struct {
	PublicToken     string `json:"publicToken"`
	InstitutionName string `json:"institutionName,omitempty"`
	InstitutionID   string `json:"institutionId,omitempty"`
}

// Remove Plaid item request format.
type removePlaidItemRequest struct {
	ItemID string `json:"itemId"`
}

// Remove Snaptrade connection request format.
type removeSnaptradeConnectionRequest struct {
	ConnectionID string `json:"connectionId"`
}

// Reconnect Plaid item request format.
type reconnectPlaidItemRequest struct {
	ItemID string `json:"itemId"`
}

// Link token response format.
type linkTokenResponse struct {
	LinkToken string `json:"linkToken"`
}

// Exchange Plaid public token response format.
type exchangeTokenResponse struct {
	ItemID string `json:"itemId"`
}

// List links response format.
type linksResponse struct {
	PlaidItems           []database.PlaidItemJSON           `json:"plaidItems"`
	SnaptradeConnections []database.SnaptradeConnectionJSON `json:"snaptradeConnections"`
}

// Connect URL response format.
type connectURLResponse struct {
	RedirectURI string `json:"redirectUri"`
}

// Error response format.
type errorResponse struct {
	Error string `json:"error"`
}

// Registers the link management routes.
func registerLinkManagementRoutes(mux *http.ServeMux, deps apiDependencies) {
	// GET /api/plaid/link-token creates a Plaid link token.
	mux.Handle("/api/plaid/link-token", serverauth.JWTAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		handleCreatePlaidLinkToken(w, r, deps)
	})))

	// POST /api/plaid/reconnect-link-token creates a Plaid link token in update mode for reconnecting.
	mux.Handle("/api/plaid/reconnect-link-token", serverauth.JWTAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		handleCreatePlaidReconnectLinkToken(w, r, deps)
	})))

	// POST /api/plaid/exchange-token exchanges a Plaid public token for an access token and item ID.
	mux.Handle("/api/plaid/exchange-token", serverauth.JWTAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		handleExchangePlaidPublicToken(w, r, deps)
	})))

	// GET /api/links lists all Plaid items and Snaptrade connections.
	mux.Handle("/api/links", serverauth.JWTAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		handleListLinks(w, r, deps)
	})))

	// POST /api/plaid/remove-item removes a Plaid item.
	mux.Handle("/api/plaid/remove-item", serverauth.JWTAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		handleRemovePlaidItem(w, r, deps)
	})))

	// POST /api/snaptrade/connect-url generates a URL for the Snaptrade Connection Portal.
	mux.Handle("/api/snaptrade/connect-url", serverauth.JWTAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		handleSnaptradeConnectURL(w, r, deps)
	})))

	// POST /api/snaptrade/sync-connections syncs Snaptrade connections from the API into the database.
	mux.Handle("/api/snaptrade/sync-connections", serverauth.JWTAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		handleSnaptradeSyncConnections(w, r, deps)
	})))

	// POST /api/snaptrade/remove-connection removes a Snaptrade connection.
	mux.Handle("/api/snaptrade/remove-connection", serverauth.JWTAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		handleRemoveSnaptradeConnection(w, r, deps)
	})))
}

// Creates a Plaid link token.
func handleCreatePlaidLinkToken(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	w.Header().Set("Content-Type", "application/json")

	if deps.plaidClient == nil {
		writeJSONError(w, http.StatusInternalServerError, "Plaid is not configured (missing environment variables)")
		return
	}

	// Get the authenticated user ID.
	userID, ok := serverauth.UserIDFromContext(r.Context())
	if !ok || userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "missing authenticated user")
		return
	}

	linkToken, err := deps.plaidClient.CreateLinkToken(r.Context(), userID)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to create Plaid link token: "+err.Error())
		return
	}

	resp := linkTokenResponse{LinkToken: linkToken}
	_ = json.NewEncoder(w).Encode(resp)
}

// Exchanges a Plaid public token for an access token and item ID.
func handleExchangePlaidPublicToken(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	w.Header().Set("Content-Type", "application/json")

	if deps.plaidClient == nil {
		writeJSONError(w, http.StatusInternalServerError, "Plaid is not configured (missing environment variables)")
		return
	}

	if deps.db == nil {
		writeJSONError(w, http.StatusInternalServerError, "Database is not configured")
		return
	}

	// Parse the request body.
	var req exchangePlaidPublicTokenRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.PublicToken == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	accessToken, itemID, err := deps.plaidClient.ExchangePublicToken(r.Context(), req.PublicToken)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to exchange public token: "+err.Error())
		return
	}

	// Fetch the Plaid accounts.
	accounts, err := deps.plaidClient.GetAccounts(r.Context(), accessToken)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to fetch Plaid accounts: "+err.Error())
		return
	}

	now := time.Now().UTC()

	// Check if an item with the same institution_id already exists (for reconnection).
	var existingItem *database.PlaidItem
	if req.InstitutionID != "" {
		existingItem, _ = deps.db.GetPlaidItemByInstitutionID(r.Context(), req.InstitutionID)
	}

	// Build the Plaid item for the database.
	item := &database.PlaidItem{
		ItemID:      itemID,
		AccessToken: accessToken,
		Status:      "OK",
		LastUpdated: now,
	}
	if req.InstitutionName != "" {
		item.InstitutionName = &req.InstitutionName
	}
	if req.InstitutionID != "" {
		item.InstitutionID = &req.InstitutionID
	}

	// If item existed, delete the old item and update accounts to point to the new item_id.
	if existingItem != nil && existingItem.ItemID != itemID {
		_ = deps.db.DeletePlaidItem(r.Context(), existingItem.ItemID)
	}

	// Save item to DB (upsert by item_id)
	err = deps.db.UpsertPlaidItem(r.Context(), item)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to save Plaid item: "+err.Error())
		return
	}

	// Build the Plaid accounts for the database.
	var dbAccounts []database.PlaidAccount
	for _, a := range accounts {
		account := database.PlaidAccount{
			PlaidItemID:    itemID,
			AccountID:      a.AccountID,
			Name:           a.Name,
			Type:           a.Type,
			CurrentBalance: a.Balances.Current,
		}
		if a.Mask != "" {
			account.Mask = &a.Mask
		}
		if a.Subtype != "" {
			account.Subtype = &a.Subtype
		}
		dbAccounts = append(dbAccounts, account)
	}

	// Save accounts to DB, reconnecting will update existing accounts (upsert by account_id)
	if err := deps.db.UpsertPlaidAccounts(r.Context(), dbAccounts); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to save Plaid accounts: "+err.Error())
		return
	}

	// Return the item ID.
	resp := exchangeTokenResponse{ItemID: itemID}
	_ = json.NewEncoder(w).Encode(resp)
}

// Lists all Plaid items and Snaptrade connections from the database.
func handleListLinks(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	w.Header().Set("Content-Type", "application/json")

	if deps.db == nil {
		writeJSONError(w, http.StatusInternalServerError, "Database is not configured")
		return
	}

	// Fetch Plaid items
	plaidItems, err := deps.db.ListPlaidItems(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list Plaid items: "+err.Error())
		return
	}

	// Fetch Snaptrade connections
	snapConns, err := deps.db.ListSnaptradeConnections(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list Snaptrade connections: "+err.Error())
		return
	}

	// Convert to JSON-safe representations.
	plaidJSON := []database.PlaidItemJSON{}
	for _, item := range plaidItems {
		plaidJSON = append(plaidJSON, item.ToJSON())
	}

	snapJSON := []database.SnaptradeConnectionJSON{}
	for _, conn := range snapConns {
		snapJSON = append(snapJSON, conn.ToJSON())
	}

	resp := linksResponse{
		PlaidItems:           plaidJSON,
		SnaptradeConnections: snapJSON,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// Removes a Plaid item.
func handleRemovePlaidItem(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	w.Header().Set("Content-Type", "application/json")

	if deps.db == nil {
		writeJSONError(w, http.StatusInternalServerError, "Database is not configured")
		return
	}

	// Parse the request body.
	var req removePlaidItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ItemID == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Fetch the item from DB to get the access_token.
	item, err := deps.db.GetPlaidItemByItemID(r.Context(), req.ItemID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to get Plaid item: "+err.Error())
		return
	}
	if item == nil {
		writeJSONError(w, http.StatusNotFound, "Plaid item not found")
		return
	}

	// Remove the item from Plaid.
	if deps.plaidClient != nil {
		if err := deps.plaidClient.RemoveItem(r.Context(), item.AccessToken); err != nil {
			writeJSONError(w, http.StatusBadGateway, "failed to remove Plaid item: "+err.Error())
			return
		}
	}

	// Delete the item from the database.
	if err := deps.db.DeletePlaidAccountsByItemID(r.Context(), req.ItemID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to delete Plaid accounts: "+err.Error())
		return
	}

	// Delete the item from the database.
	if err := deps.db.DeletePlaidItem(r.Context(), req.ItemID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to delete Plaid item: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Generates a URL for the Snaptrade Connection Portal.
func handleSnaptradeConnectURL(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	w.Header().Set("Content-Type", "application/json")

	if deps.snaptradeClient == nil {
		writeJSONError(w, http.StatusInternalServerError, "Snaptrade is not configured (missing environment variables)")
		return
	}

	if deps.db == nil {
		writeJSONError(w, http.StatusInternalServerError, "Database is not configured")
		return
	}

	// Get the authenticated user ID.
	userID, ok := serverauth.UserIDFromContext(r.Context())
	if !ok || userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "missing authenticated user")
		return
	}

	// Get existing Snaptrade user from DB.
	user, err := deps.db.GetSnaptradeUser(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to fetch Snaptrade user: "+err.Error())
		return
	}

	// If no user exists, create and register with Snaptrade and save to DB.
	if user == nil {
		userSecret, err := deps.snaptradeClient.RegisterUser(r.Context(), userID)
		if err != nil {
			writeJSONError(w, http.StatusBadGateway, "failed to register Snaptrade user: "+err.Error())
			return
		}

		user, err = deps.db.CreateSnaptradeUser(r.Context(), userID, userSecret)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to save Snaptrade user: "+err.Error())
			return
		}
	}

	redirectURI, err := deps.snaptradeClient.GenerateConnectionPortalURL(r.Context(), user.UserID, user.UserSecret)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to create Snaptrade Connect URL: "+err.Error())
		return
	}

	resp := connectURLResponse{RedirectURI: redirectURI}
	_ = json.NewEncoder(w).Encode(resp)
}

// Syncs Snaptrade connections from the Snaptrade API into the database and returns the updated list.
func handleSnaptradeSyncConnections(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	w.Header().Set("Content-Type", "application/json")

	if deps.snaptradeClient == nil {
		writeJSONError(w, http.StatusInternalServerError, "Snaptrade is not configured (missing environment variables)")
		return
	}

	if deps.db == nil {
		writeJSONError(w, http.StatusInternalServerError, "Database is not configured")
		return
	}

	// Get existing Snaptrade user from DB.
	user, err := deps.db.GetSnaptradeUser(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to fetch Snaptrade user: "+err.Error())
		return
	}

	// If no user exists, return an empty response.
	if user == nil {
		resp := linksResponse{
			PlaidItems:           nil,
			SnaptradeConnections: []database.SnaptradeConnectionJSON{},
		}
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// Fetch connections from Snaptrade API and sync to database.
	conns, err := deps.snaptradeClient.ListConnections(user.UserID, user.UserSecret)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to list Snaptrade connections: "+err.Error())
		return
	}

	// Build DB records and upsert
	now := time.Now().UTC()
	var dbConns []database.SnaptradeConnection
	for _, c := range conns {
		dbConns = append(dbConns, database.SnaptradeConnection{
			ConnID:     c.ID,
			Brokerage:  c.Brokerage.Name,
			Status:     "OK",
			LastSynced: &now,
		})
	}

	// Save the connections to the database.
	if err := deps.db.UpsertSnaptradeConnections(r.Context(), dbConns); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to save Snaptrade connections: "+err.Error())
		return
	}

	// Convert to JSON-safe representations.
	var snapJSON []database.SnaptradeConnectionJSON
	for _, c := range dbConns {
		snapJSON = append(snapJSON, c.ToJSON())
	}

	// Return the connections.
	resp := linksResponse{
		PlaidItems:           nil,
		SnaptradeConnections: snapJSON,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// Removes a Snaptrade connection.
func handleRemoveSnaptradeConnection(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	w.Header().Set("Content-Type", "application/json")

	if deps.db == nil {
		writeJSONError(w, http.StatusInternalServerError, "Database is not configured")
		return
	}

	// Parse the request body.
	var req removeSnaptradeConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ConnectionID == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Delete the connection from Snaptrade first
	if deps.snaptradeClient != nil {
		user, err := deps.db.GetSnaptradeUser(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to get Snaptrade user: "+err.Error())
			return
		}
		if user != nil {
			if err := deps.snaptradeClient.RemoveConnection(user.UserID, user.UserSecret, req.ConnectionID); err != nil {
				writeJSONError(w, http.StatusBadGateway, "failed to remove Snaptrade connection: "+err.Error())
				return
			}
		}
	}

	// Delete the connection from our database.
	if err := deps.db.DeleteSnaptradeConnection(r.Context(), req.ConnectionID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to delete Snaptrade connection: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Writes a JSON error response to the response writer.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := errorResponse{Error: message}
	_ = json.NewEncoder(w).Encode(resp)
}

// Creates a Plaid link token in update mode for reconnecting an existing item.
func handleCreatePlaidReconnectLinkToken(w http.ResponseWriter, r *http.Request, deps apiDependencies) {
	w.Header().Set("Content-Type", "application/json")

	if deps.plaidClient == nil {
		writeJSONError(w, http.StatusInternalServerError, "Plaid is not configured (missing environment variables)")
		return
	}

	if deps.db == nil {
		writeJSONError(w, http.StatusInternalServerError, "Database is not configured")
		return
	}

	// Get the authenticated user ID.
	userID, ok := serverauth.UserIDFromContext(r.Context())
	if !ok || userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "missing authenticated user")
		return
	}

	// Parse the request body.
	var req reconnectPlaidItemRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.ItemID == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get the existing item to retrieve its access token.
	existingItem, err := deps.db.GetPlaidItemByItemID(r.Context(), req.ItemID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to fetch Plaid item: "+err.Error())
		return
	}
	if existingItem == nil {
		writeJSONError(w, http.StatusNotFound, "Plaid item not found")
		return
	}

	// Create a link token in update mode using the existing access token.
	linkToken, err := deps.plaidClient.CreateLinkTokenWithAccessToken(r.Context(), userID, existingItem.AccessToken)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to create reconnect link token: "+err.Error())
		return
	}

	resp := linkTokenResponse{LinkToken: linkToken}
	_ = json.NewEncoder(w).Encode(resp)
}

// Ensures only the allowed HTTP method is used.
func methodNotAllowed(w http.ResponseWriter, allowed string) {
	w.Header().Set("Allow", allowed)
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}
