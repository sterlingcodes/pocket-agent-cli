package obsidian

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/unstablemind/pocket/internal/common/config"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "obsidian" {
		t.Errorf("expected Use 'obsidian', got %q", cmd.Use)
	}
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	expectedSubs := []string{"vaults", "notes", "read [note]", "write [note] [content]", "search [query]", "daily", "recent"}
	for _, name := range expectedSubs {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func setupTestVault(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "obsidian-test-*")
	if err != nil {
		t.Fatal(err)
	}

	// Create vault structure
	subdir := filepath.Join(tmpDir, "subfolder")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create sample notes
	notes := map[string]string{
		"note1.md":            "# Note 1\nThis is the first note.",
		"note2.md":            "# Note 2\nThis is the second note.",
		"subfolder/nested.md": "# Nested Note\nThis is nested.",
		".hidden.md":          "# Hidden\nThis should be skipped.",
	}
	for path, content := range notes {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
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
	if err := config.Set("obsidian_vault", tmpDir); err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
		os.Remove(tmpConfig.Name())
		os.Unsetenv("POCKET_CONFIG")
	}

	return tmpDir, cleanup
}

func TestVaults_List(t *testing.T) {
	_, cleanup := setupTestVault(t)
	defer cleanup()

	cmd := newVaultsCmd()
	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestVaults_NoVaults(t *testing.T) {
	tmpConfig, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatal(err)
	}
	tmpConfig.Close()
	defer os.Remove(tmpConfig.Name())

	os.Setenv("POCKET_CONFIG", tmpConfig.Name())
	defer os.Unsetenv("POCKET_CONFIG")

	cmd := newVaultsCmd()
	err = cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestNotes_List(t *testing.T) {
	_, cleanup := setupTestVault(t)
	defer cleanup()

	cmd := newNotesCmd()
	cmd.SetArgs([]string{"--limit", "10"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestNotes_WithFolder(t *testing.T) {
	_, cleanup := setupTestVault(t)
	defer cleanup()

	cmd := newNotesCmd()
	cmd.SetArgs([]string{"--folder", "subfolder"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestNotes_InvalidFolder(t *testing.T) {
	_, cleanup := setupTestVault(t)
	defer cleanup()

	cmd := newNotesCmd()
	cmd.SetArgs([]string{"--folder", "nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for invalid folder, got nil")
	}
}

func TestRead_ExistingNote(t *testing.T) {
	_, cleanup := setupTestVault(t)
	defer cleanup()

	cmd := newReadCmd()
	cmd.SetArgs([]string{"note1"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRead_WithExtension(t *testing.T) {
	_, cleanup := setupTestVault(t)
	defer cleanup()

	cmd := newReadCmd()
	cmd.SetArgs([]string{"note1.md"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRead_NotFound(t *testing.T) {
	_, cleanup := setupTestVault(t)
	defer cleanup()

	cmd := newReadCmd()
	cmd.SetArgs([]string{"nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for nonexistent note, got nil")
	}
}

func TestWrite_NewNote(t *testing.T) {
	vaultPath, cleanup := setupTestVault(t)
	defer cleanup()

	cmd := newWriteCmd()
	cmd.SetArgs([]string{"new_note", "# New Note\nContent here"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Verify note was created
	notePath := filepath.Join(vaultPath, "new_note.md")
	content, err := os.ReadFile(notePath)
	if err != nil {
		t.Errorf("expected note to be created, got error: %v", err)
	}
	if string(content) != "# New Note\nContent here" {
		t.Errorf("expected specific content, got %q", string(content))
	}
}

func TestWrite_UpdateNote(t *testing.T) {
	vaultPath, cleanup := setupTestVault(t)
	defer cleanup()

	cmd := newWriteCmd()
	cmd.SetArgs([]string{"note1", "# Updated Content"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Verify note was updated
	notePath := filepath.Join(vaultPath, "note1.md")
	content, err := os.ReadFile(notePath)
	if err != nil {
		t.Errorf("expected note to exist, got error: %v", err)
	}
	if string(content) != "# Updated Content" {
		t.Errorf("expected updated content, got %q", string(content))
	}
}

func TestWrite_AppendMode(t *testing.T) {
	_, cleanup := setupTestVault(t)
	defer cleanup()

	cmd := newWriteCmd()
	cmd.SetArgs([]string{"note1", "\nAppended content", "--append"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestWrite_NestedPath(t *testing.T) {
	vaultPath, cleanup := setupTestVault(t)
	defer cleanup()

	cmd := newWriteCmd()
	cmd.SetArgs([]string{"newfolder/newnote", "# Nested Note"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Verify note and folder were created
	notePath := filepath.Join(vaultPath, "newfolder", "newnote.md")
	_, err = os.Stat(notePath)
	if err != nil {
		t.Errorf("expected nested note to be created, got error: %v", err)
	}
}

func TestSearch_FindContent(t *testing.T) {
	_, cleanup := setupTestVault(t)
	defer cleanup()

	cmd := newSearchCmd()
	cmd.SetArgs([]string{"first", "--limit", "10"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSearch_CaseSensitive(t *testing.T) {
	_, cleanup := setupTestVault(t)
	defer cleanup()

	cmd := newSearchCmd()
	cmd.SetArgs([]string{"FIRST", "--case-sensitive"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSearch_NoResults(t *testing.T) {
	_, cleanup := setupTestVault(t)
	defer cleanup()

	cmd := newSearchCmd()
	cmd.SetArgs([]string{"nonexistentquery123456"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error for empty results, got %v", err)
	}
}

func TestDaily_NotExisting(t *testing.T) {
	_, cleanup := setupTestVault(t)
	defer cleanup()

	cmd := newDailyCmd()
	cmd.SetArgs([]string{"--offset", "100"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestDaily_Create(t *testing.T) {
	vaultPath, cleanup := setupTestVault(t)
	defer cleanup()

	// Set daily format
	if err := config.Set("obsidian_daily_format", "2006-01-02"); err != nil {
		t.Fatal(err)
	}

	cmd := newDailyCmd()
	cmd.SetArgs([]string{"--create", "--offset", "-1"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Verify a daily note was created (check that some .md file exists)
	entries, err := os.ReadDir(vaultPath)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one .md file to exist")
	}
}

func TestDaily_WithFolder(t *testing.T) {
	vaultPath, cleanup := setupTestVault(t)
	defer cleanup()

	cmd := newDailyCmd()
	cmd.SetArgs([]string{"--create", "--folder", "dailies", "--offset", "0"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Verify dailies folder was created
	dailiesPath := filepath.Join(vaultPath, "dailies")
	_, err = os.Stat(dailiesPath)
	if err != nil {
		t.Errorf("expected dailies folder to be created, got error: %v", err)
	}
}

func TestRecent_ListNotes(t *testing.T) {
	_, cleanup := setupTestVault(t)
	defer cleanup()

	cmd := newRecentCmd()
	cmd.SetArgs([]string{"--limit", "5"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRecent_WithDays(t *testing.T) {
	_, cleanup := setupTestVault(t)
	defer cleanup()

	cmd := newRecentCmd()
	cmd.SetArgs([]string{"--days", "1"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestGetVaultPath_NoConfig(t *testing.T) {
	tmpConfig, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatal(err)
	}
	tmpConfig.Close()
	defer os.Remove(tmpConfig.Name())

	os.Setenv("POCKET_CONFIG", tmpConfig.Name())
	defer os.Unsetenv("POCKET_CONFIG")

	_, err = getVaultPath("")
	if err == nil {
		t.Error("expected error for missing vault config, got nil")
	}
}

func TestGetVaultPath_InvalidPath(t *testing.T) {
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

	if err := config.Set("obsidian_vault", "/nonexistent/path/to/vault"); err != nil {
		t.Fatal(err)
	}

	_, err = getVaultPath("")
	if err == nil {
		t.Error("expected error for nonexistent vault path, got nil")
	}
}
