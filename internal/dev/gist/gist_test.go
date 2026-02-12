package gist

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "gist" {
		t.Errorf("expected Use 'gist', got %q", cmd.Use)
	}
	// Check that we have 3 subcommands
	if len(cmd.Commands()) != 3 {
		t.Errorf("expected 3 subcommands, got %d", len(cmd.Commands()))
	}
	// Verify subcommands exist by name (Use field may include args)
	subNames := make(map[string]bool)
	for _, s := range cmd.Commands() {
		subNames[s.Name()] = true
	}
	for _, name := range []string{"list", "get", "create"} {
		if !subNames[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestGetString(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]any
		key      string
		expected string
	}{
		{"existing", map[string]any{"id": "abc123"}, "id", "abc123"},
		{"missing", map[string]any{}, "id", ""},
		{"wrong type", map[string]any{"id": 123}, "id", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getString(tt.m, tt.key)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]any
		key      string
		expected int
	}{
		{"existing", map[string]any{"size": float64(1024)}, "size", 1024},
		{"missing", map[string]any{}, "size", 0},
		{"wrong type", map[string]any{"size": "1024"}, "size", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInt(tt.m, tt.key)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]any
		key      string
		expected bool
	}{
		{"true", map[string]any{"public": true}, "public", true},
		{"false", map[string]any{"public": false}, "public", false},
		{"missing", map[string]any{}, "public", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBool(tt.m, tt.key)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGhGetSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("expected Authorization header")
		}
		gists := []map[string]any{
			{
				"id":          "abc123",
				"description": "Test gist",
				"public":      true,
				"created_at":  "2024-01-15T10:30:00Z",
				"updated_at":  "2024-01-15T11:30:00Z",
				"html_url":    "https://gist.github.com/abc123",
				"files": map[string]any{
					"test.txt": map[string]any{
						"filename": "test.txt",
						"type":     "text/plain",
						"size":     float64(100),
					},
				},
			},
		}
		json.NewEncoder(w).Encode(gists)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	var result []map[string]any
	err := ghGet("test-token", srv.URL+"/gists", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 gist, got %d", len(result))
	}
}

func TestGhGetError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"401", 401},
		{"404", 404},
		{"500", 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(map[string]any{"message": "Error"})
			}))
			defer srv.Close()

			var result []map[string]any
			err := ghGet("test-token", srv.URL+"/test", &result)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestToSummary(t *testing.T) {
	gistData := map[string]any{
		"id":          "abc123",
		"description": "My test gist",
		"public":      true,
		"created_at":  "2024-01-15T10:30:00Z",
		"updated_at":  "2024-01-15T11:30:00Z",
		"html_url":    "https://gist.github.com/abc123",
		"files": map[string]any{
			"test.txt":  map[string]any{},
			"script.sh": map[string]any{},
		},
	}

	summary := toSummary(gistData)
	if summary.ID != "abc123" {
		t.Errorf("expected ID 'abc123', got %q", summary.ID)
	}
	if summary.Description != "My test gist" {
		t.Errorf("expected description 'My test gist', got %q", summary.Description)
	}
	if !summary.Public {
		t.Error("expected public to be true")
	}
	if len(summary.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(summary.Files))
	}
}

func TestToDetail(t *testing.T) {
	gistData := map[string]any{
		"id":          "abc123",
		"description": "Detailed gist",
		"public":      false,
		"created_at":  "2024-01-15T10:30:00Z",
		"html_url":    "https://gist.github.com/abc123",
		"files": map[string]any{
			"test.txt": map[string]any{
				"filename": "test.txt",
				"language": "Text",
				"size":     float64(1024),
				"content":  "Hello World",
			},
		},
	}

	detail := toDetail(gistData)
	if detail.ID != "abc123" {
		t.Errorf("expected ID 'abc123', got %q", detail.ID)
	}
	if len(detail.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(detail.Files))
	}
	if detail.Files[0].Filename != "test.txt" {
		t.Errorf("expected filename 'test.txt', got %q", detail.Files[0].Filename)
	}
	if detail.Files[0].Content != "Hello World" {
		t.Errorf("expected content 'Hello World', got %q", detail.Files[0].Content)
	}
}
