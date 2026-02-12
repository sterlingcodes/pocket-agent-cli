package domain

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "domain" {
		t.Errorf("expected Use 'domain', got %q", cmd.Use)
	}

	aliases := cmd.Aliases
	if len(aliases) != 3 {
		t.Errorf("expected 3 aliases, got %d", len(aliases))
	}

	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"dns [domain]", "whois [domain]", "ssl [domain]"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestGetString(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
		key  string
		want string
	}{
		{
			name: "string value",
			data: map[string]any{"key": "value"},
			key:  "key",
			want: "value",
		},
		{
			name: "array of strings",
			data: map[string]any{"key": []any{"val1", "val2"}},
			key:  "key",
			want: "val1, val2",
		},
		{
			name: "missing key",
			data: map[string]any{},
			key:  "missing",
			want: "",
		},
		{
			name: "empty array",
			data: map[string]any{"key": []any{}},
			key:  "key",
			want: "",
		},
		{
			name: "non-string value",
			data: map[string]any{"key": 123},
			key:  "key",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getString(tt.data, tt.key)
			if got != tt.want {
				t.Errorf("getString(%v, %q) = %q, want %q", tt.data, tt.key, got, tt.want)
			}
		})
	}
}

func TestCleanDomain(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"example.com", "example.com"},
		{"http://example.com", "example.com"},
		{"https://example.com", "example.com"},
		{"https://example.com/", "example.com"},
		{"https://example.com/path", "example.com"},
		{"example.com/path", "example.com"},
	}

	for _, tt := range tests {
		got := cleanDomain(tt.input)
		if got != tt.want {
			t.Errorf("cleanDomain(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDNSTypeToString(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{1, "A"},
		{2, "NS"},
		{5, "CNAME"},
		{6, "SOA"},
		{15, "MX"},
		{16, "TXT"},
		{28, "AAAA"},
		{999, "TYPE999"},
	}

	for _, tt := range tests {
		got := dnsTypeToString(tt.input)
		if got != tt.want {
			t.Errorf("dnsTypeToString(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDNSLookup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"Status": 0,
			"Question": []map[string]any{
				{"name": "example.com.", "type": 1},
			},
			"Answer": []map[string]any{
				{
					"name": "example.com.",
					"type": 1,
					"TTL":  3600,
					"data": "93.184.216.34",
				},
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := dnsURL
	dnsURL = srv.URL
	defer func() { dnsURL = oldURL }()

	records, err := dnsLookup("example.com", "A")
	if err != nil {
		t.Errorf("dnsLookup failed: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
	if records[0].Type != "A" {
		t.Errorf("expected type A, got %q", records[0].Type)
	}
	if records[0].Value != "93.184.216.34" {
		t.Errorf("expected value 93.184.216.34, got %q", records[0].Value)
	}
}

func TestDNSLookupNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"Status": 3,
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := dnsURL
	dnsURL = srv.URL
	defer func() { dnsURL = oldURL }()

	_, err := dnsLookup("invalid.example", "A")
	if err == nil {
		t.Error("expected error for not found domain, got nil")
	}
}

func TestWhoisCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"registrar":       "Example Registrar",
			"creation_date":   "2000-01-01",
			"expiration_date": "2030-01-01",
			"updated_date":    "2024-01-01",
			"status":          "clientTransferProhibited",
			"name_servers":    "ns1.example.com, ns2.example.com",
			"registrant":      "Example Inc",
			"country":         "US",
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := whoisURL
	whoisURL = srv.URL
	defer func() { whoisURL = oldURL }()

	cmd := newWhoisCmd()
	cmd.SetArgs([]string{"example.com"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("whois command failed: %v", err)
	}
}

func TestWhoisError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"status":  "error",
			"message": "Domain not found",
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := whoisURL
	whoisURL = srv.URL
	defer func() { whoisURL = oldURL }()

	cmd := newWhoisCmd()
	cmd.SetArgs([]string{"invalid.example"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for whois lookup, got nil")
	}
}

func TestDNSCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"Status": 0,
			"Answer": []map[string]any{
				{
					"name": "example.com.",
					"type": 1,
					"TTL":  3600,
					"data": "93.184.216.34",
				},
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := dnsURL
	dnsURL = srv.URL
	defer func() { dnsURL = oldURL }()

	cmd := newDNSCmd()
	cmd.SetArgs([]string{"example.com", "--type", "A"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("dns command failed: %v", err)
	}
}

func TestRateLimitHandling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": "rate limited"})
	}))
	defer srv.Close()

	oldURL := whoisURL
	whoisURL = srv.URL
	defer func() { whoisURL = oldURL }()

	cmd := newWhoisCmd()
	cmd.SetArgs([]string{"example.com"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected rate limit error, got nil")
	}
}
