package netdiag

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

var httpClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
	Timeout: 30 * time.Second,
}

// HeaderResult is LLM-friendly HTTP header result
type HeaderResult struct {
	URL         string            `json:"url"`
	StatusCode  int               `json:"status_code"`
	Headers     map[string]string `json:"headers"`
	Server      string            `json:"server,omitempty"`
	ContentType string            `json:"content_type,omitempty"`
	RedirectTo  string            `json:"redirect_to,omitempty"`
}

// PortResult is LLM-friendly port scan result
type PortResult struct {
	Host      string     `json:"host"`
	OpenPorts []PortInfo `json:"open_ports"`
}

// PortInfo describes a single port
type PortInfo struct {
	Port    int    `json:"port"`
	Service string `json:"service"`
	Open    bool   `json:"open"`
}

// PingResult is LLM-friendly DNS resolve result
type PingResult struct {
	Host          string   `json:"host"`
	IPs           []string `json:"ips"`
	ResolveTimeMs int64    `json:"resolve_time_ms"`
}

// portServices maps common ports to service names
var portServices = map[int]string{
	21:    "FTP",
	22:    "SSH",
	25:    "SMTP",
	53:    "DNS",
	80:    "HTTP",
	443:   "HTTPS",
	3306:  "MySQL",
	5432:  "PostgreSQL",
	6379:  "Redis",
	8080:  "HTTP-Alt",
	8443:  "HTTPS-Alt",
	27017: "MongoDB",
}

// NewCmd returns the netdiag command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "netdiag",
		Aliases: []string{"nd", "diag"},
		Short:   "Network diagnostics commands",
	}

	cmd.AddCommand(newHeadersCmd())
	cmd.AddCommand(newPortsCmd())
	cmd.AddCommand(newPingCmd())

	return cmd
}

func newHeadersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "headers [url]",
		Short: "Get HTTP response headers for a URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fetchHeaders(args[0])
		},
	}

	return cmd
}

func newPortsCmd() *cobra.Command {
	var timeout int

	cmd := &cobra.Command{
		Use:   "ports [host]",
		Short: "Scan common ports on a host",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return scanPorts(args[0], time.Duration(timeout)*time.Second)
		},
	}

	cmd.Flags().IntVarP(&timeout, "timeout", "t", 2, "Timeout per port in seconds")

	return cmd
}

func newPingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ping [host]",
		Short: "Resolve host to IPs and measure DNS resolve time",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return pingHost(args[0])
		},
	}

	return cmd
}

func fetchHeaders(rawURL string) error {
	// Ensure URL has a scheme
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "HEAD", rawURL, nil)
	if err != nil {
		return output.PrintError("fetch_failed", err.Error(), nil)
	}

	req.Header.Set("User-Agent", "Pocket-CLI/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return output.PrintError("fetch_failed", err.Error(), nil)
	}
	defer resp.Body.Close()

	headers := make(map[string]string)
	for key, values := range resp.Header {
		headers[key] = strings.Join(values, ", ")
	}

	result := HeaderResult{
		URL:         rawURL,
		StatusCode:  resp.StatusCode,
		Headers:     headers,
		Server:      resp.Header.Get("Server"),
		ContentType: resp.Header.Get("Content-Type"),
	}

	if location := resp.Header.Get("Location"); location != "" {
		result.RedirectTo = location
	}

	return output.Print(result)
}

func scanPorts(host string, timeout time.Duration) error {
	ports := []int{80, 443, 22, 21, 25, 53, 3306, 5432, 6379, 8080, 8443, 27017}

	var mu sync.Mutex
	var openPorts []PortInfo

	var wg sync.WaitGroup
	for _, port := range ports {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			address := net.JoinHostPort(host, fmt.Sprintf("%d", p))
			conn, err := net.DialTimeout("tcp", address, timeout)
			open := err == nil
			if open {
				conn.Close()
			}

			info := PortInfo{
				Port:    p,
				Service: portServices[p],
				Open:    open,
			}

			mu.Lock()
			openPorts = append(openPorts, info)
			mu.Unlock()
		}(port)
	}

	wg.Wait()

	// Filter to only open ports for cleaner output
	var open []PortInfo
	for _, p := range openPorts {
		if p.Open {
			open = append(open, p)
		}
	}

	result := PortResult{
		Host:      host,
		OpenPorts: open,
	}

	return output.Print(result)
}

func pingHost(host string) error {
	start := time.Now()

	ips, err := net.LookupHost(host)
	if err != nil {
		return output.PrintError("resolve_failed", fmt.Sprintf("Failed to resolve %s: %s", host, err.Error()), nil)
	}

	elapsed := time.Since(start)

	result := PingResult{
		Host:          host,
		IPs:           ips,
		ResolveTimeMs: elapsed.Milliseconds(),
	}

	return output.Print(result)
}
