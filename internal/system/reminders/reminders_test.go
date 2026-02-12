package reminders

import (
	"strings"
	"testing"
	"time"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "reminders" {
		t.Errorf("expected Use 'reminders', got %q", cmd.Use)
	}

	// Check aliases
	aliases := map[string]bool{"remind": true, "rem": true}
	for _, alias := range cmd.Aliases {
		if !aliases[alias] {
			t.Errorf("unexpected alias %q", alias)
		}
	}
	if len(cmd.Aliases) != 2 {
		t.Errorf("expected 2 aliases, got %d", len(cmd.Aliases))
	}

	// Check subcommands
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"lists", "list [name]", "add [title]", "complete [id or name]", "delete [id or name]", "today", "overdue"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestEscapeAppleScriptString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no special chars",
			input: "Buy groceries",
			want:  "Buy groceries",
		},
		{
			name:  "backslash",
			input: "C:\\path\\to\\file",
			want:  "C:\\\\path\\\\to\\\\file",
		},
		{
			name:  "double quotes",
			input: `Task: "Important"`,
			want:  `Task: \"Important\"`,
		},
		{
			name:  "both backslash and quotes",
			input: `C:\Users\"test"`,
			want:  `C:\\Users\\\"test\"`,
		},
		{
			name:  "single quote",
			input: "Don't forget",
			want:  "Don't forget", // Single quotes don't need escaping
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeAppleScriptString(tt.input)
			if got != tt.want {
				t.Errorf("escapeAppleScriptString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBoolTrueConst(t *testing.T) {
	if boolTrue != "true" {
		t.Errorf("expected boolTrue = %q, got %q", "true", boolTrue)
	}
}

func TestParseReminderLists(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   int // number of lists expected
	}{
		{
			name:   "empty output",
			output: "",
			want:   0,
		},
		{
			name:   "single list",
			output: "x-apple-reminder://list/ABC123\tReminders\t5\n",
			want:   1,
		},
		{
			name:   "multiple lists",
			output: "x-apple-reminder://list/ABC123\tReminders\t5\nx-apple-reminder://list/DEF456\tShopping\t3\n",
			want:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lists := parseReminderLists(tt.output)
			if len(lists) != tt.want {
				t.Errorf("parseReminderLists() returned %d lists, want %d", len(lists), tt.want)
			}

			// Check first list if present
			if len(lists) > 0 {
				if lists[0].Name == "" {
					t.Error("first list has empty name")
				}
				if lists[0].ID == "" {
					t.Error("first list has empty ID")
				}
			}
		})
	}
}

func TestParseReminders(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		defaultList string
		wantCount   int
	}{
		{
			name:        "empty output",
			output:      "",
			defaultList: "",
			wantCount:   0,
		},
		{
			name:        "single reminder with list",
			output:      "x-apple-reminder://REM123\tBuy milk\tShopping\tfalse\t0\t\tx",
			defaultList: "",
			wantCount:   1,
		},
		{
			name:        "multiple reminders",
			output:      "x-apple-reminder://REM123\tBuy milk\tShopping\tfalse\t5\tTuesday, January 7, 2025 at 9:00:00 AM\tDon't forget\nx-apple-reminder://REM124\tCall dentist\tPersonal\ttrue\t1\t\tx",
			defaultList: "",
			wantCount:   2,
		},
		{
			name:        "reminder with default list",
			output:      "x-apple-reminder://REM123\tBuy milk\tfalse\t0\t\tx",
			defaultList: "Reminders",
			wantCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reminders := parseReminders(tt.output, tt.defaultList)
			if len(reminders) != tt.wantCount {
				t.Errorf("parseReminders() returned %d reminders, want %d", len(reminders), tt.wantCount)
			}

			// Check first reminder if present
			if len(reminders) > 0 {
				if reminders[0].ID == "" {
					t.Error("first reminder has empty ID")
				}
				if reminders[0].Name == "" {
					t.Error("first reminder has empty name")
				}
				if tt.defaultList != "" && reminders[0].List != tt.defaultList {
					t.Errorf("expected list %q, got %q", tt.defaultList, reminders[0].List)
				}
			}
		})
	}
}

func TestParseReminders_CompletedStatus(t *testing.T) {
	// Test that completed status is parsed correctly
	output := "x-apple-reminder://REM123\tTask 1\tReminders\tfalse\t0\t\tx\nx-apple-reminder://REM124\tTask 2\tReminders\ttrue\t0\t\tx"
	reminders := parseReminders(output, "")

	if len(reminders) != 2 {
		t.Fatalf("expected 2 reminders, got %d", len(reminders))
	}

	if reminders[0].Completed {
		t.Error("first reminder should not be completed")
	}
	if !reminders[1].Completed {
		t.Error("second reminder should be completed")
	}
}

func TestParseFlexibleDate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(time.Time) bool
	}{
		{
			name:    "today",
			input:   "today",
			wantErr: false,
			check: func(t time.Time) bool {
				return t.Year() == now.Year() && t.Month() == now.Month() && t.Day() == now.Day()
			},
		},
		{
			name:    "tomorrow",
			input:   "tomorrow",
			wantErr: false,
			check: func(t time.Time) bool {
				tomorrow := now.AddDate(0, 0, 1)
				return t.Year() == tomorrow.Year() && t.Month() == tomorrow.Month() && t.Day() == tomorrow.Day()
			},
		},
		{
			name:    "next week",
			input:   "next week",
			wantErr: false,
			check: func(t time.Time) bool {
				nextWeek := now.AddDate(0, 0, 7)
				return t.Year() == nextWeek.Year() && t.Month() == nextWeek.Month() && t.Day() == nextWeek.Day()
			},
		},
		{
			name:    "YYYY-MM-DD format",
			input:   "2025-12-25",
			wantErr: false,
			check: func(t time.Time) bool {
				return t.Year() == 2025 && t.Month() == time.December && t.Day() == 25
			},
		},
		{
			name:    "YYYY-MM-DD HH:MM format",
			input:   "2025-12-25 15:30",
			wantErr: false,
			check: func(t time.Time) bool {
				return t.Year() == 2025 && t.Month() == time.December && t.Day() == 25 && t.Hour() == 15 && t.Minute() == 30
			},
		},
		{
			name:    "MM/DD/YYYY format",
			input:   "12/25/2025",
			wantErr: false,
			check: func(t time.Time) bool {
				return t.Year() == 2025 && t.Month() == time.December && t.Day() == 25
			},
		},
		{
			name:    "invalid format",
			input:   "not a date",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseFlexibleDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFlexibleDate(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.check != nil {
				if !tt.check(result) {
					t.Errorf("parseFlexibleDate(%q) = %v, failed validation check", tt.input, result)
				}
			}
		})
	}
}

func TestParseFlexibleDate_DefaultTime(t *testing.T) {
	// Test that dates without time default to 9:00 AM
	result, err := parseFlexibleDate("2025-12-25")
	if err != nil {
		t.Fatalf("parseFlexibleDate() error = %v", err)
	}

	if result.Hour() != 9 || result.Minute() != 0 {
		t.Errorf("expected default time 9:00 AM, got %02d:%02d", result.Hour(), result.Minute())
	}
}

func TestNewListsCmd(t *testing.T) {
	cmd := newListsCmd()
	if cmd.Use != "lists" {
		t.Errorf("expected Use 'lists', got %q", cmd.Use)
	}
}

func TestNewListCmd(t *testing.T) {
	cmd := newListCmd()
	if !strings.HasPrefix(cmd.Use, "list") {
		t.Errorf("expected Use to start with 'list', got %q", cmd.Use)
	}

	// Check flags
	completedFlag := cmd.Flags().Lookup("completed")
	if completedFlag == nil {
		t.Error("expected 'completed' flag")
	}
}

func TestNewAddCmd(t *testing.T) {
	cmd := newAddCmd()
	if !strings.HasPrefix(cmd.Use, "add") {
		t.Errorf("expected Use to start with 'add', got %q", cmd.Use)
	}

	// Check flags
	flags := []string{"list", "due", "notes", "priority"}
	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected %q flag", flagName)
		}
	}
}

func TestNewCompleteCmd(t *testing.T) {
	cmd := newCompleteCmd()
	if !strings.HasPrefix(cmd.Use, "complete") {
		t.Errorf("expected Use to start with 'complete', got %q", cmd.Use)
	}
}

func TestNewDeleteCmd(t *testing.T) {
	cmd := newDeleteCmd()
	if !strings.HasPrefix(cmd.Use, "delete") {
		t.Errorf("expected Use to start with 'delete', got %q", cmd.Use)
	}
}

func TestNewTodayCmd(t *testing.T) {
	cmd := newTodayCmd()
	if cmd.Use != "today" {
		t.Errorf("expected Use 'today', got %q", cmd.Use)
	}
}

func TestNewOverdueCmd(t *testing.T) {
	cmd := newOverdueCmd()
	if cmd.Use != "overdue" {
		t.Errorf("expected Use 'overdue', got %q", cmd.Use)
	}
}
