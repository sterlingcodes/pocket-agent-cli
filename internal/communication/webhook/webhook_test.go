package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "webhook" {
		t.Errorf("expected Use 'webhook', got %q", cmd.Use)
	}

	// Verify aliases
	expectedAliases := []string{"wh", "hook"}
	if len(cmd.Aliases) != len(expectedAliases) {
		t.Errorf("expected %d aliases, got %d", len(expectedAliases), len(cmd.Aliases))
	}

	// Check subcommands
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"send [url] [data]", "slack [webhook-url] [message]", "discord [webhook-url] [message]"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestSendCmd_POST_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	}))
	defer srv.Close()

	cmd := newSendCmd()
	cmd.SetArgs([]string{srv.URL, `{"test":"data"}`, "--method", "POST"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestSendCmd_GET_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result":"ok"}`))
	}))
	defer srv.Close()

	cmd := newSendCmd()
	cmd.SetArgs([]string{srv.URL, `ignored`, "--method", "GET"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestSendCmd_CustomHeaders(t *testing.T) {
	headerChecked := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") == "test-value" {
			headerChecked = true
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	cmd := newSendCmd()
	cmd.SetArgs([]string{srv.URL, `{"test":"data"}`, "--header", "X-Custom: test-value"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !headerChecked {
		t.Error("custom header was not set")
	}
}

func TestSendCmd_InvalidJSON(t *testing.T) {
	cmd := newSendCmd()
	cmd.SetArgs([]string{"http://example.com", `{invalid json}`, "--method", "POST"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestSendCmd_InvalidMethod(t *testing.T) {
	cmd := newSendCmd()
	cmd.SetArgs([]string{"http://example.com", `{"test":"data"}`, "--method", "INVALID"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid method, got nil")
	}
}

func TestSendCmd_InvalidHeader(t *testing.T) {
	cmd := newSendCmd()
	cmd.SetArgs([]string{"http://example.com", `{"test":"data"}`, "--header", "InvalidHeaderFormat"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid header format, got nil")
	}
}

func TestSendCmd_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`error response`))
	}))
	defer srv.Close()

	cmd := newSendCmd()
	cmd.SetArgs([]string{srv.URL, `{"test":"data"}`})

	// The command returns success even for 400, but marks status as "error"
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	// Note: webhook send command returns the response with status="error" for 400+ codes
	// but doesn't return an error from Execute
}

func TestSlackCmd_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var msg SlackMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}

		if msg.Text != "test message" {
			t.Errorf("expected text 'test message', got %q", msg.Text)
		}

		if msg.Username != "TestBot" {
			t.Errorf("expected username 'TestBot', got %q", msg.Username)
		}

		if msg.IconEmoji != ":robot:" {
			t.Errorf("expected icon_emoji ':robot:', got %q", msg.IconEmoji)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	cmd := newSlackCmd()
	cmd.SetArgs([]string{srv.URL, "test message", "--username", "TestBot", "--icon-emoji", "robot"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestSlackCmd_EmojiWrapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var msg SlackMessage
		json.NewDecoder(r.Body).Decode(&msg)

		// Should wrap emoji in colons
		if !strings.HasPrefix(msg.IconEmoji, ":") || !strings.HasSuffix(msg.IconEmoji, ":") {
			t.Errorf("expected emoji wrapped in colons, got %q", msg.IconEmoji)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	cmd := newSlackCmd()
	cmd.SetArgs([]string{srv.URL, "test", "--icon-emoji", "tada"})

	cmd.Execute()
}

func TestSlackCmd_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid_webhook"))
	}))
	defer srv.Close()

	cmd := newSlackCmd()
	cmd.SetArgs([]string{srv.URL, "test message"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for bad response, got nil")
	}
}

func TestDiscordCmd_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var msg DiscordMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}

		if msg.Content != "test message" {
			t.Errorf("expected content 'test message', got %q", msg.Content)
		}

		if msg.Username != "TestBot" {
			t.Errorf("expected username 'TestBot', got %q", msg.Username)
		}

		if msg.AvatarURL != "https://example.com/avatar.png" {
			t.Errorf("expected avatar_url, got %q", msg.AvatarURL)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	cmd := newDiscordCmd()
	cmd.SetArgs([]string{srv.URL, "test message", "--username", "TestBot", "--avatar-url", "https://example.com/avatar.png"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestDiscordCmd_MessageTooLong(t *testing.T) {
	longMessage := strings.Repeat("a", 2001)

	cmd := newDiscordCmd()
	cmd.SetArgs([]string{"http://example.com", longMessage})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for message too long, got nil")
	}
}

func TestDiscordCmd_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message":"Invalid webhook"}`))
	}))
	defer srv.Close()

	cmd := newDiscordCmd()
	cmd.SetArgs([]string{srv.URL, "test message"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for bad response, got nil")
	}
}

func TestResponse_Struct(t *testing.T) {
	resp := Response{
		Status:     "success",
		StatusCode: 200,
		URL:        "https://example.com",
		Method:     "POST",
		Response:   "ok",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal Response: %v", err)
	}

	var decoded Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal Response: %v", err)
	}

	if decoded.Status != resp.Status {
		t.Errorf("expected status %q, got %q", resp.Status, decoded.Status)
	}
}

func TestSlackMessage_Struct(t *testing.T) {
	msg := SlackMessage{
		Text:      "Hello",
		Username:  "Bot",
		IconEmoji: ":robot:",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal SlackMessage: %v", err)
	}

	if !strings.Contains(string(data), "Hello") {
		t.Error("marshaled JSON should contain message text")
	}
}

func TestDiscordMessage_Struct(t *testing.T) {
	msg := DiscordMessage{
		Content:   "Hello",
		Username:  "Bot",
		AvatarURL: "https://example.com/avatar.png",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal DiscordMessage: %v", err)
	}

	if !strings.Contains(string(data), "Hello") {
		t.Error("marshaled JSON should contain message content")
	}
}
