package server

import (
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/matthewtzong/portfolio-tracker/backend/internal/database"
)

func TestSortSnapshotDataPointsOrdersByDate(t *testing.T) {
	points := []SnapshotDataPoint{
		{Date: "2024-03-01", PortfolioValueCents: 200},
		{Date: "2024-01-01", PortfolioValueCents: 100},
		{Date: "2024-02-01", PortfolioValueCents: 150},
	}

	sortSnapshotDataPoints(points)

	if points[0].Date != "2024-01-01" || points[1].Date != "2024-02-01" || points[2].Date != "2024-03-01" {
		t.Fatalf("snapshots not sorted by date: %#v", points)
	}
}

func TestAggregateMonthlySnapshotsSumsByMonth(t *testing.T) {
	month1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	month2 := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	allMonthly := []database.MonthlySnapshot{
		{Month: database.DateOnly{Time: month1}, AccountID: "acc-1", PortfolioValueCents: 100},
		{Month: database.DateOnly{Time: month1}, AccountID: "acc-2", PortfolioValueCents: 150},
		{Month: database.DateOnly{Time: month2}, AccountID: "acc-1", PortfolioValueCents: 200},
	}

	points := aggregateMonthlySnapshotsForTest(allMonthly)

	if len(points) != 2 {
		t.Fatalf("expected 2 aggregated points, got %d", len(points))
	}

	if points[0].Date != "2024-01-01" || points[0].PortfolioValueCents != 250 {
		t.Fatalf("unexpected first point: %#v", points[0])
	}
	if points[1].Date != "2024-02-01" || points[1].PortfolioValueCents != 200 {
		t.Fatalf("unexpected second point: %#v", points[1])
	}
}

// Function for testing the aggregation of monthly snapshots.
func aggregateMonthlySnapshotsForTest(allMonthly []database.MonthlySnapshot) []SnapshotDataPoint {
	sumByMonth := make(map[string]int64)
	for _, snapshot := range allMonthly {
		month := snapshot.Month.Format(dateLayout)
		sumByMonth[month] += snapshot.PortfolioValueCents
	}

	points := make([]SnapshotDataPoint, 0, len(sumByMonth))
	for month, sum := range sumByMonth {
		points = append(points, SnapshotDataPoint{
			Date:                month,
			PortfolioValueCents: sum,
		})
	}

	sortSnapshotDataPoints(points)
	return points
}

// Test that the portfolio snapshots CSV includes headers and rows.
func TestBuildPortfolioSnapshotsCSVIncludesHeadersAndRows(t *testing.T) {
	month1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	snapshots := []database.MonthlySnapshot{
		{Month: database.DateOnly{Time: month1}, AccountID: "acc-1", PortfolioValueCents: 12345},
	}

	data, err := BuildPortfolioSnapshotsCSV(snapshots)
	if err != nil {
		t.Fatalf("BuildPortfolioSnapshotsCSV returned error: %v", err)
	}

	// Reads the CSV data.
	reader := csv.NewReader(strings.NewReader(string(data)))
	header, err := reader.Read()
	if err != nil {
		t.Fatalf("failed to read header: %v", err)
	}
	if len(header) != 3 || header[0] != "Month" || header[1] != "Account ID" || header[2] != "Portfolio Value ($)" {
		t.Fatalf("unexpected header: %#v", header)
	}

	// Reads the data row.
	row, err := reader.Read()
	if err != nil {
		t.Fatalf("failed to read data row: %v", err)
	}
	if row[0] != "2024-01" || row[1] != "acc-1" || row[2] != "123.45" {
		t.Fatalf("unexpected data row: %#v", row)
	}
}
