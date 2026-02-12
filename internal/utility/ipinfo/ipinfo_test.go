package ipinfo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "ip" {
		t.Errorf("expected Use 'ip', got %q", cmd.Use)
	}

	aliases := cmd.Aliases
	if len(aliases) != 2 {
		t.Errorf("expected 2 aliases, got %d", len(aliases))
	}

	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"lookup [ip]", "me"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestLookupCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := IPInfo{
			IP:       "8.8.8.8",
			Hostname: "dns.google",
			City:     "Mountain View",
			Region:   "California",
			Country:  "US",
			Location: "37.4056,-122.0775",
			Org:      "AS15169 Google LLC",
			Postal:   "94043",
			Timezone: "America/Los_Angeles",
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newLookupCmd()
	cmd.SetArgs([]string{"8.8.8.8"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("lookup command failed: %v", err)
	}
}

func TestLookupInvalidIP(t *testing.T) {
	cmd := newLookupCmd()
	cmd.SetArgs([]string{"not-an-ip"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for invalid IP, got nil")
	}
}

func TestMyIPCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := IPInfo{
			IP:       "1.2.3.4",
			City:     "San Francisco",
			Region:   "California",
			Country:  "US",
			Location: "37.7749,-122.4194",
			Org:      "AS12345 Example ISP",
			Timezone: "America/Los_Angeles",
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newMyIPCmd()
	err := cmd.Execute()
	if err != nil {
		t.Errorf("me command failed: %v", err)
	}
}

func TestFetchIPInfoRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": "rate limited"})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	err := fetchIPInfo("8.8.8.8")
	if err == nil {
		t.Error("expected rate limit error, got nil")
	}
}

func TestFetchIPInfoHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	err := fetchIPInfo("8.8.8.8")
	if err == nil {
		t.Error("expected HTTP error, got nil")
	}
}

func TestFetchIPInfoParseError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid json"))
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	err := fetchIPInfo("8.8.8.8")
	if err == nil {
		t.Error("expected parse error, got nil")
	}
}
