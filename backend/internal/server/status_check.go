package server

import (
	"context"
	"errors"
	"time"

	"github.com/matthewtzong/portfolio-tracker/backend/internal/database"
	"github.com/matthewtzong/portfolio-tracker/backend/internal/plaid"
	"github.com/matthewtzong/portfolio-tracker/backend/internal/snaptrade"
)

// Checks and updates the status of all Plaid items.
func checkAndUpdatePlaidItemStatuses(ctx context.Context, db *database.Client, plaidClient *plaid.Client) error {
	if db == nil || plaidClient == nil {
		return nil
	}

	// Fetch all Plaid items
	items, err := db.ListPlaidItems(ctx)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	// Check status of each item
	for i := range items {
		itemStatus, err := plaidClient.GetItem(ctx, items[i].AccessToken)
		if err != nil {
			// Check if this is an authentication error (needs reconnection)
			var plaidErr *plaid.PlaidConnectionError
			if errors.As(err, &plaidErr) && plaidErr.IsAuthError {
				items[i].Status = "ITEM_LOGIN_REQUIRED"
				items[i].LastUpdated = now
				_ = db.UpsertPlaidItem(ctx, &items[i])
			}
			continue
		}

		if itemStatus == nil {
			continue
		}

		// Update status if it changed
		if items[i].Status != itemStatus.Status {
			items[i].Status = itemStatus.Status
			items[i].LastUpdated = now
			_ = db.UpsertPlaidItem(ctx, &items[i])
		}
	}

	return nil
}

// Checks and updates the status of all Snaptrade connections.
// Uses a 2-strike system to avoid false positives from transient errors:
func checkAndUpdateSnaptradeConnectionStatuses(ctx context.Context, db *database.Client, snapClient *snaptrade.Client) error {
	if db == nil || snapClient == nil {
		return nil
	}

	// Get Snaptrade user
	user, err := db.GetSnaptradeUser(ctx)
	if err != nil || user == nil {
		return err
	}

	// Get existing connections to check current status
	existingConns, _ := db.ListSnaptradeConnections(ctx)
	existingStatusMap := make(map[string]string) // connID -> current status
	for _, conn := range existingConns {
		existingStatusMap[conn.ConnID] = conn.Status
	}

	// Fetch connections from Snaptrade API.
	conns, err := snapClient.ListConnections(user.UserID, user.UserSecret)
	if err != nil {
		// If ListConnections fails, mark all connections based on 2-strike system:
		now := time.Now().UTC()
		var dbConns []database.SnaptradeConnection

		for _, conn := range existingConns {
			newStatus := conn.Status
			switch conn.Status {
			case "OK":
				newStatus = "ACCOUNT_FETCH_ERROR"
			case "ACCOUNT_FETCH_ERROR":
				newStatus = "CONNECTION_ERROR"
			}

			dbConns = append(dbConns, database.SnaptradeConnection{
				ConnID:     conn.ConnID,
				Brokerage:  conn.Brokerage,
				Status:     newStatus,
				LastSynced: &now,
			})
		}

		if len(dbConns) > 0 {
			_ = db.UpdateSnaptradeConnectionStatuses(ctx, dbConns)
		}
		return err
	}

	// If ListConnections succeeds, try to fetch accounts to verify connections are working.
	now := time.Now().UTC()
	var dbConns []database.SnaptradeConnection
	_, err = snapClient.ListAccounts(user.UserID, user.UserSecret)
	if err != nil {
		// If we can't fetch accounts, apply 2-strike system:
		for _, c := range conns {
			currentStatus := existingStatusMap[c.ID]
			newStatus := currentStatus

			switch currentStatus {
			case "OK":
				newStatus = "ACCOUNT_FETCH_ERROR"
			case "ACCOUNT_FETCH_ERROR":
				newStatus = "CONNECTION_ERROR"
			}

			dbConns = append(dbConns, database.SnaptradeConnection{
				ConnID:     c.ID,
				Brokerage:  c.Brokerage.Name,
				Status:     newStatus,
				LastSynced: &now,
			})
		}
	} else {
		// Successfully fetched accounts, reset all connections to OK
		for _, c := range conns {
			dbConns = append(dbConns, database.SnaptradeConnection{
				ConnID:     c.ID,
				Brokerage:  c.Brokerage.Name,
				Status:     "OK",
				LastSynced: &now,
			})
		}
	}

	// Update connection statuses in database
	if len(dbConns) > 0 {
		if err := db.UpdateSnaptradeConnectionStatuses(ctx, dbConns); err != nil {
			return err
		}
	}

	return nil
}
