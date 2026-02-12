package sentry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "sentry" {
		t.Errorf("expected Use 'sentry', got %q", cmd.Use)
	}
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Name()] = true
	}
	for _, name := range []string{"projects", "issues", "issue", "events"} {
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
		{map[string]any{"id": "abc123"}, "id", "abc123"},
		{map[string]any{}, "id", ""},
		{map[string]any{"id": 123}, "id", ""},
	}
	for _, tt := range tests {
		result := getString(tt.m, tt.key)
		if result != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, result)
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
			t.Errorf("expected %d, got %d", tt.expected, result)
		}
	}
}

func TestSentryGetSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("expected Authorization header")
		}
		projects := []map[string]any{
			{
				"id":          "123456",
				"name":        "my-project",
				"slug":        "my-project",
				"platform":    "javascript",
				"dateCreated": "2024-01-15T10:30:00.000Z",
				"status":      "active",
			},
		}
		json.NewEncoder(w).Encode(projects)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	var result []map[string]any
	err := sentryGet("test-token", srv.URL+"/projects/", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 project, got %d", len(result))
	}
}

func TestSentryGetError(t *testing.T) {
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
				json.NewEncoder(w).Encode(map[string]any{"detail": "Error"})
			}))
			defer srv.Close()

			var result map[string]any
			err := sentryGet("test-token", srv.URL+"/test", &result)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestToIssue(t *testing.T) {
	issueData := map[string]any{
		"id":        "123456",
		"title":     "TypeError: Cannot read property",
		"culprit":   "app.js in handleClick",
		"level":     "error",
		"status":    "unresolved",
		"count":     "42",
		"userCount": float64(10),
		"firstSeen": "2024-01-15T10:00:00.000Z",
		"lastSeen":  "2024-01-15T12:00:00.000Z",
		"permalink": "https://sentry.io/issues/123456",
	}

	issue := toIssue(issueData)
	if issue.ID != "123456" {
		t.Errorf("expected ID '123456', got %q", issue.ID)
	}
	if issue.Title != "TypeError: Cannot read property" {
		t.Errorf("expected title, got %q", issue.Title)
	}
	if issue.Level != "error" {
		t.Errorf("expected level 'error', got %q", issue.Level)
	}
	if issue.UserCount != 10 {
		t.Errorf("expected userCount 10, got %d", issue.UserCount)
	}
}

func TestSentryProjectsResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		projects := []map[string]any{
			{
				"id":          "123",
				"name":        "frontend",
				"slug":        "frontend",
				"platform":    "javascript-react",
				"dateCreated": "2024-01-01T00:00:00.000Z",
				"status":      "active",
				"organization": map[string]any{
					"slug": "my-org",
				},
			},
			{
				"id":          "456",
				"name":        "backend",
				"slug":        "backend",
				"platform":    "python",
				"dateCreated": "2024-01-02T00:00:00.000Z",
				"status":      "active",
				"organization": map[string]any{
					"slug": "my-org",
				},
			},
		}
		json.NewEncoder(w).Encode(projects)
	}))
	defer srv.Close()

	var result []map[string]any
	err := sentryGet("test-token", srv.URL+"/projects/", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 projects, got %d", len(result))
	}
	if result[0]["name"] != "frontend" {
		t.Errorf("expected first project name 'frontend', got %v", result[0]["name"])
	}
}

func TestSentryIssuesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		issues := []map[string]any{
			{
				"id":        "issue123",
				"title":     "Database connection failed",
				"culprit":   "db.connect",
				"level":     "error",
				"status":    "unresolved",
				"count":     "15",
				"userCount": float64(5),
				"firstSeen": "2024-01-15T08:00:00.000Z",
				"lastSeen":  "2024-01-15T14:30:00.000Z",
				"permalink": "https://sentry.io/issues/issue123",
			},
		}
		json.NewEncoder(w).Encode(issues)
	}))
	defer srv.Close()

	var result []map[string]any
	err := sentryGet("test-token", srv.URL+"/issues/", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 issue, got %d", len(result))
	}

	issue := toIssue(result[0])
	if issue.Title != "Database connection failed" {
		t.Errorf("expected title 'Database connection failed', got %q", issue.Title)
	}
}
