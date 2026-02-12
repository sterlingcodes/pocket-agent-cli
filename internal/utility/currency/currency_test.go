package currency

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "currency" {
		t.Errorf("expected Use 'currency', got %q", cmd.Use)
	}

	aliases := cmd.Aliases
	if len(aliases) != 2 {
		t.Errorf("expected 2 aliases, got %d", len(aliases))
	}

	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"rate [from] [to]", "convert [amount] [from] [to]", "list"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestRateCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"base":   "USD",
			"date":   "2024-01-15",
			"rates":  map[string]float64{"EUR": 0.92},
			"amount": 1.0,
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newRateCmd()
	cmd.SetArgs([]string{"USD", "EUR"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("rate command failed: %v", err)
	}
}

func TestRateNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"base":  "USD",
			"date":  "2024-01-15",
			"rates": map[string]float64{},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newRateCmd()
	cmd.SetArgs([]string{"USD", "INVALID"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for invalid currency, got nil")
	}
}

func TestConvertCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"base":   "USD",
			"date":   "2024-01-15",
			"rates":  map[string]float64{"EUR": 92.0},
			"amount": 100.0,
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newConvertCmd()
	cmd.SetArgs([]string{"100", "USD", "EUR"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("convert command failed: %v", err)
	}
}

func TestConvertInvalidAmount(t *testing.T) {
	cmd := newConvertCmd()
	cmd.SetArgs([]string{"invalid", "USD", "EUR"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for invalid amount, got nil")
	}
}

func TestListCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]string{
			"USD": "United States Dollar",
			"EUR": "Euro",
			"GBP": "British Pound Sterling",
			"JPY": "Japanese Yen",
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newListCmd()
	err := cmd.Execute()
	if err != nil {
		t.Errorf("list command failed: %v", err)
	}
}

func TestListEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]string{}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newListCmd()
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for empty currency list, got nil")
	}
}

func TestRateLimitHandling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": "rate limited"})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newRateCmd()
	cmd.SetArgs([]string{"USD", "EUR"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected rate limit error, got nil")
	}
}

func TestNotFoundHandling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newRateCmd()
	cmd.SetArgs([]string{"INVALID", "EUR"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected not found error, got nil")
	}
}

func TestParseError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid json"))
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newRateCmd()
	cmd.SetArgs([]string{"USD", "EUR"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected parse error, got nil")
	}
}
