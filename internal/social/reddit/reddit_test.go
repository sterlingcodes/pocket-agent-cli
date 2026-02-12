package reddit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "reddit" {
		t.Errorf("expected Use 'reddit', got %q", cmd.Use)
	}
	// Check that command has subcommands
	if len(cmd.Commands()) < 5 {
		t.Errorf("expected at least 5 subcommands, got %d", len(cmd.Commands()))
	}
}

func TestParseListingResponse(t *testing.T) {
	jsonData := `{
		"data": {
			"children": [
				{
					"data": {
						"id": "abc123",
						"title": "Test Post",
						"author": "testuser",
						"subreddit": "golang",
						"score": 42,
						"num_comments": 10,
						"url": "https://example.com",
						"permalink": "/r/golang/comments/abc123",
						"created_utc": 1609459200,
						"over_18": false
					}
				}
			]
		}
	}`

	posts, err := parseListingResponse([]byte(jsonData))
	if err != nil {
		t.Fatalf("parseListingResponse failed: %v", err)
	}

	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}

	p := posts[0]
	if p.ID != "abc123" {
		t.Errorf("expected ID 'abc123', got %q", p.ID)
	}
	if p.Title != "Test Post" {
		t.Errorf("expected Title 'Test Post', got %q", p.Title)
	}
	if p.Author != "testuser" {
		t.Errorf("expected Author 'testuser', got %q", p.Author)
	}
	if p.Score != 42 {
		t.Errorf("expected Score 42, got %d", p.Score)
	}
	if p.Comments != 10 {
		t.Errorf("expected Comments 10, got %d", p.Comments)
	}
}

func TestFormatPosts(t *testing.T) {
	posts := []redditPost{
		{
			ID:        "post1",
			Title:     "First Post",
			Author:    "user1",
			Subreddit: "test",
			Score:     100,
			Comments:  5,
			URL:       "https://example.com",
			Permalink: "/r/test/comments/post1",
			Created:   1609459200,
			IsNSFW:    false,
		},
	}

	formatted := formatPosts(posts)
	if len(formatted) != 1 {
		t.Fatalf("expected 1 formatted post, got %d", len(formatted))
	}

	f := formatted[0]
	if f["id"] != "post1" {
		t.Errorf("expected id 'post1', got %v", f["id"])
	}
	if f["title"] != "First Post" {
		t.Errorf("expected title 'First Post', got %v", f["title"])
	}
	if f["author"] != "user1" {
		t.Errorf("expected author 'user1', got %v", f["author"])
	}
	if f["score"] != 100 {
		t.Errorf("expected score 100, got %v", f["score"])
	}
	if f["nsfw"] != false {
		t.Errorf("expected nsfw false, got %v", f["nsfw"])
	}
	if permalink, ok := f["permalink"].(string); !ok || permalink != "https://reddit.com/r/test/comments/post1" {
		t.Errorf("expected full permalink, got %v", f["permalink"])
	}
}

func TestDoRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check headers
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization header 'Bearer test-token', got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("User-Agent") != userAgent {
			t.Errorf("expected User-Agent %q, got %q", userAgent, r.Header.Get("User-Agent"))
		}

		// Return mock listing response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"children": []map[string]any{
					{
						"data": map[string]any{
							"id":           "test123",
							"title":        "Test Title",
							"author":       "testuser",
							"subreddit":    "golang",
							"score":        50,
							"num_comments": 20,
							"url":          "https://example.com",
							"permalink":    "/r/golang/comments/test123",
							"created_utc":  1609459200,
							"over_18":      false,
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	client := &redditClient{
		token:      "test-token",
		httpClient: &http.Client{},
	}

	body, err := client.doRequest("/test")
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}

	posts, err := parseListingResponse(body)
	if err != nil {
		t.Fatalf("parseListingResponse failed: %v", err)
	}

	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}

	if posts[0].ID != "test123" {
		t.Errorf("expected post ID 'test123', got %q", posts[0].ID)
	}
}

func TestDoRequestError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid_token"}`))
	}))
	defer srv.Close()

	oldURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldURL }()

	client := &redditClient{
		token:      "bad-token",
		httpClient: &http.Client{},
	}

	_, err := client.doRequest("/test")
	if err == nil {
		t.Error("expected error for 401 response, got nil")
	}
}

func TestRefreshAccessToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check method
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		// Check headers
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("expected Content-Type 'application/x-www-form-urlencoded', got %q", r.Header.Get("Content-Type"))
		}

		// Return mock token response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "new-access-token",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	oldTokenURL := tokenURL
	tokenURL = srv.URL
	defer func() { tokenURL = oldTokenURL }()

	token, err := refreshAccessToken("test-client-id", "test-refresh-token")
	if err != nil {
		t.Fatalf("refreshAccessToken failed: %v", err)
	}

	if token != "new-access-token" {
		t.Errorf("expected token 'new-access-token', got %q", token)
	}
}

func TestRefreshAccessTokenError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "invalid_grant",
		})
	}))
	defer srv.Close()

	oldTokenURL := tokenURL
	tokenURL = srv.URL
	defer func() { tokenURL = oldTokenURL }()

	_, err := refreshAccessToken("test-client-id", "bad-refresh-token")
	if err == nil {
		t.Error("expected error for bad refresh token, got nil")
	}
}
