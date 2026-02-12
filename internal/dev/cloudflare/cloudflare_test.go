package cloudflare

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "cloudflare" {
		t.Errorf("expected Use 'cloudflare', got %q", cmd.Use)
	}
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Name()] = true
	}
	for _, name := range []string{"zones", "zone", "dns", "purge", "analytics"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestGetString(t *testing.T) {
	tests := []struct {
		m        map[string]any
		key      string
		expected string
	}{
		{map[string]any{"name": "test"}, "name", "test"},
		{map[string]any{}, "name", ""},
		{map[string]any{"name": 123}, "name", ""},
	}
	for _, tt := range tests {
		result := getString(tt.m, tt.key)
		if result != tt.expected {
			t.Errorf("getString(%v, %q) = %q, want %q", tt.m, tt.key, result, tt.expected)
		}
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		m        map[string]any
		key      string
		expected int
	}{
		{map[string]any{"count": float64(42)}, "count", 42},
		{map[string]any{}, "count", 0},
		{map[string]any{"count": "42"}, "count", 0},
	}
	for _, tt := range tests {
		result := getInt(tt.m, tt.key)
		if result != tt.expected {
			t.Errorf("getInt(%v, %q) = %d, want %d", tt.m, tt.key, result, tt.expected)
		}
	}
}

func TestGetInt64(t *testing.T) {
	tests := []struct {
		m        map[string]any
		key      string
		expected int64
	}{
		{map[string]any{"size": float64(1024)}, "size", 1024},
		{map[string]any{}, "size", 0},
	}
	for _, tt := range tests {
		result := getInt64(tt.m, tt.key)
		if result != tt.expected {
			t.Errorf("getInt64(%v, %q) = %d, want %d", tt.m, tt.key, result, tt.expected)
		}
	}
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		m        map[string]any
		key      string
		expected bool
	}{
		{map[string]any{"active": true}, "active", true},
		{map[string]any{"active": false}, "active", false},
		{map[string]any{}, "active", false},
	}
	for _, tt := range tests {
		result := getBool(tt.m, tt.key)
		if result != tt.expected {
			t.Errorf("getBool(%v, %q) = %v, want %v", tt.m, tt.key, result, tt.expected)
		}
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2024-01-15T10:30:00Z", "2024-01-15 10:30:00"},
		{"invalid", "invalid"},
	}
	for _, tt := range tests {
		result := parseTime(tt.input)
		if result != tt.expected {
			t.Errorf("parseTime(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestCfGetSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("expected Authorization header")
		}
		resp := cfResponse{
			Success: true,
			Result:  json.RawMessage(`{"id":"abc123","name":"example.com"}`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	var result cfResponse
	err := cfGet("test-token", srv.URL+"/zones", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.Success {
		t.Error("expected success to be true")
	}
}

func TestCfGetError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"400", 400},
		{"401", 401},
		{"500", 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				resp := cfResponse{
					Success: false,
					Errors: []cfError{
						{Code: tt.statusCode, Message: "Error"},
					},
				}
				json.NewEncoder(w).Encode(resp)
			}))
			defer srv.Close()

			var result cfResponse
			err := cfGet("test-token", srv.URL+"/test", &result)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestToZone(t *testing.T) {
	zoneData := map[string]any{
		"id":     "abc123",
		"name":   "example.com",
		"status": "active",
		"paused": false,
		"type":   "full",
		"name_servers": []any{
			"ns1.cloudflare.com",
			"ns2.cloudflare.com",
		},
		"plan": map[string]any{
			"name": "Free",
		},
		"created_on":  time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		"modified_on": time.Now().Format(time.RFC3339),
	}

	zone := toZone(zoneData)
	if zone.ID != "abc123" {
		t.Errorf("expected ID 'abc123', got %q", zone.ID)
	}
	if zone.Name != "example.com" {
		t.Errorf("expected name 'example.com', got %q", zone.Name)
	}
	if zone.Status != "active" {
		t.Errorf("expected status 'active', got %q", zone.Status)
	}
	if len(zone.NameServers) != 2 {
		t.Errorf("expected 2 nameservers, got %d", len(zone.NameServers))
	}
	if zone.Plan != "Free" {
		t.Errorf("expected plan 'Free', got %q", zone.Plan)
	}
}

func TestToDNSRecord(t *testing.T) {
	recordData := map[string]any{
		"id":       "rec123",
		"type":     "A",
		"name":     "example.com",
		"content":  "1.2.3.4",
		"proxied":  true,
		"ttl":      float64(1),
		"priority": float64(10),
		"locked":   false,
	}

	record := toDNSRecord(recordData)
	if record.ID != "rec123" {
		t.Errorf("expected ID 'rec123', got %q", record.ID)
	}
	if record.Type != "A" {
		t.Errorf("expected type 'A', got %q", record.Type)
	}
	if record.Content != "1.2.3.4" {
		t.Errorf("expected content '1.2.3.4', got %q", record.Content)
	}
	if !record.Proxied {
		t.Error("expected proxied to be true")
	}
}
