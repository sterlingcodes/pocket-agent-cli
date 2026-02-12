package diskhealth

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/pkg/output"
)

// DiskStatus is the LLM-friendly disk status
type DiskStatus struct {
	Disks []DiskEntry `json:"disks"`
}

// DiskEntry holds info about a single disk
type DiskEntry struct {
	Name        string `json:"name"`
	MediaType   string `json:"media_type,omitempty"`
	Protocol    string `json:"protocol,omitempty"`
	Size        string `json:"size,omitempty"`
	SMARTStatus string `json:"smart_status,omitempty"`
	Internal    bool   `json:"internal,omitempty"`
}

// DiskDetail holds detailed disk info
type DiskDetail struct {
	Name        string            `json:"name"`
	DeviceNode  string            `json:"device_node,omitempty"`
	MediaType   string            `json:"media_type,omitempty"`
	Protocol    string            `json:"protocol,omitempty"`
	Size        string            `json:"size,omitempty"`
	SMARTStatus string            `json:"smart_status,omitempty"`
	Partitions  []PartitionEntry  `json:"partitions,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

// PartitionEntry holds partition info
type PartitionEntry struct {
	Name       string `json:"name"`
	MountPoint string `json:"mount_point,omitempty"`
	FileSystem string `json:"file_system,omitempty"`
	Size       string `json:"size,omitempty"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "diskhealth",
		Aliases: []string{"dh", "smart"},
		Short:   "Disk health and S.M.A.R.T. status",
	}

	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newInfoCmd())

	return cmd
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show disk health status overview",
		RunE: func(cmd *cobra.Command, args []string) error {
			return getStatus()
		},
	}
}

func newInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info [disk]",
		Short: "Detailed disk information",
		Long:  "Get detailed info for a specific disk. Use 'disk0' on macOS or '/dev/sda' on Linux.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			disk := ""
			if len(args) > 0 {
				disk = args[0]
			}
			return getInfo(disk)
		},
	}
}

func getStatus() error {
	switch runtime.GOOS {
	case "darwin":
		return getStatusDarwin()
	case "linux":
		return getStatusLinux()
	default:
		return output.PrintError("platform_unsupported",
			fmt.Sprintf("disk health not supported on %s", runtime.GOOS),
			map[string]string{"supported": "macOS, Linux"})
	}
}

func getInfo(disk string) error {
	switch runtime.GOOS {
	case "darwin":
		if disk == "" {
			disk = "disk0"
		}
		return getInfoDarwin(disk)
	case "linux":
		if disk == "" {
			disk = "/dev/sda"
		}
		return getInfoLinux(disk)
	default:
		return output.PrintError("platform_unsupported",
			fmt.Sprintf("disk info not supported on %s", runtime.GOOS),
			map[string]string{"supported": "macOS, Linux"})
	}
}

// macOS implementations
func getStatusDarwin() error {
	out, err := exec.Command("system_profiler", "SPStorageDataType", "-json").Output()
	if err != nil {
		return output.PrintError("diskhealth_error",
			fmt.Sprintf("system_profiler failed: %v", err), nil)
	}

	var spData map[string]any
	if err := json.Unmarshal(out, &spData); err != nil {
		// Fallback to diskutil
		return getStatusDarwinDiskutil()
	}

	var disks []DiskEntry
	if items, ok := spData["SPStorageDataType"].([]any); ok {
		for _, item := range items {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			entry := DiskEntry{
				Name: getString(m, "_name"),
			}
			if mt := getString(m, "physical_drive_media_type"); mt != "" {
				entry.MediaType = mt
			}
			if proto := getString(m, "physical_drive_protocol"); proto != "" {
				entry.Protocol = proto
			}
			if size := getString(m, "size_in_bytes"); size != "" {
				entry.Size = formatBytes(size)
			} else if size := getString(m, "free_space_in_bytes"); size != "" {
				entry.Size = "available: " + formatBytes(size)
			}
			if smart := getString(m, "smart_status"); smart != "" {
				entry.SMARTStatus = smart
			}
			disks = append(disks, entry)
		}
	}

	if len(disks) == 0 {
		return getStatusDarwinDiskutil()
	}

	return output.Print(DiskStatus{Disks: disks})
}

//nolint:gocyclo // complex but clear sequential logic
func getStatusDarwinDiskutil() error {
	out, err := exec.Command("diskutil", "list").Output()
	if err != nil {
		return output.PrintError("diskhealth_error",
			fmt.Sprintf("diskutil failed: %v", err), nil)
	}

	var disks []DiskEntry
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "/dev/disk") {
			parts := strings.Fields(line)
			name := parts[0]
			rest := ""
			if len(parts) > 1 {
				rest = strings.Join(parts[1:], " ")
			}

			entry := DiskEntry{
				Name: name,
			}
			if strings.Contains(rest, "internal") {
				entry.Internal = true
			}
			switch {
			case strings.Contains(rest, "physical"):
				entry.MediaType = "physical"
			case strings.Contains(rest, "synthesized"):
				entry.MediaType = "synthesized"
			case strings.Contains(rest, "virtual"):
				entry.MediaType = "virtual"
			}

			// Get SMART status
			smartOut, err := exec.Command("diskutil", "info", name).Output()
			if err == nil {
				smartScanner := bufio.NewScanner(strings.NewReader(string(smartOut)))
				for smartScanner.Scan() {
					sLine := smartScanner.Text()
					if strings.Contains(sLine, "SMART Status") {
						parts := strings.SplitN(sLine, ":", 2)
						if len(parts) == 2 {
							entry.SMARTStatus = strings.TrimSpace(parts[1])
						}
					}
					if strings.Contains(sLine, "Disk Size") {
						parts := strings.SplitN(sLine, ":", 2)
						if len(parts) == 2 {
							entry.Size = strings.TrimSpace(parts[1])
						}
					}
					if strings.Contains(sLine, "Protocol") {
						parts := strings.SplitN(sLine, ":", 2)
						if len(parts) == 2 {
							entry.Protocol = strings.TrimSpace(parts[1])
						}
					}
				}
			}

			disks = append(disks, entry)
		}
	}

	return output.Print(DiskStatus{Disks: disks})
}

//nolint:gocyclo // complex but clear sequential logic
func getInfoDarwin(disk string) error {
	out, err := exec.Command("diskutil", "info", disk).Output()
	if err != nil {
		return output.PrintError("diskhealth_error",
			fmt.Sprintf("diskutil info failed for %s: %v", disk, err), nil)
	}

	detail := DiskDetail{
		Name:       disk,
		Attributes: make(map[string]string),
	}

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
		case "Device Node":
			detail.DeviceNode = val
		case "Protocol":
			detail.Protocol = val
		case "Disk Size":
			detail.Size = val
		case "SMART Status":
			detail.SMARTStatus = val
		case "Solid State", "Media Type":
			detail.MediaType = val
		default:
			if val != "" {
				detail.Attributes[key] = val
			}
		}
	}

	// Get partitions
	listOut, err := exec.Command("diskutil", "list", disk).Output()
	if err == nil {
		partScanner := bufio.NewScanner(strings.NewReader(string(listOut)))
		for partScanner.Scan() {
			pLine := partScanner.Text()
			pLine = strings.TrimSpace(pLine)
			// Partition lines start with a number
			if pLine != "" && pLine[0] >= '0' && pLine[0] <= '9' {
				fields := strings.Fields(pLine)
				if len(fields) >= 3 {
					p := PartitionEntry{
						Name: strings.Join(fields[1:len(fields)-2], " "),
						Size: fields[len(fields)-2] + " " + fields[len(fields)-1],
					}
					detail.Partitions = append(detail.Partitions, p)
				}
			}
		}
	}

	return output.Print(detail)
}

// Linux implementations
func getStatusLinux() error {
	out, err := exec.Command("lsblk", "-Jdo", "NAME,SIZE,TYPE,TRAN,MODEL").Output()
	if err != nil {
		return output.PrintError("diskhealth_error",
			fmt.Sprintf("lsblk failed: %v", err),
			map[string]string{"suggestion": "Ensure lsblk is available"})
	}

	var lsblkData struct {
		Blockdevices []struct {
			Name  string `json:"name"`
			Size  string `json:"size"`
			Type  string `json:"type"`
			Tran  string `json:"tran"`
			Model string `json:"model"`
		} `json:"blockdevices"`
	}

	if err := json.Unmarshal(out, &lsblkData); err != nil {
		return output.PrintError("parse_error", fmt.Sprintf("failed to parse lsblk: %v", err), nil)
	}

	disks := make([]DiskEntry, 0, len(lsblkData.Blockdevices))
	for _, bd := range lsblkData.Blockdevices {
		if bd.Type != "disk" {
			continue
		}

		entry := DiskEntry{
			Name:     bd.Name,
			Size:     bd.Size,
			Protocol: bd.Tran,
		}

		if bd.Model != "" {
			entry.Name = bd.Name + " (" + strings.TrimSpace(bd.Model) + ")"
		}

		if bd.Tran == "nvme" || bd.Tran == "sata" {
			entry.MediaType = bd.Tran
		}

		// Try to get SMART status
		smartOut, err := exec.Command("smartctl", "-H", "/dev/"+bd.Name).Output() //nolint:gosec // bd.Name comes from system disk listing, not user input
		if err == nil {
			if strings.Contains(string(smartOut), "PASSED") {
				entry.SMARTStatus = "Verified"
			} else if strings.Contains(string(smartOut), "FAILED") {
				entry.SMARTStatus = "Failing"
			}
		}

		disks = append(disks, entry)
	}

	return output.Print(DiskStatus{Disks: disks})
}

func getInfoLinux(disk string) error {
	// Try smartctl for detailed info
	out, err := exec.Command("smartctl", "-a", disk).Output()
	if err != nil {
		// Fallback to basic lsblk info
		return getInfoLinuxBasic(disk)
	}

	detail := DiskDetail{
		Name:       disk,
		Attributes: make(map[string]string),
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			if val != "" {
				detail.Attributes[key] = val
			}
			switch key {
			case "Device Model", "Model Number":
				detail.Name = val
			case "User Capacity":
				detail.Size = val
			case "SMART overall-health self-assessment test result":
				detail.SMARTStatus = val
			case "Transport protocol":
				detail.Protocol = val
			case "Rotation Rate":
				if strings.Contains(val, "Solid State") {
					detail.MediaType = "SSD"
				} else {
					detail.MediaType = "HDD (" + val + ")"
				}
			}
		}
	}

	return output.Print(detail)
}

func getInfoLinuxBasic(disk string) error {
	detail := DiskDetail{
		Name:       disk,
		Attributes: make(map[string]string),
	}

	// Get basic info from lsblk
	out, err := exec.Command("lsblk", "-Jdo", "NAME,SIZE,TYPE,TRAN,MODEL,SERIAL", disk).Output()
	if err == nil {
		var data struct {
			Blockdevices []struct {
				Name   string `json:"name"`
				Size   string `json:"size"`
				Tran   string `json:"tran"`
				Model  string `json:"model"`
				Serial string `json:"serial"`
			} `json:"blockdevices"`
		}
		if json.Unmarshal(out, &data) == nil && len(data.Blockdevices) > 0 {
			bd := data.Blockdevices[0]
			detail.Size = bd.Size
			detail.Protocol = bd.Tran
			if bd.Model != "" {
				detail.Name = strings.TrimSpace(bd.Model)
			}
			if bd.Serial != "" {
				detail.Attributes["Serial"] = bd.Serial
			}
		}
	}

	detail.Attributes["note"] = "Install smartmontools for detailed S.M.A.R.T. data"

	return output.Print(detail)
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func formatBytes(bytesStr string) string {
	// Remove any non-numeric characters
	bytesStr = strings.TrimSpace(bytesStr)
	// Simple pass-through for now
	return bytesStr
}
