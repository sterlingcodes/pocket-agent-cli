package telegram

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
	tmpFile := "/tmp/pocket-telegram-test-config.json"
	os.Setenv("POCKET_CONFIG", tmpFile)
	defer os.Remove(tmpFile)
	os.Exit(m.Run())
}

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "telegram" {
		t.Errorf("expected Use 'telegram', got %q", cmd.Use)
	}

	// Verify aliases
	expectedAliases := []string{"tg"}
	if len(cmd.Aliases) != len(expectedAliases) {
		t.Errorf("expected %d aliases, got %d", len(expectedAliases), len(cmd.Aliases))
	}

	// Check subcommands
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"me", "updates", "send [message]", "chats", "forward [from-chat-id] [message-id] [to-chat-id]"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestMeCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if !strings.Contains(r.URL.Path, "/getMe") {
			t.Errorf("expected path to contain '/getMe', got %q", r.URL.Path)
		}

		resp := apiResponse{
			OK: true,
			Result: json.RawMessage(`{
				"id": 123456789,
				"is_bot": true,
				"first_name": "TestBot",
				"username": "testbot",
				"can_join_groups": true,
				"can_read_all_group_messages": false,
				"supports_inline_queries": true
			}`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL + "/"
	defer func() { baseURL = oldURL }()

	config.Set("telegram_token", "test-token")

	cmd := newMeCmd()
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestMeCmd_NoToken(t *testing.T) {
	config.Set("telegram_token", "")

	cmd := newMeCmd()
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing token, got nil")
	}
}

func TestUpdatesCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/getUpdates") {
			t.Errorf("expected path to contain '/getUpdates', got %q", r.URL.Path)
		}

		resp := apiResponse{
			OK: true,
			Result: json.RawMessage(`[
				{
					"update_id": 123456,
					"message": {
						"message_id": 1,
						"date": 1234567890,
						"chat": {"id": 111, "type": "private", "first_name": "Test"},
						"from": {"id": 111, "is_bot": false, "first_name": "Test"},
						"text": "Hello"
					}
				}
			]`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL + "/"
	defer func() { baseURL = oldURL }()

	config.Set("telegram_token", "test-token")

	cmd := newUpdatesCmd()
	cmd.SetArgs([]string{"--limit", "50"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestUpdatesCmd_WithOffset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		if body["offset"] != float64(100) {
			t.Errorf("expected offset 100, got %v", body["offset"])
		}

		resp := apiResponse{
			OK:     true,
			Result: json.RawMessage(`[]`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL + "/"
	defer func() { baseURL = oldURL }()

	config.Set("telegram_token", "test-token")

	cmd := newUpdatesCmd()
	cmd.SetArgs([]string{"--offset", "100"})

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

		if !strings.Contains(r.URL.Path, "/sendMessage") {
			t.Errorf("expected path to contain '/sendMessage', got %q", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		if body["chat_id"] != "123456" {
			t.Errorf("expected chat_id '123456', got %v", body["chat_id"])
		}

		if body["text"] != "test message" {
			t.Errorf("expected text 'test message', got %v", body["text"])
		}

		resp := apiResponse{
			OK: true,
			Result: json.RawMessage(`{
				"message_id": 999,
				"date": 1234567890,
				"chat": {"id": 123456, "type": "private"},
				"text": "test message"
			}`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL + "/"
	defer func() { baseURL = oldURL }()

	config.Set("telegram_token", "test-token")

	cmd := newSendCmd()
	cmd.SetArgs([]string{"test message", "--chat", "123456"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestSendCmd_WithParseMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		if body["parse_mode"] != "Markdown" {
			t.Errorf("expected parse_mode 'Markdown', got %v", body["parse_mode"])
		}

		resp := apiResponse{
			OK: true,
			Result: json.RawMessage(`{
				"message_id": 999,
				"date": 1234567890,
				"chat": {"id": 123456, "type": "private"},
				"text": "*bold*"
			}`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL + "/"
	defer func() { baseURL = oldURL }()

	config.Set("telegram_token", "test-token")

	cmd := newSendCmd()
	cmd.SetArgs([]string{"*bold*", "--chat", "123456", "--parse", "Markdown"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestSendCmd_Silent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		if body["disable_notification"] != true {
			t.Errorf("expected disable_notification true, got %v", body["disable_notification"])
		}

		resp := apiResponse{
			OK: true,
			Result: json.RawMessage(`{
				"message_id": 999,
				"date": 1234567890,
				"chat": {"id": 123456, "type": "private"},
				"text": "silent"
			}`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL + "/"
	defer func() { baseURL = oldURL }()

	config.Set("telegram_token", "test-token")

	cmd := newSendCmd()
	cmd.SetArgs([]string{"silent", "--chat", "123456", "--silent"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestSendCmd_NoChat(t *testing.T) {
	config.Set("telegram_token", "test-token")

	cmd := newSendCmd()
	cmd.SetArgs([]string{"test message"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing chat, got nil")
	}
}

func TestChatsCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := apiResponse{
			OK: true,
			Result: json.RawMessage(`[
				{
					"update_id": 1,
					"message": {
						"message_id": 1,
						"date": 1234567890,
						"chat": {"id": 111, "type": "private", "first_name": "User1"},
						"text": "msg1"
					}
				},
				{
					"update_id": 2,
					"message": {
						"message_id": 2,
						"date": 1234567891,
						"chat": {"id": 222, "type": "group", "title": "Group1"},
						"text": "msg2"
					}
				},
				{
					"update_id": 3,
					"message": {
						"message_id": 3,
						"date": 1234567892,
						"chat": {"id": 111, "type": "private", "first_name": "User1"},
						"text": "msg3"
					}
				}
			]`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL + "/"
	defer func() { baseURL = oldURL }()

	config.Set("telegram_token", "test-token")

	cmd := newChatsCmd()
	cmd.SetArgs([]string{"--limit", "10"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestForwardCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/forwardMessage") {
			t.Errorf("expected path to contain '/forwardMessage', got %q", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		if body["from_chat_id"] != "111" {
			t.Errorf("expected from_chat_id '111', got %v", body["from_chat_id"])
		}

		if body["message_id"] != float64(999) {
			t.Errorf("expected message_id 999, got %v", body["message_id"])
		}

		if body["chat_id"] != "222" {
			t.Errorf("expected chat_id '222', got %v", body["chat_id"])
		}

		resp := apiResponse{
			OK: true,
			Result: json.RawMessage(`{
				"message_id": 1000,
				"date": 1234567890,
				"chat": {"id": 222, "type": "private"}
			}`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL + "/"
	defer func() { baseURL = oldURL }()

	config.Set("telegram_token", "test-token")

	cmd := newForwardCmd()
	cmd.SetArgs([]string{"111", "999", "222"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestForwardCmd_InvalidMessageID(t *testing.T) {
	config.Set("telegram_token", "test-token")

	cmd := newForwardCmd()
	cmd.SetArgs([]string{"111", "invalid", "222"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid message ID, got nil")
	}
}

func TestCallAPI_RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := apiResponse{
			OK:          false,
			ErrorCode:   429,
			Description: "Too Many Requests: retry after 30",
			Parameters:  &responseParams{RetryAfter: 30},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL + "/"
	defer func() { baseURL = oldURL }()

	_, err := callAPI("test-token", "getMe", nil)
	if err == nil {
		t.Fatal("expected rate limit error, got nil")
	}
}

func TestCallAPI_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := apiResponse{
			OK:          false,
			ErrorCode:   401,
			Description: "Unauthorized",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL + "/"
	defer func() { baseURL = oldURL }()

	_, err := callAPI("bad-token", "getMe", nil)
	if err == nil {
		t.Fatal("expected unauthorized error, got nil")
	}
}

func TestCallAPI_ChatNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := apiResponse{
			OK:          false,
			ErrorCode:   400,
			Description: "Bad Request: chat not found",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL + "/"
	defer func() { baseURL = oldURL }()

	_, err := callAPI("test-token", "sendMessage", map[string]any{"chat_id": "999"})
	if err == nil {
		t.Fatal("expected chat not found error, got nil")
	}
}

func TestConvertMessage(t *testing.T) {
	apiMsg := &apiMessage{
		MessageID: 123,
		Date:      1234567890,
		Chat: apiChat{
			ID:        111,
			Type:      "private",
			FirstName: "Test",
		},
		From: &apiUser{
			ID:        222,
			IsBot:     false,
			FirstName: "User",
		},
		Text: "Hello",
	}

	msg := convertMessage(apiMsg)
	if msg == nil {
		t.Fatal("convertMessage returned nil")
	}

	if msg.MessageID != 123 {
		t.Errorf("expected MessageID 123, got %d", msg.MessageID)
	}

	if msg.Chat.ID != 111 {
		t.Errorf("expected Chat.ID 111, got %d", msg.Chat.ID)
	}

	if msg.From == nil {
		t.Fatal("expected From to be non-nil")
	}

	if msg.From.ID != 222 {
		t.Errorf("expected From.ID 222, got %d", msg.From.ID)
	}

	// Test with nil message
	if convertMessage(nil) != nil {
		t.Error("expected nil for nil input")
	}
}

func TestFormatUnixTime(t *testing.T) {
	// Test with valid timestamp
	result := formatUnixTime(1234567890)
	if result == "" {
		t.Error("formatUnixTime returned empty string")
	}

	if !strings.Contains(result, "2009") {
		t.Errorf("expected year 2009 in result, got %q", result)
	}

	// Test with zero
	result = formatUnixTime(0)
	if !strings.Contains(result, "1970") {
		t.Errorf("expected year 1970 for epoch 0, got %q", result)
	}
}
