package timezone

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "timezone" {
		t.Errorf("expected Use 'timezone', got %q", cmd.Use)
	}

	aliases := cmd.Aliases
	if len(aliases) != 2 {
		t.Errorf("expected 2 aliases, got %d", len(aliases))
	}

	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"get [timezone]", "ip [ip-address]", "list"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestGetCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"timezone":     "America/New_York",
			"datetime":     "2024-01-15T10:30:45-05:00",
			"utc_offset":   "-05:00",
			"day_of_week":  1,
			"week_number":  3,
			"dst":          false,
			"abbreviation": "EST",
			"unixtime":     1705328445,
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newGetCmd()
	cmd.SetArgs([]string{"America/New_York"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("get command failed: %v", err)
	}
}

func TestGetNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "timezone not found"})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newGetCmd()
	cmd.SetArgs([]string{"Invalid/Timezone"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for invalid timezone, got nil")
	}
}

func TestIPCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"timezone":     "America/Los_Angeles",
			"datetime":     "2024-01-15T07:30:45-08:00",
			"utc_offset":   "-08:00",
			"day_of_week":  1,
			"week_number":  3,
			"dst":          false,
			"abbreviation": "PST",
			"unixtime":     1705328445,
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newIPCmd()
	cmd.SetArgs([]string{"8.8.8.8"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("ip command failed: %v", err)
	}
}

func TestListCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := []string{
			"America/New_York",
			"America/Los_Angeles",
			"Europe/London",
			"Europe/Paris",
			"Asia/Tokyo",
			"Asia/Shanghai",
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

func TestFetchTimezoneHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	err := fetchTimezone(srv.URL + "/test")
	if err == nil {
		t.Error("expected HTTP error, got nil")
	}
}

func TestFetchTimezoneParseError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid json"))
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	err := fetchTimezone(srv.URL + "/test")
	if err == nil {
		t.Error("expected parse error, got nil")
	}
}

func TestListTimezonesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	err := listTimezones()
	if err == nil {
		t.Error("expected HTTP error, got nil")
	}
}

func TestListTimezonesParseError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid json"))
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	err := listTimezones()
	if err == nil {
		t.Error("expected parse error, got nil")
	}
}
