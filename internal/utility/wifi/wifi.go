package wifi

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/pkg/output"
)

// Network represents a WiFi network
type Network struct {
	SSID     string `json:"ssid"`
	BSSID    string `json:"bssid,omitempty"`
	RSSI     int    `json:"rssi,omitempty"`
	Channel  int    `json:"channel,omitempty"`
	Security string `json:"security,omitempty"`
}

// ScanResult holds WiFi scan results
type ScanResult struct {
	Networks []Network `json:"networks"`
	Count    int       `json:"count"`
}

// ConnectionInfo holds current WiFi connection details
type ConnectionInfo struct {
	SSID      string `json:"ssid"`
	BSSID     string `json:"bssid,omitempty"`
	RSSI      int    `json:"rssi,omitempty"`
	Noise     int    `json:"noise,omitempty"`
	Channel   int    `json:"channel,omitempty"`
	TxRate    string `json:"tx_rate,omitempty"`
	Security  string `json:"security,omitempty"`
	Connected bool   `json:"connected"`
}

const airportPath = "/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport"

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "wifi",
		Aliases: []string{"wf"},
		Short:   "WiFi network analysis commands",
	}

	cmd.AddCommand(newScanCmd())
	cmd.AddCommand(newCurrentCmd())

	return cmd
}

func newScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Scan nearby WiFi networks with signal strength",
		RunE: func(cmd *cobra.Command, args []string) error {
			return scanNetworks()
		},
	}
}

func newCurrentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show current WiFi connection details",
		RunE: func(cmd *cobra.Command, args []string) error {
			return currentConnection()
		},
	}
}

func scanNetworks() error {
	switch runtime.GOOS {
	case "darwin":
		return scanDarwin()
	case "linux":
		return scanLinux()
	default:
		return output.PrintError("platform_unsupported",
			fmt.Sprintf("WiFi scan not supported on %s", runtime.GOOS),
			map[string]string{"supported": "macOS, Linux"})
	}
}

func currentConnection() error {
	switch runtime.GOOS {
	case "darwin":
		return currentDarwin()
	case "linux":
		return currentLinux()
	default:
		return output.PrintError("platform_unsupported",
			fmt.Sprintf("WiFi info not supported on %s", runtime.GOOS),
			map[string]string{"supported": "macOS, Linux"})
	}
}

// macOS implementation using airport utility
func scanDarwin() error {
	out, err := exec.Command(airportPath, "-s").CombinedOutput()
	if err != nil {
		return output.PrintError("wifi_scan_error",
			fmt.Sprintf("airport scan failed: %v", err),
			map[string]string{"suggestion": "WiFi may be disabled"})
	}

	networks := parseAirportScan(string(out))

	return output.Print(ScanResult{
		Networks: networks,
		Count:    len(networks),
	})
}

func currentDarwin() error {
	out, err := exec.Command(airportPath, "-I").CombinedOutput()
	if err != nil {
		return output.PrintError("wifi_info_error",
			fmt.Sprintf("airport info failed: %v", err),
			map[string]string{"suggestion": "WiFi may be disabled"})
	}

	info := parseAirportInfo(string(out))
	return output.Print(info)
}

func parseAirportScan(data string) []Network {
	var networks []Network
	scanner := bufio.NewScanner(strings.NewReader(data))

	// Skip header line
	scanner.Scan()

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		n := parseAirportScanLine(line)
		if n.SSID != "" || n.BSSID != "" {
			networks = append(networks, n)
		}
	}
	// scanner.Err() always nil for strings.NewReader, but good practice
	_ = scanner.Err()

	return networks
}

func parseAirportScanLine(line string) Network {
	// airport -s output format (fixed-width columns):
	// SSID                 BSSID             RSSI CHANNEL HT CC SECURITY
	// The SSID can be up to 32 chars and right-aligned to column 33

	n := Network{}

	// Find BSSID pattern to determine column positions
	bssidRegex := regexp.MustCompile(`([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}`)
	bssidLoc := bssidRegex.FindStringIndex(line)
	if bssidLoc == nil {
		return n
	}

	n.SSID = strings.TrimSpace(line[:bssidLoc[0]])
	n.BSSID = strings.TrimSpace(line[bssidLoc[0]:bssidLoc[1]])

	// Parse remaining fields after BSSID
	rest := strings.TrimSpace(line[bssidLoc[1]:])
	fields := strings.Fields(rest)

	if len(fields) >= 1 {
		if rssi, err := strconv.Atoi(fields[0]); err == nil {
			n.RSSI = rssi
		}
	}
	if len(fields) >= 2 {
		// Channel may be like "6" or "36,+1"
		chStr := strings.Split(fields[1], ",")[0]
		if ch, err := strconv.Atoi(chStr); err == nil {
			n.Channel = ch
		}
	}
	if len(fields) >= 5 {
		n.Security = strings.Join(fields[4:], " ")
	}

	return n
}

func parseAirportInfo(data string) ConnectionInfo {
	info := ConnectionInfo{}
	scanner := bufio.NewScanner(strings.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "SSID":
			info.SSID = val
			info.Connected = true
		case "BSSID":
			info.BSSID = val
		case "agrCtlRSSI":
			if v, err := strconv.Atoi(val); err == nil {
				info.RSSI = v
			}
		case "agrCtlNoise":
			if v, err := strconv.Atoi(val); err == nil {
				info.Noise = v
			}
		case "channel":
			chStr := strings.Split(val, ",")[0]
			if v, err := strconv.Atoi(chStr); err == nil {
				info.Channel = v
			}
		case "lastTxRate":
			info.TxRate = val + " Mbps"
		case "link auth":
			info.Security = val
		}
	}

	return info
}

// Linux implementation using nmcli
func scanLinux() error {
	out, err := exec.Command("nmcli", "-t", "-f", "SSID,BSSID,SIGNAL,CHAN,SECURITY", "dev", "wifi", "list").CombinedOutput()
	if err != nil {
		return output.PrintError("wifi_scan_error",
			fmt.Sprintf("nmcli scan failed: %v", err),
			map[string]string{"suggestion": "Ensure NetworkManager is installed and WiFi is enabled"})
	}

	var networks []Network
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, ":", 5)
		if len(fields) < 5 {
			continue
		}

		n := Network{
			SSID:     fields[0],
			BSSID:    fields[1],
			Security: fields[4],
		}
		if sig, err := strconv.Atoi(fields[2]); err == nil {
			// nmcli reports signal strength as percentage, convert to approximate dBm
			n.RSSI = sig - 100
		}
		if ch, err := strconv.Atoi(fields[3]); err == nil {
			n.Channel = ch
		}
		networks = append(networks, n)
	}

	return output.Print(ScanResult{
		Networks: networks,
		Count:    len(networks),
	})
}

func currentLinux() error {
	out, err := exec.Command("nmcli", "-t", "-f", "GENERAL.CONNECTION,WIFI.SSID,WIFI.BSSID,WIFI.CHAN,WIFI.RATE,WIFI.SIGNAL,WIFI.SECURITY", "dev", "show", "wlan0").CombinedOutput()
	if err != nil {
		// Try common alternative interface names
		out, err = exec.Command("nmcli", "-t", "-f", "active,ssid,bssid,signal,chan,security", "dev", "wifi").CombinedOutput()
		if err != nil {
			return output.PrintError("wifi_info_error",
				fmt.Sprintf("nmcli failed: %v", err), nil)
		}
	}

	info := ConnectionInfo{}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "WIFI.SSID", "GENERAL.CONNECTION":
			if val != "" && val != "--" {
				info.SSID = val
				info.Connected = true
			}
		case "WIFI.BSSID":
			info.BSSID = val
		case "WIFI.SIGNAL":
			if v, err := strconv.Atoi(val); err == nil {
				info.RSSI = v - 100
			}
		case "WIFI.CHAN":
			if v, err := strconv.Atoi(val); err == nil {
				info.Channel = v
			}
		case "WIFI.RATE":
			info.TxRate = val
		case "WIFI.SECURITY":
			info.Security = val
		}
	}

	return output.Print(info)
}
