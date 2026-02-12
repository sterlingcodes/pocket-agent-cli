package diskhealth

import (
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()

	if cmd.Use != "diskhealth" {
		t.Errorf("expected Use=diskhealth, got %s", cmd.Use)
	}

	aliases := map[string]bool{"dh": false, "smart": false}
	for _, a := range cmd.Aliases {
		aliases[a] = true
	}
	for alias, found := range aliases {
		if !found {
			t.Errorf("missing alias: %s", alias)
		}
	}

	subs := map[string]bool{"status": false}
	infoFound := false
	for _, sub := range cmd.Commands() {
		if sub.Use == "status" {
			subs["status"] = true
		}
		if sub.Use == "info [disk]" {
			infoFound = true
		}
	}
	if !subs["status"] {
		t.Error("missing subcommand: status")
	}
	if !infoFound {
		t.Error("missing subcommand: info")
	}
}

func TestInfoCmdDefaultDisk(t *testing.T) {
	// Verify info command accepts max 1 arg
	cmd := NewCmd()
	for _, sub := range cmd.Commands() {
		if sub.Use == "info [disk]" {
			// cobra.MaximumNArgs(1) means 0 or 1 args
			if err := sub.Args(sub, []string{}); err != nil {
				t.Error("info should accept 0 args")
			}
			if err := sub.Args(sub, []string{"disk0"}); err != nil {
				t.Error("info should accept 1 arg")
			}
			if err := sub.Args(sub, []string{"disk0", "extra"}); err == nil {
				t.Error("info should reject 2 args")
			}
			return
		}
	}
	t.Error("info subcommand not found")
}

func TestGetString(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]any
		key  string
		want string
	}{
		{
			name: "string value",
			m:    map[string]any{"key": "value"},
			key:  "key",
			want: "value",
		},
		{
			name: "missing key",
			m:    map[string]any{},
			key:  "key",
			want: "",
		},
		{
			name: "non-string value",
			m:    map[string]any{"key": 42},
			key:  "key",
			want: "",
		},
		{
			name: "nil map",
			m:    nil,
			key:  "key",
			want: "",
		},
		{
			name: "empty string",
			m:    map[string]any{"key": ""},
			key:  "key",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getString(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("getString(%v, %q) = %q, want %q", tt.m, tt.key, got, tt.want)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1000000000", "1000000000"},
		{"  500000  ", "500000"},
		{"0", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := formatBytes(tt.input)
			if got != tt.want {
				t.Errorf("formatBytes(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDiskStatusTypes(t *testing.T) {
	status := DiskStatus{
		Disks: []DiskEntry{
			{
				Name:        "Macintosh HD",
				MediaType:   "SSD",
				Protocol:    "Apple Fabric",
				Size:        "500 GB",
				SMARTStatus: "Verified",
				Internal:    true,
			},
		},
	}

	if len(status.Disks) != 1 {
		t.Fatalf("expected 1 disk, got %d", len(status.Disks))
	}
	if status.Disks[0].SMARTStatus != "Verified" {
		t.Errorf("expected SMART=Verified, got %s", status.Disks[0].SMARTStatus)
	}
	if !status.Disks[0].Internal {
		t.Error("expected Internal=true")
	}
}

func TestDiskDetailTypes(t *testing.T) {
	detail := DiskDetail{
		Name:        "disk0",
		DeviceNode:  "/dev/disk0",
		MediaType:   "SSD",
		Protocol:    "Apple Fabric",
		Size:        "500.11 GB",
		SMARTStatus: "Verified",
		Partitions: []PartitionEntry{
			{
				Name:       "Apple_APFS_ISC",
				MountPoint: "",
				FileSystem: "APFS",
				Size:       "500 MB",
			},
			{
				Name:       "APFS Container",
				MountPoint: "/",
				FileSystem: "APFS",
				Size:       "494 GB",
			},
		},
		Attributes: map[string]string{
			"Device Block Size": "4096",
		},
	}

	if detail.Name != "disk0" {
		t.Errorf("expected Name=disk0, got %s", detail.Name)
	}
	if len(detail.Partitions) != 2 {
		t.Errorf("expected 2 partitions, got %d", len(detail.Partitions))
	}
	if detail.Attributes["Device Block Size"] != "4096" {
		t.Errorf("expected block size=4096, got %s", detail.Attributes["Device Block Size"])
	}
}

func TestPartitionEntryTypes(t *testing.T) {
	p := PartitionEntry{
		Name:       "EFI",
		MountPoint: "/boot/efi",
		FileSystem: "vfat",
		Size:       "200 MB",
	}

	if p.Name != "EFI" {
		t.Errorf("expected Name=EFI, got %s", p.Name)
	}
	if p.MountPoint != "/boot/efi" {
		t.Errorf("expected MountPoint=/boot/efi, got %s", p.MountPoint)
	}
}
