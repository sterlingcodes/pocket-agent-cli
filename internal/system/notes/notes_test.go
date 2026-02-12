package notes

import (
	"strings"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "notes" {
		t.Errorf("expected Use 'notes', got %q", cmd.Use)
	}

	// Check aliases
	aliases := map[string]bool{"note": true, "applenotes": true}
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
	for _, name := range []string{"list", "folders", "read [name]", "create [name] [body]", "search [query]", "append [name] [text]"} {
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
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "backslash",
			input: "path\\to\\file",
			want:  "path\\\\to\\\\file",
		},
		{
			name:  "double quotes",
			input: `he said "hello"`,
			want:  `he said \"hello\"`,
		},
		{
			name:  "both backslash and quotes",
			input: `C:\Users\"test"`,
			want:  `C:\\Users\\\"test\"`,
		},
		{
			name:  "single quote",
			input: "it's a test",
			want:  "it's a test", // Single quotes don't need escaping
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "multiple quotes",
			input: `"first" and "second"`,
			want:  `\"first\" and \"second\"`,
		},
		{
			name:  "multiple backslashes",
			input: `\\server\share`,
			want:  `\\\\server\\share`,
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

func TestHtmlToPlaintext(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "paragraph tags",
			input: "<p>First paragraph</p><p>Second paragraph</p>",
			want:  "First paragraph\nSecond paragraph",
		},
		{
			name:  "line breaks",
			input: "Line 1<br/>Line 2<br />Line 3",
			want:  "Line 1\nLine 2\nLine 3",
		},
		{
			name:  "list items",
			input: "<ul><li>Item 1</li><li>Item 2</li></ul>",
			want:  "• Item 1\n• Item 2",
		},
		{
			name:  "div tags",
			input: "<div>Block 1</div><div>Block 2</div>",
			want:  "Block 1\nBlock 2",
		},
		{
			name:  "html entities",
			input: "&lt;tag&gt; &amp; &quot;quoted&quot; &#39;apostrophe&#39; &nbsp;",
			want:  "<tag> & \"quoted\" 'apostrophe'",
		},
		{
			name:  "html entities without semicolon",
			input: "&amp test",
			want:  "& test",
		},
		{
			name:  "headers",
			input: "<h1>Title</h1><h2>Subtitle</h2>",
			want:  "Title\nSubtitle",
		},
		{
			name:  "multiple newlines collapse",
			input: "Line 1\n\n\n\nLine 2",
			want:  "Line 1\n\nLine 2",
		},
		{
			name:  "mixed content",
			input: "<div>Hello <strong>world</strong>!</div><p>New paragraph</p>",
			want:  "Hello world!\nNew paragraph",
		},
		{
			name:  "carriage returns",
			input: "Line 1\r\nLine 2\rLine 3",
			want:  "Line 1\nLine 2\nLine 3",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "self-closing tags",
			input: "<div/>Content<br/>More",
			want:  "Content\nMore",
		},
		{
			name:  "table rows",
			input: "<tr><td>Cell 1</td></tr><tr><td>Cell 2</td></tr>",
			want:  "Cell 1\nCell 2",
		},
		{
			name:  "whitespace trimming",
			input: "   \n   Content   \n   ",
			want:  "Content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := htmlToPlaintext(tt.input)
			if got != tt.want {
				t.Errorf("htmlToPlaintext() =\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}

func TestHtmlToPlaintext_ComplexHTML(t *testing.T) {
	input := `<div>
		<h1>Meeting Notes</h1>
		<p>Attended by: John &amp; Jane</p>
		<ul>
			<li>Item 1</li>
			<li>Item 2</li>
		</ul>
		<p>See &lt;README&gt; for details.</p>
	</div>`

	result := htmlToPlaintext(input)

	// Check that key elements are present
	if !strings.Contains(result, "Meeting Notes") {
		t.Error("Expected 'Meeting Notes' in result")
	}
	if !strings.Contains(result, "John & Jane") {
		t.Error("Expected 'John & Jane' (unescaped ampersand) in result")
	}
	if !strings.Contains(result, "• Item 1") {
		t.Error("Expected '• Item 1' (bullet list) in result")
	}
	if !strings.Contains(result, "<README>") {
		t.Error("Expected '<README>' (unescaped HTML entities) in result")
	}

	// Check that HTML tags are removed
	if strings.Contains(result, "<div>") || strings.Contains(result, "</div>") {
		t.Error("HTML tags should be removed")
	}
}

func TestNewListCmd(t *testing.T) {
	cmd := newListCmd()
	if cmd.Use != "list" {
		t.Errorf("expected Use 'list', got %q", cmd.Use)
	}

	// Check flags
	folderFlag := cmd.Flags().Lookup("folder")
	if folderFlag == nil {
		t.Error("expected 'folder' flag")
	}
	limitFlag := cmd.Flags().Lookup("limit")
	if limitFlag == nil {
		t.Error("expected 'limit' flag")
	}
}

func TestNewReadCmd(t *testing.T) {
	cmd := newReadCmd()
	if cmd.Use != "read [name]" {
		t.Errorf("expected Use 'read [name]', got %q", cmd.Use)
	}

	// Check flags
	folderFlag := cmd.Flags().Lookup("folder")
	if folderFlag == nil {
		t.Error("expected 'folder' flag")
	}
}

func TestNewCreateCmd(t *testing.T) {
	cmd := newCreateCmd()
	if cmd.Use != "create [name] [body]" {
		t.Errorf("expected Use 'create [name] [body]', got %q", cmd.Use)
	}

	// Check flags
	folderFlag := cmd.Flags().Lookup("folder")
	if folderFlag == nil {
		t.Error("expected 'folder' flag")
	}
}

func TestNewSearchCmd(t *testing.T) {
	cmd := newSearchCmd()
	if cmd.Use != "search [query]" {
		t.Errorf("expected Use 'search [query]', got %q", cmd.Use)
	}

	// Check flags
	limitFlag := cmd.Flags().Lookup("limit")
	if limitFlag == nil {
		t.Error("expected 'limit' flag")
	}
}

func TestNewAppendCmd(t *testing.T) {
	cmd := newAppendCmd()
	if cmd.Use != "append [name] [text]" {
		t.Errorf("expected Use 'append [name] [text]', got %q", cmd.Use)
	}

	// Check flags
	folderFlag := cmd.Flags().Lookup("folder")
	if folderFlag == nil {
		t.Error("expected 'folder' flag")
	}
}

func TestNewFoldersCmd(t *testing.T) {
	cmd := newFoldersCmd()
	if cmd.Use != "folders" {
		t.Errorf("expected Use 'folders', got %q", cmd.Use)
	}
}
