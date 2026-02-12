package mastodon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "mastodon" {
		t.Errorf("expected Use 'mastodon', got %q", cmd.Use)
	}
	// Check that command has subcommands
	if len(cmd.Commands()) < 4 {
		t.Errorf("expected at least 4 subcommands, got %d", len(cmd.Commands()))
	}
	// Check aliases
	if len(cmd.Aliases) < 1 {
		t.Error("expected aliases")
	}
}

func TestFormatStatuses(t *testing.T) {
	statuses := []status{
		{
			ID:        "status1",
			CreatedAt: "2024-01-01T12:00:00Z",
			Content:   "<p>Test content</p>",
			URL:       "https://mastodon.example/status1",
			Account: struct {
				Username    string `json:"username"`
				DisplayName string `json:"display_name"`
				Acct        string `json:"acct"`
			}{
				Username:    "testuser",
				DisplayName: "Test User",
				Acct:        "testuser@example.com",
			},
			ReblogsCount:    5,
			FavouritesCount: 10,
			RepliesCount:    2,
			Reblog:          nil,
		},
	}

	formatted := formatStatuses(statuses)
	if len(formatted) != 1 {
		t.Fatalf("expected 1 formatted status, got %d", len(formatted))
	}

	f := formatted[0]
	if f["id"] != "status1" {
		t.Errorf("expected id 'status1', got %v", f["id"])
	}
	if f["author"] != "testuser@example.com" {
		t.Errorf("expected author 'testuser@example.com', got %v", f["author"])
	}
	if f["display"] != "Test User" {
		t.Errorf("expected display 'Test User', got %v", f["display"])
	}
	if f["reblogs"] != 5 {
		t.Errorf("expected reblogs 5, got %v", f["reblogs"])
	}
	if f["favourites"] != 10 {
		t.Errorf("expected favourites 10, got %v", f["favourites"])
	}
	if f["is_boost"] != false {
		t.Errorf("expected is_boost false, got %v", f["is_boost"])
	}
}

func TestFormatStatusesWithReblog(t *testing.T) {
	reblog := &status{
		ID:        "reblog1",
		CreatedAt: "2024-01-01T11:00:00Z",
		Content:   "<p>Original content</p>",
		URL:       "https://mastodon.example/reblog1",
		Account: struct {
			Username    string `json:"username"`
			DisplayName string `json:"display_name"`
			Acct        string `json:"acct"`
		}{
			Username:    "original",
			DisplayName: "Original User",
			Acct:        "original@example.com",
		},
		ReblogsCount:    100,
		FavouritesCount: 200,
		RepliesCount:    50,
	}

	statuses := []status{
		{
			ID:        "status1",
			CreatedAt: "2024-01-01T12:00:00Z",
			Content:   "",
			URL:       "https://mastodon.example/status1",
			Account: struct {
				Username    string `json:"username"`
				DisplayName string `json:"display_name"`
				Acct        string `json:"acct"`
			}{
				Username:    "booster",
				DisplayName: "Booster User",
				Acct:        "booster@example.com",
			},
			Reblog: reblog,
		},
	}

	formatted := formatStatuses(statuses)
	if len(formatted) != 1 {
		t.Fatalf("expected 1 formatted status, got %d", len(formatted))
	}

	f := formatted[0]
	// Should use reblog's content and stats
	if f["author"] != "original@example.com" {
		t.Errorf("expected author 'original@example.com', got %v", f["author"])
	}
	if f["reblogs"] != 100 {
		t.Errorf("expected reblogs 100, got %v", f["reblogs"])
	}
	if f["is_boost"] != true {
		t.Errorf("expected is_boost true, got %v", f["is_boost"])
	}
}

func TestDoRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authorization header
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization 'Bearer test-token', got %q", r.Header.Get("Authorization"))
		}

		// Check endpoint
		if r.URL.Path != "/api/v1/timelines/home" {
			t.Errorf("expected path '/api/v1/timelines/home', got %q", r.URL.Path)
		}

		// Return mock statuses
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":         "test1",
				"created_at": "2024-01-01T12:00:00Z",
				"content":    "Test post",
				"url":        "https://mastodon.example/test1",
				"account": map[string]any{
					"username":     "testuser",
					"display_name": "Test User",
					"acct":         "testuser@example.com",
				},
				"reblogs_count":    5,
				"favourites_count": 10,
				"replies_count":    2,
			},
		})
	}))
	defer srv.Close()

	client := &mastoClient{
		server:     srv.URL,
		token:      "test-token",
		httpClient: &http.Client{},
	}

	body, err := client.doRequest("GET", "/timelines/home", nil)
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}

	var statuses []status
	if err := json.Unmarshal(body, &statuses); err != nil {
		t.Fatalf("failed to unmarshal statuses: %v", err)
	}

	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}

	if statuses[0].ID != "test1" {
		t.Errorf("expected status ID 'test1', got %q", statuses[0].ID)
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

		if payload["status"] != "Hello, Fediverse!" {
			t.Errorf("expected status 'Hello, Fediverse!', got %v", payload["status"])
		}

		// Return created status
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":         "new123",
			"content":    payload["status"],
			"url":        "https://mastodon.example/new123",
			"created_at": "2024-01-01T12:00:00Z",
		})
	}))
	defer srv.Close()

	client := &mastoClient{
		server:     srv.URL,
		token:      "test-token",
		httpClient: &http.Client{},
	}

	payload := map[string]any{
		"status":     "Hello, Fediverse!",
		"visibility": "public",
	}

	body, err := client.doRequest("POST", "/statuses", payload)
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}

	var result status
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if result.ID != "new123" {
		t.Errorf("expected status ID 'new123', got %q", result.ID)
	}
}

func TestDoRequestError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "Unauthorized",
		})
	}))
	defer srv.Close()

	client := &mastoClient{
		server:     srv.URL,
		token:      "bad-token",
		httpClient: &http.Client{},
	}

	_, err := client.doRequest("GET", "/timelines/home", nil)
	if err == nil {
		t.Error("expected error for 401 response, got nil")
	}
}
