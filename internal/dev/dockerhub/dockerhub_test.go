package dockerhub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "dockerhub" {
		t.Errorf("expected Use 'dockerhub', got %q", cmd.Use)
	}
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Name()] = true
	}
	for _, name := range []string{"search", "image", "tags", "inspect"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
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
		{"exactly five", 13, "exactly five"},
		{"this is too long", 10, "this is..."},
	}
	for _, tt := range tests {
		result := truncate(tt.s, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, result, tt.expected)
		}
	}
}

func TestTruncateDigest(t *testing.T) {
	tests := []struct {
		digest   string
		expected string
	}{
		{"sha256:abc123def456", "sha256:abc123def456"},
		{"sha256:abc123def456ghi789jkl012mno345pqr678stu901", "sha256:abc123def456..."},
		{"short", "short"},
	}
	for _, tt := range tests {
		result := truncateDigest(tt.digest)
		if result != tt.expected {
			t.Errorf("truncateDigest(%q) = %q, want %q", tt.digest, result, tt.expected)
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
		{now.Add(-5 * time.Minute), "5m"},
		{now.Add(-3 * time.Hour), "3h"},
		{now.Add(-10 * 24 * time.Hour), "10d"},
	}
	for _, tt := range tests {
		result := timeAgo(tt.t)
		if result != tt.expected {
			t.Errorf("timeAgo() = %q, want %q", result, tt.expected)
		}
	}
}

func TestNormalizeImageName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"nginx", "library/nginx"},
		{"library/nginx", "library/nginx"},
		{"user/image", "user/image"},
	}
	for _, tt := range tests {
		result := normalizeImageName(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeImageName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestParseNameTag(t *testing.T) {
	tests := []struct {
		input        string
		expectedName string
		expectedTag  string
	}{
		{"nginx", "nginx", "latest"},
		{"nginx:1.21", "nginx", "1.21"},
		{"user/image:tag", "user/image", "tag"},
	}
	for _, tt := range tests {
		name, tag := parseNameTag(tt.input)
		if name != tt.expectedName || tag != tt.expectedTag {
			t.Errorf("parseNameTag(%q) = (%q, %q), want (%q, %q)", tt.input, name, tag, tt.expectedName, tt.expectedTag)
		}
	}
}

func TestDockerHubSearchResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"results": []map[string]any{
				{
					"repo_name":         "nginx",
					"short_description": "Official build of Nginx.",
					"star_count":        float64(15000),
					"is_official":       true,
					"is_automated":      false,
				},
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	resp, err := http.Get(srv.URL + "/search/repositories/?query=nginx")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var data struct {
		Results []struct {
			RepoName  string `json:"repo_name"`
			StarCount int    `json:"star_count"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if len(data.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(data.Results))
	}
	if data.Results[0].RepoName != "nginx" {
		t.Errorf("expected repo_name 'nginx', got %q", data.Results[0].RepoName)
	}
}

func TestDockerHubImageResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"name":            "nginx",
			"namespace":       "library",
			"description":     "Official build of Nginx.",
			"star_count":      float64(15000),
			"pull_count":      float64(1000000000),
			"is_automated":    false,
			"last_updated":    "2024-01-15T10:30:00.000Z",
			"repository_type": "image",
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/repositories/library/nginx/")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var data struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		StarCount int    `json:"star_count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if data.Name != "nginx" {
		t.Errorf("expected name 'nginx', got %q", data.Name)
	}
	if data.Namespace != "library" {
		t.Errorf("expected namespace 'library', got %q", data.Namespace)
	}
}
