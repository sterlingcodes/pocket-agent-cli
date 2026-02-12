package virustotal

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/unstablemind/pocket/internal/common/config"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "virustotal" {
		t.Errorf("expected Use 'virustotal', got %q", cmd.Use)
	}
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	expectedSubs := []string{"url [url]", "domain [domain]", "ip [ip]", "hash [hash]"}
	for _, name := range expectedSubs {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func setupTestConfig(t *testing.T) func() {
	tmpFile, err := os.CreateTemp("", "vt-config-*.json")
	if err != nil {
		t.Fatal(err)
	}
	// Write empty JSON object to initialize config file
	if _, err := tmpFile.Write([]byte("{}")); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	os.Setenv("POCKET_CONFIG", tmpFile.Name())

	// Set API key
	if err := config.Set("virustotal_api_key", "test-api-key"); err != nil {
		t.Fatal(err)
	}

	return func() {
		os.Unsetenv("POCKET_CONFIG")
		os.Remove(tmpFile.Name())
	}
}

func TestDomain_Success(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-apikey") != "test-api-key" {
			t.Error("missing or incorrect API key header")
		}
		if r.URL.Path != "/domains/example.com" {
			t.Errorf("expected path /domains/example.com, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"data": {
				"attributes": {
					"reputation": 10,
					"last_analysis_stats": {
						"malicious": 0,
						"suspicious": 0,
						"harmless": 80,
						"undetected": 5
					},
					"categories": {
						"Forcepoint ThreatSeeker": "search engines and portals"
					},
					"last_analysis_date": 1609459200
				}
			}
		}`))
	}))
	defer srv.Close()

	oldBaseURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldBaseURL }()

	cmd := newDomainCmd()
	cmd.SetArgs([]string{"example.com"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestIP_Success(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ip_addresses/8.8.8.8" {
			t.Errorf("expected path /ip_addresses/8.8.8.8, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"data": {
				"attributes": {
					"as_owner": "Google LLC",
					"country": "US",
					"reputation": 0,
					"last_analysis_stats": {
						"malicious": 0,
						"suspicious": 0,
						"harmless": 70,
						"undetected": 10
					},
					"last_analysis_date": 1609459200
				}
			}
		}`))
	}))
	defer srv.Close()

	oldBaseURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldBaseURL }()

	cmd := newIPCmd()
	cmd.SetArgs([]string{"8.8.8.8"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestHash_Success(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/files/d41d8cd98f00b204e9800998ecf8427e" {
			t.Errorf("expected path /files/d41d8cd98f00b204e9800998ecf8427e, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"data": {
				"attributes": {
					"sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					"meaningful_name": "test.txt",
					"type_description": "Text",
					"size": 0,
					"reputation": 0,
					"last_analysis_stats": {
						"malicious": 0,
						"suspicious": 0,
						"harmless": 60,
						"undetected": 10
					},
					"first_submission_date": 1609459200,
					"last_analysis_date": 1609459200
				}
			}
		}`))
	}))
	defer srv.Close()

	oldBaseURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldBaseURL }()

	cmd := newHashCmd()
	cmd.SetArgs([]string{"d41d8cd98f00b204e9800998ecf8427e"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestDomain_APIError(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"code": "Unauthorized", "message": "Invalid API key"}}`))
	}))
	defer srv.Close()

	oldBaseURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldBaseURL }()

	cmd := newDomainCmd()
	cmd.SetArgs([]string{"example.com"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestDomain_InvalidJSON(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	oldBaseURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldBaseURL }()

	cmd := newDomainCmd()
	cmd.SetArgs([]string{"example.com"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGetAPIKey_NoKey(t *testing.T) {
	// Force config to reload by creating a new isolated environment
	tmpFile, err := os.CreateTemp("", "vt-config-nokey-*.json")
	if err != nil {
		t.Fatal(err)
	}
	// Write empty JSON object
	if _, err := tmpFile.Write([]byte("{}")); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Clear any environment that might be set
	oldVal := os.Getenv("POCKET_CONFIG")
	os.Setenv("POCKET_CONFIG", tmpFile.Name())
	defer func() {
		if oldVal != "" {
			os.Setenv("POCKET_CONFIG", oldVal)
		} else {
			os.Unsetenv("POCKET_CONFIG")
		}
	}()

	// Force config package to reload by creating a fresh instance
	// (Note: config package uses sync.Once, so it won't reload in same test run)
	// This test verifies getAPIKey logic when key is empty string
	key, err := config.Get("virustotal_api_key")
	if key != "" {
		// Skip if previous test left state
		t.Skip("config state persists across tests due to sync.Once")
	}

	_, err = getAPIKey()
	if err == nil {
		t.Error("expected error for missing API key, got nil")
	}
}
