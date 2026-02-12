package traceroute

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

// TraceResult is the LLM-friendly traceroute result
type TraceResult struct {
	Host     string `json:"host"`
	Hops     []Hop  `json:"hops"`
	HopCount int    `json:"hop_count"`
	Complete bool   `json:"complete"`
}

// Hop represents a single hop in the traceroute
type Hop struct {
	Number  int      `json:"number"`
	Host    string   `json:"host,omitempty"`
	IP      string   `json:"ip,omitempty"`
	RTTs    []string `json:"rtts,omitempty"`
	Timeout bool     `json:"timeout,omitempty"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "traceroute",
		Aliases: []string{"trace", "tr"},
		Short:   "Network path tracing commands",
	}

	cmd.AddCommand(newRunCmd())

	return cmd
}

func newRunCmd() *cobra.Command {
	var maxHops int

	cmd := &cobra.Command{
		Use:   "run [host]",
		Short: "Trace the network path to a host",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTraceroute(args[0], maxHops)
		},
	}

	cmd.Flags().IntVarP(&maxHops, "max-hops", "m", 30, "Maximum number of hops")

	return cmd
}

func runTraceroute(host string, maxHops int) error {
	if maxHops < 1 || maxHops > 255 {
		maxHops = 30
	}

	var cmdName string
	var cmdArgs []string

	switch runtime.GOOS {
	case "darwin", "linux":
		cmdName = "traceroute"
		cmdArgs = []string{"-m", strconv.Itoa(maxHops), "-n", host}
	case "windows":
		cmdName = "tracert"
		cmdArgs = []string{"-h", strconv.Itoa(maxHops), "-d", host}
	default:
		return output.PrintError("platform_unsupported",
			fmt.Sprintf("traceroute not supported on %s", runtime.GOOS), nil)
	}

	cmd := exec.Command(cmdName, cmdArgs...)
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		// traceroute may return non-zero even on partial success
		if len(outBytes) == 0 {
			return output.PrintError("traceroute_error",
				fmt.Sprintf("traceroute failed: %v", err),
				map[string]string{"suggestion": "Ensure traceroute is installed"})
		}
	}

	hops := parseTraceroute(string(outBytes))

	complete := false
	if len(hops) > 0 {
		lastHop := hops[len(hops)-1]
		complete = !lastHop.Timeout && lastHop.IP != ""
	}

	return output.Print(TraceResult{
		Host:     host,
		Hops:     hops,
		HopCount: len(hops),
		Complete: complete,
	})
}

var (
	// Matches lines like: " 1  192.168.1.1  1.234 ms  0.987 ms  1.123 ms"
	hopRegex = regexp.MustCompile(`^\s*(\d+)\s+(.+)$`)
	// Matches IP addresses
	ipRegex = regexp.MustCompile(`(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})`)
	// Matches RTT values like "1.234 ms"
	rttRegex = regexp.MustCompile(`([\d.]+)\s*ms`)
	// Matches hostname (ip) patterns
	hostIPRegex = regexp.MustCompile(`(\S+)\s+\((\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\)`)
)

func parseTraceroute(data string) []Hop {
	var hops []Hop
	scanner := bufio.NewScanner(strings.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()

		match := hopRegex.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		hopNum, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}

		rest := match[2]

		hop := Hop{
			Number: hopNum,
		}

		// Check for timeout
		if strings.Contains(rest, "* * *") {
			hop.Timeout = true
			hops = append(hops, hop)
			continue
		}

		// Try to extract hostname (ip) pattern
		hostMatch := hostIPRegex.FindStringSubmatch(rest)
		if hostMatch != nil {
			hop.Host = hostMatch[1]
			hop.IP = hostMatch[2]
		} else {
			// Try just IP
			ipMatch := ipRegex.FindStringSubmatch(rest)
			if ipMatch != nil {
				hop.IP = ipMatch[1]
			}
		}

		// Extract RTTs
		rttMatches := rttRegex.FindAllStringSubmatch(rest, -1)
		for _, rm := range rttMatches {
			hop.RTTs = append(hop.RTTs, rm[1]+" ms")
		}

		hops = append(hops, hop)
	}

	return hops
}
