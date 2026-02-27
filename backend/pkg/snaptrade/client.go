package snaptrade

import (
	"context"
	"errors"
	"os"
	"time"

	sdk "github.com/passiv/snaptrade-sdks/sdks/go"
)

// SnapTrade Go SDK client.
type Client struct {
	api *sdk.APIClient
}

// SnapTrade brokerage connection.
type Connection struct {
	ID        string              `json:"id"`
	CreatedAt string              `json:"created_date"`
	Brokerage ConnectionBrokerage `json:"brokerage"`
}

// Connection brokerage info.
type ConnectionBrokerage struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Snaptrade account with balances.
type Account struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Number          string  `json:"number"`
	Institution     string  `json:"institution_name"`
	BalanceAmount   float64 `json:"balance_amount"`
	BalanceCurrency string  `json:"balance_currency"`
}

// Account Holdings Position.
type Position struct {
	Symbol     string  `json:"symbol"`
	Quantity   float64 `json:"quantity"`
	ValueCents int64   `json:"value_cents"`
}

// Constructs a Snaptrade client from environment variables.
func NewClientFromEnv() (*Client, error) {
	clientID := os.Getenv("SNAPTRADE_CLIENT_ID")
	clientSecret := os.Getenv("SNAPTRADE_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return nil, errors.New("SNAPTRADE_CLIENT_ID and SNAPTRADE_CLIENT_SECRET must be set")
	}

	cfg := sdk.NewConfiguration()
	cfg.SetPartnerClientId(clientID)
	cfg.SetConsumerKey(clientSecret)

	apiClient := sdk.NewAPIClient(cfg)
	return &Client{api: apiClient}, nil
}

// Registers a new SnapTrade user and returns their userSecret.
func (c *Client) RegisterUser(ctx context.Context, userID string) (string, error) {
	body := sdk.NewSnapTradeRegisterUserRequestBody(userID)
	req := c.api.AuthenticationApi.RegisterSnapTradeUser(*body)

	resp, _, err := req.Execute()
	if err != nil {
		return "", err
	}

	secret := resp.GetUserSecret()
	if secret == "" {
		return "", errors.New("snaptrade: empty userSecret in registerUser response")
	}
	return secret, nil
}

// Generates a URL for the SnapTrade Connection Portal.
func (c *Client) GenerateConnectionPortalURL(ctx context.Context, userID, userSecret string) (string, error) {
	body := sdk.NewSnapTradeLoginUserRequestBody()
	body.SetConnectionType("READ")

	// Login the user and get the redirect URI.
	req := c.api.AuthenticationApi.LoginSnapTradeUser(userID, userSecret)
	resp, _, err := (&req).
		SnapTradeLoginUserRequestBody(*body).
		Execute()

	if err != nil {
		return "", err
	}

	// Get the redirect URI from the response.
	login := resp.GetActualInstance().(*sdk.LoginRedirectURI)
	redirectURI := login.GetRedirectURI()
	if redirectURI == "" {
		return "", errors.New("snaptrade: empty redirectURI in login response")
	}
	return redirectURI, nil
}

// Lists all brokerage connections for the given user.
func (c *Client) ListConnections(userID, userSecret string) ([]Connection, error) {
	// List the authenticated connections.
	req := c.api.ConnectionsApi.ListBrokerageAuthorizations(userID, userSecret)
	autenticatedConnections, _, err := req.Execute()
	if err != nil {
		return nil, err
	}

	// Convert the authenticated connections to Brokerage Struct
	var connections []Connection
	for _, a := range autenticatedConnections {
		conn := Connection{
			ID:        a.GetId(),
			CreatedAt: "",
			Brokerage: ConnectionBrokerage{},
		}

		if a.HasCreatedDate() {
			conn.CreatedAt = a.GetCreatedDate().Format(time.RFC3339)
		}

		b, ok := a.GetBrokerageOk()
		if ok {
			conn.Brokerage.Name = b.GetName()
			conn.Brokerage.Slug = b.GetSlug()
		}
		connections = append(connections, conn)
	}
	return connections, nil
}

// Deletes a Snaptrade connection.
func (c *Client) RemoveConnection(userID, userSecret, authorizationID string) error {
	req := c.api.ConnectionsApi.RemoveBrokerageAuthorization(authorizationID, userID, userSecret)
	_, err := req.Execute()
	if err != nil {
		return err
	}
	return nil
}

// Lists all accounts for the given user with balances.
func (c *Client) ListAccounts(userID, userSecret string) ([]Account, error) {
	req := c.api.AccountInformationApi.ListUserAccounts(userID, userSecret)
	userAccounts, _, err := req.Execute()
	if err != nil {
		return nil, err
	}

	// Convert the user accounts to Account Struct.
	var accounts []Account
	for _, a := range userAccounts {
		account := Account{
			ID:              a.GetId(),
			Name:            a.GetName(),
			Number:          a.GetNumber(),
			Institution:     a.GetInstitutionName(),
			BalanceAmount:   0,
			BalanceCurrency: "USD",
		}

		// Checks if the account has a valid balance.
		balance := a.GetBalance()
		if total, ok := balance.GetTotalOk(); ok {
			account.BalanceAmount = float64(total.GetAmount())
			account.BalanceCurrency = total.GetCurrency()
		}

		accounts = append(accounts, account)
	}
	return accounts, nil
}

// Lists positions (holdings) for a specific account.
func (c *Client) ListAccountPositions(userID, userSecret, accountID string) ([]Position, error) {
	req := c.api.AccountInformationApi.GetUserAccountPositions(accountID, userID, userSecret)
	positions, _, err := req.Execute()
	if err != nil {
		return nil, err
	}

	// Convert the positions to Position Struct.
	var result []Position
	for _, pos := range positions {
		// Get the symbol from the position.
		symbol := ""
		if sym, ok := pos.GetSymbolOk(); ok && sym != nil {
			universal := sym.GetSymbol()
			symbol = universal.GetSymbol()
		}

		// Get the number of shares from the position.
		quantity := 0.0
		if pos.HasUnits() {
			quantity = float64(pos.GetUnits())
		}

		// Get the price per share from the position.
		pricePerShare := float32(0)
		if pos.HasPrice() {
			pricePerShare = pos.GetPrice()
		}

		// Get the value in cents from the position.
		valueCents := int64(0)
		if quantity > 0 && pricePerShare > 0 {
			valueCents = int64(float64(pricePerShare) * quantity * 100)
		}

		// Add the position to the result.
		result = append(result, Position{
			Symbol:     symbol,
			Quantity:   quantity,
			ValueCents: valueCents,
		})
	}

	return result, nil
}
