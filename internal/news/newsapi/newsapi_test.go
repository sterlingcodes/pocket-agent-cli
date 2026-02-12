package newsapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

var testConfigPath string

func TestMain(m *testing.M) {
	// Set up a temp config path BEFORE any config.Path() call.
	dir, err := os.MkdirTemp("", "newsapi-test-*")
	if err != nil {
		panic(err)
	}
	testConfigPath = filepath.Join(dir, "config.json")
	os.Setenv("POCKET_CONFIG", testConfigPath)

	code := m.Run()

	os.Unsetenv("POCKET_CONFIG")
	os.RemoveAll(dir)
	os.Exit(code)
}

func writeTestConfig(t *testing.T, values map[string]string) {
	t.Helper()
	data, err := json.Marshal(values)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(testConfigPath, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func clearTestConfig(t *testing.T) {
	t.Helper()
	os.Remove(testConfigPath)
}

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "newsapi" {
		t.Errorf("expected Use 'newsapi', got %q", cmd.Use)
	}

	// Check aliases
	expectedAliases := []string{"news-api"}
	if len(cmd.Aliases) != len(expectedAliases) {
		t.Errorf("expected %d aliases, got %d", len(expectedAliases), len(cmd.Aliases))
	}

	// Check subcommands exist
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"headlines", "search [query]", "sources"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestHeadlinesCmd(t *testing.T) {
	writeTestConfig(t, map[string]string{"newsapi_key": "test-api-key"})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify API key header
		if r.Header.Get("X-Api-Key") != "test-api-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if r.URL.Path != "/top-headlines" {
			http.Error(w, "wrong path", http.StatusNotFound)
			return
		}

		resp := map[string]any{
			"status":       "ok",
			"totalResults": 2,
			"articles": []article{
				{
					Source: struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					}{ID: "test-source", Name: "Test Source"},
					Author:      "Test Author",
					Title:       "Test Headline 1",
					Description: "Test description 1",
					URL:         "https://example.com/1",
					ImageURL:    "https://example.com/img1.jpg",
					PublishedAt: "2025-01-01T12:00:00Z",
				},
				{
					Source: struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					}{ID: "test-source2", Name: "Test Source 2"},
					Title:       "Test Headline 2",
					Description: "Test description 2",
					URL:         "https://example.com/2",
					PublishedAt: "2025-01-01T13:00:00Z",
				},
			},
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	cmd := newHeadlinesCmd()
	cmd.SetArgs([]string{"--country", "us", "--limit", "20"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHeadlinesWithCategory(t *testing.T) {
	writeTestConfig(t, map[string]string{"newsapi_key": "test-api-key"})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "test-api-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Verify category parameter
		if r.URL.Query().Get("category") != "technology" {
			http.Error(w, "bad category", http.StatusBadRequest)
			return
		}

		resp := map[string]any{
			"status":       "ok",
			"totalResults": 1,
			"articles": []article{
				{
					Title: "Tech News",
					URL:   "https://example.com/tech",
				},
			},
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	cmd := newHeadlinesCmd()
	cmd.SetArgs([]string{"--category", "technology"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSearchCmd(t *testing.T) {
	writeTestConfig(t, map[string]string{"newsapi_key": "test-api-key"})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "test-api-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if r.URL.Path != "/everything" {
			http.Error(w, "wrong path", http.StatusNotFound)
			return
		}

		// Verify query parameter
		if r.URL.Query().Get("q") == "" {
			http.Error(w, "missing query", http.StatusBadRequest)
			return
		}

		resp := map[string]any{
			"status":       "ok",
			"totalResults": 100,
			"articles": []article{
				{
					Title:       "Search Result 1",
					Description: "Description 1",
					URL:         "https://example.com/result1",
					PublishedAt: "2025-01-02T10:00:00Z",
				},
			},
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	cmd := newSearchCmd()
	cmd.SetArgs([]string{"golang", "--limit", "20", "--sort", "publishedAt"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSearchWithDateRange(t *testing.T) {
	writeTestConfig(t, map[string]string{"newsapi_key": "test-api-key"})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "test-api-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Verify date parameters
		if r.URL.Query().Get("from") != "2025-01-01" {
			http.Error(w, "bad from date", http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("to") != "2025-01-31" {
			http.Error(w, "bad to date", http.StatusBadRequest)
			return
		}

		resp := map[string]any{
			"status":       "ok",
			"totalResults": 5,
			"articles":     []article{},
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	cmd := newSearchCmd()
	cmd.SetArgs([]string{"test", "--from", "2025-01-01", "--to", "2025-01-31"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSourcesCmd(t *testing.T) {
	writeTestConfig(t, map[string]string{"newsapi_key": "test-api-key"})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "test-api-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if r.URL.Path != "/top-headlines/sources" {
			http.Error(w, "wrong path", http.StatusNotFound)
			return
		}

		resp := map[string]any{
			"status": "ok",
			"sources": []map[string]any{
				{
					"id":          "bbc-news",
					"name":        "BBC News",
					"description": "Use BBC News for up-to-the-minute news",
					"url":         "https://www.bbc.co.uk/news",
					"category":    "general",
					"language":    "en",
					"country":     "gb",
				},
				{
					"id":          "cnn",
					"name":        "CNN",
					"description": "View the latest news and videos",
					"url":         "https://www.cnn.com",
					"category":    "general",
					"language":    "en",
					"country":     "us",
				},
			},
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	cmd := newSourcesCmd()
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSourcesWithFilters(t *testing.T) {
	writeTestConfig(t, map[string]string{"newsapi_key": "test-api-key"})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "test-api-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Verify filter parameters
		if r.URL.Query().Get("category") != "technology" {
			http.Error(w, "bad category", http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("language") != "en" {
			http.Error(w, "bad language", http.StatusBadRequest)
			return
		}

		resp := map[string]any{
			"status":  "ok",
			"sources": []map[string]any{},
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	cmd := newSourcesCmd()
	cmd.SetArgs([]string{"--category", "technology", "--lang", "en"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAPIError(t *testing.T) {
	writeTestConfig(t, map[string]string{"newsapi_key": "test-api-key"})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"code":    "apiKeyInvalid",
			"message": "Your API key is invalid",
		})
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	cmd := newHeadlinesCmd()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for invalid API key")
	}
}

func TestHTTPError(t *testing.T) {
	writeTestConfig(t, map[string]string{"newsapi_key": "test-api-key"})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	cmd := newHeadlinesCmd()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestMalformedJSON(t *testing.T) {
	writeTestConfig(t, map[string]string{"newsapi_key": "test-api-key"})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not valid json"))
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	cmd := newHeadlinesCmd()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestMissingAPIKey(t *testing.T) {
	clearTestConfig(t)

	cmd := newHeadlinesCmd()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for missing API key")
	}
}

func TestFormatArticles(t *testing.T) {
	articles := []article{
		{
			Source: struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{ID: "source1", Name: "Source 1"},
			Author:      "Author 1",
			Title:       "Title 1",
			Description: "Description 1",
			URL:         "https://example.com/1",
			ImageURL:    "https://example.com/img1.jpg",
			PublishedAt: "2025-01-01T12:00:00Z",
		},
	}

	result := formatArticles(articles)

	if len(result) != 1 {
		t.Errorf("expected 1 article, got %d", len(result))
	}

	if result[0]["source"] != "Source 1" {
		t.Errorf("expected source 'Source 1', got %v", result[0]["source"])
	}

	if result[0]["title"] != "Title 1" {
		t.Errorf("expected title 'Title 1', got %v", result[0]["title"])
	}
}
