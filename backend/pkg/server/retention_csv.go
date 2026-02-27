package server

import (
	"bytes"
	"context"
	"encoding/csv"
	"strconv"
	"time"

	"github.com/matthewtzong/portfolio-tracker/backend/pkg/database"
)

// Builds CSV for the given month's transactions.
func BuildTransactionsCSV(ctx context.Context, db *database.Client, month time.Time) ([]byte, error) {
	// Lists transactions for the given month.
	transactions, err := db.ListTransactionsForMonth(ctx, month)
	if err != nil {
		return nil, err
	}

	// Lists categories.
	categories, err := db.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	// Maps categories to their names.
	categoryMap := make(map[int64]string)
	for _, cat := range categories {
		categoryMap[cat.ID] = cat.Name
	}

	// Creates a new CSV writer.
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	// Writes the headers.
	csvHeaders := []string{"Date", "Name", "Merchant", "Category", "Amount ($)", "Pending"}
	err = writer.Write(csvHeaders)
	if err != nil {
		return nil, err
	}

	// Writes the rows.
	for _, transaction := range transactions {
		categoryName := "Uncategorized"
		if transaction.CategoryID != nil {
			name, ok := categoryMap[*transaction.CategoryID]
			if ok {
				categoryName = name
			}
		}
		merchant := ""
		if transaction.MerchantName != nil {
			merchant = *transaction.MerchantName
		}
		amountDollars := float64(transaction.AmountCents) / 100.0
		pending := "No"
		if transaction.Pending {
			pending = "Yes"
		}
		row := []string{
			transaction.Date.Format("2006-01-02"),
			transaction.Name,
			merchant,
			categoryName,
			strconv.FormatFloat(amountDollars, 'f', 2, 64),
			pending,
		}
		err = writer.Write(row)
		if err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return buffer.Bytes(), writer.Error()
}

// Builds CSV for the given year's monthly snapshots.
func BuildPortfolioSnapshotsCSV(snapshots []database.MonthlySnapshot) ([]byte, error) {
	// Creates a new CSV writer.
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	csvHeaders := []string{"Month", "Account ID", "Portfolio Value ($)"}
	err := writer.Write(csvHeaders)
	if err != nil {
		return nil, err
	}
	// Writes the rows
	for _, snapshot := range snapshots {
		row := []string{
			snapshot.Month.Format("2006-01"),
			snapshot.AccountID,
			strconv.FormatFloat(float64(snapshot.PortfolioValueCents)/100.0, 'f', 2, 64),
		}
		err = writer.Write(row)
		if err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return buffer.Bytes(), writer.Error()
}
