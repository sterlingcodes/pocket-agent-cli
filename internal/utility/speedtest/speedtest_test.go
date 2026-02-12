package speedtest

import (
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()

	if cmd.Use != "speedtest" {
		t.Errorf("expected Use=speedtest, got %s", cmd.Use)
	}

	// Check aliases
	found := false
	for _, a := range cmd.Aliases {
		if a == "speed" {
			found = true
		}
	}
	if !found {
		t.Error("expected alias 'speed'")
	}

	// Check subcommands exist
	subs := map[string]bool{
		"run":      false,
		"download": false,
		"upload":   false,
		"latency":  false,
	}
	for _, sub := range cmd.Commands() {
		if _, ok := subs[sub.Use]; ok {
			subs[sub.Use] = true
		}
	}
	for name, present := range subs {
		if !present {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

func TestZeroReader(t *testing.T) {
	z := &zeroReader{}
	buf := make([]byte, 64)
	n, err := z.Read(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 64 {
		t.Errorf("expected 64 bytes, got %d", n)
	}
	for i, b := range buf {
		if b != 0 {
			t.Errorf("expected zero at index %d, got %d", i, b)
		}
	}
}

func TestZeroReaderSmallBuffer(t *testing.T) {
	z := &zeroReader{}
	buf := make([]byte, 1)
	n, err := z.Read(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 byte, got %d", n)
	}
	if buf[0] != 0 {
		t.Errorf("expected zero byte, got %d", buf[0])
	}
}

func TestSpeedResultTypes(t *testing.T) {
	// Ensure result types are constructable with expected fields
	result := SpeedResult{
		Download: &BandwidthResult{
			SpeedMbps:  100.5,
			Bytes:      25 * 1024 * 1024,
			DurationMs: 2000,
		},
		Upload: &BandwidthResult{
			SpeedMbps:  50.2,
			Bytes:      10 * 1024 * 1024,
			DurationMs: 1600,
		},
		Latency: &LatencyResult{
			MinMs: 5.0,
			AvgMs: 10.0,
			MaxMs: 15.0,
		},
		TestedAt:    "2025-01-01T00:00:00Z",
		TestType:    "full",
		DurationSec: 5.5,
	}

	if result.Download.SpeedMbps != 100.5 {
		t.Errorf("expected download speed 100.5, got %f", result.Download.SpeedMbps)
	}
	if result.Upload.Bytes != 10*1024*1024 {
		t.Errorf("expected upload bytes %d, got %d", 10*1024*1024, result.Upload.Bytes)
	}
	if result.Latency.AvgMs != 10.0 {
		t.Errorf("expected latency avg 10.0, got %f", result.Latency.AvgMs)
	}
	if result.TestType != "full" {
		t.Errorf("expected test type 'full', got %s", result.TestType)
	}
}
