package wifi

import (
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()

	if cmd.Use != "wifi" {
		t.Errorf("expected Use=wifi, got %s", cmd.Use)
	}

	found := false
	for _, a := range cmd.Aliases {
		if a == "wf" {
			found = true
		}
	}
	if !found {
		t.Error("expected alias 'wf'")
	}

	subs := map[string]bool{"scan": false, "current": false}
	for _, sub := range cmd.Commands() {
		subs[sub.Use] = true
	}
	for name, present := range subs {
		if !present {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

func TestParseAirportScanLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		ssid    string
		bssid   string
		rssi    int
		channel int
	}{
		{
			name:    "normal network",
			line:    "                   MyNetwork aa:bb:cc:dd:ee:ff -65  6       Y  -- WPA2(PSK/AES/AES)",
			ssid:    "MyNetwork",
			bssid:   "aa:bb:cc:dd:ee:ff",
			rssi:    -65,
			channel: 6,
		},
		{
			name:    "network with spaces in name",
			line:    "          My Home WiFi 11:22:33:44:55:66 -42  36,+1   Y  -- WPA2(PSK/AES/AES)",
			ssid:    "My Home WiFi",
			bssid:   "11:22:33:44:55:66",
			rssi:    -42,
			channel: 36,
		},
		{
			name: "no BSSID match",
			line: "just some random text",
			ssid: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := parseAirportScanLine(tt.line)
			if n.SSID != tt.ssid {
				t.Errorf("SSID = %q, want %q", n.SSID, tt.ssid)
			}
			if tt.bssid != "" && n.BSSID != tt.bssid {
				t.Errorf("BSSID = %q, want %q", n.BSSID, tt.bssid)
			}
			if n.RSSI != tt.rssi {
				t.Errorf("RSSI = %d, want %d", n.RSSI, tt.rssi)
			}
			if n.Channel != tt.channel {
				t.Errorf("Channel = %d, want %d", n.Channel, tt.channel)
			}
		})
	}
}

func TestParseAirportScan(t *testing.T) {
	input := `                            SSID BSSID             RSSI CHANNEL HT CC SECURITY (auth/unicast/group, 802.1X/EAP)
                   HomeNetwork aa:bb:cc:dd:ee:ff -55  11      Y  -- WPA2(PSK/AES/AES)
                    GuestWiFi  11:22:33:44:55:66 -72  6       Y  -- WPA2(PSK/AES/AES)`

	networks := parseAirportScan(input)

	if len(networks) != 2 {
		t.Fatalf("expected 2 networks, got %d", len(networks))
	}

	if networks[0].SSID != "HomeNetwork" {
		t.Errorf("network 0 SSID = %q, want 'HomeNetwork'", networks[0].SSID)
	}
}

func TestParseAirportScanEmpty(t *testing.T) {
	networks := parseAirportScan("")
	if len(networks) != 0 {
		t.Errorf("expected 0 networks for empty input, got %d", len(networks))
	}
}

func TestParseAirportScanHeaderOnly(t *testing.T) {
	input := `                            SSID BSSID             RSSI CHANNEL HT CC SECURITY`
	networks := parseAirportScan(input)
	if len(networks) != 0 {
		t.Errorf("expected 0 networks for header-only input, got %d", len(networks))
	}
}

func TestParseAirportInfo(t *testing.T) {
	input := `     agrCtlRSSI: -55
     agrExtRSSI: 0
    agrCtlNoise: -90
    agrExtNoise: 0
          state: running
        op mode: station
     lastTxRate: 866
        maxRate: 866
lastAssocStatus: 0
    802.11 auth: open
      link auth: wpa2-psk
          BSSID: aa:bb:cc:dd:ee:ff
           SSID: MyNetwork
            MCS: 9
  guardInterval: 800
            NSS: 2
        channel: 149,80`

	info := parseAirportInfo(input)

	if !info.Connected {
		t.Error("expected connected=true")
	}
	if info.SSID != "MyNetwork" {
		t.Errorf("SSID = %q, want 'MyNetwork'", info.SSID)
	}
	if info.BSSID != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("BSSID = %q, want 'aa:bb:cc:dd:ee:ff'", info.BSSID)
	}
	if info.RSSI != -55 {
		t.Errorf("RSSI = %d, want -55", info.RSSI)
	}
	if info.Noise != -90 {
		t.Errorf("Noise = %d, want -90", info.Noise)
	}
	if info.Channel != 149 {
		t.Errorf("Channel = %d, want 149", info.Channel)
	}
	if info.TxRate != "866 Mbps" {
		t.Errorf("TxRate = %q, want '866 Mbps'", info.TxRate)
	}
	if info.Security != "wpa2-psk" {
		t.Errorf("Security = %q, want 'wpa2-psk'", info.Security)
	}
}

func TestParseAirportInfoDisconnected(t *testing.T) {
	input := `     agrCtlRSSI: 0
    agrCtlNoise: 0
          state: init`

	info := parseAirportInfo(input)
	if info.Connected {
		t.Error("expected connected=false for disconnected state")
	}
	if info.SSID != "" {
		t.Errorf("expected empty SSID, got %q", info.SSID)
	}
}

func TestConnectionInfoTypes(t *testing.T) {
	info := ConnectionInfo{
		SSID:      "TestNet",
		BSSID:     "aa:bb:cc:dd:ee:ff",
		RSSI:      -45,
		Noise:     -90,
		Channel:   36,
		TxRate:    "866 Mbps",
		Security:  "wpa3",
		Connected: true,
	}

	if info.SSID != "TestNet" {
		t.Errorf("expected SSID=TestNet, got %s", info.SSID)
	}
	if !info.Connected {
		t.Error("expected connected=true")
	}
}

func TestScanResultTypes(t *testing.T) {
	result := ScanResult{
		Networks: []Network{
			{SSID: "Net1", RSSI: -50, Channel: 6},
			{SSID: "Net2", RSSI: -70, Channel: 11},
		},
		Count: 2,
	}

	if result.Count != 2 {
		t.Errorf("expected count=2, got %d", result.Count)
	}
	if len(result.Networks) != 2 {
		t.Errorf("expected 2 networks, got %d", len(result.Networks))
	}
}
