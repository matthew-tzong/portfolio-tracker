package plaid

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Plaid API Client
type Client struct {
	httpClient *http.Client
	baseURL    string
	clientID   string
	secret     string
}

// Constructs a Plaid client from environment variables.
func NewClientFromEnv() (*Client, error) {
	clientID := os.Getenv("PLAID_CLIENT_ID")
	secret := os.Getenv("PLAID_SECRET")
	env := os.Getenv("PLAID_ENV")
	// Sets the environment to sandbox if not set -> will eventually only be development
	if env == "" {
		env = "sandbox"
	}
	var baseURL string
	switch env {
	case "sandbox":
		baseURL = "https://sandbox.plaid.com"
	case "development":
		baseURL = "https://development.plaid.com"
	case "production":
		baseURL = "https://production.plaid.com"
	default:
		return nil, fmt.Errorf("unsupported PLAID_ENV: %s", env)
	}

	if clientID == "" || secret == "" {
		return nil, errors.New("PLAID_CLIENT_ID and PLAID_SECRET must be set")
	}

	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    baseURL,
		clientID:   clientID,
		secret:     secret,
	}, nil
}

// Creates a Plaid Link token for the given user.
func (c *Client) CreateLinkToken(ctx context.Context, userID string) (string, error) {
	// Constructs the request body for the Plaid Link token create request.
	reqBody := linkTokenCreateRequest{
		ClientID:   c.clientID,
		Secret:     c.secret,
		ClientName: "Portfolio Tracker",
		User: linkTokenUser{
			ClientUserID: userID,
		},
		Products:     []string{"transactions"},
		CountryCodes: []string{"US"},
		Language:     "en",
	}

	var resp linkTokenCreateResponse
	err := c.postJSON(ctx, "/link/token/create", reqBody, &resp)
	if err != nil {
		return "", err
	}
	if resp.LinkToken == "" {
		return "", errors.New("plaid: empty link_token in response")
	}
	return resp.LinkToken, nil
}

// Exchanges a public token for an access token and item ID.
func (c *Client) ExchangePublicToken(ctx context.Context, publicToken string) (accessToken, itemID string, err error) {
	// Constructs the request body for the Plaid public token exchange request.
	reqBody := itemPublicTokenExchangeRequest{
		ClientID:    c.clientID,
		Secret:      c.secret,
		PublicToken: publicToken,
	}

	var resp itemPublicTokenExchangeResponse
	err = c.postJSON(ctx, "/item/public_token/exchange", reqBody, &resp)
	if err != nil {
		return "", "", err
	}
	if resp.AccessToken == "" || resp.ItemID == "" {
		return "", "", errors.New("plaid: missing access_token or item_id in exchange response")
	}
	return resp.AccessToken, resp.ItemID, nil
}

// Returns accounts for a given access token.
func (c *Client) GetAccounts(ctx context.Context, accessToken string) ([]Account, error) {
	// Constructs the request body for the Plaid accounts get request.
	reqBody := accountsGetRequest{
		ClientID:    c.clientID,
		Secret:      c.secret,
		AccessToken: accessToken,
	}
	var resp accountsGetResponse
	if err := c.postJSON(ctx, "/accounts/get", reqBody, &resp); err != nil {
		return nil, err
	}
	return resp.Accounts, nil
}

// Posts a JSON body and decodes a JSON response.
func (c *Client) postJSON(ctx context.Context, path string, input, output interface{}) error {
	// Marshals the request body into a JSON string.
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}

	// Creates a new request with the context, method, URL, and body reader.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Checks the status code of the response.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var plaidErr plaidErrorResponse
		_ = json.NewDecoder(resp.Body).Decode(&plaidErr)
		if plaidErr.ErrorMessage != "" {
			return fmt.Errorf("plaid API error: %s (%s)", plaidErr.ErrorMessage, plaidErr.ErrorCode)
		}
		return fmt.Errorf("plaid API error: status %d", resp.StatusCode)
	}

	// Decodes the response body into the output.
	if output != nil {
		return json.NewDecoder(resp.Body).Decode(output)
	}
	return nil
}

// Request body for creating a Plaid Link token.
type linkTokenCreateRequest struct {
	ClientID     string        `json:"client_id"`
	Secret       string        `json:"secret"`
	ClientName   string        `json:"client_name"`
	User         linkTokenUser `json:"user"`
	Products     []string      `json:"products"`
	CountryCodes []string      `json:"country_codes"`
	Language     string        `json:"language"`
	Webhook      string        `json:"webhook,omitempty"`
}

// User object for the link token create request.
type linkTokenUser struct {
	ClientUserID string `json:"client_user_id"`
}

// Response body for creating a Plaid Link token.
type linkTokenCreateResponse struct {
	LinkToken string `json:"link_token"`
}

// Request body for exchanging a public token for an access token and item ID.
type itemPublicTokenExchangeRequest struct {
	ClientID    string `json:"client_id"`
	Secret      string `json:"secret"`
	PublicToken string `json:"public_token"`
}

// Response body for exchanging a public token for an access token and item ID.
type itemPublicTokenExchangeResponse struct {
	AccessToken string `json:"access_token"`
	ItemID      string `json:"item_id"`
}

// Request body for fetching accounts for a given access token.
type accountsGetRequest struct {
	ClientID    string `json:"client_id"`
	Secret      string `json:"secret"`
	AccessToken string `json:"access_token"`
}

// Response body for fetching accounts for a given access token.
type accountsGetResponse struct {
	Accounts []Account `json:"accounts"`
}

// Represents a Plaid account used for link management and balances.
type Account struct {
	AccountID string          `json:"account_id"`
	Name      string          `json:"name"`
	Mask      string          `json:"mask"`
	Type      string          `json:"type"`
	Subtype   string          `json:"subtype"`
	Balances  accountBalances `json:"balances"`
}

// Represents the balances of a Plaid account.
type accountBalances struct {
	Current float64 `json:"current"`
}

// Represents an error response from the Plaid API.
type plaidErrorResponse struct {
	ErrorType        string `json:"error_type"`
	ErrorCode        string `json:"error_code"`
	ErrorMessage     string `json:"error_message"`
	DisplayMessage   string `json:"display_message"`
	RequestID        string `json:"request_id"`
	SuggestedAction  string `json:"suggested_action"`
	DocumentationURL string `json:"documentation_url"`
}
