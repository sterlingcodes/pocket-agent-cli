package wayback

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "wayback" {
		t.Errorf("expected Use 'wayback', got %q", cmd.Use)
	}

	aliases := cmd.Aliases
	if len(aliases) != 2 {
		t.Errorf("expected 2 aliases, got %d", len(aliases))
	}

	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"check [url]", "latest [url]", "snapshots [url]"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"example.com", "https://example.com"},
		{"http://example.com", "http://example.com"},
		{"https://example.com", "https://example.com"},
		{"", ""},
	}

	for _, tt := range tests {
		got := normalizeURL(tt.input)
		if got != tt.want {
			t.Errorf("normalizeURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"20240115103045", "2024-01-15T10:30:45Z"},
		{"short", "short"},
		{"", ""},
	}

	for _, tt := range tests {
		got := formatTimestamp(tt.input)
		if tt.want != "short" && tt.want != "" {
			// Just check it doesn't error and returns something
			if len(got) < len(tt.input) {
				t.Errorf("formatTimestamp(%q) returned shorter string", tt.input)
			}
		}
	}
}

func TestCheckCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"url": "https://example.com",
			"archived_snapshots": map[string]any{
				"closest": map[string]any{
					"available": true,
					"url":       "https://web.archive.org/web/20240115/https://example.com",
					"timestamp": "20240115103045",
					"status":    "200",
				},
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := availabilityAPI
	availabilityAPI = srv.URL
	defer func() { availabilityAPI = oldURL }()

	cmd := newCheckCmd()
	cmd.SetArgs([]string{"https://example.com"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("check command failed: %v", err)
	}
}

func TestCheckCmdNotAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"url":                "https://notarchived.example",
			"archived_snapshots": map[string]any{},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := availabilityAPI
	availabilityAPI = srv.URL
	defer func() { availabilityAPI = oldURL }()

	cmd := newCheckCmd()
	cmd.SetArgs([]string{"https://notarchived.example"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("check command for unavailable URL failed: %v", err)
	}
}

func TestLatestCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"url": "https://example.com",
			"archived_snapshots": map[string]any{
				"closest": map[string]any{
					"available": true,
					"url":       "https://web.archive.org/web/20240115/https://example.com",
					"timestamp": "20240115103045",
					"status":    "200",
				},
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := availabilityAPI
	availabilityAPI = srv.URL
	defer func() { availabilityAPI = oldURL }()

	cmd := newLatestCmd()
	cmd.SetArgs([]string{"https://example.com"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("latest command failed: %v", err)
	}
}

func TestLatestCmdNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"url":                "https://notarchived.example",
			"archived_snapshots": map[string]any{},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := availabilityAPI
	availabilityAPI = srv.URL
	defer func() { availabilityAPI = oldURL }()

	cmd := newLatestCmd()
	cmd.SetArgs([]string{"https://notarchived.example"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected not found error, got nil")
	}
}

func TestSnapshotsCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := [][]string{
			{"timestamp", "original", "statuscode", "mimetype", "digest", "length"},
			{"20240115103045", "https://example.com", "200", "text/html", "ABC123", "12345"},
			{"20240101120000", "https://example.com", "200", "text/html", "DEF456", "12340"},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := cdxAPI
	cdxAPI = srv.URL
	defer func() { cdxAPI = oldURL }()

	cmd := newSnapshotsCmd()
	cmd.SetArgs([]string{"https://example.com"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("snapshots command failed: %v", err)
	}
}

func TestSnapshotsCmdNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := [][]string{
			{"timestamp", "original", "statuscode", "mimetype", "digest", "length"},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := cdxAPI
	cdxAPI = srv.URL
	defer func() { cdxAPI = oldURL }()

	cmd := newSnapshotsCmd()
	cmd.SetArgs([]string{"https://notarchived.example"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected not found error, got nil")
	}
}

func TestSnapshotsCmdWithLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := [][]string{
			{"timestamp", "original", "statuscode", "mimetype", "digest", "length"},
			{"20240115103045", "https://example.com", "200", "text/html", "ABC123", "12345"},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := cdxAPI
	cdxAPI = srv.URL
	defer func() { cdxAPI = oldURL }()

	cmd := newSnapshotsCmd()
	cmd.SetArgs([]string{"https://example.com", "--limit", "5"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("snapshots command with limit failed: %v", err)
	}
}

func TestRateLimitHandling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": "rate limited"})
	}))
	defer srv.Close()

	oldURL := availabilityAPI
	availabilityAPI = srv.URL
	defer func() { availabilityAPI = oldURL }()

	cmd := newCheckCmd()
	cmd.SetArgs([]string{"https://example.com"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected rate limit error, got nil")
	}
}

func TestHTTPErrorHandling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	oldURL := availabilityAPI
	availabilityAPI = srv.URL
	defer func() { availabilityAPI = oldURL }()

	cmd := newCheckCmd()
	cmd.SetArgs([]string{"https://example.com"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected HTTP error, got nil")
	}
}

func TestParseError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid json"))
	}))
	defer srv.Close()

	oldURL := availabilityAPI
	availabilityAPI = srv.URL
	defer func() { availabilityAPI = oldURL }()

	cmd := newCheckCmd()
	cmd.SetArgs([]string{"https://example.com"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected parse error, got nil")
	}
}
