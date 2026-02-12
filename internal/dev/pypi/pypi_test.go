package pypi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "pypi" {
		t.Errorf("expected Use 'pypi', got %q", cmd.Use)
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
		{"short", "test", 10, "test"},
		{"exact", "12345", 5, "12345"},
		{"long", "this is a long string", 10, "this is..."},
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
		{"minutes", now.Add(-10 * time.Minute), "10m"},
		{"hours", now.Add(-5 * time.Hour), "5h"},
		{"days", now.Add(-7 * 24 * time.Hour), "7d"},
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

func TestPypiGetSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"info": map[string]any{
				"name":    "requests",
				"version": "2.31.0",
				"summary": "Python HTTP for Humans.",
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	var result pypiResponse
	err := pypiGet(srv.URL+"/pypi/requests/json", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Info.Name != "requests" {
		t.Errorf("expected name 'requests', got %q", result.Info.Name)
	}
}

func TestPypiGetError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"404", 404},
		{"500", 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer srv.Close()

			var result pypiResponse
			err := pypiGet(srv.URL+"/test", &result)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestPypiInfoResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"info": map[string]any{
				"name":            "flask",
				"version":         "3.0.0",
				"summary":         "A simple framework for building complex web applications.",
				"author":          "Armin Ronacher",
				"license":         "BSD-3-Clause",
				"home_page":       "https://palletsprojects.com/p/flask",
				"keywords":        "wsgi,web,framework",
				"requires_python": ">=3.8",
				"requires_dist": []string{
					"Werkzeug>=3.0.0",
					"Jinja2>=3.1.2",
				},
				"project_urls": map[string]string{
					"Repository": "https://github.com/pallets/flask",
				},
			},
			"releases": map[string][]any{
				"3.0.0": {
					map[string]any{
						"upload_time":     "2024-01-15T10:30:00",
						"requires_python": ">=3.8",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	var result pypiResponse
	err := pypiGet(srv.URL+"/pypi/flask/json", &result)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if result.Info.Name != "flask" {
		t.Errorf("expected name 'flask', got %q", result.Info.Name)
	}
	if result.Info.Version != "3.0.0" {
		t.Errorf("expected version '3.0.0', got %q", result.Info.Version)
	}
	if len(result.Info.RequiresDist) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(result.Info.RequiresDist))
	}
}
