package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "github" {
		t.Errorf("expected Use 'github', got %q", cmd.Use)
	}
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	// Check that we have the expected number of subcommands
	if len(subs) != 8 {
		t.Errorf("expected 8 subcommands, got %d: %v", len(subs), subs)
	}
	// Check key subcommands exist
	for _, name := range []string{"repos", "issues", "prs", "notifications"} {
		if !subs[name] {
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
		{"existing key", map[string]any{"name": "test"}, "name", "test"},
		{"missing key", map[string]any{}, "name", ""},
		{"wrong type", map[string]any{"name": 123}, "name", ""},
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
		{"existing key", map[string]any{"count": float64(42)}, "count", 42},
		{"missing key", map[string]any{}, "count", 0},
		{"wrong type", map[string]any{"count": "42"}, "count", 0},
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
		{"true", map[string]any{"private": true}, "private", true},
		{"false", map[string]any{"private": false}, "private", false},
		{"missing key", map[string]any{}, "private", false},
		{"wrong type", map[string]any{"private": "true"}, "private", false},
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

func TestTimeAgo(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		t        time.Time
		expected string
	}{
		{"now", now, "now"},
		{"5 minutes ago", now.Add(-5 * time.Minute), "5m"},
		{"2 hours ago", now.Add(-2 * time.Hour), "2h"},
		{"3 days ago", now.Add(-3 * 24 * time.Hour), "3d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := timeAgo(tt.t)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		maxLen   int
		expected string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello..."},
		{"with whitespace", "  hello  ", 5, "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.s, tt.maxLen)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGhGetSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization header with Bearer token")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"name": "test-repo"})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	var result map[string]any
	err := ghGet("test-token", srv.URL+"/test", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result["name"] != "test-repo" {
		t.Errorf("expected name 'test-repo', got %v", result["name"])
	}
}

func TestGhGetError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   map[string]any
	}{
		{"401 unauthorized", 401, map[string]any{"message": "Bad credentials"}},
		{"404 not found", 404, map[string]any{"message": "Not Found"}},
		{"500 server error", 500, map[string]any{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer srv.Close()

			var result map[string]any
			err := ghGet("test-token", srv.URL+"/test", &result)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestToRepo(t *testing.T) {
	repoData := map[string]any{
		"name":             "test-repo",
		"full_name":        "user/test-repo",
		"description":      "A test repository",
		"private":          false,
		"stargazers_count": float64(42),
		"forks_count":      float64(10),
		"language":         "Go",
		"html_url":         "https://github.com/user/test-repo",
		"updated_at":       time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
	}

	repo := toRepo(repoData)
	if repo.Name != "test-repo" {
		t.Errorf("expected name 'test-repo', got %q", repo.Name)
	}
	if repo.Stars != 42 {
		t.Errorf("expected 42 stars, got %d", repo.Stars)
	}
	if repo.Forks != 10 {
		t.Errorf("expected 10 forks, got %d", repo.Forks)
	}
}

func TestToIssue(t *testing.T) {
	issueData := map[string]any{
		"number":     float64(123),
		"title":      "Test Issue",
		"state":      "open",
		"html_url":   "https://github.com/user/repo/issues/123",
		"created_at": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		"user": map[string]any{
			"login": "testuser",
		},
		"labels": []any{
			map[string]any{"name": "bug"},
		},
	}

	issue := toIssue(issueData, false)
	if issue.Number != 123 {
		t.Errorf("expected number 123, got %d", issue.Number)
	}
	if issue.Title != "Test Issue" {
		t.Errorf("expected title 'Test Issue', got %q", issue.Title)
	}
	if issue.Author != "testuser" {
		t.Errorf("expected author 'testuser', got %q", issue.Author)
	}
	if len(issue.Labels) != 1 || issue.Labels[0] != "bug" {
		t.Errorf("expected labels [bug], got %v", issue.Labels)
	}
}

func TestToPR(t *testing.T) {
	prData := map[string]any{
		"number":     float64(456),
		"title":      "Test PR",
		"state":      "open",
		"draft":      true,
		"mergeable":  true,
		"html_url":   "https://github.com/user/repo/pull/456",
		"created_at": time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		"user": map[string]any{
			"login": "contributor",
		},
		"labels": []any{},
	}

	pr := toPR(prData, false)
	if pr.Number != 456 {
		t.Errorf("expected number 456, got %d", pr.Number)
	}
	if pr.Title != "Test PR" {
		t.Errorf("expected title 'Test PR', got %q", pr.Title)
	}
	if !pr.Draft {
		t.Error("expected draft to be true")
	}
	if pr.Mergeable != "yes" {
		t.Errorf("expected mergeable 'yes', got %q", pr.Mergeable)
	}
}
