package gdrive

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "gdrive" {
		t.Errorf("expected Use 'gdrive', got %q", cmd.Use)
	}
	// Check that command has subcommands
	if len(cmd.Commands()) < 2 {
		t.Errorf("expected at least 2 subcommands, got %d", len(cmd.Commands()))
	}
}

func TestFormatTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		input    string
		expected string
	}{
		{now.Add(-30 * time.Second).Format(time.RFC3339), "now"},
		{now.Add(-5 * time.Minute).Format(time.RFC3339), "5m ago"},
		{now.Add(-2 * time.Hour).Format(time.RFC3339), "2h ago"},
		{now.Add(-3 * 24 * time.Hour).Format(time.RFC3339), "3d ago"},
		{now.Add(-10 * 24 * time.Hour).Format(time.RFC3339), "1w ago"},
		{now.Add(-40 * 24 * time.Hour).Format(time.RFC3339), "1mo ago"},
		{now.Add(-400 * 24 * time.Hour).Format(time.RFC3339), "1y ago"},
		{"", ""},
		{"invalid", "invalid"},
	}

	for _, tt := range tests {
		result := formatTime(tt.input)
		if result != tt.expected {
			t.Errorf("formatTime(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"500", "500 B"},
		{"1024", "1.0 KB"},
		{"1048576", "1.0 MB"},
		{"1073741824", "1.0 GB"},
		{"2147483648", "2.0 GB"},
		{"invalid", "invalid"},
	}

	for _, tt := range tests {
		result := formatSize(tt.input)
		if result != tt.expected {
			t.Errorf("formatSize(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestDoRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check x-goog-api-key header
		if r.Header.Get("x-goog-api-key") != "test-api-key" {
			t.Errorf("expected x-goog-api-key 'test-api-key', got %q", r.Header.Get("x-goog-api-key"))
		}

		// Check User-Agent
		if r.Header.Get("User-Agent") != "Pocket-CLI/1.0" {
			t.Errorf("expected User-Agent 'Pocket-CLI/1.0', got %q", r.Header.Get("User-Agent"))
		}

		// Return mock files
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"files": []map[string]any{
				{
					"id":           "file1",
					"name":         "Test Document.pdf",
					"mimeType":     "application/pdf",
					"size":         "1048576",
					"createdTime":  "2024-01-01T12:00:00Z",
					"modifiedTime": "2024-01-02T12:00:00Z",
					"webViewLink":  "https://drive.google.com/file/d/file1",
				},
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	data, err := doRequest(srv.URL+"/files", "test-api-key")
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}

	var resp struct {
		Files []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"files"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(resp.Files))
	}

	if resp.Files[0].ID != "file1" {
		t.Errorf("expected file ID 'file1', got %q", resp.Files[0].ID)
	}
	if resp.Files[0].Name != "Test Document.pdf" {
		t.Errorf("expected file name 'Test Document.pdf', got %q", resp.Files[0].Name)
	}
}

func TestDoRequestForbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Access denied",
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	_, err := doRequest(srv.URL, "bad-key")
	if err == nil {
		t.Error("expected error for 403 response, got nil")
	}
}

func TestDoRequestNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "File not found",
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	_, err := doRequest(srv.URL, "test-key")
	if err == nil {
		t.Error("expected error for 404 response, got nil")
	}
}

func TestSearchResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check query params
		params := r.URL.Query()
		if params.Get("q") != "name contains 'test'" {
			t.Errorf("expected query 'name contains 'test'', got %q", params.Get("q"))
		}
		if params.Get("pageSize") != "10" {
			t.Errorf("expected pageSize '10', got %q", params.Get("pageSize"))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"files": []map[string]any{
				{
					"id":           "search1",
					"name":         "test.txt",
					"mimeType":     "text/plain",
					"size":         "1024",
					"createdTime":  "2024-01-01T12:00:00Z",
					"modifiedTime": "2024-01-01T13:00:00Z",
					"webViewLink":  "https://drive.google.com/file/d/search1",
				},
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	data, err := doRequest(srv.URL+"?q=name+contains+'test'&pageSize=10", "test-key")
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}

	var resp struct {
		Files []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			MimeType string `json:"mimeType"`
		} `json:"files"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(resp.Files))
	}

	if resp.Files[0].Name != "test.txt" {
		t.Errorf("expected file name 'test.txt', got %q", resp.Files[0].Name)
	}
}

func TestFileInfoResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/files/file123" {
			t.Errorf("expected path '/files/file123', got %q", r.URL.Path)
		}

		// Check fields param
		params := r.URL.Query()
		if params.Get("fields") == "" {
			t.Error("expected 'fields' query parameter")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":           "file123",
			"name":         "Document.docx",
			"mimeType":     "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"size":         "2097152",
			"createdTime":  "2024-01-01T10:00:00Z",
			"modifiedTime": "2024-01-05T15:30:00Z",
			"webViewLink":  "https://drive.google.com/file/d/file123",
			"description":  "Important document",
			"owners": []map[string]any{
				{"displayName": "John Doe"},
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	data, err := doRequest(srv.URL+"/files/file123?fields=id,name,mimeType,size,createdTime,modifiedTime,webViewLink,description,owners", "test-key")
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}

	var resp struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Size        string `json:"size"`
		Description string `json:"description"`
		Owners      []struct {
			DisplayName string `json:"displayName"`
		} `json:"owners"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.ID != "file123" {
		t.Errorf("expected file ID 'file123', got %q", resp.ID)
	}
	if resp.Name != "Document.docx" {
		t.Errorf("expected file name 'Document.docx', got %q", resp.Name)
	}
	if resp.Description != "Important document" {
		t.Errorf("expected description 'Important document', got %q", resp.Description)
	}
	if len(resp.Owners) != 1 || resp.Owners[0].DisplayName != "John Doe" {
		t.Error("expected owner 'John Doe'")
	}
}

func TestDoRequestAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Invalid query",
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	_, err := doRequest(srv.URL, "test-key")
	if err == nil {
		t.Error("expected error for 400 response, got nil")
	}
}
