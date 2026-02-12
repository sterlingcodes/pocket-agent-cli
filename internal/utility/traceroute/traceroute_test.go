package traceroute

import (
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()

	if cmd.Use != "traceroute" {
		t.Errorf("expected Use=traceroute, got %s", cmd.Use)
	}

	aliases := map[string]bool{"trace": false, "tr": false}
	for _, a := range cmd.Aliases {
		aliases[a] = true
	}
	for alias, found := range aliases {
		if !found {
			t.Errorf("missing alias: %s", alias)
		}
	}

	// Check run subcommand exists
	found := false
	for _, sub := range cmd.Commands() {
		if sub.Use == "run [host]" {
			found = true
		}
	}
	if !found {
		t.Error("missing 'run' subcommand")
	}
}

func TestParseTracerouteNormal(t *testing.T) {
	input := `traceroute to 8.8.8.8 (8.8.8.8), 30 hops max, 60 byte packets
 1  192.168.1.1  1.234 ms  0.987 ms  1.123 ms
 2  10.0.0.1  5.678 ms  4.321 ms  5.555 ms
 3  * * *
 4  dns.google (8.8.8.8)  12.345 ms  11.222 ms  12.001 ms`

	hops := parseTraceroute(input)

	if len(hops) != 4 {
		t.Fatalf("expected 4 hops, got %d", len(hops))
	}

	// Hop 1
	if hops[0].Number != 1 {
		t.Errorf("hop 1: expected number 1, got %d", hops[0].Number)
	}
	if hops[0].IP != "192.168.1.1" {
		t.Errorf("hop 1: expected IP 192.168.1.1, got %s", hops[0].IP)
	}
	if len(hops[0].RTTs) != 3 {
		t.Errorf("hop 1: expected 3 RTTs, got %d", len(hops[0].RTTs))
	}
	if hops[0].Timeout {
		t.Error("hop 1: should not be timeout")
	}

	// Hop 3 - timeout
	if !hops[2].Timeout {
		t.Error("hop 3: expected timeout")
	}
	if hops[2].Number != 3 {
		t.Errorf("hop 3: expected number 3, got %d", hops[2].Number)
	}

	// Hop 4 - hostname with IP
	if hops[3].Host != "dns.google" {
		t.Errorf("hop 4: expected host dns.google, got %s", hops[3].Host)
	}
	if hops[3].IP != "8.8.8.8" {
		t.Errorf("hop 4: expected IP 8.8.8.8, got %s", hops[3].IP)
	}
}

func TestParseTracerouteEmpty(t *testing.T) {
	hops := parseTraceroute("")
	if len(hops) != 0 {
		t.Errorf("expected 0 hops for empty input, got %d", len(hops))
	}
}

func TestParseTracerouteHeaderOnly(t *testing.T) {
	input := `traceroute to 8.8.8.8 (8.8.8.8), 30 hops max, 60 byte packets`
	hops := parseTraceroute(input)
	if len(hops) != 0 {
		t.Errorf("expected 0 hops for header-only input, got %d", len(hops))
	}
}

func TestParseTracerouteAllTimeouts(t *testing.T) {
	input := ` 1  * * *
 2  * * *
 3  * * *`

	hops := parseTraceroute(input)
	if len(hops) != 3 {
		t.Fatalf("expected 3 hops, got %d", len(hops))
	}
	for i, hop := range hops {
		if !hop.Timeout {
			t.Errorf("hop %d: expected timeout", i+1)
		}
	}
}

func TestParseTracerouteIPOnly(t *testing.T) {
	input := ` 1  10.0.0.1  2.345 ms  1.234 ms  2.000 ms`

	hops := parseTraceroute(input)
	if len(hops) != 1 {
		t.Fatalf("expected 1 hop, got %d", len(hops))
	}
	if hops[0].IP != "10.0.0.1" {
		t.Errorf("expected IP 10.0.0.1, got %s", hops[0].IP)
	}
}

func TestParseTracerouteRTTExtraction(t *testing.T) {
	input := ` 1  192.168.1.1  1.234 ms  0.987 ms  1.123 ms`

	hops := parseTraceroute(input)
	if len(hops) != 1 {
		t.Fatalf("expected 1 hop, got %d", len(hops))
	}

	expectedRTTs := []string{"1.234 ms", "0.987 ms", "1.123 ms"}
	if len(hops[0].RTTs) != len(expectedRTTs) {
		t.Fatalf("expected %d RTTs, got %d", len(expectedRTTs), len(hops[0].RTTs))
	}
	for i, rtt := range hops[0].RTTs {
		if rtt != expectedRTTs[i] {
			t.Errorf("RTT[%d]: expected %s, got %s", i, expectedRTTs[i], rtt)
		}
	}
}

func TestTraceResultComplete(t *testing.T) {
	tests := []struct {
		name     string
		hops     []Hop
		complete bool
	}{
		{
			name:     "empty",
			hops:     []Hop{},
			complete: false,
		},
		{
			name: "last hop has IP",
			hops: []Hop{
				{Number: 1, IP: "10.0.0.1"},
				{Number: 2, IP: "8.8.8.8"},
			},
			complete: true,
		},
		{
			name: "last hop timeout",
			hops: []Hop{
				{Number: 1, IP: "10.0.0.1"},
				{Number: 2, Timeout: true},
			},
			complete: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			complete := false
			if len(tt.hops) > 0 {
				last := tt.hops[len(tt.hops)-1]
				complete = !last.Timeout && last.IP != ""
			}
			if complete != tt.complete {
				t.Errorf("expected complete=%v, got %v", tt.complete, complete)
			}
		})
	}
}

func TestMaxHopsBounds(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{0, 30},
		{-1, 30},
		{256, 30},
		{1, 1},
		{255, 255},
		{30, 30},
	}

	for _, tt := range tests {
		maxHops := tt.input
		if maxHops < 1 || maxHops > 255 {
			maxHops = 30
		}
		if maxHops != tt.want {
			t.Errorf("maxHops(%d) = %d, want %d", tt.input, maxHops, tt.want)
		}
	}
}

func TestHopRegex(t *testing.T) {
	tests := []struct {
		line  string
		match bool
	}{
		{" 1  192.168.1.1  1.234 ms", true},
		{"10  10.0.0.1  5.0 ms", true},
		{"traceroute to 8.8.8.8", false},
		{"", false},
		{" 1  * * *", true},
	}

	for _, tt := range tests {
		m := hopRegex.FindStringSubmatch(tt.line)
		got := m != nil
		if got != tt.match {
			t.Errorf("hopRegex(%q) match=%v, want %v", tt.line, got, tt.match)
		}
	}
}

func TestIPRegex(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"192.168.1.1  1.234 ms", "192.168.1.1"},
		{"10.0.0.1  5.678 ms", "10.0.0.1"},
		{"* * *", ""},
	}

	for _, tt := range tests {
		m := ipRegex.FindStringSubmatch(tt.input)
		got := ""
		if m != nil {
			got = m[1]
		}
		if got != tt.want {
			t.Errorf("ipRegex(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
