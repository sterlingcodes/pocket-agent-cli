package stocks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/unstablemind/pocket/internal/common/config"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "stocks" {
		t.Errorf("expected Use 'stocks', got %q", cmd.Use)
	}

	aliases := cmd.Aliases
	if len(aliases) != 2 {
		t.Errorf("expected 2 aliases, got %d", len(aliases))
	}

	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"quote [symbol]", "search [query]", "info [symbol]"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func setupTestConfig(t *testing.T) func() {
	t.Helper()

	tmpfile, err := os.CreateTemp("", "stocks_test_*.json")
	if err != nil {
		t.Fatal(err)
	}

	// Initialize with empty JSON object
	if _, err := tmpfile.Write([]byte("{}")); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	oldEnv := os.Getenv("POCKET_CONFIG")
	os.Setenv("POCKET_CONFIG", tmpfile.Name())

	// Set test API key
	if err := config.Set("alphavantage_key", "test_api_key"); err != nil {
		t.Fatal(err)
	}

	return func() {
		if oldEnv != "" {
			os.Setenv("POCKET_CONFIG", oldEnv)
		} else {
			os.Unsetenv("POCKET_CONFIG")
		}
		os.Remove(tmpfile.Name())
	}
}

func TestQuoteCmd(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"Global Quote": map[string]string{
				"01. symbol":             "AAPL",
				"02. open":               "150.00",
				"03. high":               "152.00",
				"04. low":                "149.00",
				"05. price":              "151.50",
				"06. volume":             "50000000",
				"07. latest trading day": "2024-01-15",
				"08. previous close":     "150.50",
				"09. change":             "1.00",
				"10. change percent":     "0.66%",
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newQuoteCmd()
	cmd.SetArgs([]string{"AAPL"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("quote command failed: %v", err)
	}
}

func TestQuoteRateLimit(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"Note": "API rate limit reached",
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newQuoteCmd()
	cmd.SetArgs([]string{"AAPL"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected rate limit error, got nil")
	}
}

func TestSearchCmd(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"bestMatches": []map[string]string{
				{
					"1. symbol":   "AAPL",
					"2. name":     "Apple Inc.",
					"3. type":     "Equity",
					"4. region":   "United States",
					"8. currency": "USD",
				},
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newSearchCmd()
	cmd.SetArgs([]string{"apple"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("search command failed: %v", err)
	}
}

func TestInfoCmd(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"Symbol":               "AAPL",
			"Name":                 "Apple Inc.",
			"Description":          "Apple Inc. designs and manufactures consumer electronics.",
			"Exchange":             "NASDAQ",
			"Sector":               "Technology",
			"Industry":             "Consumer Electronics",
			"MarketCapitalization": "2500000000000",
			"PERatio":              "28.5",
			"DividendYield":        "0.005",
			"EPS":                  "5.89",
			"52WeekHigh":           "180.00",
			"52WeekLow":            "120.00",
			"Country":              "USA",
			"OfficialSite":         "https://apple.com",
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newInfoCmd()
	cmd.SetArgs([]string{"AAPL"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("info command failed: %v", err)
	}
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"123.45", 123.45},
		{"0", 0},
		{"-10.5", -10.5},
		{"invalid", 0},
		{"", 0},
	}

	for _, tt := range tests {
		got := parseFloat(tt.input)
		if got != tt.want {
			t.Errorf("parseFloat(%q) = %f, want %f", tt.input, got, tt.want)
		}
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"12345", 12345},
		{"0", 0},
		{"-100", -100},
		{"invalid", 0},
		{"", 0},
	}

	for _, tt := range tests {
		got := parseInt(tt.input)
		if got != tt.want {
			t.Errorf("parseInt(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
