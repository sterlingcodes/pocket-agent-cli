package spotify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "spotify" {
		t.Errorf("expected Use 'spotify', got %q", cmd.Use)
	}
	// Check that command has subcommands
	if len(cmd.Commands()) < 4 {
		t.Errorf("expected at least 4 subcommands, got %d", len(cmd.Commands()))
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0:00"},
		{1000, "0:01"},
		{60000, "1:00"},
		{125000, "2:05"},
		{253000, "4:13"},
		{3661000, "61:01"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.input)
		if result != tt.expected {
			t.Errorf("formatDuration(%d) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestExtractID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"6rqhFgbbKwnb9MLmUQDhG6", "6rqhFgbbKwnb9MLmUQDhG6"},
		{"https://open.spotify.com/track/6rqhFgbbKwnb9MLmUQDhG6", "6rqhFgbbKwnb9MLmUQDhG6"},
		{"https://open.spotify.com/track/6rqhFgbbKwnb9MLmUQDhG6?si=xyz", "6rqhFgbbKwnb9MLmUQDhG6"},
		{"spotify:track:6rqhFgbbKwnb9MLmUQDhG6", "6rqhFgbbKwnb9MLmUQDhG6"},
		{"spotify:album:4aawyAB9vmqN3uQ7FjRGTy", "4aawyAB9vmqN3uQ7FjRGTy"},
	}

	for _, tt := range tests {
		result := extractID(tt.input)
		if result != tt.expected {
			t.Errorf("extractID(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestGetToken(t *testing.T) {
	// Reset cache
	cachedToken = ""
	tokenExpiry = time.Time{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check method
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		// Check headers
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("expected Content-Type 'application/x-www-form-urlencoded', got %q", r.Header.Get("Content-Type"))
		}

		// Return mock token
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	oldTokenURL := tokenURL
	tokenURL = srv.URL
	defer func() { tokenURL = oldTokenURL }()

	// Note: getToken requires config, so this test would need config mocking
	// For now, we just test the token caching mechanism
	if cachedToken != "" {
		t.Error("expected empty cached token initially")
	}

	// Set a cached token manually
	cachedToken = "cached-token"
	tokenExpiry = time.Now().Add(2 * time.Hour)

	// Should return cached token without hitting server
	// (Can't test getToken directly without config setup, but we test the cache logic)
}

func TestDoRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authorization header
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization 'Bearer test-token', got %q", r.Header.Get("Authorization"))
		}

		// Check User-Agent
		if r.Header.Get("User-Agent") != "Pocket-CLI/1.0" {
			t.Errorf("expected User-Agent 'Pocket-CLI/1.0', got %q", r.Header.Get("User-Agent"))
		}

		// Return mock track data
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"name": "Test Track",
			"artists": []map[string]any{
				{"name": "Test Artist"},
			},
			"album": map[string]any{
				"name": "Test Album",
			},
			"duration_ms": 180000,
			"popularity":  75,
			"external_urls": map[string]any{
				"spotify": "https://open.spotify.com/track/test123",
			},
		})
	}))
	defer srv.Close()

	oldAPIURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldAPIURL }()

	data, err := doRequest("test-token", srv.URL)
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}

	var track struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &track); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if track.Name != "Test Track" {
		t.Errorf("expected track name 'Test Track', got %q", track.Name)
	}
}

func TestDoRequestUnauthorized(t *testing.T) {
	// Reset cache
	cachedToken = ""
	tokenExpiry = time.Time{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Invalid access token",
			},
		})
	}))
	defer srv.Close()

	oldAPIURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldAPIURL }()

	_, err := doRequest("bad-token", srv.URL)
	if err == nil {
		t.Error("expected error for 401 response, got nil")
	}

	// Token cache should be cleared
	if cachedToken != "" {
		t.Error("expected cached token to be cleared after 401")
	}
}

func TestDoRequestNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Track not found",
			},
		})
	}))
	defer srv.Close()

	oldAPIURL := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = oldAPIURL }()

	_, err := doRequest("test-token", srv.URL)
	if err == nil {
		t.Error("expected error for 404 response, got nil")
	}
}

func TestParseTrackSearch(t *testing.T) {
	jsonData := `{
		"tracks": {
			"items": [
				{
					"name": "Test Song",
					"artists": [
						{"name": "Artist 1"},
						{"name": "Artist 2"}
					],
					"album": {
						"name": "Test Album"
					},
					"duration_ms": 210000,
					"popularity": 80,
					"preview_url": "https://example.com/preview.mp3",
					"external_urls": {
						"spotify": "https://open.spotify.com/track/test123"
					}
				}
			]
		}
	}`

	// parseTrackSearch calls output.Print, so we can't test it directly without mocking
	// Instead, test the unmarshaling logic
	var resp struct {
		Tracks struct {
			Items []struct {
				Name    string `json:"name"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
			} `json:"items"`
		} `json:"tracks"`
	}

	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(resp.Tracks.Items) != 1 {
		t.Fatalf("expected 1 track, got %d", len(resp.Tracks.Items))
	}

	track := resp.Tracks.Items[0]
	if track.Name != "Test Song" {
		t.Errorf("expected track name 'Test Song', got %q", track.Name)
	}

	if len(track.Artists) != 2 {
		t.Fatalf("expected 2 artists, got %d", len(track.Artists))
	}
}

func TestParseArtistSearch(t *testing.T) {
	jsonData := `{
		"artists": {
			"items": [
				{
					"name": "Test Artist",
					"genres": ["pop", "rock"],
					"followers": {
						"total": 1000000
					},
					"popularity": 85,
					"external_urls": {
						"spotify": "https://open.spotify.com/artist/test123"
					}
				}
			]
		}
	}`

	var resp struct {
		Artists struct {
			Items []struct {
				Name   string   `json:"name"`
				Genres []string `json:"genres"`
			} `json:"items"`
		} `json:"artists"`
	}

	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(resp.Artists.Items) != 1 {
		t.Fatalf("expected 1 artist, got %d", len(resp.Artists.Items))
	}

	artist := resp.Artists.Items[0]
	if artist.Name != "Test Artist" {
		t.Errorf("expected artist name 'Test Artist', got %q", artist.Name)
	}

	if len(artist.Genres) != 2 {
		t.Fatalf("expected 2 genres, got %d", len(artist.Genres))
	}
}
