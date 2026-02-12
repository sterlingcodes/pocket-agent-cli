package twilio

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
	tmpFile := "/tmp/pocket-twilio-test-config.json"
	os.Setenv("POCKET_CONFIG", tmpFile)
	defer os.Remove(tmpFile)
	os.Exit(m.Run())
}

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "twilio" {
		t.Errorf("expected Use 'twilio', got %q", cmd.Use)
	}

	// Verify aliases
	expectedAliases := []string{"sms"}
	if len(cmd.Aliases) != len(expectedAliases) {
		t.Errorf("expected %d aliases, got %d", len(expectedAliases), len(cmd.Aliases))
	}

	// Check subcommands
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"send [message]", "messages", "message [sid]", "account"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestSendCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		// Check Basic Auth header
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Basic ") {
			t.Errorf("expected Basic auth, got %q", authHeader)
		}

		// Check form data
		r.ParseForm()
		if r.FormValue("From") != "+15551234567" {
			t.Errorf("expected From '+15551234567', got %q", r.FormValue("From"))
		}

		if r.FormValue("To") != "+15559876543" {
			t.Errorf("expected To '+15559876543', got %q", r.FormValue("To"))
		}

		if r.FormValue("Body") != "test message" {
			t.Errorf("expected Body 'test message', got %q", r.FormValue("Body"))
		}

		resp := twilioAPIMessage{
			SID:         "SM123456789",
			From:        "+15551234567",
			To:          "+15559876543",
			Body:        "test message",
			Status:      "queued",
			Direction:   "outbound-api",
			DateCreated: "Wed, 05 Feb 2025 14:30:00 +0000",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	config.Set("twilio_sid", "test-sid")
	config.Set("twilio_token", "test-token")
	config.Set("twilio_phone", "+15551234567")

	cmd := newSendCmd()
	cmd.SetArgs([]string{"test message", "--to", "+15559876543"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestSendCmd_NoConfig(t *testing.T) {
	config.Set("twilio_sid", "")
	config.Set("twilio_token", "")
	config.Set("twilio_phone", "")

	cmd := newSendCmd()
	cmd.SetArgs([]string{"test", "--to", "+15551234567"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing config, got nil")
	}
}

func TestSendCmd_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		resp := twilioAPIError{
			Code:     21211,
			Message:  "The 'To' number is not a valid phone number",
			MoreInfo: "https://www.twilio.com/docs/errors/21211",
			Status:   400,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	config.Set("twilio_sid", "test-sid")
	config.Set("twilio_token", "test-token")
	config.Set("twilio_phone", "+15551234567")

	cmd := newSendCmd()
	cmd.SetArgs([]string{"test", "--to", "invalid"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for bad request, got nil")
	}
}

func TestMessagesCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		if !strings.Contains(r.URL.RawQuery, "PageSize") {
			t.Error("expected PageSize query param")
		}

		resp := twilioAPIMessagesResponse{
			Messages: []twilioAPIMessage{
				{
					SID:         "SM123",
					From:        "+15551234567",
					To:          "+15559876543",
					Body:        "Hello",
					Status:      "delivered",
					Direction:   "outbound-api",
					DateCreated: "Wed, 05 Feb 2025 14:30:00 +0000",
					DateSent:    "Wed, 05 Feb 2025 14:30:01 +0000",
				},
			},
			PageSize: 20,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	config.Set("twilio_sid", "test-sid")
	config.Set("twilio_token", "test-token")
	config.Set("twilio_phone", "+15551234567")

	cmd := newMessagesCmd()
	cmd.SetArgs([]string{"--limit", "20"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestMessagesCmd_WithDirection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.RawQuery
		if !strings.Contains(query, "From=") && !strings.Contains(query, "To=") {
			t.Error("expected direction filter in query")
		}

		resp := twilioAPIMessagesResponse{
			Messages: []twilioAPIMessage{},
			PageSize: 0,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	config.Set("twilio_sid", "test-sid")
	config.Set("twilio_token", "test-token")
	config.Set("twilio_phone", "+15551234567")

	cmd := newMessagesCmd()
	cmd.SetArgs([]string{"--direction", "outbound"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestMessageCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		if !strings.Contains(r.URL.Path, "/Messages/SM123") {
			t.Errorf("expected path to contain '/Messages/SM123', got %q", r.URL.Path)
		}

		resp := twilioAPIMessage{
			SID:         "SM123",
			From:        "+15551234567",
			To:          "+15559876543",
			Body:        "Hello world",
			Status:      "delivered",
			Direction:   "outbound-api",
			DateCreated: "Wed, 05 Feb 2025 14:30:00 +0000",
			DateSent:    "Wed, 05 Feb 2025 14:30:01 +0000",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	config.Set("twilio_sid", "test-sid")
	config.Set("twilio_token", "test-token")
	config.Set("twilio_phone", "+15551234567")

	cmd := newMessageCmd()
	cmd.SetArgs([]string{"SM123"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestAccountCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		resp := twilioAPIAccount{
			SID:          "AC123456789",
			FriendlyName: "Test Account",
			Status:       "active",
			Type:         "Full",
			DateCreated:  "Wed, 01 Jan 2020 00:00:00 +0000",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	config.Set("twilio_sid", "test-sid")
	config.Set("twilio_token", "test-token")
	config.Set("twilio_phone", "+15551234567")

	cmd := newAccountCmd()
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestGetTwilioConfig_Complete(t *testing.T) {
	config.Set("twilio_sid", "AC123")
	config.Set("twilio_token", "token123")
	config.Set("twilio_phone", "+15551234567")

	sid, token, phone, err := getTwilioConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if sid != "AC123" {
		t.Errorf("expected sid 'AC123', got %q", sid)
	}

	if token != "token123" {
		t.Errorf("expected token 'token123', got %q", token)
	}

	if phone != "+15551234567" {
		t.Errorf("expected phone '+15551234567', got %q", phone)
	}
}

func TestGetTwilioConfig_MissingSID(t *testing.T) {
	config.Set("twilio_sid", "")
	config.Set("twilio_token", "token123")
	config.Set("twilio_phone", "+15551234567")

	_, _, _, err := getTwilioConfig()
	if err == nil {
		t.Fatal("expected error for missing SID, got nil")
	}
}

func TestGetTwilioConfig_MissingToken(t *testing.T) {
	config.Set("twilio_sid", "AC123")
	config.Set("twilio_token", "")
	config.Set("twilio_phone", "+15551234567")

	_, _, _, err := getTwilioConfig()
	if err == nil {
		t.Fatal("expected error for missing token, got nil")
	}
}

func TestGetTwilioConfig_MissingPhone(t *testing.T) {
	config.Set("twilio_sid", "AC123")
	config.Set("twilio_token", "token123")
	config.Set("twilio_phone", "")

	_, _, _, err := getTwilioConfig()
	if err == nil {
		t.Fatal("expected error for missing phone, got nil")
	}
}

func TestBasicAuth(t *testing.T) {
	result := basicAuth("username", "password")
	if result == "" {
		t.Error("basicAuth returned empty string")
	}

	// Basic auth of "username:password" should be "dXNlcm5hbWU6cGFzc3dvcmQ="
	expected := "dXNlcm5hbWU6cGFzc3dvcmQ="
	if result != expected {
		t.Errorf("basicAuth() = %q, want %q", result, expected)
	}
}

func TestFormatDate(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Wed, 05 Feb 2025 14:30:00 +0000", "2025-02-05 14:30:00"},
		{"", ""},
	}

	for _, tt := range tests {
		result := formatDate(tt.input)
		if result != tt.expected {
			t.Errorf("formatDate(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestMessage_WithPriceAndError(t *testing.T) {
	// Test that price and error fields are properly handled
	price := "-0.00750"
	errCode := 30003
	errMsg := "Unreachable destination"

	msg := twilioAPIMessage{
		SID:          "SM123",
		Price:        &price,
		PriceUnit:    "USD",
		ErrorCode:    &errCode,
		ErrorMessage: &errMsg,
	}

	if msg.Price == nil || *msg.Price != "-0.00750" {
		t.Error("price not properly stored")
	}

	if msg.ErrorCode == nil || *msg.ErrorCode != 30003 {
		t.Error("error code not properly stored")
	}

	if msg.ErrorMessage == nil || *msg.ErrorMessage != "Unreachable destination" {
		t.Error("error message not properly stored")
	}
}

func TestMessagesCmd_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		resp := twilioAPIError{
			Code:     20003,
			Message:  "Authentication error",
			MoreInfo: "https://www.twilio.com/docs/errors/20003",
			Status:   401,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	config.Set("twilio_sid", "bad-sid")
	config.Set("twilio_token", "bad-token")
	config.Set("twilio_phone", "+15551234567")

	cmd := newMessagesCmd()
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unauthorized, got nil")
	}
}

func TestGetConfigPhone(t *testing.T) {
	config.Set("twilio_phone", "+15551234567")

	phone := getConfigPhone()
	if phone != "+15551234567" {
		t.Errorf("expected phone '+15551234567', got %q", phone)
	}

	config.Set("twilio_phone", "")
	phone = getConfigPhone()
	if phone != "" {
		t.Errorf("expected empty phone, got %q", phone)
	}
}
