package dnsbench

import (
	"net"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()

	if cmd.Use != "dnsbench" {
		t.Errorf("expected Use=dnsbench, got %s", cmd.Use)
	}

	// Check aliases
	aliases := map[string]bool{"dnsb": false, "dns-bench": false}
	for _, a := range cmd.Aliases {
		aliases[a] = true
	}
	for alias, found := range aliases {
		if !found {
			t.Errorf("missing alias: %s", alias)
		}
	}

	// Check subcommands
	subs := map[string]bool{"run": false, "test [resolver-ip]": false}
	for _, sub := range cmd.Commands() {
		subs[sub.Use] = true
	}
	for name, present := range subs {
		if !present {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

func TestDefaultResolvers(t *testing.T) {
	if len(defaultResolvers) == 0 {
		t.Fatal("expected at least one default resolver")
	}

	for _, r := range defaultResolvers {
		if r.Name == "" {
			t.Error("resolver has empty name")
		}
		// Validate address format (host:port)
		host, port, err := net.SplitHostPort(r.Address)
		if err != nil {
			t.Errorf("resolver %s has invalid address %s: %v", r.Name, r.Address, err)
		}
		if host == "" {
			t.Errorf("resolver %s has empty host", r.Name)
		}
		if port != "53" {
			t.Errorf("resolver %s expected port 53, got %s", r.Name, port)
		}
	}
}

func TestTestDomains(t *testing.T) {
	if len(testDomains) == 0 {
		t.Fatal("expected at least one test domain")
	}

	for _, d := range testDomains {
		if d == "" {
			t.Error("test domain is empty")
		}
		// Should not have protocol prefix
		if len(d) > 8 && d[:8] == "https://" {
			t.Errorf("test domain should not have protocol: %s", d)
		}
	}
}

func TestBenchResultTypes(t *testing.T) {
	// Verify types work correctly
	result := BenchResult{
		Name:        "TestDNS",
		Address:     "1.2.3.4:53",
		AvgMs:       15.5,
		MinMs:       10.0,
		MaxMs:       20.0,
		SuccessRate: 100.0,
		Queries:     5,
		Failures:    0,
	}

	if result.SuccessRate != 100.0 {
		t.Errorf("expected success rate 100.0, got %f", result.SuccessRate)
	}
	if result.Failures != 0 {
		t.Errorf("expected 0 failures, got %d", result.Failures)
	}
}

func TestBenchReportTypes(t *testing.T) {
	report := BenchReport{
		Resolvers: []BenchResult{
			{Name: "Fast", AvgMs: 5.0, SuccessRate: 100},
			{Name: "Slow", AvgMs: 50.0, SuccessRate: 80},
		},
		Fastest:    "Fast",
		TestDomain: "multiple domains",
		TestedAt:   "2025-01-01T00:00:00Z",
	}

	if report.Fastest != "Fast" {
		t.Errorf("expected fastest=Fast, got %s", report.Fastest)
	}
	if len(report.Resolvers) != 2 {
		t.Errorf("expected 2 resolvers, got %d", len(report.Resolvers))
	}
}

func TestQueryResultTypes(t *testing.T) {
	qr := QueryResult{
		Domain: "example.com",
		TimeMs: 12.5,
		IPs:    []string{"93.184.216.34"},
	}
	if qr.Domain != "example.com" {
		t.Errorf("expected domain example.com, got %s", qr.Domain)
	}
	if len(qr.IPs) != 1 {
		t.Errorf("expected 1 IP, got %d", len(qr.IPs))
	}

	// Error case
	qrErr := QueryResult{
		Domain: "fail.example",
		TimeMs: 5000,
		Error:  "timeout",
	}
	if qrErr.Error != "timeout" {
		t.Errorf("expected error 'timeout', got %s", qrErr.Error)
	}
}

func TestAddressPortAppend(t *testing.T) {
	// testResolver adds :53 if not present - test the logic
	tests := []struct {
		input string
		want  string
	}{
		{"1.1.1.1", "1.1.1.1:53"},
		{"1.1.1.1:53", "1.1.1.1:53"},
		{"8.8.8.8:5353", "8.8.8.8:5353"},
		{"[::1]", "[::1]:53"},
	}

	for _, tt := range tests {
		got := tt.input
		if _, _, err := net.SplitHostPort(got); err != nil {
			got += ":53"
		}
		if got != tt.want {
			t.Errorf("address(%s) = %s, want %s", tt.input, got, tt.want)
		}
	}
}
