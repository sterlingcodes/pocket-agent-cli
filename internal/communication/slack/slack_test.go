package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/unstablemind/pocket/internal/common/config"
)

func TestMain(m *testing.M) {
	// Use a temporary config file for tests
	tmpFile := "/tmp/pocket-slack-test-config.json"
	os.Setenv("POCKET_CONFIG", tmpFile)
	defer os.Remove(tmpFile)
	os.Exit(m.Run())
}

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "slack" {
		t.Errorf("expected Use 'slack', got %q", cmd.Use)
	}

	// Check subcommands
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"channels", "messages [channel]", "send [message]", "users", "dm [user-id] [message]", "search [query]"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestChannelsCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization 'Bearer test-token', got %q", r.Header.Get("Authorization"))
		}

		if !strings.Contains(r.URL.Path, "conversations.list") {
			t.Errorf("expected path to contain 'conversations.list', got %q", r.URL.Path)
		}

		resp := map[string]any{
			"ok": true,
			"channels": []map[string]any{
				{
					"id":          "C123456",
					"name":        "general",
					"is_private":  false,
					"is_archived": false,
					"is_member":   true,
					"num_members": 10,
					"topic":       map[string]string{"value": "General discussion"},
					"purpose":     map[string]string{"value": "Company-wide announcements"},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	config.Set("slack_token", "test-token")

	cmd := newChannelsCmd()
	cmd.SetArgs([]string{"--limit", "50"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestChannelsCmd_NoToken(t *testing.T) {
	config.Set("slack_token", "")

	cmd := newChannelsCmd()
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing token, got nil")
	}
}

func TestMessagesCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, "conversations.history") {
			resp := map[string]any{
				"ok": true,
				"messages": []map[string]any{
					{
						"type": "message",
						"user": "U123",
						"text": "Hello world",
						"ts":   "1234567890.123456",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		} else if strings.Contains(path, "users.info") {
			resp := map[string]any{
				"ok": true,
				"user": map[string]any{
					"name":      "testuser",
					"real_name": "Test User",
					"profile":   map[string]string{"display_name": "Test"},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	config.Set("slack_token", "test-token")

	cmd := newMessagesCmd()
	cmd.SetArgs([]string{"C123456", "--limit", "10"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestSendCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if !strings.Contains(r.URL.Path, "chat.postMessage") {
			t.Errorf("expected path to contain 'chat.postMessage', got %q", r.URL.Path)
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)

		if body["channel"] != "C123456" {
			t.Errorf("expected channel 'C123456', got %q", body["channel"])
		}

		if body["text"] != "test message" {
			t.Errorf("expected text 'test message', got %q", body["text"])
		}

		resp := map[string]any{
			"ok":      true,
			"channel": "C123456",
			"ts":      "1234567890.123456",
			"message": map[string]string{"text": "test message", "ts": "1234567890.123456"},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	config.Set("slack_token", "test-token")

	cmd := newSendCmd()
	cmd.SetArgs([]string{"test message", "--channel", "C123456"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestSendCmd_WithThread(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)

		if body["thread_ts"] != "1234567890.111111" {
			t.Errorf("expected thread_ts, got %q", body["thread_ts"])
		}

		resp := map[string]any{
			"ok":      true,
			"channel": "C123456",
			"ts":      "1234567890.123456",
			"message": map[string]string{"text": "reply", "ts": "1234567890.123456"},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	config.Set("slack_token", "test-token")

	cmd := newSendCmd()
	cmd.SetArgs([]string{"reply", "--channel", "C123456", "--thread", "1234567890.111111"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestUsersCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "users.list") {
			t.Errorf("expected path to contain 'users.list', got %q", r.URL.Path)
		}

		resp := map[string]any{
			"ok": true,
			"members": []map[string]any{
				{
					"id":        "U123",
					"name":      "testuser",
					"real_name": "Test User",
					"deleted":   false,
					"is_bot":    false,
					"is_admin":  true,
					"profile": map[string]string{
						"display_name": "Test",
						"email":        "test@example.com",
						"status_text":  "Working",
						"status_emoji": ":coffee:",
					},
					"tz": "America/New_York",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	config.Set("slack_token", "test-token")

	cmd := newUsersCmd()
	cmd.SetArgs([]string{"--limit", "50"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestDMCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, "conversations.open") {
			resp := map[string]any{
				"ok":      true,
				"channel": map[string]string{"id": "D123456"},
			}
			json.NewEncoder(w).Encode(resp)
		} else if strings.Contains(path, "chat.postMessage") {
			resp := map[string]any{
				"ok":      true,
				"channel": "D123456",
				"ts":      "1234567890.123456",
				"message": map[string]string{"text": "DM message"},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	config.Set("slack_token", "test-token")

	cmd := newDMCmd()
	cmd.SetArgs([]string{"U123456", "DM message"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestSearchCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "search.messages") {
			t.Errorf("expected path to contain 'search.messages', got %q", r.URL.Path)
		}

		params, _ := url.ParseQuery(r.URL.RawQuery)
		if params.Get("query") != "test query" {
			t.Errorf("expected query 'test query', got %q", params.Get("query"))
		}

		resp := map[string]any{
			"ok": true,
			"messages": map[string]any{
				"total": 1,
				"matches": []map[string]any{
					{
						"channel":   map[string]string{"id": "C123", "name": "general"},
						"user":      "U123",
						"username":  "testuser",
						"text":      "This is a test message",
						"ts":        "1234567890.123456",
						"permalink": "https://slack.com/...",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	config.Set("slack_token", "test-token")

	cmd := newSearchCmd()
	cmd.SetArgs([]string{"test query", "--limit", "10"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestSlackGet_RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{"ok": false})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	params := url.Values{}
	var result map[string]any
	err := slackGet("test-token", "test.method", params, &result)
	if err == nil {
		t.Fatal("expected rate limit error, got nil")
	}
}

func TestSlackPost_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": "invalid_auth",
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	body := map[string]string{"test": "data"}
	var result map[string]any
	err := slackPost("test-token", "test.method", body, &result)
	if err == nil {
		t.Fatal("expected error for 400 response, got nil")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a very long string", 10, "this is..."},
		{"exactly10c", 10, "exactly10c"},
		{"", 10, ""},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestFormatSlackTime(t *testing.T) {
	// Test with a timestamp
	result := formatSlackTime("1234567890.123456")
	if result == "" {
		t.Error("formatSlackTime returned empty string")
	}

	// Test with invalid timestamp
	result = formatSlackTime("invalid")
	if result != "" {
		t.Errorf("expected empty string for invalid timestamp, got %q", result)
	}
}

func TestGetErrorHint(t *testing.T) {
	tests := []struct {
		errCode  string
		wantHint bool
	}{
		{"not_authed", true},
		{"invalid_auth", true},
		{"channel_not_found", true},
		{"unknown_error", false},
	}

	for _, tt := range tests {
		hint := getErrorHint(tt.errCode)
		hasHint := hint != "" && !strings.HasPrefix(hint, "Check Slack API")
		if hasHint != tt.wantHint {
			t.Errorf("getErrorHint(%q) hasHint=%v, want %v", tt.errCode, hasHint, tt.wantHint)
		}
	}
}
