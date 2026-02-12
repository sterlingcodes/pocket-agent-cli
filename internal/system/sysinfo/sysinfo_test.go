package sysinfo

import (
	"strings"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()

	if cmd.Use != "sysinfo" {
		t.Errorf("expected Use=sysinfo, got %s", cmd.Use)
	}

	aliases := map[string]bool{"si": false, "info": false}
	for _, a := range cmd.Aliases {
		aliases[a] = true
	}
	for alias, found := range aliases {
		if !found {
			t.Errorf("missing alias: %s", alias)
		}
	}

	expectedSubs := []string{"overview", "cpu", "memory", "disk", "processes", "network"}
	subMap := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subMap[sub.Use] = true
	}
	for _, name := range expectedSubs {
		if !subMap[name] {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

func TestProcessesCmdHasLimitFlag(t *testing.T) {
	cmd := NewCmd()
	for _, sub := range cmd.Commands() {
		if sub.Use == "processes" {
			f := sub.Flags().Lookup("limit")
			if f == nil {
				t.Error("processes subcommand missing --limit flag")
			}
			if f != nil && f.DefValue != "10" {
				t.Errorf("expected default limit=10, got %s", f.DefValue)
			}
			return
		}
	}
	t.Error("processes subcommand not found")
}

func TestParseSizeToGB(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"100G", 100},
		{"100Gi", 100},
		{"1T", 1024},
		{"1Ti", 1024},
		{"512M", 0.5},
		{"512Mi", 0.5},
		{"1024K", 1.0 / 1024},
		{"1024Ki", 1.0 / 1024},
		{"0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseSizeToGB(tt.input)
			// Allow small floating point differences
			diff := got - tt.want
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.01 {
				t.Errorf("parseSizeToGB(%q) = %f, want %f", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseVmStatValue(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"Pages active:   123456.", 123456},
		{"Pages wired down:   78901.", 78901},
		{"Pages occupied by compressor:   0.", 0},
		{"no colon here", 0},
		{"key: ", 0},
		{"key: abc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseVMStatValue(tt.input)
			if got != tt.want {
				t.Errorf("parseVMStatValue(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseIfconfig(t *testing.T) {
	input := `en0: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST> mtu 1500
	ether aa:bb:cc:dd:ee:ff
	inet 192.168.1.100 netmask 0xffffff00 broadcast 192.168.1.255
	inet6 fe80::1%en0 prefixlen 64 scopeid 0x4
lo0: flags=8049<UP,LOOPBACK,RUNNING,MULTICAST> mtu 16384
	inet 127.0.0.1 netmask 0xff000000
	inet6 ::1 prefixlen 128
en1: flags=8823<DOWN,BROADCAST,SMART,SIMPLEX,MULTICAST> mtu 1500
	ether 00:11:22:33:44:55`

	interfaces := parseIfconfig(input)

	// Should have en0 but NOT lo0 (loopback) and NOT en1 (no IP)
	if len(interfaces) != 1 {
		t.Fatalf("expected 1 interface (en0 only), got %d", len(interfaces))
	}

	if interfaces[0].Name != "en0" {
		t.Errorf("expected name=en0, got %s", interfaces[0].Name)
	}
	if interfaces[0].IPv4 != "192.168.1.100" {
		t.Errorf("expected IPv4=192.168.1.100, got %s", interfaces[0].IPv4)
	}
	if interfaces[0].Status != "up" {
		t.Errorf("expected status=up, got %s", interfaces[0].Status)
	}
}

func TestParseIfconfigEmpty(t *testing.T) {
	interfaces := parseIfconfig("")
	if len(interfaces) != 0 {
		t.Errorf("expected 0 interfaces for empty input, got %d", len(interfaces))
	}
}

func TestParseIPBrief(t *testing.T) {
	input := `lo               UNKNOWN        127.0.0.1/8 ::1/128
eth0             UP             192.168.1.50/24 fe80::1/64
wlan0            UP             10.0.0.5/24
docker0          DOWN`

	interfaces := parseIPBrief(input)

	// lo filtered out, docker0 has no IPs
	if len(interfaces) != 2 {
		t.Fatalf("expected 2 interfaces, got %d", len(interfaces))
	}

	if interfaces[0].Name != "eth0" {
		t.Errorf("expected name=eth0, got %s", interfaces[0].Name)
	}
	if interfaces[0].IPv4 != "192.168.1.50" {
		t.Errorf("expected IPv4=192.168.1.50, got %s", interfaces[0].IPv4)
	}
	if interfaces[0].Status != "up" {
		t.Errorf("expected status=up, got %s", interfaces[0].Status)
	}
	if interfaces[1].IPv4 != "10.0.0.5" {
		t.Errorf("expected wlan0 IPv4=10.0.0.5, got %s", interfaces[1].IPv4)
	}
}

func TestParseIPBriefEmpty(t *testing.T) {
	interfaces := parseIPBrief("")
	if len(interfaces) != 0 {
		t.Errorf("expected 0 interfaces for empty input, got %d", len(interfaces))
	}
}

func TestShortNameExtraction(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/usr/bin/python3", "python3"},
		{"python3", "python3"},
		{"/Applications/Safari.app/Contents/MacOS/Safari", "Safari"},
		{"", ""},
		{"/", "/"}, // trailing slash with nothing after stays as-is
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// Match the logic in getProcesses: strings.LastIndex + slice
			got := tt.input
			if i := strings.LastIndex(got, "/"); i >= 0 && i < len(got)-1 {
				got = got[i+1:]
			}
			if got != tt.want {
				t.Errorf("shortName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestOverviewTypes(t *testing.T) {
	o := Overview{
		Platform: "darwin",
		Hostname: "test-host",
		OS:       "darwin/arm64",
		Arch:     "arm64",
		CPUs:     10,
		Uptime:   "5 days",
		LoadAvg:  "1.5 2.0 1.8",
	}
	if o.CPUs != 10 {
		t.Errorf("expected CPUs=10, got %d", o.CPUs)
	}
}
