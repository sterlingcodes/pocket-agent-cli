package battery

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/pkg/output"
)

const (
	powerSourceAC      = "AC Power"
	powerSourceBattery = "Battery"
	conditionNormal    = "Normal"
)

// Status holds current battery status
type Status struct {
	Charging    bool   `json:"charging"`
	Percentage  int    `json:"percentage"`
	State       string `json:"state"`
	PowerSource string `json:"power_source"`
	TimeLeft    string `json:"time_left,omitempty"`
	HasBattery  bool   `json:"has_battery"`
}

// Health holds battery health details
type Health struct {
	Condition      string  `json:"condition"`
	MaxCapacityPct int     `json:"max_capacity_percent,omitempty"`
	CycleCount     int     `json:"cycle_count,omitempty"`
	DesignCapacity int     `json:"design_capacity_mah,omitempty"`
	CurrentMax     int     `json:"current_max_capacity_mah,omitempty"`
	HealthPct      float64 `json:"health_percent,omitempty"`
	Temperature    string  `json:"temperature,omitempty"`
	HasBattery     bool    `json:"has_battery"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "battery",
		Aliases: []string{"bat", "batt"},
		Short:   "Battery status and health",
	}

	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newHealthCmd())

	return cmd
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Current battery charge and state",
		RunE: func(cmd *cobra.Command, args []string) error {
			return getStatus()
		},
	}
}

func newHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Battery health, cycle count, and capacity",
		RunE: func(cmd *cobra.Command, args []string) error {
			return getHealth()
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
			fmt.Sprintf("battery status not supported on %s", runtime.GOOS),
			map[string]string{"supported": "macOS, Linux"})
	}
}

func getHealth() error {
	switch runtime.GOOS {
	case "darwin":
		return getHealthDarwin()
	case "linux":
		return getHealthLinux()
	default:
		return output.PrintError("platform_unsupported",
			fmt.Sprintf("battery health not supported on %s", runtime.GOOS),
			map[string]string{"supported": "macOS, Linux"})
	}
}

// macOS implementations
func getStatusDarwin() error {
	out, err := exec.Command("pmset", "-g", "batt").Output()
	if err != nil {
		return output.PrintError("battery_error",
			fmt.Sprintf("pmset failed: %v", err), nil)
	}

	status := parsePmset(string(out))
	return output.Print(status)
}

//nolint:gocyclo // complex but clear sequential logic
func parsePmset(data string) Status {
	status := Status{}

	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, powerSourceAC) {
			status.PowerSource = powerSourceAC
		} else if strings.Contains(line, "Battery Power") {
			status.PowerSource = powerSourceBattery
		}

		if strings.Contains(line, "InternalBattery") {
			status.HasBattery = true

			// Parse percentage
			for _, part := range strings.Split(line, ";") {
				part = strings.TrimSpace(part)
				if strings.HasSuffix(part, "%") {
					// Find the percentage value
					fields := strings.Fields(part)
					for _, f := range fields {
						if strings.HasSuffix(f, "%") {
							pctStr := strings.TrimSuffix(f, "%")
							if v, err := strconv.Atoi(pctStr); err == nil {
								status.Percentage = v
							}
						}
					}
				}
				switch {
				case strings.Contains(part, "not charging"):
					status.State = "not charging"
				case strings.Contains(part, "discharging"):
					status.State = "discharging"
				case strings.Contains(part, "charging"):
					status.State = "charging"
					status.Charging = true
				case strings.Contains(part, "charged"):
					status.State = "charged"
				}
				if strings.Contains(part, "remaining") || strings.Contains(part, "until") {
					status.TimeLeft = strings.TrimSpace(part)
				}
			}
		}
	}

	return status
}

func getHealthDarwin() error {
	out, err := exec.Command("system_profiler", "SPPowerDataType").Output()
	if err != nil {
		return output.PrintError("battery_error",
			fmt.Sprintf("system_profiler failed: %v", err), nil)
	}

	health := parseSystemProfilerPower(string(out))
	return output.Print(health)
}

//nolint:gocyclo // complex but clear sequential logic
func parseSystemProfilerPower(data string) Health {
	health := Health{}

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
		case "Cycle Count":
			if v, err := strconv.Atoi(val); err == nil {
				health.CycleCount = v
				health.HasBattery = true
			}
		case "Condition":
			health.Condition = val
			health.HasBattery = true
		case "Maximum Capacity":
			// e.g., "87%"
			pctStr := strings.TrimSuffix(val, "%")
			if v, err := strconv.Atoi(strings.TrimSpace(pctStr)); err == nil {
				health.MaxCapacityPct = v
				health.HealthPct = float64(v)
			}
		case "Full Charge Capacity (mAh)":
			if v, err := strconv.Atoi(val); err == nil {
				health.CurrentMax = v
			}
		case "Design Capacity (mAh)":
			// Older macOS format
			if v, err := strconv.Atoi(val); err == nil {
				health.DesignCapacity = v
			}
		case "Temperature":
			health.Temperature = val
		}
	}

	// Calculate health if we have capacity data but no MaxCapacityPct
	if health.HealthPct == 0 && health.DesignCapacity > 0 && health.CurrentMax > 0 {
		health.HealthPct = float64(health.CurrentMax) / float64(health.DesignCapacity) * 100
	}

	return health
}

// Linux implementations
func getStatusLinux() error {
	status := Status{}

	// Check if battery exists
	batPath := "/sys/class/power_supply/BAT0"
	if _, err := os.Stat(batPath); os.IsNotExist(err) {
		batPath = "/sys/class/power_supply/BAT1"
		if _, err := os.Stat(batPath); os.IsNotExist(err) {
			return output.Print(Status{HasBattery: false})
		}
	}

	status.HasBattery = true

	if data, err := os.ReadFile(batPath + "/capacity"); err == nil {
		if v, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			status.Percentage = v
		}
	}

	if data, err := os.ReadFile(batPath + "/status"); err == nil {
		s := strings.TrimSpace(string(data))
		status.State = strings.ToLower(s)
		status.Charging = s == "Charging"
		if s == "Discharging" {
			status.PowerSource = powerSourceBattery
		} else {
			status.PowerSource = powerSourceAC
		}
	}

	return output.Print(status)
}

//nolint:gocyclo // complex but clear sequential logic
func getHealthLinux() error {
	health := Health{}

	batPath := "/sys/class/power_supply/BAT0"
	if _, err := os.Stat(batPath); os.IsNotExist(err) {
		batPath = "/sys/class/power_supply/BAT1"
		if _, err := os.Stat(batPath); os.IsNotExist(err) {
			return output.Print(Health{HasBattery: false})
		}
	}

	health.HasBattery = true

	if data, err := os.ReadFile(batPath + "/cycle_count"); err == nil {
		if v, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			health.CycleCount = v
		}
	}

	// Try energy_full / energy_full_design (in microwatt-hours)
	var fullCharge, designCharge int64
	if data, err := os.ReadFile(batPath + "/energy_full"); err == nil {
		fullCharge, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	}
	if data, err := os.ReadFile(batPath + "/energy_full_design"); err == nil {
		designCharge, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	}

	// Fallback to charge_full / charge_full_design (in microamp-hours)
	if fullCharge == 0 {
		if data, err := os.ReadFile(batPath + "/charge_full"); err == nil {
			fullCharge, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		}
	}
	if designCharge == 0 {
		if data, err := os.ReadFile(batPath + "/charge_full_design"); err == nil {
			designCharge, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		}
	}

	if fullCharge > 0 {
		health.CurrentMax = int(fullCharge / 1000) // Convert to mAh
	}
	if designCharge > 0 {
		health.DesignCapacity = int(designCharge / 1000)
	}

	if designCharge > 0 && fullCharge > 0 {
		health.HealthPct = float64(fullCharge) / float64(designCharge) * 100
		health.Condition = conditionNormal
		if health.HealthPct < 80 {
			health.Condition = "Service Recommended"
		}
		if health.HealthPct < 50 {
			health.Condition = "Replace Soon"
		}
	}

	return output.Print(health)
}
