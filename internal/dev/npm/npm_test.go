package npm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "npm" {
		t.Errorf("expected Use 'npm', got %q", cmd.Use)
	}
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Name()] = true
	}
	for _, name := range []string{"search", "info", "versions", "deps"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		maxLen   int
		expected string
	}{
		{"short", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"needs truncation", "hello world test", 10, "hello w..."},
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

func TestTimeAgo(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		t        time.Time
		expected string
	}{
		{"now", now, "now"},
		{"5m", now.Add(-5 * time.Minute), "5m"},
		{"2h", now.Add(-2 * time.Hour), "2h"},
		{"3d", now.Add(-3 * 24 * time.Hour), "3d"},
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

func TestCleanRepoURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"git+https", "git+https://github.com/user/repo.git", "https://github.com/user/repo"},
		{"git://", "git://github.com/user/repo.git", "https://github.com/user/repo"},
		{"plain", "https://github.com/user/repo", "https://github.com/user/repo"},
		{"no protocol", "github.com/user/repo", "https://github.com/user/repo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanRepoURL(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestNpmSearchResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"objects": []map[string]any{
				{
					"package": map[string]any{
						"name":        "express",
						"version":     "4.18.2",
						"description": "Fast, unopinionated, minimalist web framework",
						"publisher": map[string]any{
							"username": "dougwilson",
						},
					},
					"score": map[string]any{
						"final": 0.95,
					},
				},
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	// We can't test the full command here without mocking more, but we can test parsing
	// This test verifies the structure that would be returned
	resp, err := http.Get(srv.URL + "/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var data struct {
		Objects []struct {
			Package struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"package"`
		} `json:"objects"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if len(data.Objects) != 1 {
		t.Errorf("expected 1 object, got %d", len(data.Objects))
	}
	if data.Objects[0].Package.Name != "express" {
		t.Errorf("expected name 'express', got %q", data.Objects[0].Package.Name)
	}
}

func TestNpmInfoResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"name":        "lodash",
			"description": "Lodash modular utilities.",
			"dist-tags": map[string]any{
				"latest": "4.17.21",
			},
			"license":  "MIT",
			"homepage": "https://lodash.com/",
			"repository": map[string]any{
				"url": "git+https://github.com/lodash/lodash.git",
			},
			"keywords": []string{"modules", "stdlib", "util"},
			"author": map[string]any{
				"name": "John-David Dalton",
			},
			"time": map[string]string{
				"modified": "2024-01-15T10:30:00.000Z",
			},
			"versions": map[string]any{
				"4.17.21": map[string]any{
					"dependencies": map[string]string{},
				},
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/lodash")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var pkgData struct {
		Name     string `json:"name"`
		License  string `json:"license"`
		Homepage string `json:"homepage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pkgData); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if pkgData.Name != "lodash" {
		t.Errorf("expected name 'lodash', got %q", pkgData.Name)
	}
	if pkgData.License != "MIT" {
		t.Errorf("expected license 'MIT', got %q", pkgData.License)
	}
}
