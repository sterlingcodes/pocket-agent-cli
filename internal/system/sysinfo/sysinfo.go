package sysinfo

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/pkg/output"
)

const (
	osDarwin = "darwin"
	osLinux  = "linux"
)

// Overview is the LLM-friendly system overview
type Overview struct {
	Platform string   `json:"platform"`
	Hostname string   `json:"hostname"`
	OS       string   `json:"os"`
	Arch     string   `json:"arch"`
	CPUs     int      `json:"cpus"`
	Uptime   string   `json:"uptime,omitempty"`
	Memory   *MemInfo `json:"memory,omitempty"`
	LoadAvg  string   `json:"load_avg,omitempty"`
}

// CPUInfo holds CPU details
type CPUInfo struct {
	Model        string  `json:"model,omitempty"`
	Cores        int     `json:"cores"`
	LogicalCPUs  int     `json:"logical_cpus"`
	UsagePercent float64 `json:"usage_percent,omitempty"`
	Arch         string  `json:"arch"`
}

// MemInfo holds memory info
type MemInfo struct {
	TotalMB int64   `json:"total_mb"`
	UsedMB  int64   `json:"used_mb"`
	FreeMB  int64   `json:"free_mb"`
	UsedPct float64 `json:"used_percent"`
}

// DiskInfo holds disk partition info
type DiskInfo struct {
	Partitions []DiskPartition `json:"partitions"`
}

// DiskPartition is a single disk partition
type DiskPartition struct {
	Filesystem string  `json:"filesystem"`
	MountPoint string  `json:"mount_point"`
	TotalGB    float64 `json:"total_gb"`
	UsedGB     float64 `json:"used_gb"`
	FreeGB     float64 `json:"free_gb"`
	UsedPct    string  `json:"used_percent"`
}

// ProcessInfo holds info about a process
type ProcessInfo struct {
	PID    int     `json:"pid"`
	Name   string  `json:"name"`
	CPUPct float64 `json:"cpu_percent"`
	MemPct float64 `json:"mem_percent"`
	RSS    string  `json:"rss"`
	User   string  `json:"user,omitempty"`
}

// ProcessList holds top processes
type ProcessList struct {
	ByCPU    []ProcessInfo `json:"by_cpu"`
	ByMemory []ProcessInfo `json:"by_memory"`
}

// NetInterface holds network interface info
type NetInterface struct {
	Name   string `json:"name"`
	IPv4   string `json:"ipv4,omitempty"`
	IPv6   string `json:"ipv6,omitempty"`
	Status string `json:"status,omitempty"`
}

// NetInfo holds network info
type NetInfo struct {
	Interfaces []NetInterface `json:"interfaces"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sysinfo",
		Aliases: []string{"si", "info"},
		Short:   "System information and monitoring",
	}

	cmd.AddCommand(newOverviewCmd())
	cmd.AddCommand(newCPUCmd())
	cmd.AddCommand(newMemoryCmd())
	cmd.AddCommand(newDiskCmd())
	cmd.AddCommand(newProcessesCmd())
	cmd.AddCommand(newNetworkCmd())

	return cmd
}

func newOverviewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "overview",
		Short: "System overview (CPU, RAM, uptime)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return getOverview()
		},
	}
}

func newCPUCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cpu",
		Short: "CPU information and usage",
		RunE: func(cmd *cobra.Command, args []string) error {
			return getCPU()
		},
	}
}

func newMemoryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "memory",
		Short: "Memory usage details",
		RunE: func(cmd *cobra.Command, args []string) error {
			return getMemory()
		},
	}
}

func newDiskCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disk",
		Short: "Disk usage by partition",
		RunE: func(cmd *cobra.Command, args []string) error {
			return getDisk()
		},
	}
}

func newProcessesCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "processes",
		Short: "Top processes by CPU and memory",
		RunE: func(cmd *cobra.Command, args []string) error {
			return getProcesses(limit)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of top processes to show")

	return cmd
}

func newNetworkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "network",
		Short: "Network interface information",
		RunE: func(cmd *cobra.Command, args []string) error {
			return getNetwork()
		},
	}
}

func getOverview() error {
	hostname, _ := os.Hostname()

	info := Overview{
		Platform: runtime.GOOS,
		Hostname: hostname,
		OS:       runtime.GOOS + "/" + runtime.GOARCH,
		Arch:     runtime.GOARCH,
		CPUs:     runtime.NumCPU(),
	}

	// Get uptime
	switch runtime.GOOS {
	case osDarwin:
		if out, err := exec.Command("uptime").Output(); err == nil {
			info.Uptime = strings.TrimSpace(string(out))
		}
		// Get load average
		if out, err := exec.Command("sysctl", "-n", "vm.loadavg").Output(); err == nil {
			info.LoadAvg = strings.TrimSpace(string(out))
		}
	case osLinux:
		if data, err := os.ReadFile("/proc/uptime"); err == nil {
			fields := strings.Fields(string(data))
			if len(fields) > 0 {
				if secs, err := strconv.ParseFloat(fields[0], 64); err == nil {
					d := time.Duration(secs) * time.Second
					info.Uptime = d.String()
				}
			}
		}
		if data, err := os.ReadFile("/proc/loadavg"); err == nil {
			info.LoadAvg = strings.TrimSpace(string(data))
		}
	}

	// Get memory
	mem, _ := getMemInfo()
	info.Memory = mem

	return output.Print(info)
}

func getCPU() error {
	info := CPUInfo{
		Cores:       runtime.NumCPU(),
		LogicalCPUs: runtime.NumCPU(),
		Arch:        runtime.GOARCH,
	}

	switch runtime.GOOS {
	case osDarwin:
		if out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output(); err == nil {
			info.Model = strings.TrimSpace(string(out))
		}
		if out, err := exec.Command("sysctl", "-n", "hw.physicalcpu").Output(); err == nil {
			if v, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil {
				info.Cores = v
			}
		}
	case osLinux:
		if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
			scanner := bufio.NewScanner(strings.NewReader(string(data)))
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "model name") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						info.Model = strings.TrimSpace(parts[1])
						break
					}
				}
			}
		}
	}

	return output.Print(info)
}

func getMemory() error {
	mem, err := getMemInfo()
	if err != nil {
		return output.PrintError("memory_error", err.Error(), nil)
	}
	return output.Print(mem)
}

func getMemInfo() (*MemInfo, error) {
	switch runtime.GOOS {
	case osDarwin:
		return getMemDarwin()
	case osLinux:
		return getMemLinux()
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func getMemDarwin() (*MemInfo, error) {
	// Get total memory
	var totalBytes int64
	if out, err := exec.Command("sysctl", "-n", "hw.memsize").Output(); err == nil {
		totalBytes, _ = strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	}

	totalMB := totalBytes / (1024 * 1024)

	// Get memory usage from vm_stat
	var usedMB int64
	if out, err := exec.Command("vm_stat").Output(); err == nil {
		pageSize := int64(16384) // Apple Silicon default
		if ps, err := exec.Command("sysctl", "-n", "hw.pagesize").Output(); err == nil {
			pageSize, _ = strconv.ParseInt(strings.TrimSpace(string(ps)), 10, 64)
		}

		var active, wired, compressed int64
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := scanner.Text()
			switch {
			case strings.Contains(line, "Pages active"):
				active = parseVMStatValue(line)
			case strings.Contains(line, "Pages wired"):
				wired = parseVMStatValue(line)
			case strings.Contains(line, "Pages occupied by compressor"):
				compressed = parseVMStatValue(line)
			}
		}
		usedMB = (active + wired + compressed) * pageSize / (1024 * 1024)
	}

	freeMB := totalMB - usedMB
	usedPct := 0.0
	if totalMB > 0 {
		usedPct = float64(usedMB) / float64(totalMB) * 100
	}

	return &MemInfo{
		TotalMB: totalMB,
		UsedMB:  usedMB,
		FreeMB:  freeMB,
		UsedPct: usedPct,
	}, nil
}

func parseVMStatValue(line string) int64 {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return 0
	}
	s := strings.TrimSpace(parts[1])
	s = strings.TrimSuffix(s, ".")
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func getMemLinux() (*MemInfo, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return nil, err
	}

	values := make(map[string]int64)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		valStr := strings.TrimSpace(parts[1])
		valStr = strings.TrimSuffix(valStr, " kB")
		v, _ := strconv.ParseInt(strings.TrimSpace(valStr), 10, 64)
		values[key] = v
	}

	totalKB := values["MemTotal"]
	availableKB := values["MemAvailable"]
	usedKB := totalKB - availableKB

	totalMB := totalKB / 1024
	usedMB := usedKB / 1024
	freeMB := availableKB / 1024
	usedPct := 0.0
	if totalMB > 0 {
		usedPct = float64(usedMB) / float64(totalMB) * 100
	}

	return &MemInfo{
		TotalMB: totalMB,
		UsedMB:  usedMB,
		FreeMB:  freeMB,
		UsedPct: usedPct,
	}, nil
}

func getDisk() error {
	out, err := exec.Command("df", "-h").Output()
	if err != nil {
		return output.PrintError("disk_error", fmt.Sprintf("df failed: %v", err), nil)
	}

	var partitions []DiskPartition
	scanner := bufio.NewScanner(strings.NewReader(string(out)))

	// Skip header
	scanner.Scan()

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		mount := fields[len(fields)-1]
		// Filter to interesting mounts
		if !strings.HasPrefix(mount, "/") {
			continue
		}
		// Skip system mounts on macOS
		if strings.HasPrefix(mount, "/System") || strings.HasPrefix(mount, "/private/var/vm") {
			continue
		}

		p := DiskPartition{
			Filesystem: fields[0],
			MountPoint: mount,
			UsedPct:    fields[len(fields)-2],
		}

		p.TotalGB = parseSizeToGB(fields[1])
		p.UsedGB = parseSizeToGB(fields[2])
		p.FreeGB = parseSizeToGB(fields[3])

		partitions = append(partitions, p)
	}

	return output.Print(DiskInfo{Partitions: partitions})
}

func parseSizeToGB(s string) float64 {
	s = strings.TrimSpace(s)
	multiplier := 1.0

	switch {
	case strings.HasSuffix(s, "Ti") || strings.HasSuffix(s, "T"):
		multiplier = 1024
		s = strings.TrimRight(s, "TiB")
	case strings.HasSuffix(s, "Gi") || strings.HasSuffix(s, "G"):
		multiplier = 1
		s = strings.TrimRight(s, "GiB")
	case strings.HasSuffix(s, "Mi") || strings.HasSuffix(s, "M"):
		multiplier = 1.0 / 1024
		s = strings.TrimRight(s, "MiB")
	case strings.HasSuffix(s, "Ki") || strings.HasSuffix(s, "K"):
		multiplier = 1.0 / (1024 * 1024)
		s = strings.TrimRight(s, "KiB")
	}

	v, _ := strconv.ParseFloat(s, 64)
	return v * multiplier
}

func getProcesses(limit int) error {
	var out []byte
	var err error

	switch runtime.GOOS {
	case osDarwin, osLinux:
		out, err = exec.Command("ps", "axo", "pid,pcpu,pmem,rss,comm", "-r").Output()
	default:
		return output.PrintError("platform_unsupported",
			fmt.Sprintf("process listing not supported on %s", runtime.GOOS), nil)
	}

	if err != nil {
		return output.PrintError("process_error", fmt.Sprintf("ps failed: %v", err), nil)
	}

	var procs []ProcessInfo
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	scanner.Scan() // skip header

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		pid, _ := strconv.Atoi(fields[0])
		cpuPct, _ := strconv.ParseFloat(fields[1], 64)
		memPct, _ := strconv.ParseFloat(fields[2], 64)
		rssKB, _ := strconv.ParseInt(fields[3], 10, 64)
		name := strings.Join(fields[4:], " ")

		// Get just the binary name
		shortName := name
		if i := strings.LastIndex(name, "/"); i >= 0 && i < len(name)-1 {
			shortName = name[i+1:]
		}

		rssStr := fmt.Sprintf("%d MB", rssKB/1024)
		if rssKB < 1024 {
			rssStr = fmt.Sprintf("%d KB", rssKB)
		}

		procs = append(procs, ProcessInfo{
			PID:    pid,
			Name:   shortName,
			CPUPct: cpuPct,
			MemPct: memPct,
			RSS:    rssStr,
		})
	}

	// Top by CPU
	byCPU := make([]ProcessInfo, len(procs))
	copy(byCPU, procs)
	sort.Slice(byCPU, func(i, j int) bool {
		return byCPU[i].CPUPct > byCPU[j].CPUPct
	})
	if len(byCPU) > limit {
		byCPU = byCPU[:limit]
	}

	// Top by Memory
	byMem := make([]ProcessInfo, len(procs))
	copy(byMem, procs)
	sort.Slice(byMem, func(i, j int) bool {
		return byMem[i].MemPct > byMem[j].MemPct
	})
	if len(byMem) > limit {
		byMem = byMem[:limit]
	}

	return output.Print(ProcessList{
		ByCPU:    byCPU,
		ByMemory: byMem,
	})
}

func getNetwork() error {
	var interfaces []NetInterface

	switch runtime.GOOS {
	case osDarwin:
		out, err := exec.Command("ifconfig").Output()
		if err != nil {
			return output.PrintError("network_error", fmt.Sprintf("ifconfig failed: %v", err), nil)
		}
		interfaces = parseIfconfig(string(out))
	case osLinux:
		out, err := exec.Command("ip", "-brief", "addr").Output()
		if err != nil {
			// Fallback to ifconfig
			out, err = exec.Command("ifconfig").Output()
			if err != nil {
				return output.PrintError("network_error", "could not get network info", nil)
			}
			interfaces = parseIfconfig(string(out))
		} else {
			interfaces = parseIPBrief(string(out))
		}
	default:
		return output.PrintError("platform_unsupported",
			fmt.Sprintf("network info not supported on %s", runtime.GOOS), nil)
	}

	return output.Print(NetInfo{Interfaces: interfaces})
}

//nolint:gocyclo // complex but clear sequential logic
func parseIfconfig(data string) []NetInterface {
	var interfaces []NetInterface
	var current *NetInterface

	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()

		// New interface starts without leading whitespace
		if line != "" && line[0] != ' ' && line[0] != '\t' {
			if current != nil {
				interfaces = append(interfaces, *current)
			}
			parts := strings.SplitN(line, ":", 2)
			name := strings.TrimSpace(parts[0])
			current = &NetInterface{Name: name}
			if strings.Contains(line, "UP") {
				current.Status = "up"
			} else {
				current.Status = "down"
			}
		}

		if current == nil {
			continue
		}

		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "inet ") {
			fields := strings.Fields(trimmed)
			if len(fields) >= 2 {
				current.IPv4 = fields[1]
			}
		} else if strings.HasPrefix(trimmed, "inet6 ") {
			fields := strings.Fields(trimmed)
			if len(fields) >= 2 && current.IPv6 == "" {
				current.IPv6 = fields[1]
			}
		}
	}

	if current != nil {
		interfaces = append(interfaces, *current)
	}

	// Filter out loopback and uninteresting interfaces
	filtered := make([]NetInterface, 0, len(interfaces))
	for _, iface := range interfaces {
		if iface.Name == "lo" || iface.Name == "lo0" {
			continue
		}
		if iface.IPv4 == "" && iface.IPv6 == "" {
			continue
		}
		filtered = append(filtered, iface)
	}

	return filtered
}

func parseIPBrief(data string) []NetInterface {
	var interfaces []NetInterface

	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		name := fields[0]
		status := strings.ToLower(fields[1])

		if name == "lo" {
			continue
		}

		iface := NetInterface{
			Name:   name,
			Status: status,
		}

		for _, addr := range fields[2:] {
			addr = strings.Split(addr, "/")[0]
			if strings.Contains(addr, ":") {
				if iface.IPv6 == "" {
					iface.IPv6 = addr
				}
			} else if strings.Contains(addr, ".") {
				iface.IPv4 = addr
			}
		}

		if iface.IPv4 != "" || iface.IPv6 != "" {
			interfaces = append(interfaces, iface)
		}
	}

	return interfaces
}
