package cleanup

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/pkg/output"
)

// ScanResult holds the full cleanup scan results
type ScanResult struct {
	Categories  []Category `json:"categories"`
	TotalSizeMB float64    `json:"total_size_mb"`
	TotalItems  int        `json:"total_items"`
	ScannedAt   string     `json:"scanned_at"`
	Note        string     `json:"note"`
}

// Category is a group of cleanable directories
type Category struct {
	Name    string     `json:"name"`
	SizeMB  float64    `json:"size_mb"`
	Items   int        `json:"items"`
	Entries []DirEntry `json:"entries"`
}

// DirEntry is a single directory with its size
type DirEntry struct {
	Path       string  `json:"path"`
	SizeMB     float64 `json:"size_mb"`
	Items      int     `json:"items"`
	OldestDays int     `json:"oldest_days,omitempty"`
	Exists     bool    `json:"exists"`
}

const maxFiles = 100000 // Safety cap for directory walking

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cleanup",
		Aliases: []string{"clean", "cl"},
		Short:   "Cache/temp/log diagnostic (read-only, never deletes)",
		Long:    "Scans cleanable directories and reports sizes. Does NOT delete anything.",
	}

	cmd.AddCommand(newScanCmd())
	cmd.AddCommand(newCachesCmd())
	cmd.AddCommand(newLogsCmd())
	cmd.AddCommand(newTempCmd())

	return cmd
}

func newScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Full scan of all cleanable directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fullScan()
		},
	}
}

func newCachesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "caches",
		Short: "Scan cache directories only",
		RunE: func(cmd *cobra.Command, args []string) error {
			return scanCategory("caches")
		},
	}
}

func newLogsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logs",
		Short: "Scan log directories only",
		RunE: func(cmd *cobra.Command, args []string) error {
			return scanCategory("logs")
		},
	}
}

func newTempCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "temp",
		Short: "Scan temp directories only",
		RunE: func(cmd *cobra.Command, args []string) error {
			return scanCategory("temp")
		},
	}
}

func fullScan() error {
	categories := getAllCategories()

	var totalSize float64
	var totalItems int

	for i := range categories {
		scanCategoryDirs(&categories[i])
		totalSize += categories[i].SizeMB
		totalItems += categories[i].Items
	}

	return output.Print(ScanResult{
		Categories:  categories,
		TotalSizeMB: totalSize,
		TotalItems:  totalItems,
		ScannedAt:   time.Now().Format(time.RFC3339),
		Note:        "This is a read-only scan. No files were deleted.",
	})
}

func scanCategory(name string) error {
	categories := getAllCategories()

	for i := range categories {
		if categories[i].Name == name {
			scanCategoryDirs(&categories[i])
			return output.Print(categories[i])
		}
	}

	return output.PrintError("not_found",
		fmt.Sprintf("category '%s' not found", name),
		map[string]string{"available": "caches, logs, temp"})
}

func getAllCategories() []Category {
	home, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "darwin":
		return getDarwinCategories(home)
	case "linux":
		return getLinuxCategories(home)
	default:
		return getGenericCategories(home)
	}
}

func getDarwinCategories(home string) []Category {
	return []Category{
		{
			Name: "caches",
			Entries: []DirEntry{
				{Path: filepath.Join(home, "Library", "Caches")},
				{Path: "/Library/Caches"},
				{Path: filepath.Join(home, "Library", "Developer", "Xcode", "DerivedData")},
				{Path: filepath.Join(home, "Library", "Caches", "Homebrew")},
				{Path: filepath.Join(home, ".cache")},
			},
		},
		{
			Name: "logs",
			Entries: []DirEntry{
				{Path: filepath.Join(home, "Library", "Logs")},
				{Path: "/Library/Logs"},
				{Path: "/var/log"},
			},
		},
		{
			Name: "temp",
			Entries: []DirEntry{
				{Path: os.TempDir()},
				{Path: "/tmp"},
				{Path: filepath.Join(home, "Library", "Saved Application State")},
			},
		},
	}
}

func getLinuxCategories(home string) []Category {
	return []Category{
		{
			Name: "caches",
			Entries: []DirEntry{
				{Path: filepath.Join(home, ".cache")},
				{Path: "/var/cache"},
			},
		},
		{
			Name: "logs",
			Entries: []DirEntry{
				{Path: "/var/log"},
				{Path: filepath.Join(home, ".local", "share", "logs")},
			},
		},
		{
			Name: "temp",
			Entries: []DirEntry{
				{Path: "/tmp"},
				{Path: "/var/tmp"},
			},
		},
	}
}

func getGenericCategories(home string) []Category {
	return []Category{
		{
			Name: "caches",
			Entries: []DirEntry{
				{Path: filepath.Join(home, ".cache")},
			},
		},
		{
			Name: "temp",
			Entries: []DirEntry{
				{Path: os.TempDir()},
			},
		},
	}
}

func scanCategoryDirs(cat *Category) {
	var totalSize float64
	var totalItems int

	// Deduplicate paths (e.g., /tmp and os.TempDir() may be the same)
	seen := make(map[string]bool)

	entries := make([]DirEntry, 0, len(cat.Entries))
	for _, entry := range cat.Entries {
		// Resolve symlinks to avoid double-counting
		resolved, err := filepath.EvalSymlinks(entry.Path)
		if err != nil {
			resolved = entry.Path
		}

		if seen[resolved] {
			continue
		}
		seen[resolved] = true

		scanned := scanDir(entry.Path)
		entries = append(entries, scanned)
		totalSize += scanned.SizeMB
		totalItems += scanned.Items
	}

	// Sort by size descending
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].SizeMB > entries[j].SizeMB
	})

	cat.Entries = entries
	cat.SizeMB = totalSize
	cat.Items = totalItems
}

func scanDir(path string) DirEntry {
	entry := DirEntry{
		Path: path,
	}

	info, err := os.Stat(path)
	if err != nil {
		entry.Exists = false
		return entry
	}

	if !info.IsDir() {
		entry.Exists = true
		entry.SizeMB = float64(info.Size()) / (1024 * 1024)
		entry.Items = 1
		return entry
	}

	entry.Exists = true

	var totalBytes int64
	var count int
	var oldestMod time.Time
	now := time.Now()

	_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			// Skip permission-denied directories
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		count++
		if count > maxFiles {
			return filepath.SkipAll
		}

		if !d.IsDir() {
			if fi, err := d.Info(); err == nil {
				totalBytes += fi.Size()
				if oldestMod.IsZero() || fi.ModTime().Before(oldestMod) {
					oldestMod = fi.ModTime()
				}
			}
		}

		return nil
	})

	entry.SizeMB = float64(totalBytes) / (1024 * 1024)
	entry.Items = count

	if !oldestMod.IsZero() {
		entry.OldestDays = int(now.Sub(oldestMod).Hours() / 24)
	}

	// Clean up display path
	if home, err := os.UserHomeDir(); err == nil {
		if strings.HasPrefix(entry.Path, home) {
			entry.Path = "~" + entry.Path[len(home):]
		}
	}

	return entry
}
