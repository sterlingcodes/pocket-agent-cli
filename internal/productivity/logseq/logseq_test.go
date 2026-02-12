package logseq

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/unstablemind/pocket/internal/common/config"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "logseq" {
		t.Errorf("expected Use 'logseq', got %q", cmd.Use)
	}
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	expectedSubs := []string{"graphs", "pages", "read [page]", "write [page] [content]", "search [query]", "journal", "recent"}
	for _, name := range expectedSubs {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func setupTestGraph(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "logseq-test-*")
	if err != nil {
		t.Fatal(err)
	}

	// Create graph structure
	pagesDir := filepath.Join(tmpDir, "pages")
	journalsDir := filepath.Join(tmpDir, "journals")
	if err := os.MkdirAll(pagesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(journalsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create sample pages
	pages := map[string]string{
		"test_page.md":      "# Test Page\n- This is a test\n- [[linked page]]",
		"another.md":        "# Another Page\n- More content",
		"special%2Fchar.md": "# Special/Char\n- Testing encoding",
	}
	for name, content := range pages {
		err := os.WriteFile(filepath.Join(pagesDir, name), []byte(content), 0600)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Create sample journals
	journals := map[string]string{
		"2024_01_01.md": "- Daily note for Jan 1",
		"2024_01_02.md": "- Daily note for Jan 2",
	}
	for name, content := range journals {
		err := os.WriteFile(filepath.Join(journalsDir, name), []byte(content), 0600)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Setup config
	tmpConfig, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatal(err)
	}
	// Write empty JSON object
	if _, err := tmpConfig.Write([]byte("{}")); err != nil {
		t.Fatal(err)
	}
	tmpConfig.Close()

	os.Setenv("POCKET_CONFIG", tmpConfig.Name())
	if err := config.Set("logseq_graph", tmpDir); err != nil {
		t.Fatal(err)
	}
	if err := config.Set("logseq_format", "md"); err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
		os.Remove(tmpConfig.Name())
		os.Unsetenv("POCKET_CONFIG")
	}

	return tmpDir, cleanup
}

func TestPages_List(t *testing.T) {
	_, cleanup := setupTestGraph(t)
	defer cleanup()

	cmd := newPagesCmd()
	cmd.SetArgs([]string{"--limit", "10"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRead_ExistingPage(t *testing.T) {
	_, cleanup := setupTestGraph(t)
	defer cleanup()

	cmd := newReadCmd()
	cmd.SetArgs([]string{"test_page"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRead_EncodedPage(t *testing.T) {
	_, cleanup := setupTestGraph(t)
	defer cleanup()

	cmd := newReadCmd()
	cmd.SetArgs([]string{"special/char"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRead_NotFound(t *testing.T) {
	_, cleanup := setupTestGraph(t)
	defer cleanup()

	cmd := newReadCmd()
	cmd.SetArgs([]string{"nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for nonexistent page, got nil")
	}
}

func TestWrite_NewPage(t *testing.T) {
	graphPath, cleanup := setupTestGraph(t)
	defer cleanup()

	cmd := newWriteCmd()
	cmd.SetArgs([]string{"new_page", "New content for testing"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Verify page was created
	pagePath := filepath.Join(graphPath, "pages", "new_page.md")
	content, err := os.ReadFile(pagePath)
	if err != nil {
		t.Errorf("expected page to be created, got error: %v", err)
	}
	if string(content) != "New content for testing" {
		t.Errorf("expected content 'New content for testing', got %q", string(content))
	}
}

func TestWrite_AppendMode(t *testing.T) {
	_, cleanup := setupTestGraph(t)
	defer cleanup()

	cmd := newWriteCmd()
	cmd.SetArgs([]string{"test_page", "Appended content", "--append"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSearch_FindContent(t *testing.T) {
	_, cleanup := setupTestGraph(t)
	defer cleanup()

	cmd := newSearchCmd()
	cmd.SetArgs([]string{"test", "--limit", "10"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSearch_CaseSensitive(t *testing.T) {
	_, cleanup := setupTestGraph(t)
	defer cleanup()

	cmd := newSearchCmd()
	cmd.SetArgs([]string{"TEST", "--case-sensitive"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestJournal_Existing(t *testing.T) {
	_, cleanup := setupTestGraph(t)
	defer cleanup()

	cmd := newJournalCmd()
	cmd.SetArgs([]string{"--date", "2024-01-01"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestJournal_NotExisting(t *testing.T) {
	_, cleanup := setupTestGraph(t)
	defer cleanup()

	cmd := newJournalCmd()
	cmd.SetArgs([]string{"--date", "2025-12-31"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestJournal_WithContent(t *testing.T) {
	graphPath, cleanup := setupTestGraph(t)
	defer cleanup()

	cmd := newJournalCmd()
	cmd.SetArgs([]string{"--date", "2024-01-15", "--content", "New journal entry"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Verify journal was created
	journalPath := filepath.Join(graphPath, "journals", "2024_01_15.md")
	content, err := os.ReadFile(journalPath)
	if err != nil {
		t.Errorf("expected journal to be created, got error: %v", err)
	}
	if string(content) != "New journal entry" {
		t.Errorf("expected content 'New journal entry', got %q", string(content))
	}
}

func TestRecent_ListPages(t *testing.T) {
	_, cleanup := setupTestGraph(t)
	defer cleanup()

	cmd := newRecentCmd()
	cmd.SetArgs([]string{"--limit", "5", "--days", "30"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestEncodeDecodPageName(t *testing.T) {
	tests := []struct {
		decoded string
		encoded string
	}{
		{"simple", "simple"},
		{"with/slash", "with%2Fslash"},
		{"with:colon", "with%3Acolon"},
		{"with?question", "with%3Fquestion"},
		{"with#hash", "with%23hash"},
		{"with&ampersand", "with%26ampersand"},
		{"with%percent", "with%25percent"},
	}

	for _, tt := range tests {
		encoded := encodePageName(tt.decoded)
		if encoded != tt.encoded {
			t.Errorf("encodePageName(%q) = %q, want %q", tt.decoded, encoded, tt.encoded)
		}

		decoded := decodePageName(tt.encoded)
		if decoded != tt.decoded {
			t.Errorf("decodePageName(%q) = %q, want %q", tt.encoded, decoded, tt.decoded)
		}
	}
}

func TestGetGraphPath_NoConfig(t *testing.T) {
	tmpConfig, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatal(err)
	}
	// Write empty JSON object
	if _, err := tmpConfig.Write([]byte("{}")); err != nil {
		t.Fatal(err)
	}
	tmpConfig.Close()
	defer os.Remove(tmpConfig.Name())

	os.Setenv("POCKET_CONFIG", tmpConfig.Name())
	defer os.Unsetenv("POCKET_CONFIG")

	// Check if previous tests left state (due to sync.Once in config package)
	graph, _ := config.Get("logseq_graph")
	if graph != "" {
		t.Skip("config state persists across tests due to sync.Once")
	}

	_, _, err = getGraphPath("")
	if err == nil {
		t.Error("expected error for missing graph config, got nil")
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2024-01-15", "2024-01-15"},
		{"2024/01/15", "2024-01-15"},
		{"01-15-2024", "2024-01-15"},
		{"01/15/2024", "2024-01-15"},
	}

	for _, tt := range tests {
		result, err := parseDate(tt.input)
		if err != nil {
			t.Errorf("parseDate(%q) returned error: %v", tt.input, err)
			continue
		}
		if result.Format("2006-01-02") != tt.expected {
			t.Errorf("parseDate(%q) = %s, want %s", tt.input, result.Format("2006-01-02"), tt.expected)
		}
	}
}

func TestParseDate_Invalid(t *testing.T) {
	_, err := parseDate("not a date")
	if err == nil {
		t.Error("expected error for invalid date, got nil")
	}
}
