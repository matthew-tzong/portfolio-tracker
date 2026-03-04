package server

import (
	"strings"
	"testing"
)

// Tests parseCents function.
func TestParseCents(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"$1,234.56", 123456},
		{"1234.56", 123456},
		{"$0.01", 1},
		{"-$1.00", -100},
		{"--", 0},
		{"", 0},
		{" 45.67 ", 4567},
	}

	for _, tt := range tests {
		got := parseCents(tt.input)
		if got != tt.expected {
			t.Errorf("parseCents(%q) = %d; want %d", tt.input, got, tt.expected)
		}
	}
}

// Tests parseFidelityMonthlyCSV function.
func TestParseFidelityMonthlyCSV(t *testing.T) {
	csvData := `Brokerage Account Summary
Account Number,XXXX-1234

Individual,
Symbol/CUSIP,Description,Quantity,Price,Beginning Value,Ending Value,Cost Basis
AAPL,APPLE INC,10.000,$150.00,$1450.00,$1500.00,$1400.00
MSFT,MICROSOFT CORP,5.000,$300.00,$1400.00,$1500.00,$1300.00
SPAXX,FIDELITY GOVERNMENT MONEY MARKET,1000.000,$1.00,$1000.00,$1000.00,not applicable

Subtotal,Individual,,,$3850.00,$4000.00,
`
	holdings, err := parseFidelityMonthlyCSV(strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(holdings) != 3 {
		t.Errorf("expected 3 holdings, got %d", len(holdings))
	}

	expected := []struct {
		symbol string
		qty    float64
		value  int64
	}{
		{"AAPL", 10.0, 150000},
		{"MSFT", 5.0, 150000},
		{"SPAXX", 1000.0, 100000},
	}

	for i, h := range holdings {
		if h.Symbol != expected[i].symbol {
			t.Errorf("holding %d: expected symbol %s, got %s", i, expected[i].symbol, h.Symbol)
		}
		if h.Quantity != expected[i].qty {
			t.Errorf("holding %d: expected quantity %f, got %f", i, expected[i].qty, h.Quantity)
		}
		if h.ValueCents != expected[i].value {
			t.Errorf("holding %d: expected value %d, got %d", i, expected[i].value, h.ValueCents)
		}
	}
}

// Tests parseFidelityHoldingsCSV function.
func TestParseFidelityHoldingsCSV(t *testing.T) {
	// Fidelity Portfolio Positions CSV header starts with "Account Name/Number"
	csvData := `Account Name/Number,Account Type,Symbol,Description,Quantity,Price,Last Price Change,Current Value,Today's Gain/Loss $,Today's Gain/Loss %,Total Gain/Loss $,Total Gain/Loss %,Percent Of Account,Cost Basis Per Share,Cost Basis Total,Type
"XXXX1234",Individual,AAPL,APPLE INC,10,$150.00,+$1.00,$1500.00,+$10.00,+0.67%,+$100.00,+7.14%,37.5%,$140.00,$1400.00,Cash
"XXXX1234",Individual,SPAXX**,FIDELITY GOVERNMENT MONEY MARKET,1000,$1.00,--,$1000.00,--,--,--,--,25.0%,n/a,n/a,Cash
"XXXX1234",Individual,Pending Activity,PENDING ACTIVITY,n/a,n/a,n/a,$500.00,n/a,n/a,n/a,n/a,12.5%,n/a,n/a,Cash
`
	holdings, err := parseFidelityHoldingsCSV(strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Should ignore "Pending Activity"
	if len(holdings) != 2 {
		t.Errorf("expected 2 holdings, got %d", len(holdings))
	}

	if holdings[0].Symbol != "AAPL" {
		t.Errorf("expected AAPL, got %s", holdings[0].Symbol)
	}
	if holdings[1].Symbol != "SPAXX" { // should strip **
		t.Errorf("expected SPAXX, got %s", holdings[1].Symbol)
	}
	if holdings[1].ValueCents != 100000 {
		t.Errorf("expected 100000 cents for SPAXX, got %d", holdings[1].ValueCents)
	}
}
