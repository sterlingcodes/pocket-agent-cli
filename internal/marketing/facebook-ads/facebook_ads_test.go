package facebookads

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstablemind/pocket/internal/common/config"
)

// testConfigDir is set once in TestMain and reused by all tests.
var testConfigPath string

func TestMain(m *testing.M) {
	// Set up a temp config path BEFORE any config.Path() call.
	dir, err := os.MkdirTemp("", "fbads-test-*")
	if err != nil {
		panic(err)
	}
	testConfigPath = filepath.Join(dir, "config.json")
	os.Setenv("POCKET_CONFIG", testConfigPath)

	code := m.Run()

	os.Unsetenv("POCKET_CONFIG")
	os.RemoveAll(dir)
	os.Exit(code)
}

// writeTestConfig writes config values to the shared test config file.
func writeTestConfig(t *testing.T, values map[string]string) {
	t.Helper()
	data, err := json.Marshal(values)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(testConfigPath, data, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// clearTestConfig removes the config file so Load() returns empty config.
func clearTestConfig(t *testing.T) {
	t.Helper()
	os.Remove(testConfigPath)
}

func TestActIDPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"123456", "act_123456"},
		{"act_123456", "act_123456"},
		{"act_", "act_"},
		{"999", "act_999"},
	}

	for _, tt := range tests {
		c := &fbClient{accountID: tt.input}
		got := c.actID()
		if got != tt.want {
			t.Errorf("actID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseAPIError(t *testing.T) {
	resp := map[string]any{
		"error": map[string]any{
			"message":       "Invalid OAuth access token",
			"type":          "OAuthException",
			"code":          float64(190),
			"error_subcode": float64(463),
			"fbtrace_id":    "ABC123xyz",
		},
	}

	err := parseAPIError(resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	msg := err.Error()
	for _, want := range []string{"Invalid OAuth access token", "OAuthException", "190", "463", "ABC123xyz"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message %q missing expected substring %q", msg, want)
		}
	}
}

func TestParseAPIErrorNoError(t *testing.T) {
	resp := map[string]any{
		"data": []any{},
	}

	err := parseAPIError(resp)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestGetDataArray(t *testing.T) {
	resp := map[string]any{
		"data": []any{
			map[string]any{"id": "1"},
			map[string]any{"id": "2"},
		},
	}

	data := getDataArray(resp)
	if len(data) != 2 {
		t.Fatalf("expected 2 items, got %d", len(data))
	}

	// No "data" key
	noData := map[string]any{"other": "value"}
	if got := getDataArray(noData); got != nil {
		t.Errorf("expected nil for no data key, got %v", got)
	}
}

func TestHelpers(t *testing.T) {
	m := map[string]any{
		"name":   "test",
		"count":  float64(42),
		"active": true,
	}

	if got := getString(m, "name"); got != "test" {
		t.Errorf("getString(name) = %q, want %q", got, "test")
	}
	if got := getString(m, "missing"); got != "" {
		t.Errorf("getString(missing) = %q, want empty", got)
	}

	if got := getInt(m, "count"); got != 42 {
		t.Errorf("getInt(count) = %d, want 42", got)
	}
	if got := getInt(m, "missing"); got != 0 {
		t.Errorf("getInt(missing) = %d, want 0", got)
	}

	if got := getBool(m, "active"); got != true {
		t.Errorf("getBool(active) = %v, want true", got)
	}
	if got := getBool(m, "missing"); got != false {
		t.Errorf("getBool(missing) = %v, want false", got)
	}
}

func TestNewCmdSubcommands(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "facebook-ads" {
		t.Errorf("expected Use=facebook-ads, got %s", cmd.Use)
	}

	want := map[string]bool{
		"account":         false,
		"campaigns":       false,
		"campaign-create": false,
		"campaign-update": false,
		"adsets":          false,
		"adset-create":    false,
		"ads":             false,
		"insights":        false,
	}

	for _, sub := range cmd.Commands() {
		use := sub.Use
		// Strip args from Use (e.g., "campaign-update [campaign-id]" → "campaign-update")
		if idx := strings.IndexByte(use, ' '); idx != -1 {
			use = use[:idx]
		}
		if _, ok := want[use]; ok {
			want[use] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q not found in NewCmd()", name)
		}
	}
}

func TestDoGetSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test_token_123" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer test_token_123")
		}
		if r.Method != "GET" {
			t.Errorf("method = %s, want GET", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []any{
				map[string]any{"id": "123", "name": "Test Campaign"},
			},
		})
	}))
	defer ts.Close()

	origURL := baseURL
	baseURL = ts.URL
	defer func() { baseURL = origURL }()

	c := &fbClient{token: "test_token_123", accountID: "act_999"}
	result, err := c.doGet("act_999/campaigns", nil)
	if err != nil {
		t.Fatalf("doGet failed: %v", err)
	}

	data := getDataArray(result)
	if len(data) != 1 {
		t.Fatalf("expected 1 item, got %d", len(data))
	}

	item := data[0].(map[string]any)
	if getString(item, "id") != "123" {
		t.Errorf("id = %q, want %q", getString(item, "id"), "123")
	}
}

func TestDoGetAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Invalid token",
				"type":    "OAuthException",
				"code":    float64(190),
			},
		})
	}))
	defer ts.Close()

	origURL := baseURL
	baseURL = ts.URL
	defer func() { baseURL = origURL }()

	c := &fbClient{token: "bad_token", accountID: "act_999"}
	_, err := c.doGet("act_999/campaigns", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "Invalid token") {
		t.Errorf("error %q should contain 'Invalid token'", err.Error())
	}
	if !strings.Contains(err.Error(), "OAuthException") {
		t.Errorf("error %q should contain 'OAuthException'", err.Error())
	}
}

func TestDoGetHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("internal error"))
	}))
	defer ts.Close()

	origURL := baseURL
	baseURL = ts.URL
	defer func() { baseURL = origURL }()

	c := &fbClient{token: "tok", accountID: "act_1"}
	_, err := c.doGet("endpoint", nil)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error %q should contain '500'", err.Error())
	}
}

func TestDoPostSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer post_token" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer post_token")
		}
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", ct)
		}

		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "name=Test") {
			t.Errorf("body %q should contain 'name=Test'", string(body))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "123"})
	}))
	defer ts.Close()

	origURL := baseURL
	baseURL = ts.URL
	defer func() { baseURL = origURL }()

	c := &fbClient{token: "post_token", accountID: "act_1"}
	result, err := c.doPost("act_1/campaigns", map[string]string{"name": "Test"})
	if err != nil {
		t.Fatalf("doPost failed: %v", err)
	}

	if getString(result, "id") != "123" {
		t.Errorf("id = %q, want %q", getString(result, "id"), "123")
	}
}

func TestDoPostHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte("unauthorized"))
	}))
	defer ts.Close()

	origURL := baseURL
	baseURL = ts.URL
	defer func() { baseURL = origURL }()

	c := &fbClient{token: "bad", accountID: "act_1"}
	_, err := c.doPost("endpoint", map[string]string{"k": "v"})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}

	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error %q should contain '401'", err.Error())
	}
}

func TestNewClientMissingToken(t *testing.T) {
	writeTestConfig(t, map[string]string{
		"facebook_ads_account_id": "123",
	})

	_, err := newClient()
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestNewClientMissingAccountID(t *testing.T) {
	writeTestConfig(t, map[string]string{
		"facebook_ads_token": "test_token",
	})

	_, err := newClient()
	if err == nil {
		t.Fatal("expected error for missing account ID")
	}
}

func TestNewClientSuccess(t *testing.T) {
	writeTestConfig(t, map[string]string{
		"facebook_ads_token":      "valid_token",
		"facebook_ads_account_id": "12345",
	})

	c, err := newClient()
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if c.token != "valid_token" {
		t.Errorf("token = %q, want %q", c.token, "valid_token")
	}
	if c.accountID != "12345" {
		t.Errorf("accountID = %q, want %q", c.accountID, "12345")
	}
}

func TestFacebookAdsConfigKeys(t *testing.T) {
	clearTestConfig(t)

	// Verify round-trip through config.Set/Get
	keys := map[string]string{
		"facebook_ads_token":      "test_fb_token_abc",
		"facebook_ads_account_id": "987654321",
	}

	for k, v := range keys {
		if err := config.Set(k, v); err != nil {
			t.Fatalf("config.Set(%q) failed: %v", k, err)
		}
	}

	for k, want := range keys {
		got, err := config.Get(k)
		if err != nil {
			t.Fatalf("config.Get(%q) failed: %v", k, err)
		}
		if got != want {
			t.Errorf("config.Get(%q) = %q, want %q", k, got, want)
		}
	}
}
