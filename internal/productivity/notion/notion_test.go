package notion

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "notion" {
		t.Errorf("expected Use 'notion', got %q", cmd.Use)
	}
	// Check that command has subcommands
	if len(cmd.Commands()) < 4 {
		t.Errorf("expected at least 4 subcommands, got %d", len(cmd.Commands()))
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		props    map[string]any
		expected string
	}{
		{
			name: "page with title property",
			props: map[string]any{
				"title": []interface{}{
					map[string]any{
						"plain_text": "Test Page",
					},
				},
			},
			expected: "Test Page",
		},
		{
			name: "database item with Name property",
			props: map[string]any{
				"Name": map[string]any{
					"title": []interface{}{
						map[string]any{
							"plain_text": "Test Item",
						},
					},
				},
			},
			expected: "Test Item",
		},
		{
			name:     "empty properties",
			props:    map[string]any{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTitle(tt.props)
			if result != tt.expected {
				t.Errorf("extractTitle() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestDoRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authorization header
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization 'Bearer test-token', got %q", r.Header.Get("Authorization"))
		}

		// Check Notion-Version header
		if r.Header.Get("Notion-Version") != "2022-06-28" {
			t.Errorf("expected Notion-Version '2022-06-28', got %q", r.Header.Get("Notion-Version"))
		}

		// Return mock search results
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{
					"id":     "page1",
					"object": "page",
					"url":    "https://notion.so/page1",
					"properties": map[string]any{
						"title": []map[string]any{
							{"plain_text": "Test Page"},
						},
					},
				},
			},
			"has_more": false,
		})
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	client := &notionClient{
		token:      "test-token",
		httpClient: &http.Client{},
	}

	body, err := client.doRequest("GET", "/search", nil)
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}

	var result struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}

	if result.Results[0].ID != "page1" {
		t.Errorf("expected page ID 'page1', got %q", result.Results[0].ID)
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

		if payload["query"] != "test query" {
			t.Errorf("expected query 'test query', got %v", payload["query"])
		}

		// Return mock results
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{
					"id":     "result1",
					"object": "page",
				},
			},
		})
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	client := &notionClient{
		token:      "test-token",
		httpClient: &http.Client{},
	}

	payload := map[string]any{
		"query": "test query",
	}

	body, err := client.doRequest("POST", "/search", payload)
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}

	var result struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
}

func TestDoRequestError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"message": "Unauthorized",
			"code":    "unauthorized",
		})
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	client := &notionClient{
		token:      "bad-token",
		httpClient: &http.Client{},
	}

	_, err := client.doRequest("GET", "/search", nil)
	if err == nil {
		t.Error("expected error for 401 response, got nil")
	}
}

func TestDoRequestNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"message": "Page not found",
			"code":    "object_not_found",
		})
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	client := &notionClient{
		token:      "test-token",
		httpClient: &http.Client{},
	}

	_, err := client.doRequest("GET", "/pages/nonexistent", nil)
	if err == nil {
		t.Error("expected error for 404 response, got nil")
	}
}

func TestPageResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock page metadata response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":  "page123",
			"url": "https://notion.so/page123",
			"properties": map[string]any{
				"title": []map[string]any{
					{"plain_text": "Test Page Title"},
				},
			},
		})
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	client := &notionClient{
		token:      "test-token",
		httpClient: &http.Client{},
	}

	body, err := client.doRequest("GET", "/pages/page123", nil)
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}

	var page struct {
		ID         string         `json:"id"`
		URL        string         `json:"url"`
		Properties map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatalf("failed to unmarshal page: %v", err)
	}

	if page.ID != "page123" {
		t.Errorf("expected page ID 'page123', got %q", page.ID)
	}

	title := extractTitle(page.Properties)
	if title != "Test Page Title" {
		t.Errorf("expected title 'Test Page Title', got %q", title)
	}
}

func TestDatabaseQueryResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock database query response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{
					"id":  "item1",
					"url": "https://notion.so/item1",
					"properties": map[string]any{
						"Name": map[string]any{
							"title": []map[string]any{
								{"plain_text": "Database Item 1"},
							},
						},
					},
				},
			},
			"has_more": false,
		})
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	client := &notionClient{
		token:      "test-token",
		httpClient: &http.Client{},
	}

	body, err := client.doRequest("POST", "/databases/db123/query", map[string]any{"page_size": 10})
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}

	var result struct {
		Results []struct {
			ID         string         `json:"id"`
			Properties map[string]any `json:"properties"`
		} `json:"results"`
		HasMore bool `json:"has_more"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}

	title := extractTitle(result.Results[0].Properties)
	if title != "Database Item 1" {
		t.Errorf("expected title 'Database Item 1', got %q", title)
	}
}
