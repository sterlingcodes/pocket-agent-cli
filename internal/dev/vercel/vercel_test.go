package vercel

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "vercel" {
		t.Errorf("expected Use 'vercel', got %q", cmd.Use)
	}
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Name()] = true
	}
	for _, name := range []string{"projects", "project", "deployments", "deployment", "domains", "env"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestGetString(t *testing.T) {
	tests := []struct {
		m        map[string]any
		key      string
		expected string
	}{
		{map[string]any{"name": "test"}, "name", "test"},
		{map[string]any{}, "name", ""},
		{map[string]any{"name": 123}, "name", ""},
	}
	for _, tt := range tests {
		result := getString(tt.m, tt.key)
		if result != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, result)
		}
	}
}

func TestGetInt64(t *testing.T) {
	tests := []struct {
		m        map[string]any
		key      string
		expected int64
	}{
		{map[string]any{"timestamp": float64(1000000)}, "timestamp", 1000000},
		{map[string]any{}, "timestamp", 0},
	}
	for _, tt := range tests {
		result := getInt64(tt.m, tt.key)
		if result != tt.expected {
			t.Errorf("expected %d, got %d", tt.expected, result)
		}
	}
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		m        map[string]any
		key      string
		expected bool
	}{
		{map[string]any{"verified": true}, "verified", true},
		{map[string]any{"verified": false}, "verified", false},
		{map[string]any{}, "verified", false},
	}
	for _, tt := range tests {
		result := getBool(tt.m, tt.key)
		if result != tt.expected {
			t.Errorf("expected %v, got %v", tt.expected, result)
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
		{now.Add(-10 * time.Minute), "10m"},
		{now.Add(-4 * time.Hour), "4h"},
		{now.Add(-5 * 24 * time.Hour), "5d"},
	}
	for _, tt := range tests {
		result := timeAgo(tt.t)
		if result != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, result)
		}
	}
}

func TestVcGetSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("expected Authorization header")
		}
		json.NewEncoder(w).Encode(map[string]any{"id": "prj_123", "name": "my-project"})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	var result map[string]any
	err := vcGet("test-token", srv.URL+"/v9/projects/my-project", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result["name"] != "my-project" {
		t.Errorf("expected name 'my-project', got %v", result["name"])
	}
}

func TestVcGetError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"401", 401},
		{"404", 404},
		{"500", 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]any{"message": "Error"},
				})
			}))
			defer srv.Close()

			var result map[string]any
			err := vcGet("test-token", srv.URL+"/test", &result)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestToProject(t *testing.T) {
	projectData := map[string]any{
		"id":        "prj_abc123",
		"name":      "my-website",
		"framework": "nextjs",
		"updatedAt": float64(time.Now().Add(-2 * time.Hour).UnixMilli()),
	}

	proj := toProject(projectData)
	if proj.ID != "prj_abc123" {
		t.Errorf("expected ID 'prj_abc123', got %q", proj.ID)
	}
	if proj.Name != "my-website" {
		t.Errorf("expected name 'my-website', got %q", proj.Name)
	}
	if proj.Framework != "nextjs" {
		t.Errorf("expected framework 'nextjs', got %q", proj.Framework)
	}
	if proj.URL == "" {
		t.Error("expected URL to be set")
	}
}

func TestToDeployment(t *testing.T) {
	depData := map[string]any{
		"uid":     "dpl_xyz789",
		"name":    "my-website",
		"url":     "my-website-abc.vercel.app",
		"state":   "READY",
		"created": float64(time.Now().Add(-1 * time.Hour).UnixMilli()),
		"target":  "production",
	}

	dep := toDeployment(depData)
	if dep.ID != "dpl_xyz789" {
		t.Errorf("expected ID 'dpl_xyz789', got %q", dep.ID)
	}
	if dep.State != "READY" {
		t.Errorf("expected state 'READY', got %q", dep.State)
	}
	if dep.Target != "production" {
		t.Errorf("expected target 'production', got %q", dep.Target)
	}
	if dep.URL[:8] != "https://" {
		t.Errorf("expected URL to start with https://, got %q", dep.URL)
	}
}

func TestToDomain(t *testing.T) {
	domData := map[string]any{
		"name":       "example.com",
		"configured": true,
		"verified":   true,
		"createdAt":  float64(time.Now().Add(-7 * 24 * time.Hour).UnixMilli()),
	}

	dom := toDomain(domData)
	if dom.Name != "example.com" {
		t.Errorf("expected name 'example.com', got %q", dom.Name)
	}
	if !dom.Configured {
		t.Error("expected configured to be true")
	}
	if !dom.Verified {
		t.Error("expected verified to be true")
	}
}

func TestToEnvVar(t *testing.T) {
	envData := map[string]any{
		"id":   "env_123",
		"key":  "API_KEY",
		"type": "encrypted",
		"target": []any{
			"production",
			"preview",
		},
	}

	env := toEnvVar(envData)
	if env.ID != "env_123" {
		t.Errorf("expected ID 'env_123', got %q", env.ID)
	}
	if env.Key != "API_KEY" {
		t.Errorf("expected key 'API_KEY', got %q", env.Key)
	}
	if len(env.Targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(env.Targets))
	}
}
