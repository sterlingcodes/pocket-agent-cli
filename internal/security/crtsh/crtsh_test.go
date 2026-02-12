package crtsh

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "crtsh" {
		t.Errorf("expected Use 'crtsh', got %q", cmd.Use)
	}
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	if !subs["lookup [domain]"] {
		t.Error("missing subcommand 'lookup [domain]'")
	}
}

func TestLookup_Success(t *testing.T) {
	// Mock server returning certificate transparency logs
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("q") != "example.com" {
			t.Errorf("expected q=example.com, got %s", query.Get("q"))
		}
		if query.Get("output") != "json" {
			t.Errorf("expected output=json, got %s", query.Get("output"))
		}
		if query.Get("exclude") != "expired" {
			t.Errorf("expected exclude=expired, got %s", query.Get("exclude"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{
				"id": 123456789,
				"issuer_ca_id": 12345,
				"issuer_name": "C=US, O=Let's Encrypt, CN=R3",
				"common_name": "example.com",
				"name_value": "example.com\n*.example.com",
				"not_before": "2024-01-01T00:00:00",
				"not_after": "2024-04-01T00:00:00",
				"serial_number": "03abcdef123456789"
			}
		]`))
	}))
	defer srv.Close()

	oldBaseURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldBaseURL }()

	cmd := newLookupCmd()
	cmd.SetArgs([]string{"example.com", "--limit", "20"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestLookup_WithExpired(t *testing.T) {
	// Mock server verifying expired flag is not set
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Has("exclude") {
			t.Error("expected no exclude param when --expired is set")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	oldBaseURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldBaseURL }()

	cmd := newLookupCmd()
	cmd.SetArgs([]string{"example.com", "--expired"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestLookup_WithLimit(t *testing.T) {
	// Mock server returning multiple results
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{"id": 1, "issuer_ca_id": 12345, "issuer_name": "CA1", "common_name": "test1.com", "name_value": "test1.com", "not_before": "2024-01-01", "not_after": "2024-12-31", "serial_number": "01"},
			{"id": 2, "issuer_ca_id": 12345, "issuer_name": "CA1", "common_name": "test2.com", "name_value": "test2.com", "not_before": "2024-01-01", "not_after": "2024-12-31", "serial_number": "02"},
			{"id": 3, "issuer_ca_id": 12345, "issuer_name": "CA1", "common_name": "test3.com", "name_value": "test3.com", "not_before": "2024-01-01", "not_after": "2024-12-31", "serial_number": "03"}
		]`))
	}))
	defer srv.Close()

	oldBaseURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldBaseURL }()

	cmd := newLookupCmd()
	cmd.SetArgs([]string{"example.com", "--limit", "2"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
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
	cmd.SetArgs([]string{"example.com"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error, got nil")
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
	cmd.SetArgs([]string{"example.com"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestLookup_EmptyResults(t *testing.T) {
	// Mock server returning empty array
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	oldBaseURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldBaseURL }()

	cmd := newLookupCmd()
	cmd.SetArgs([]string{"nonexistent.example"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error for empty results, got %v", err)
	}
}
