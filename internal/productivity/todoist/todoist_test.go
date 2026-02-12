package todoist

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "todoist" {
		t.Errorf("expected Use 'todoist', got %q", cmd.Use)
	}
	// Check that command has subcommands
	if len(cmd.Commands()) < 4 {
		t.Errorf("expected at least 4 subcommands, got %d", len(cmd.Commands()))
	}
}

func TestFormatTasks(t *testing.T) {
	tasks := []task{
		{
			ID:          "task1",
			Content:     "Test task",
			Description: "Test description",
			ProjectID:   "proj1",
			Priority:    3,
			Due: &struct {
				Date      string `json:"date"`
				Datetime  string `json:"datetime,omitempty"`
				String    string `json:"string"`
				Recurring bool   `json:"is_recurring"`
			}{
				Date:      "2024-01-15",
				String:    "Jan 15",
				Recurring: false,
			},
			Labels:    []string{"work", "urgent"},
			CreatedAt: "2024-01-01T12:00:00Z",
			URL:       "https://todoist.com/task1",
		},
	}

	formatted := formatTasks(tasks)
	if len(formatted) != 1 {
		t.Fatalf("expected 1 formatted task, got %d", len(formatted))
	}

	f := formatted[0]
	if f["id"] != "task1" {
		t.Errorf("expected id 'task1', got %v", f["id"])
	}
	if f["content"] != "Test task" {
		t.Errorf("expected content 'Test task', got %v", f["content"])
	}
	if f["priority"] != 3 {
		t.Errorf("expected priority 3, got %v", f["priority"])
	}
	if f["due"] != "Jan 15" {
		t.Errorf("expected due 'Jan 15', got %v", f["due"])
	}
	if f["due_date"] != "2024-01-15" {
		t.Errorf("expected due_date '2024-01-15', got %v", f["due_date"])
	}
	if f["recurring"] != false {
		t.Errorf("expected recurring false, got %v", f["recurring"])
	}
}

func TestFormatTasksNoDue(t *testing.T) {
	tasks := []task{
		{
			ID:          "task2",
			Content:     "No due date task",
			Description: "",
			ProjectID:   "proj1",
			Priority:    1,
			Due:         nil,
			Labels:      []string{},
			URL:         "https://todoist.com/task2",
		},
	}

	formatted := formatTasks(tasks)
	if len(formatted) != 1 {
		t.Fatalf("expected 1 formatted task, got %d", len(formatted))
	}

	f := formatted[0]
	if _, hasDue := f["due"]; hasDue {
		t.Error("expected no 'due' field for task without due date")
	}
	if _, hasDueDate := f["due_date"]; hasDueDate {
		t.Error("expected no 'due_date' field for task without due date")
	}
}

func TestDoRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authorization header
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization 'Bearer test-token', got %q", r.Header.Get("Authorization"))
		}

		// Return mock tasks
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{
					"id":          "task1",
					"content":     "Test Task",
					"project_id":  "proj1",
					"priority":    2,
					"labels":      []string{"test"},
					"url":         "https://todoist.com/task1",
					"added_at":    "2024-01-01T12:00:00Z",
					"description": "Test description",
				},
			},
		})
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	client := &todoistClient{
		token:      "test-token",
		httpClient: &http.Client{},
	}

	body, err := client.doRequest("GET", "/tasks", nil)
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}

	var resp struct {
		Results []task `json:"results"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 task, got %d", len(resp.Results))
	}

	if resp.Results[0].ID != "task1" {
		t.Errorf("expected task ID 'task1', got %q", resp.Results[0].ID)
	}
}

func TestDoRequestPOST(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check method
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		// Check content type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got %q", r.Header.Get("Content-Type"))
		}

		// Parse body
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if payload["content"] != "New task" {
			t.Errorf("expected content 'New task', got %v", payload["content"])
		}

		// Return created task
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":         "new123",
			"content":    payload["content"],
			"project_id": payload["project_id"],
			"url":        "https://todoist.com/new123",
		})
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	client := &todoistClient{
		token:      "test-token",
		httpClient: &http.Client{},
	}

	payload := map[string]any{
		"content":    "New task",
		"project_id": "proj1",
	}

	body, err := client.doRequest("POST", "/tasks", payload)
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}

	var result task
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if result.ID != "new123" {
		t.Errorf("expected task ID 'new123', got %q", result.ID)
	}
}

func TestDoRequestDELETE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check method
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got %s", r.Method)
		}

		// Return success (no content)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	client := &todoistClient{
		token:      "test-token",
		httpClient: &http.Client{},
	}

	_, err := client.doRequest("DELETE", "/tasks/task123", nil)
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}
}

func TestDoRequestError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Invalid token"}`))
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	client := &todoistClient{
		token:      "bad-token",
		httpClient: &http.Client{},
	}

	_, err := client.doRequest("GET", "/tasks", nil)
	if err == nil {
		t.Error("expected error for 401 response, got nil")
	}
}

func TestProjectsResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{
					"id":            "proj1",
					"name":          "Work",
					"color":         "red",
					"parent_id":     "",
					"child_order":   1,
					"is_favorite":   true,
					"inbox_project": false,
					"url":           "https://todoist.com/proj1",
				},
			},
		})
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	client := &todoistClient{
		token:      "test-token",
		httpClient: &http.Client{},
	}

	body, err := client.doRequest("GET", "/projects", nil)
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}

	var resp struct {
		Results []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal projects: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 project, got %d", len(resp.Results))
	}

	if resp.Results[0].Name != "Work" {
		t.Errorf("expected project name 'Work', got %q", resp.Results[0].Name)
	}
}
