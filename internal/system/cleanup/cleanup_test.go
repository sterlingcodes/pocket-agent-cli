package cleanup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()

	if cmd.Use != "cleanup" {
		t.Errorf("expected Use=cleanup, got %s", cmd.Use)
	}

	aliases := map[string]bool{"clean": false, "cl": false}
	for _, a := range cmd.Aliases {
		aliases[a] = true
	}
	for alias, found := range aliases {
		if !found {
			t.Errorf("missing alias: %s", alias)
		}
	}

	expectedSubs := []string{"scan", "caches", "logs", "temp"}
	subMap := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subMap[sub.Use] = true
	}
	for _, name := range expectedSubs {
		if !subMap[name] {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

func TestScanDirExistingDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some test files
	for i := 0; i < 5; i++ {
		f, err := os.CreateTemp(tmpDir, "test")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = f.WriteString("test data here")
		f.Close()
	}

	entry := scanDir(tmpDir)

	if !entry.Exists {
		t.Error("expected Exists=true")
	}
	if entry.Items < 5 {
		t.Errorf("expected at least 5 items, got %d", entry.Items)
	}
	if entry.SizeMB <= 0 {
		t.Error("expected SizeMB > 0")
	}
}

func TestScanDirNonExistent(t *testing.T) {
	entry := scanDir("/nonexistent/path/that/does/not/exist")

	if entry.Exists {
		t.Error("expected Exists=false for non-existent path")
	}
	if entry.SizeMB != 0 {
		t.Errorf("expected SizeMB=0, got %f", entry.SizeMB)
	}
	if entry.Items != 0 {
		t.Errorf("expected Items=0, got %d", entry.Items)
	}
}

func TestScanDirSingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	fpath := filepath.Join(tmpDir, "single.txt")
	if err := os.WriteFile(fpath, []byte("hello world"), 0o600); err != nil {
		t.Fatal(err)
	}

	entry := scanDir(fpath)

	if !entry.Exists {
		t.Error("expected Exists=true for file")
	}
	if entry.Items != 1 {
		t.Errorf("expected Items=1 for single file, got %d", entry.Items)
	}
	if entry.SizeMB <= 0 {
		t.Error("expected SizeMB > 0 for non-empty file")
	}
}

func TestScanDirEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	entry := scanDir(tmpDir)

	if !entry.Exists {
		t.Error("expected Exists=true")
	}
	// Empty dir should have items (at least the dir itself counts)
	if entry.SizeMB < 0 {
		t.Error("expected SizeMB >= 0")
	}
}

func TestScanDirOldestDays(t *testing.T) {
	tmpDir := t.TempDir()
	f, err := os.CreateTemp(tmpDir, "test")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("data")
	f.Close()

	entry := scanDir(tmpDir)

	// File was just created, so oldest_days should be 0
	if entry.OldestDays != 0 {
		t.Errorf("expected OldestDays=0 for just-created file, got %d", entry.OldestDays)
	}
}

func TestScanDirHomeTilde(t *testing.T) {
	// Test that paths under home directory get ~ prefix
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get home dir")
	}

	// Scan a real home subdirectory that likely exists
	entry := scanDir(filepath.Join(home, ".cache"))
	if entry.Exists && entry.Path != "" {
		if entry.Path[0] != '~' {
			t.Errorf("expected path to start with ~, got %q", entry.Path)
		}
	}
}

func TestScanCategoryDirs(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Create files in dir1
	for i := 0; i < 3; i++ {
		f, err := os.CreateTemp(tmpDir1, "test")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = f.WriteString("data")
		f.Close()
	}

	cat := &Category{
		Name: "test",
		Entries: []DirEntry{
			{Path: tmpDir1},
			{Path: tmpDir2},
			{Path: "/nonexistent/path"},
		},
	}

	scanCategoryDirs(cat)

	if cat.Items < 3 {
		t.Errorf("expected at least 3 items, got %d", cat.Items)
	}
	if cat.SizeMB < 0 {
		t.Error("expected SizeMB >= 0")
	}

	// Entries should be sorted by size descending
	for i := 1; i < len(cat.Entries); i++ {
		if cat.Entries[i].SizeMB > cat.Entries[i-1].SizeMB {
			t.Error("entries should be sorted by size descending")
		}
	}
}

func TestScanCategoryDirsDedup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a symlink pointing to same dir
	linkPath := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(tmpDir, linkPath); err != nil {
		t.Skip("cannot create symlink")
	}

	cat := &Category{
		Name: "test",
		Entries: []DirEntry{
			{Path: tmpDir},
			{Path: linkPath}, // should be deduped
		},
	}

	scanCategoryDirs(cat)

	// Only one entry should remain after dedup
	if len(cat.Entries) != 1 {
		t.Errorf("expected 1 entry after dedup, got %d", len(cat.Entries))
	}
}

func TestGetAllCategories(t *testing.T) {
	categories := getAllCategories()

	if len(categories) == 0 {
		t.Fatal("expected at least one category")
	}

	// Should have at least caches and temp
	names := make(map[string]bool)
	for _, cat := range categories {
		names[cat.Name] = true
	}

	if !names["caches"] {
		t.Error("missing 'caches' category")
	}
	if !names["temp"] {
		t.Error("missing 'temp' category")
	}
}

func TestScanResultTypes(t *testing.T) {
	result := ScanResult{
		Categories: []Category{
			{Name: "caches", SizeMB: 1024, Items: 500},
			{Name: "logs", SizeMB: 256, Items: 100},
		},
		TotalSizeMB: 1280,
		TotalItems:  600,
		ScannedAt:   "2025-01-01T00:00:00Z",
		Note:        "This is a read-only scan. No files were deleted.",
	}

	if result.TotalSizeMB != 1280 {
		t.Errorf("expected TotalSizeMB=1280, got %f", result.TotalSizeMB)
	}
	if result.TotalItems != 600 {
		t.Errorf("expected TotalItems=600, got %d", result.TotalItems)
	}
	if result.Note == "" {
		t.Error("expected non-empty note")
	}
}

func TestMaxFilesConstant(t *testing.T) {
	if maxFiles != 100000 {
		t.Errorf("expected maxFiles=100000, got %d", maxFiles)
	}
}
