package gitlab

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "gitlab" {
		t.Errorf("expected Use 'gitlab', got %q", cmd.Use)
	}
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"projects", "issues", "mrs", "user"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestGitLabClientDoRequest(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   any
		expectErr  bool
	}{
		{"success", 200, []map[string]any{{"name": "test"}}, false},
		{"400 error", 400, map[string]any{"message": "Bad Request"}, true},
		{"404 error", 404, map[string]any{"error": "Not Found"}, true},
		{"500 error", 500, map[string]any{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("PRIVATE-TOKEN") == "" {
					t.Error("expected PRIVATE-TOKEN header")
				}
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer srv.Close()

			client := &gitlabClient{
				baseURL:    srv.URL,
				token:      "test-token",
				httpClient: &http.Client{},
			}

			body, err := client.doRequest("/api/v4/test")
			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if !tt.expectErr && len(body) == 0 {
				t.Error("expected response body, got empty")
			}
		})
	}
}

func TestProjectsParsing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		projects := []map[string]any{
			{
				"id":                  float64(1),
				"name":                "Test Project",
				"name_with_namespace": "Group / Test Project",
				"path_with_namespace": "group/test-project",
				"description":         "A test project",
				"web_url":             "https://gitlab.com/group/test-project",
				"default_branch":      "main",
				"visibility":          "private",
				"star_count":          float64(5),
				"forks_count":         float64(2),
				"last_activity_at":    "2024-01-15T10:30:00.000Z",
			},
		}
		json.NewEncoder(w).Encode(projects)
	}))
	defer srv.Close()

	client := &gitlabClient{
		baseURL:    srv.URL,
		token:      "test-token",
		httpClient: &http.Client{},
	}

	body, err := client.doRequest("/api/v4/projects")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var projects []struct {
		ID                int    `json:"id"`
		Name              string `json:"name"`
		NameWithNamespace string `json:"name_with_namespace"`
	}
	if err := json.Unmarshal(body, &projects); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(projects))
	}
	if projects[0].Name != "Test Project" {
		t.Errorf("expected name 'Test Project', got %q", projects[0].Name)
	}
}

func TestIssuesParsing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		issues := []map[string]any{
			{
				"id":     float64(10),
				"iid":    float64(1),
				"title":  "Test Issue",
				"state":  "opened",
				"labels": []string{"bug", "urgent"},
				"author": map[string]any{
					"username": "testuser",
				},
				"assignees": []map[string]any{
					{"username": "assignee1"},
				},
				"created_at": "2024-01-15T10:30:00.000Z",
				"updated_at": "2024-01-15T11:30:00.000Z",
				"web_url":    "https://gitlab.com/group/project/issues/1",
			},
		}
		json.NewEncoder(w).Encode(issues)
	}))
	defer srv.Close()

	client := &gitlabClient{
		baseURL:    srv.URL,
		token:      "test-token",
		httpClient: &http.Client{},
	}

	body, err := client.doRequest("/api/v4/issues")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var issues []struct {
		Title  string   `json:"title"`
		State  string   `json:"state"`
		Labels []string `json:"labels"`
	}
	if err := json.Unmarshal(body, &issues); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Title != "Test Issue" {
		t.Errorf("expected title 'Test Issue', got %q", issues[0].Title)
	}
	if len(issues[0].Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(issues[0].Labels))
	}
}
