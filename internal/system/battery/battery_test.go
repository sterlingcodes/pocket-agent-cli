package battery

import (
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()

	if cmd.Use != "battery" {
		t.Errorf("expected Use=battery, got %s", cmd.Use)
	}

	aliases := map[string]bool{"bat": false, "batt": false}
	for _, a := range cmd.Aliases {
		aliases[a] = true
	}
	for alias, found := range aliases {
		if !found {
			t.Errorf("missing alias: %s", alias)
		}
	}

	subs := map[string]bool{"status": false, "health": false}
	for _, sub := range cmd.Commands() {
		subs[sub.Use] = true
	}
	for name, present := range subs {
		if !present {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

func TestParsePmsetACPower(t *testing.T) {
	input := `Now drawing from 'AC Power'
 -InternalBattery-0 (id=12345678)	100%; charged; 0:00 remaining present: true`

	status := parsePmset(input)

	if !status.HasBattery {
		t.Error("expected HasBattery=true")
	}
	if status.PowerSource != "AC Power" {
		t.Errorf("expected PowerSource='AC Power', got %q", status.PowerSource)
	}
	if status.Percentage != 100 {
		t.Errorf("expected Percentage=100, got %d", status.Percentage)
	}
	if status.State != "charged" {
		t.Errorf("expected State=charged, got %q", status.State)
	}
	if status.Charging {
		t.Error("expected Charging=false for charged state")
	}
}

func TestParsePmsetBatteryCharging(t *testing.T) {
	input := `Now drawing from 'Battery Power'
 -InternalBattery-0 (id=12345678)	65%; charging; 1:30 remaining present: true`

	status := parsePmset(input)

	if status.PowerSource != "Battery" {
		t.Errorf("expected PowerSource='Battery', got %q", status.PowerSource)
	}
	if status.Percentage != 65 {
		t.Errorf("expected Percentage=65, got %d", status.Percentage)
	}
	if status.State != "charging" {
		t.Errorf("expected State=charging, got %q", status.State)
	}
	if !status.Charging {
		t.Error("expected Charging=true")
	}
}

func TestParsePmsetDischarging(t *testing.T) {
	input := `Now drawing from 'Battery Power'
 -InternalBattery-0 (id=12345678)	45%; discharging; 3:15 remaining present: true`

	status := parsePmset(input)

	if status.Percentage != 45 {
		t.Errorf("expected Percentage=45, got %d", status.Percentage)
	}
	if status.State != "discharging" {
		t.Errorf("expected State=discharging, got %q", status.State)
	}
	if status.Charging {
		t.Error("expected Charging=false for discharging")
	}
}

func TestParsePmsetNotCharging(t *testing.T) {
	input := `Now drawing from 'AC Power'
 -InternalBattery-0 (id=12345678)	80%; not charging present: true`

	status := parsePmset(input)

	if status.State != "not charging" {
		t.Errorf("expected State='not charging', got %q", status.State)
	}
	if status.Charging {
		t.Error("expected Charging=false")
	}
}

func TestParsePmsetNoBattery(t *testing.T) {
	input := `Now drawing from 'AC Power'`

	status := parsePmset(input)

	if status.HasBattery {
		t.Error("expected HasBattery=false")
	}
	if status.PowerSource != "AC Power" {
		t.Errorf("expected PowerSource='AC Power', got %q", status.PowerSource)
	}
}

func TestParsePmsetEmpty(t *testing.T) {
	status := parsePmset("")
	if status.HasBattery {
		t.Error("expected HasBattery=false for empty input")
	}
}

func TestParseSystemProfilerPower(t *testing.T) {
	input := `Power:

      Battery Information:

          Model Information:
              Manufacturer: Apple
              Device Name: bq20z451
              Pack Lot Code: 0
              PCB Lot Code: 0
              Firmware Version: 702
              Hardware Revision: 1
              Cell Revision: 3171
          Charge Information:
              Charge Remaining (mAh): 4500
              Fully Charged: Yes
              Charging: No
              Full Charge Capacity (mAh): 5100
          Health Information:
              Cycle Count: 287
              Condition: Normal
              Maximum Capacity: 87%
              Temperature: 30.5`

	health := parseSystemProfilerPower(input)

	if !health.HasBattery {
		t.Error("expected HasBattery=true")
	}
	if health.CycleCount != 287 {
		t.Errorf("expected CycleCount=287, got %d", health.CycleCount)
	}
	if health.Condition != "Normal" {
		t.Errorf("expected Condition=Normal, got %q", health.Condition)
	}
	if health.MaxCapacityPct != 87 {
		t.Errorf("expected MaxCapacityPct=87, got %d", health.MaxCapacityPct)
	}
	if health.HealthPct != 87.0 {
		t.Errorf("expected HealthPct=87.0, got %f", health.HealthPct)
	}
	if health.CurrentMax != 5100 {
		t.Errorf("expected CurrentMax=5100, got %d", health.CurrentMax)
	}
	if health.Temperature != "30.5" {
		t.Errorf("expected Temperature=30.5, got %q", health.Temperature)
	}
}

func TestParseSystemProfilerPowerEmpty(t *testing.T) {
	health := parseSystemProfilerPower("")
	if health.HasBattery {
		t.Error("expected HasBattery=false for empty input")
	}
}

func TestParseSystemProfilerPowerCapacityCalc(t *testing.T) {
	// Test that health percentage is calculated from capacity when no explicit max capacity
	input := `          Health Information:
              Cycle Count: 100
              Condition: Normal
          Full Charge Capacity (mAh): 4000
          Design Capacity (mAh): 5000`

	health := parseSystemProfilerPower(input)

	// Should calculate: 4000/5000 * 100 = 80%
	if health.HealthPct != 80.0 {
		t.Errorf("expected HealthPct=80.0, got %f", health.HealthPct)
	}
}

func TestStatusTypes(t *testing.T) {
	status := Status{
		Charging:    true,
		Percentage:  75,
		State:       "charging",
		PowerSource: "AC Power",
		TimeLeft:    "1:30 remaining",
		HasBattery:  true,
	}

	if !status.HasBattery {
		t.Error("expected HasBattery=true")
	}
	if status.Percentage != 75 {
		t.Errorf("expected Percentage=75, got %d", status.Percentage)
	}
}

func TestHealthTypes(t *testing.T) {
	health := Health{
		Condition:      "Normal",
		MaxCapacityPct: 92,
		CycleCount:     150,
		DesignCapacity: 5000,
		CurrentMax:     4600,
		HealthPct:      92.0,
		Temperature:    "29.5",
		HasBattery:     true,
	}

	if health.Condition != "Normal" {
		t.Errorf("expected Condition=Normal, got %s", health.Condition)
	}
	if health.CycleCount != 150 {
		t.Errorf("expected CycleCount=150, got %d", health.CycleCount)
	}
}
