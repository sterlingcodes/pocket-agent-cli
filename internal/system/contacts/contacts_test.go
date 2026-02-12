package contacts

import (
	"strings"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "contacts" {
		t.Errorf("expected Use 'contacts', got %q", cmd.Use)
	}

	// Check aliases
	aliases := map[string]bool{"contact": true, "addr": true, "addressbook": true}
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
	for _, name := range []string{"list", "search [query]", "get [name]", "groups", "group [name]", "create [name]"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestEscapeAppleScript(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no special chars",
			input: "John Doe",
			want:  "John Doe",
		},
		{
			name:  "backslash",
			input: "C:\\path",
			want:  "C:\\\\path",
		},
		{
			name:  "double quotes",
			input: `Name: "John"`,
			want:  `Name: \"John\"`,
		},
		{
			name:  "both backslash and quotes",
			input: `C:\Users\"test"`,
			want:  `C:\\Users\\\"test\"`,
		},
		{
			name:  "single quote",
			input: "O'Brien",
			want:  "O'Brien", // Single quotes don't need escaping
		},
		{
			name:  "email address",
			input: "test@example.com",
			want:  "test@example.com",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeAppleScript(tt.input)
			if got != tt.want {
				t.Errorf("escapeAppleScript(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCleanLabel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "home label",
			input: "_$!<Home>!$_",
			want:  "Home",
		},
		{
			name:  "work label",
			input: "_$!<Work>!$_",
			want:  "Work",
		},
		{
			name:  "mobile label",
			input: "_$!<Mobile>!$_",
			want:  "Mobile",
		},
		{
			name:  "plain label",
			input: "Custom",
			want:  "Custom",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "partial format",
			input: "_$!<Test",
			want:  "Test",
		},
		{
			name:  "other partial format",
			input: "Test>!$_",
			want:  "Test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanLabel(tt.input)
			if got != tt.want {
				t.Errorf("cleanLabel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNewListCmd(t *testing.T) {
	cmd := newListCmd()
	if cmd.Use != "list" {
		t.Errorf("expected Use 'list', got %q", cmd.Use)
	}

	// Check flags
	limitFlag := cmd.Flags().Lookup("limit")
	if limitFlag == nil {
		t.Error("expected 'limit' flag")
	}
}

func TestNewSearchCmd(t *testing.T) {
	cmd := newSearchCmd()
	if !strings.HasPrefix(cmd.Use, "search") {
		t.Errorf("expected Use to start with 'search', got %q", cmd.Use)
	}

	// Check flags
	limitFlag := cmd.Flags().Lookup("limit")
	if limitFlag == nil {
		t.Error("expected 'limit' flag")
	}
}

func TestNewGetCmd(t *testing.T) {
	cmd := newGetCmd()
	if !strings.HasPrefix(cmd.Use, "get") {
		t.Errorf("expected Use to start with 'get', got %q", cmd.Use)
	}
}

func TestNewGroupsCmd(t *testing.T) {
	cmd := newGroupsCmd()
	if cmd.Use != "groups" {
		t.Errorf("expected Use 'groups', got %q", cmd.Use)
	}
}

func TestNewGroupCmd(t *testing.T) {
	cmd := newGroupCmd()
	if !strings.HasPrefix(cmd.Use, "group") {
		t.Errorf("expected Use to start with 'group', got %q", cmd.Use)
	}

	// Check flags
	limitFlag := cmd.Flags().Lookup("limit")
	if limitFlag == nil {
		t.Error("expected 'limit' flag")
	}
}

func TestNewCreateCmd(t *testing.T) {
	cmd := newCreateCmd()
	if !strings.HasPrefix(cmd.Use, "create") {
		t.Errorf("expected Use to start with 'create', got %q", cmd.Use)
	}

	// Check flags
	flags := []string{"email", "phone", "company", "note"}
	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected %q flag", flagName)
		}
	}
}

func TestContactStructs(t *testing.T) {
	// Test Contact struct
	contact := Contact{
		Name:      "John Doe",
		FirstName: "John",
		LastName:  "Doe",
		Company:   "Acme Inc",
		Emails: []Email{
			{Label: "work", Value: "john@acme.com"},
		},
		Phones: []Phone{
			{Label: "mobile", Value: "555-1234"},
		},
	}

	if contact.Name != "John Doe" {
		t.Error("Contact.Name not set correctly")
	}
	if len(contact.Emails) != 1 {
		t.Error("Contact.Emails not set correctly")
	}
	if len(contact.Phones) != 1 {
		t.Error("Contact.Phones not set correctly")
	}
}

func TestEmailStruct(t *testing.T) {
	email := Email{
		Label: "work",
		Value: "test@example.com",
	}

	if email.Label != "work" || email.Value != "test@example.com" {
		t.Error("Email struct fields not set correctly")
	}
}

func TestPhoneStruct(t *testing.T) {
	phone := Phone{
		Label: "mobile",
		Value: "555-1234",
	}

	if phone.Label != "mobile" || phone.Value != "555-1234" {
		t.Error("Phone struct fields not set correctly")
	}
}

func TestAddressStruct(t *testing.T) {
	addr := Address{
		Label:   "home",
		Street:  "123 Main St",
		City:    "Springfield",
		State:   "IL",
		Zip:     "62701",
		Country: "USA",
	}

	if addr.Street != "123 Main St" || addr.City != "Springfield" {
		t.Error("Address struct fields not set correctly")
	}
}

func TestGroupStruct(t *testing.T) {
	group := Group{
		Name:  "Friends",
		Count: 10,
	}

	if group.Name != "Friends" || group.Count != 10 {
		t.Error("Group struct fields not set correctly")
	}
}

func TestContactSummaryStruct(t *testing.T) {
	summary := ContactSummary{
		Name:    "John Doe",
		Email:   "john@example.com",
		Phone:   "555-1234",
		Company: "Acme Inc",
	}

	if summary.Name != "John Doe" || summary.Email != "john@example.com" {
		t.Error("ContactSummary struct fields not set correctly")
	}
}

func TestCleanLabel_MultipleCalls(t *testing.T) {
	// Test that cleanLabel is idempotent
	input := "_$!<Home>!$_"
	first := cleanLabel(input)
	second := cleanLabel(first)

	if first != "Home" {
		t.Errorf("first call: got %q, want 'Home'", first)
	}
	if second != "Home" {
		t.Errorf("second call: got %q, want 'Home' (should be idempotent)", second)
	}
}

func TestEscapeAppleScript_LongString(t *testing.T) {
	// Test with a long string containing multiple special characters
	input := strings.Repeat(`"test\path"`, 100)
	result := escapeAppleScript(input)

	// Should have escaped all quotes and backslashes
	expectedQuoteCount := strings.Count(input, `"`) * 2 // Each quote becomes \"
	resultQuoteCount := strings.Count(result, `\`)

	if resultQuoteCount < expectedQuoteCount {
		t.Error("Not all special characters were escaped in long string")
	}
}
