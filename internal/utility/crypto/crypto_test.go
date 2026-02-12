package crypto

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "crypto" {
		t.Errorf("expected Use 'crypto', got %q", cmd.Use)
	}

	aliases := cmd.Aliases
	if len(aliases) != 2 {
		t.Errorf("expected 2 aliases, got %d", len(aliases))
	}

	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"price [coins...]", "info [coin]", "top", "trending", "search [query]"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestPriceCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := []map[string]any{
			{
				"id":                          "bitcoin",
				"symbol":                      "btc",
				"name":                        "Bitcoin",
				"current_price":               45000.0,
				"price_change_percentage_24h": 2.5,
				"market_cap":                  850000000000.0,
				"total_volume":                25000000000.0,
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newPriceCmd()
	cmd.SetArgs([]string{"bitcoin"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("price command failed: %v", err)
	}
}

func TestPriceNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := []map[string]any{}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newPriceCmd()
	cmd.SetArgs([]string{"invalid"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for not found coin, got nil")
	}
}

func TestInfoCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"id":     "bitcoin",
			"symbol": "btc",
			"name":   "Bitcoin",
			"description": map[string]string{
				"en": "Bitcoin is a cryptocurrency.",
			},
			"links": map[string][]string{
				"homepage": {"https://bitcoin.org"},
			},
			"market_cap_rank": 1,
			"market_data": map[string]any{
				"current_price":               map[string]float64{"usd": 45000.0},
				"market_cap":                  map[string]float64{"usd": 850000000000.0},
				"total_volume":                map[string]float64{"usd": 25000000000.0},
				"high_24h":                    map[string]float64{"usd": 46000.0},
				"low_24h":                     map[string]float64{"usd": 44000.0},
				"price_change_percentage_24h": 2.5,
				"price_change_percentage_7d":  5.0,
				"ath":                         map[string]float64{"usd": 69000.0},
				"ath_date":                    map[string]string{"usd": "2021-11-10T14:24:11.849Z"},
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newInfoCmd()
	cmd.SetArgs([]string{"bitcoin"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("info command failed: %v", err)
	}
}

func TestInfoNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{"error": "not found"})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newInfoCmd()
	cmd.SetArgs([]string{"invalid"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for not found coin, got nil")
	}
}

func TestTopCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := []map[string]any{
			{
				"id":                          "bitcoin",
				"symbol":                      "btc",
				"name":                        "Bitcoin",
				"current_price":               45000.0,
				"market_cap":                  850000000000.0,
				"total_volume":                25000000000.0,
				"price_change_percentage_24h": 2.5,
			},
			{
				"id":                          "ethereum",
				"symbol":                      "eth",
				"name":                        "Ethereum",
				"current_price":               3000.0,
				"market_cap":                  350000000000.0,
				"total_volume":                15000000000.0,
				"price_change_percentage_24h": 1.8,
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newTopCmd()
	err := cmd.Execute()
	if err != nil {
		t.Errorf("top command failed: %v", err)
	}
}

func TestTrendingCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"coins": []map[string]any{
				{
					"item": map[string]any{
						"id":              "bitcoin",
						"symbol":          "btc",
						"name":            "Bitcoin",
						"market_cap_rank": 1,
						"price_btc":       1.0,
					},
				},
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newTrendingCmd()
	err := cmd.Execute()
	if err != nil {
		t.Errorf("trending command failed: %v", err)
	}
}

func TestSearchCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"coins": []map[string]any{
				{
					"id":              "bitcoin",
					"symbol":          "btc",
					"name":            "Bitcoin",
					"market_cap_rank": 1,
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
	cmd.SetArgs([]string{"bitcoin"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("search command failed: %v", err)
	}
}

func TestRateLimitHandling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{"error": "rate limited"})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newPriceCmd()
	cmd.SetArgs([]string{"bitcoin"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected rate limit error, got nil")
	}
}
