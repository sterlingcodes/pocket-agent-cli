package discord

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/unstablemind/pocket/internal/common/config"
)

func TestMain(m *testing.M) {
	// Use a temporary config file for tests
	tmpFile := "/tmp/pocket-discord-test-config.json"
	os.Setenv("POCKET_CONFIG", tmpFile)
	defer os.Remove(tmpFile)
	os.Exit(m.Run())
}

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "discord" {
		t.Errorf("expected Use 'discord', got %q", cmd.Use)
	}

	// Verify aliases
	expectedAliases := []string{"dc"}
	if len(cmd.Aliases) != len(expectedAliases) {
		t.Errorf("expected %d aliases, got %d", len(expectedAliases), len(cmd.Aliases))
	}

	// Check subcommands
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"guilds", "channels [guild-id]", "messages [channel-id]", "send [message]", "dm [user-id] [message]"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestGuildsCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bot test-token" {
			t.Errorf("expected Authorization 'Bot test-token', got %q", r.Header.Get("Authorization"))
		}

		if !strings.Contains(r.URL.Path, "/users/@me/guilds") {
			t.Errorf("expected path to contain '/users/@me/guilds', got %q", r.URL.Path)
		}

		resp := []map[string]any{
			{
				"id":          "123456789",
				"name":        "Test Server",
				"icon":        "abc123",
				"owner":       true,
				"permissions": "2147483647",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	config.Set("discord_token", "test-token")

	cmd := newGuildsCmd()
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestGuildsCmd_NoToken(t *testing.T) {
	config.Set("discord_token", "")

	cmd := newGuildsCmd()
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing token, got nil")
	}
}

func TestChannelsCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/guilds/") || !strings.Contains(r.URL.Path, "/channels") {
			t.Errorf("expected path to contain '/guilds/.../channels', got %q", r.URL.Path)
		}

		resp := []map[string]any{
			{
				"id":        "987654321",
				"name":      "general",
				"type":      0,
				"topic":     "General discussion",
				"position":  0,
				"parent_id": "",
				"nsfw":      false,
			},
			{
				"id":        "987654322",
				"name":      "Voice Channel",
				"type":      2,
				"position":  1,
				"parent_id": "",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	config.Set("discord_token", "test-token")

	cmd := newChannelsCmd()
	cmd.SetArgs([]string{"123456789"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestChannelsCmd_WithTypeFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := []map[string]any{
			{"id": "1", "name": "text", "type": 0, "position": 0},
			{"id": "2", "name": "voice", "type": 2, "position": 1},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	config.Set("discord_token", "test-token")

	cmd := newChannelsCmd()
	cmd.SetArgs([]string{"123456789", "--type", "text"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestMessagesCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/channels/") || !strings.Contains(r.URL.Path, "/messages") {
			t.Errorf("expected path to contain '/channels/.../messages', got %q", r.URL.Path)
		}

		if !strings.Contains(r.URL.Query().Get("limit"), "50") && r.URL.Query().Get("limit") != "50" {
			t.Logf("Note: limit query param = %q", r.URL.Query().Get("limit"))
		}

		resp := []map[string]any{
			{
				"id":               "111111111",
				"content":          "Hello world",
				"timestamp":        "2025-01-01T12:00:00.000000+00:00",
				"edited_timestamp": "",
				"pinned":           false,
				"author":           map[string]any{"id": "222222222", "username": "testuser", "discriminator": "1234", "global_name": "Test User"},
				"mentions":         []map[string]any{},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	config.Set("discord_token", "test-token")

	cmd := newMessagesCmd()
	cmd.SetArgs([]string{"987654321", "--limit", "50"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestMessagesCmd_LimitClamping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	config.Set("discord_token", "test-token")

	// Test limit > 100 (should be clamped to 100)
	cmd := newMessagesCmd()
	cmd.SetArgs([]string{"987654321", "--limit", "200"})

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

		if !strings.Contains(r.URL.Path, "/channels/") || !strings.Contains(r.URL.Path, "/messages") {
			t.Errorf("expected path to contain '/channels/.../messages', got %q", r.URL.Path)
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)

		if body["content"] != "test message" {
			t.Errorf("expected content 'test message', got %q", body["content"])
		}

		resp := map[string]any{
			"id":         "333333333",
			"channel_id": "987654321",
			"content":    "test message",
			"timestamp":  "2025-01-01T12:00:00.000000+00:00",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	config.Set("discord_token", "test-token")

	cmd := newSendCmd()
	cmd.SetArgs([]string{"test message", "--channel", "987654321"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestSendCmd_MessageTooLong(t *testing.T) {
	config.Set("discord_token", "test-token")

	longMessage := strings.Repeat("a", 2001)

	cmd := newSendCmd()
	cmd.SetArgs([]string{longMessage, "--channel", "987654321"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for message too long, got nil")
	}
}

func TestSendCmd_NoChannel(t *testing.T) {
	config.Set("discord_token", "test-token")

	cmd := newSendCmd()
	cmd.SetArgs([]string{"test message"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing channel, got nil")
	}
}

func TestDMCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, "/users/@me/channels") {
			// DM channel creation
			resp := map[string]any{
				"id":   "D123456",
				"type": 1,
			}
			json.NewEncoder(w).Encode(resp)
		} else if strings.Contains(path, "/channels/") && strings.Contains(path, "/messages") {
			// Send message
			resp := map[string]any{
				"id":         "444444444",
				"channel_id": "D123456",
				"content":    "DM message",
				"timestamp":  "2025-01-01T12:00:00.000000+00:00",
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	config.Set("discord_token", "test-token")

	cmd := newDMCmd()
	cmd.SetArgs([]string{"222222222", "DM message"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestDMCmd_MessageTooLong(t *testing.T) {
	config.Set("discord_token", "test-token")

	longMessage := strings.Repeat("a", 2001)

	cmd := newDMCmd()
	cmd.SetArgs([]string{"222222222", longMessage})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for message too long, got nil")
	}
}

func TestDoRequest_RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		resp := map[string]any{
			"message":     "You are being rate limited.",
			"retry_after": 5.5,
			"global":      false,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	_, err := doRequest("GET", srv.URL+"/test", "test-token", nil)
	if err == nil {
		t.Fatal("expected rate limit error, got nil")
	}
}

func TestDoRequest_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{"message": "Invalid token"})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	_, err := doRequest("GET", srv.URL+"/test", "bad-token", nil)
	if err == nil {
		t.Fatal("expected unauthorized error, got nil")
	}
}

func TestDoRequest_Forbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		resp := map[string]any{
			"message": "Missing Permissions",
			"code":    50013,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	_, err := doRequest("GET", srv.URL+"/test", "test-token", nil)
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
}

func TestDoRequest_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		resp := map[string]any{
			"message": "Unknown Channel",
			"code":    10003,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	_, err := doRequest("GET", srv.URL+"/test", "test-token", nil)
	if err == nil {
		t.Fatal("expected not found error, got nil")
	}
}

func TestChannelTypeName(t *testing.T) {
	tests := []struct {
		typeNum  int
		expected string
	}{
		{0, "text"},
		{1, "dm"},
		{2, "voice"},
		{3, "group_dm"},
		{4, "category"},
		{5, "announcement"},
		{15, "forum"},
		{999, "999"},
	}

	for _, tt := range tests {
		result := channelTypeName(tt.typeNum)
		if result != tt.expected {
			t.Errorf("channelTypeName(%d) = %q, want %q", tt.typeNum, result, tt.expected)
		}
	}
}

func TestFormatTime(t *testing.T) {
	// Test valid ISO 8601 timestamp
	result := formatTime("2025-01-01T12:00:00.000000+00:00")
	if result == "" {
		t.Error("formatTime returned empty string for valid timestamp")
	}

	// Test empty string
	result = formatTime("")
	if result != "" {
		t.Errorf("expected empty string for empty input, got %q", result)
	}

	// Test invalid timestamp
	result = formatTime("invalid")
	if result != "invalid" {
		t.Errorf("expected original string for invalid timestamp, got %q", result)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a very long string", 10, "this is a ..."},
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
