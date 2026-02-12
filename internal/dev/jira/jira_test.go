package jira

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
	if cmd.Use != "jira" {
		t.Errorf("expected Use 'jira', got %q", cmd.Use)
	}
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Name()] = true
	}
	for _, name := range []string{"issues", "issue", "projects", "create", "transition"} {
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
		{map[string]any{"key": "PROJ-123"}, "key", "PROJ-123"},
		{map[string]any{}, "key", ""},
		{map[string]any{"key": 123}, "key", ""},
	}
	for _, tt := range tests {
		result := getString(tt.m, tt.key)
		if result != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, result)
		}
	}
}

func TestTimeAgo(t *testing.T) {
	now := time.Now()
	tests := []struct {
		t        time.Time
		expected string
	}{
		{now, "now"},
		{now.Add(-8 * time.Minute), "8m"},
		{now.Add(-6 * time.Hour), "6h"},
		{now.Add(-12 * 24 * time.Hour), "12d"},
	}
	for _, tt := range tests {
		result := timeAgo(tt.t)
		if result != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, result)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s        string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exact", 5, "exact"},
		{"this is too long for the limit", 15, "this is too ..."},
	}
	for _, tt := range tests {
		result := truncate(tt.s, tt.maxLen)
		if result != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, result)
		}
	}
}

func TestJiraGetSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Basic ") {
			t.Error("expected Basic auth header")
		}
		json.NewEncoder(w).Encode(map[string]any{
			"key": "PROJ-123",
			"fields": map[string]any{
				"summary": "Test issue",
			},
		})
	}))
	defer srv.Close()

	var result map[string]any
	err := jiraGet("test@example.com", "test-token", srv.URL+"/rest/api/3/issue/PROJ-123", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result["key"] != "PROJ-123" {
		t.Errorf("expected key 'PROJ-123', got %v", result["key"])
	}
}

func TestJiraGetError(t *testing.T) {
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
				json.NewEncoder(w).Encode(map[string]any{
					"errorMessages": []string{"Error occurred"},
				})
			}))
			defer srv.Close()

			var result map[string]any
			err := jiraGet("test@example.com", "test-token", srv.URL+"/test", &result)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestToIssue(t *testing.T) {
	issueData := map[string]any{
		"key": "PROJ-123",
		"fields": map[string]any{
			"summary": "Test Issue",
			"status": map[string]any{
				"name": "In Progress",
			},
			"issuetype": map[string]any{
				"name": "Bug",
			},
			"priority": map[string]any{
				"name": "High",
			},
			"assignee": map[string]any{
				"displayName": "John Doe",
			},
			"reporter": map[string]any{
				"displayName": "Jane Smith",
			},
			"project": map[string]any{
				"key": "PROJ",
			},
			"labels":  []any{"urgent", "customer"},
			"created": time.Now().Add(-24 * time.Hour).Format("2006-01-02T15:04:05.000-0700"),
			"updated": time.Now().Add(-2 * time.Hour).Format("2006-01-02T15:04:05.000-0700"),
		},
	}

	issue := toIssue("https://jira.example.com", issueData, false)
	if issue.Key != "PROJ-123" {
		t.Errorf("expected key 'PROJ-123', got %q", issue.Key)
	}
	if issue.Summary != "Test Issue" {
		t.Errorf("expected summary 'Test Issue', got %q", issue.Summary)
	}
	if issue.Status != "In Progress" {
		t.Errorf("expected status 'In Progress', got %q", issue.Status)
	}
	if issue.Type != "Bug" {
		t.Errorf("expected type 'Bug', got %q", issue.Type)
	}
	if len(issue.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(issue.Labels))
	}
}

func TestToProject(t *testing.T) {
	projectData := map[string]any{
		"key":            "PROJ",
		"name":           "Test Project",
		"projectTypeKey": "software",
		"lead": map[string]any{
			"displayName": "Project Lead",
		},
		"avatarUrls": map[string]any{
			"48x48": "https://example.com/avatar.png",
		},
	}

	proj := toProject("https://jira.example.com", projectData)
	if proj.Key != "PROJ" {
		t.Errorf("expected key 'PROJ', got %q", proj.Key)
	}
	if proj.Name != "Test Project" {
		t.Errorf("expected name 'Test Project', got %q", proj.Name)
	}
	if proj.Lead != "Project Lead" {
		t.Errorf("expected lead 'Project Lead', got %q", proj.Lead)
	}
	if proj.Avatar == "" {
		t.Error("expected avatar to be set")
	}
}

func TestExtractTextFromADF(t *testing.T) {
	adf := map[string]any{
		"type":    "doc",
		"version": float64(1),
		"content": []any{
			map[string]any{
				"type": "paragraph",
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "This is a test",
					},
				},
			},
			map[string]any{
				"type": "paragraph",
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "Second paragraph",
					},
				},
			},
		},
	}

	text := extractTextFromADF(adf)
	if !strings.Contains(text, "This is a test") {
		t.Errorf("expected text to contain 'This is a test', got %q", text)
	}
	if !strings.Contains(text, "Second paragraph") {
		t.Errorf("expected text to contain 'Second paragraph', got %q", text)
	}
}
