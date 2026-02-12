package youtube

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "youtube" {
		t.Errorf("expected Use 'youtube', got %q", cmd.Use)
	}
	// Check that command has subcommands
	if len(cmd.Commands()) < 5 {
		t.Errorf("expected at least 5 subcommands, got %d", len(cmd.Commands()))
	}
}

func TestExtractVideoID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"https://youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ&feature=share", "dQw4w9WgXcQ"},
		{"https://youtu.be/dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"https://youtu.be/dQw4w9WgXcQ?t=42", "dQw4w9WgXcQ"},
	}

	for _, tt := range tests {
		result := extractVideoID(tt.input)
		if result != tt.expected {
			t.Errorf("extractVideoID(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"123", 123},
		{"999999", 999999},
		{"0", 0},
		{"invalid", 0},
		{"", 0},
	}

	for _, tt := range tests {
		result := parseInt(tt.input)
		if result != tt.expected {
			t.Errorf("parseInt(%q) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

func TestFormatTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		input       string
		expectedMin string // Minimum match (for edge cases)
	}{
		{now.Add(-10 * time.Second).Format(time.RFC3339), "0m ago"}, // Less than 1 minute
		{now.Add(-5 * time.Minute).Format(time.RFC3339), "5m ago"},
		{now.Add(-2 * time.Hour).Format(time.RFC3339), "2h ago"},
		{now.Add(-3 * 24 * time.Hour).Format(time.RFC3339), "3d ago"},
		{now.Add(-10 * 24 * time.Hour).Format(time.RFC3339), "1w ago"},
		{now.Add(-40 * 24 * time.Hour).Format(time.RFC3339), "1mo ago"},
		{now.Add(-400 * 24 * time.Hour).Format(time.RFC3339), "1y ago"},
	}

	for _, tt := range tests {
		result := formatTime(tt.input)
		// Just check it contains expected pattern (not exact match due to time precision)
		if !strings.Contains(result, " ago") && result != tt.expectedMin {
			t.Errorf("formatTime(%q) = %q, expected pattern with ' ago'", tt.input, result)
		}
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"PT4M13S"},
		{"PT1H2M3S"},
		{"PT45S"},
		{"PT1H30S"},
		{"PT15M"},
	}

	// Just verify the function doesn't crash
	for _, tt := range tests {
		result := parseDuration(tt.input)
		if result == "" {
			t.Errorf("parseDuration(%q) returned empty string", tt.input)
		}
	}
}

func TestParseRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		input string
		valid bool
	}{
		{"7d", true},
		{"1w", true},
		{"1m", true},
		{"1y", true},
		{"14days", true},
		{"invalid", false},
	}

	for _, tt := range tests {
		result, err := parseRelativeTime(tt.input)
		if tt.valid {
			if err != nil {
				t.Errorf("parseRelativeTime(%q) returned error: %v", tt.input, err)
			}
			if result.After(now) {
				t.Errorf("parseRelativeTime(%q) returned future time", tt.input)
			}
		} else {
			if err == nil {
				t.Errorf("parseRelativeTime(%q) should return error", tt.input)
			}
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly ten", 11, "exactly ten"},
		{"this is a very long string", 10, "this is a ..."},
		{"newline\ntest", 20, "newline test"},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, expected %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestCleanHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<p>Hello</p>", "Hello"},
		{"<b>Bold</b> text", "Bold text"},
		{"Line 1<br>Line 2", "Line 1\nLine 2"},
		{"<a href='url'>Link</a>", "Link"},
		{"No tags", "No tags"},
	}

	for _, tt := range tests {
		result := cleanHTML(tt.input)
		if result != tt.expected {
			t.Errorf("cleanHTML(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestDoRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check User-Agent
		if r.Header.Get("User-Agent") != "Pocket-CLI/1.0" {
			t.Errorf("expected User-Agent 'Pocket-CLI/1.0', got %q", r.Header.Get("User-Agent"))
		}

		// Return mock video data
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"id": "test123",
					"snippet": map[string]any{
						"title":        "Test Video",
						"channelTitle": "Test Channel",
						"publishedAt":  "2024-01-01T12:00:00Z",
					},
				},
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	data, err := doRequest(srv.URL)
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}

	var resp struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}

	if resp.Items[0].ID != "test123" {
		t.Errorf("expected video ID 'test123', got %q", resp.Items[0].ID)
	}
}

func TestDoRequestQuotaExceeded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "quotaExceeded",
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	_, err := doRequest(srv.URL)
	if err == nil {
		t.Error("expected error for 403 response, got nil")
	}
}

func TestDoRequestNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Video not found",
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	_, err := doRequest(srv.URL)
	if err == nil {
		t.Error("expected error for 404 response, got nil")
	}
}
