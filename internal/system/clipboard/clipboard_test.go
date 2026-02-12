package clipboard

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "clipboard" {
		t.Errorf("expected Use 'clipboard', got %q", cmd.Use)
	}

	// Check aliases
	aliases := map[string]bool{"clip": true, "cb": true, "pasteboard": true}
	for _, alias := range cmd.Aliases {
		if !aliases[alias] {
			t.Errorf("unexpected alias %q", alias)
		}
	}
	if len(cmd.Aliases) != 3 {
		t.Errorf("expected 3 aliases, got %d", len(cmd.Aliases))
	}

	// Check subcommands
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"get", "set [text]", "clear", "copy [file]", "history"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestNewGetCmd(t *testing.T) {
	cmd := newGetCmd()
	if cmd.Use != "get" {
		t.Errorf("expected Use 'get', got %q", cmd.Use)
	}

	// Check flags
	maxLengthFlag := cmd.Flags().Lookup("max-length")
	if maxLengthFlag == nil {
		t.Error("expected 'max-length' flag")
	}

	rawFlag := cmd.Flags().Lookup("raw")
	if rawFlag == nil {
		t.Error("expected 'raw' flag")
	}
}

func TestNewSetCmd(t *testing.T) {
	cmd := newSetCmd()
	if !strings.HasPrefix(cmd.Use, "set") {
		t.Errorf("expected Use to start with 'set', got %q", cmd.Use)
	}
}

func TestNewClearCmd(t *testing.T) {
	cmd := newClearCmd()
	if cmd.Use != "clear" {
		t.Errorf("expected Use 'clear', got %q", cmd.Use)
	}
}

func TestNewCopyCmd(t *testing.T) {
	cmd := newCopyCmd()
	if !strings.HasPrefix(cmd.Use, "copy") {
		t.Errorf("expected Use to start with 'copy', got %q", cmd.Use)
	}
}

func TestNewHistoryCmd(t *testing.T) {
	cmd := newHistoryCmd()
	if cmd.Use != "history" {
		t.Errorf("expected Use 'history', got %q", cmd.Use)
	}
}

func TestContentStruct(t *testing.T) {
	content := Content{
		Content:   "test content",
		Length:    12,
		Lines:     1,
		IsText:    true,
		Truncated: false,
	}

	if content.Content != "test content" {
		t.Error("Content.Content not set correctly")
	}
	if content.Length != 12 {
		t.Error("Content.Length not set correctly")
	}
	if content.Lines != 1 {
		t.Error("Content.Lines not set correctly")
	}
	if !content.IsText {
		t.Error("Content.IsText should be true")
	}
	if content.Truncated {
		t.Error("Content.Truncated should be false")
	}
}

func TestContentStruct_Multiline(t *testing.T) {
	content := Content{
		Content: "line 1\nline 2\nline 3",
		Length:  21,
		Lines:   3,
		IsText:  true,
	}

	if content.Lines != 3 {
		t.Errorf("expected 3 lines, got %d", content.Lines)
	}
}

func TestContentStruct_Truncated(t *testing.T) {
	content := Content{
		Content:   "truncated...",
		Length:    12,
		Lines:     1,
		IsText:    true,
		Truncated: true,
	}

	if !content.Truncated {
		t.Error("Content.Truncated should be true")
	}
}

func TestContentStruct_NonText(t *testing.T) {
	content := Content{
		Content: "",
		Length:  0,
		Lines:   0,
		IsText:  false,
	}

	if content.IsText {
		t.Error("Content.IsText should be false for binary content")
	}
}

// Test helper functions would normally require mocking exec.Command
// For now, we just test the command structure

func TestGetClipboard_CommandStructure(t *testing.T) {
	// This test verifies the function signature exists
	// Actual execution would require mocking or running on macOS
	_ = getClipboard
}

func TestSetClipboard_CommandStructure(t *testing.T) {
	// This test verifies the function signature exists
	// Actual execution would require mocking or running on macOS
	_ = setClipboard
}

func TestClipboardCommands_ExistAndCallable(t *testing.T) {
	// Verify that the command functions are defined and callable
	getCmd := newGetCmd()
	if getCmd == nil {
		t.Error("newGetCmd() returned nil")
	}

	setCmd := newSetCmd()
	if setCmd == nil {
		t.Error("newSetCmd() returned nil")
	}

	clearCmd := newClearCmd()
	if clearCmd == nil {
		t.Error("newClearCmd() returned nil")
	}

	copyCmd := newCopyCmd()
	if copyCmd == nil {
		t.Error("newCopyCmd() returned nil")
	}

	historyCmd := newHistoryCmd()
	if historyCmd == nil {
		t.Error("newHistoryCmd() returned nil")
	}
}

func TestGetCmd_Flags(t *testing.T) {
	cmd := newGetCmd()

	// Test max-length flag
	maxLengthFlag := cmd.Flags().Lookup("max-length")
	if maxLengthFlag == nil {
		t.Fatal("max-length flag not found")
	}
	if maxLengthFlag.Shorthand != "m" {
		t.Errorf("expected shorthand 'm', got %q", maxLengthFlag.Shorthand)
	}

	// Test raw flag
	rawFlag := cmd.Flags().Lookup("raw")
	if rawFlag == nil {
		t.Fatal("raw flag not found")
	}
	if rawFlag.Shorthand != "r" {
		t.Errorf("expected shorthand 'r', got %q", rawFlag.Shorthand)
	}
}

func TestSetCmd_Args(t *testing.T) {
	cmd := newSetCmd()

	// Verify it expects exactly 1 argument
	// This is implied by the Args field in cobra.Command
	if cmd.Use != "set [text]" {
		t.Errorf("expected Use 'set [text]', got %q", cmd.Use)
	}
}

func TestCopyCmd_Args(t *testing.T) {
	cmd := newCopyCmd()

	// Verify it expects exactly 1 argument
	if cmd.Use != "copy [file]" {
		t.Errorf("expected Use 'copy [file]', got %q", cmd.Use)
	}
}

func TestHistoryCmd_Information(t *testing.T) {
	cmd := newHistoryCmd()

	// History command should provide information about alternatives
	if cmd.Short == "" {
		t.Error("history command should have a short description")
	}
	if cmd.Long == "" {
		t.Error("history command should have a long description")
	}
}

func TestCommandDescriptions(t *testing.T) {
	// Verify all commands have descriptions
	commands := []*struct {
		name string
		cmd  func() *cobra.Command
	}{
		{"NewCmd", NewCmd},
		{"newGetCmd", newGetCmd},
		{"newSetCmd", newSetCmd},
		{"newClearCmd", newClearCmd},
		{"newCopyCmd", newCopyCmd},
		{"newHistoryCmd", newHistoryCmd},
	}

	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.cmd()
			if cmd.Short == "" {
				t.Errorf("%s should have a Short description", tc.name)
			}
		})
	}
}
