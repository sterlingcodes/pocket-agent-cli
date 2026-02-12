package shodan

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "shodan" {
		t.Errorf("expected Use 'shodan', got %q", cmd.Use)
	}
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	if !subs["lookup [ip]"] {
		t.Error("missing subcommand 'lookup [ip]'")
	}
}

func TestLookup_Success(t *testing.T) {
	// Mock server returning InternetDB response
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/8.8.8.8" {
			t.Errorf("expected path /8.8.8.8, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"ip": "8.8.8.8",
			"ports": [53, 443],
			"hostnames": ["dns.google"],
			"cpes": ["cpe:/a:vendor:product"],
			"tags": ["cloud"],
			"vulns": []
		}`))
	}))
	defer srv.Close()

	oldBaseURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldBaseURL }()

	cmd := newLookupCmd()
	cmd.SetArgs([]string{"8.8.8.8"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestLookup_NotFound(t *testing.T) {
	// Mock server returning 404
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not found"))
	}))
	defer srv.Close()

	oldBaseURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldBaseURL }()

	cmd := newLookupCmd()
	cmd.SetArgs([]string{"192.168.1.1"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestLookup_InvalidIP(t *testing.T) {
	cmd := newLookupCmd()
	cmd.SetArgs([]string{"not-an-ip"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for invalid IP, got nil")
	}
}

func TestLookup_APIError(t *testing.T) {
	// Mock server returning error
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server error"))
	}))
	defer srv.Close()

	oldBaseURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldBaseURL }()

	cmd := newLookupCmd()
	cmd.SetArgs([]string{"1.2.3.4"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestLookup_EmptyResponse(t *testing.T) {
	// Mock server returning empty JSON object with nil slices
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"ip": "1.2.3.4",
			"ports": null,
			"hostnames": null,
			"cpes": null,
			"tags": null,
			"vulns": null
		}`))
	}))
	defer srv.Close()

	oldBaseURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldBaseURL }()

	cmd := newLookupCmd()
	cmd.SetArgs([]string{"1.2.3.4"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestLookup_InvalidJSON(t *testing.T) {
	// Mock server returning invalid JSON
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	oldBaseURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldBaseURL }()

	cmd := newLookupCmd()
	cmd.SetArgs([]string{"1.2.3.4"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error, got nil")
	}
}
