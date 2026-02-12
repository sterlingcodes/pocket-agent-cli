package holidays

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "holidays" {
		t.Errorf("expected Use 'holidays', got %q", cmd.Use)
	}

	aliases := cmd.Aliases
	if len(aliases) != 2 {
		t.Errorf("expected 2 aliases, got %d", len(aliases))
	}

	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"list [country-code] [year]", "next [country-code]", "countries"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestListCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := []map[string]any{
			{
				"date":        "2024-01-01",
				"localName":   "New Year's Day",
				"name":        "New Year's Day",
				"countryCode": "US",
				"fixed":       true,
				"global":      true,
				"counties":    nil,
				"types":       []string{"Public"},
			},
			{
				"date":        "2024-07-04",
				"localName":   "Independence Day",
				"name":        "Independence Day",
				"countryCode": "US",
				"fixed":       true,
				"global":      true,
				"counties":    nil,
				"types":       []string{"Public"},
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newListCmd()
	cmd.SetArgs([]string{"US", "2024"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("list command failed: %v", err)
	}
}

func TestListCmdCurrentYear(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := []map[string]any{
			{
				"date":        "2024-01-01",
				"localName":   "New Year's Day",
				"name":        "New Year's Day",
				"countryCode": "US",
				"fixed":       true,
				"global":      true,
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newListCmd()
	cmd.SetArgs([]string{"US"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("list command with current year failed: %v", err)
	}
}

func TestListCmdInvalidYear(t *testing.T) {
	cmd := newListCmd()
	cmd.SetArgs([]string{"US", "invalid"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for invalid year, got nil")
	}
}

func TestListCmdNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "country not found"})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newListCmd()
	cmd.SetArgs([]string{"INVALID", "2024"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected not found error, got nil")
	}
}

func TestNextCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := []map[string]any{
			{
				"date":        "2099-01-01",
				"localName":   "New Year's Day",
				"name":        "New Year's Day",
				"countryCode": "US",
				"fixed":       true,
				"global":      true,
			},
			{
				"date":        "2099-07-04",
				"localName":   "Independence Day",
				"name":        "Independence Day",
				"countryCode": "US",
				"fixed":       true,
				"global":      true,
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newNextCmd()
	cmd.SetArgs([]string{"US"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("next command failed: %v", err)
	}
}

func TestNextCmdWithLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := []map[string]any{
			{
				"date":        "2099-01-01",
				"localName":   "New Year's Day",
				"name":        "New Year's Day",
				"countryCode": "US",
				"fixed":       true,
				"global":      true,
			},
			{
				"date":        "2099-07-04",
				"localName":   "Independence Day",
				"name":        "Independence Day",
				"countryCode": "US",
				"fixed":       true,
				"global":      true,
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newNextCmd()
	cmd.SetArgs([]string{"US", "--limit", "1"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("next command with limit failed: %v", err)
	}
}

func TestCountriesCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := []map[string]string{
			{"countryCode": "US", "name": "United States"},
			{"countryCode": "GB", "name": "United Kingdom"},
			{"countryCode": "DE", "name": "Germany"},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newCountriesCmd()
	err := cmd.Execute()
	if err != nil {
		t.Errorf("countries command failed: %v", err)
	}
}

func TestCountriesEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := []map[string]string{}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newCountriesCmd()
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for empty countries list, got nil")
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

	cmd := newListCmd()
	cmd.SetArgs([]string{"US", "2024"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected rate limit error, got nil")
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

	cmd := newListCmd()
	cmd.SetArgs([]string{"US", "2024"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected parse error, got nil")
	}
}
