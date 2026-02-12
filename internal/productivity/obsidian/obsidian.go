package obsidian

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

// NoteInfo represents metadata about a note
type NoteInfo struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	Folder   string    `json:"folder"`
	Modified time.Time `json:"modified"`
	Size     int64     `json:"size"`
}

// VaultInfo represents an Obsidian vault
type VaultInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// NewCmd returns the obsidian command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "obsidian",
		Aliases: []string{"obs", "vault"},
		Short:   "Obsidian vault commands",
		Long:    `Interact with local Obsidian vaults (markdown files).`,
	}

	cmd.AddCommand(newVaultsCmd())
	cmd.AddCommand(newNotesCmd())
	cmd.AddCommand(newReadCmd())
	cmd.AddCommand(newWriteCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newDailyCmd())
	cmd.AddCommand(newRecentCmd())

	return cmd
}

// getVaultPath returns the configured vault path
func getVaultPath(vaultName string) (string, error) {
	// If a specific vault name is provided, look it up in the vaults list
	if vaultName != "" {
		vaultsJSON, err := config.Get("obsidian_vaults")
		if err == nil && vaultsJSON != "" {
			var vaults []VaultInfo
			if err := json.Unmarshal([]byte(vaultsJSON), &vaults); err == nil {
				for _, v := range vaults {
					if v.Name == vaultName {
						return v.Path, nil
					}
				}
			}
		}
		// Treat vaultName as a path if not found in vaults list
		if _, err := os.Stat(vaultName); err == nil {
			return vaultName, nil
		}
		return "", fmt.Errorf("vault not found: %s", vaultName)
	}

	// Use default vault
	vaultPath, err := config.Get("obsidian_vault")
	if err != nil {
		return "", err
	}
	if vaultPath == "" {
		return "", fmt.Errorf("obsidian vault not configured (use: pocket config set obsidian_vault /path/to/vault)")
	}

	// Expand ~ to home directory
	if strings.HasPrefix(vaultPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		vaultPath = filepath.Join(home, vaultPath[1:])
	}

	// Verify vault exists
	if _, err := os.Stat(vaultPath); os.IsNotExist(err) {
		return "", fmt.Errorf("vault path does not exist: %s", vaultPath)
	}

	return vaultPath, nil
}

// getDailyFormat returns the daily note format
func getDailyFormat() string {
	format, err := config.Get("obsidian_daily_format")
	if err != nil || format == "" {
		return "2006-01-02" // Default Go date format (YYYY-MM-DD)
	}
	return format
}

// newVaultsCmd lists configured vaults
func newVaultsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vaults",
		Short: "List configured vaults",
		RunE: func(cmd *cobra.Command, args []string) error {
			vaults := []VaultInfo{}

			// Get default vault
			defaultPath, _ := config.Get("obsidian_vault")
			if defaultPath != "" {
				vaults = append(vaults, VaultInfo{
					Name: "default",
					Path: defaultPath,
				})
			}

			// Get additional vaults
			vaultsJSON, _ := config.Get("obsidian_vaults")
			if vaultsJSON != "" {
				var additionalVaults []VaultInfo
				if err := json.Unmarshal([]byte(vaultsJSON), &additionalVaults); err == nil {
					vaults = append(vaults, additionalVaults...)
				}
			}

			if len(vaults) == 0 {
				return output.Print(map[string]any{
					"vaults":  []VaultInfo{},
					"message": "No vaults configured. Use: pocket config set obsidian_vault /path/to/vault",
				})
			}

			return output.Print(map[string]any{
				"vaults": vaults,
				"count":  len(vaults),
			})
		},
	}

	return cmd
}

// newNotesCmd lists notes in vault
func newNotesCmd() *cobra.Command {
	var folder string
	var vault string
	var limit int

	cmd := &cobra.Command{
		Use:   "notes",
		Short: "List notes in vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath, err := getVaultPath(vault)
			if err != nil {
				return output.PrintError("vault_error", err.Error(), nil)
			}

			searchPath := vaultPath
			if folder != "" {
				searchPath = filepath.Join(vaultPath, folder)
				if _, err := os.Stat(searchPath); os.IsNotExist(err) {
					return output.PrintError("folder_not_found", "Folder does not exist: "+folder, nil)
				}
			}

			var notes []NoteInfo
			err = filepath.WalkDir(searchPath, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil // Skip errors
				}

				// Skip hidden files and directories
				if strings.HasPrefix(d.Name(), ".") {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}

				// Only include markdown files
				if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
					info, err := d.Info()
					if err != nil {
						return nil
					}

					relPath, _ := filepath.Rel(vaultPath, path)
					folderPath := filepath.Dir(relPath)
					if folderPath == "." {
						folderPath = ""
					}

					notes = append(notes, NoteInfo{
						Name:     strings.TrimSuffix(d.Name(), ".md"),
						Path:     relPath,
						Folder:   folderPath,
						Modified: info.ModTime(),
						Size:     info.Size(),
					})
				}
				return nil
			})

			if err != nil {
				return output.PrintError("walk_error", "Failed to list notes: "+err.Error(), nil)
			}

			// Sort by name
			sort.Slice(notes, func(i, j int) bool {
				return notes[i].Name < notes[j].Name
			})

			// Apply limit
			if limit > 0 && len(notes) > limit {
				notes = notes[:limit]
			}

			return output.Print(map[string]any{
				"notes":  notes,
				"count":  len(notes),
				"vault":  vaultPath,
				"folder": folder,
			})
		},
	}

	cmd.Flags().StringVarP(&folder, "folder", "f", "", "Filter by folder path")
	cmd.Flags().StringVar(&vault, "vault", "", "Vault name or path (default: configured vault)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 0, "Limit number of results")

	return cmd
}

// newReadCmd reads a note's content
func newReadCmd() *cobra.Command {
	var vault string

	cmd := &cobra.Command{
		Use:   "read [note]",
		Short: "Read a note's content",
		Long:  `Read a note by name or path. If no extension is provided, .md is assumed.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath, err := getVaultPath(vault)
			if err != nil {
				return output.PrintError("vault_error", err.Error(), nil)
			}

			notePath := args[0]
			// Add .md extension if not present
			if !strings.HasSuffix(strings.ToLower(notePath), ".md") {
				notePath += ".md"
			}

			fullPath := filepath.Join(vaultPath, notePath)

			// Check if file exists
			info, err := os.Stat(fullPath)
			if os.IsNotExist(err) {
				return output.PrintError("not_found", "Note not found: "+notePath, nil)
			}
			if err != nil {
				return output.PrintError("read_error", "Failed to access note: "+err.Error(), nil)
			}

			// Read content
			content, err := os.ReadFile(fullPath)
			if err != nil {
				return output.PrintError("read_error", "Failed to read note: "+err.Error(), nil)
			}

			relPath, _ := filepath.Rel(vaultPath, fullPath)
			folderPath := filepath.Dir(relPath)
			if folderPath == "." {
				folderPath = ""
			}

			return output.Print(map[string]any{
				"name":     strings.TrimSuffix(filepath.Base(notePath), ".md"),
				"path":     relPath,
				"folder":   folderPath,
				"content":  string(content),
				"modified": info.ModTime(),
				"size":     info.Size(),
			})
		},
	}

	cmd.Flags().StringVar(&vault, "vault", "", "Vault name or path (default: configured vault)")

	return cmd
}

// newWriteCmd creates or updates a note
func newWriteCmd() *cobra.Command {
	var vault string
	var appendMode bool

	cmd := &cobra.Command{
		Use:   "write [note] [content]",
		Short: "Create or update a note",
		Long:  `Create a new note or update an existing one. Use --append to add to existing content.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath, err := getVaultPath(vault)
			if err != nil {
				return output.PrintError("vault_error", err.Error(), nil)
			}

			notePath := args[0]
			content := args[1]

			// Add .md extension if not present
			if !strings.HasSuffix(strings.ToLower(notePath), ".md") {
				notePath += ".md"
			}

			fullPath := filepath.Join(vaultPath, notePath)

			// Ensure parent directory exists
			parentDir := filepath.Dir(fullPath)
			if err := os.MkdirAll(parentDir, 0o755); err != nil {
				return output.PrintError("write_error", "Failed to create directory: "+err.Error(), nil)
			}

			var finalContent string
			existed := false

			// Check if file exists
			if _, err := os.Stat(fullPath); err == nil {
				existed = true
				if appendMode {
					// Read existing content and append
					existing, err := os.ReadFile(fullPath)
					if err != nil {
						return output.PrintError("read_error", "Failed to read existing note: "+err.Error(), nil)
					}
					finalContent = string(existing) + "\n" + content
				} else {
					finalContent = content
				}
			} else {
				finalContent = content
			}

			// Write content
			if err := os.WriteFile(fullPath, []byte(finalContent), 0o600); err != nil {
				return output.PrintError("write_error", "Failed to write note: "+err.Error(), nil)
			}

			relPath, _ := filepath.Rel(vaultPath, fullPath)
			action := "created"
			if existed {
				if appendMode {
					action = "appended"
				} else {
					action = "updated"
				}
			}

			return output.Print(map[string]any{
				"status":  "success",
				"action":  action,
				"name":    strings.TrimSuffix(filepath.Base(notePath), ".md"),
				"path":    relPath,
				"size":    len(finalContent),
				"existed": existed,
			})
		},
	}

	cmd.Flags().StringVar(&vault, "vault", "", "Vault name or path (default: configured vault)")
	cmd.Flags().BoolVarP(&appendMode, "append", "a", false, "Append to existing note instead of overwriting")

	return cmd
}

// newSearchCmd searches notes by content
//
//nolint:gocyclo // complex but clear sequential logic
func newSearchCmd() *cobra.Command {
	var vault string
	var limit int
	var caseSensitive bool

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search notes by content",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath, err := getVaultPath(vault)
			if err != nil {
				return output.PrintError("vault_error", err.Error(), nil)
			}

			query := args[0]
			if !caseSensitive {
				query = strings.ToLower(query)
			}

			type SearchResult struct {
				Name     string    `json:"name"`
				Path     string    `json:"path"`
				Folder   string    `json:"folder"`
				Modified time.Time `json:"modified"`
				Matches  []string  `json:"matches"`
			}

			var results []SearchResult

			err = filepath.WalkDir(vaultPath, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}

				// Skip hidden files and directories
				if strings.HasPrefix(d.Name(), ".") {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}

				// Only search markdown files
				if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
					content, err := os.ReadFile(path)
					if err != nil {
						return nil
					}

					searchContent := string(content)
					if !caseSensitive {
						searchContent = strings.ToLower(searchContent)
					}

					if strings.Contains(searchContent, query) {
						info, _ := d.Info()
						relPath, _ := filepath.Rel(vaultPath, path)
						folderPath := filepath.Dir(relPath)
						if folderPath == "." {
							folderPath = ""
						}

						// Extract matching lines
						var matches []string
						lines := strings.Split(string(content), "\n")
						for _, line := range lines {
							searchLine := line
							if !caseSensitive {
								searchLine = strings.ToLower(line)
							}
							if strings.Contains(searchLine, query) {
								// Trim and limit line length
								trimmed := strings.TrimSpace(line)
								if len(trimmed) > 100 {
									trimmed = trimmed[:100] + "..."
								}
								matches = append(matches, trimmed)
								if len(matches) >= 3 {
									break
								}
							}
						}

						modTime := time.Time{}
						if info != nil {
							modTime = info.ModTime()
						}

						results = append(results, SearchResult{
							Name:     strings.TrimSuffix(d.Name(), ".md"),
							Path:     relPath,
							Folder:   folderPath,
							Modified: modTime,
							Matches:  matches,
						})
					}
				}
				return nil
			})

			if err != nil {
				return output.PrintError("search_error", "Search failed: "+err.Error(), nil)
			}

			// Sort by modification time (newest first)
			sort.Slice(results, func(i, j int) bool {
				return results[i].Modified.After(results[j].Modified)
			})

			// Apply limit
			if limit > 0 && len(results) > limit {
				results = results[:limit]
			}

			return output.Print(map[string]any{
				"results": results,
				"count":   len(results),
				"query":   args[0],
				"vault":   vaultPath,
			})
		},
	}

	cmd.Flags().StringVar(&vault, "vault", "", "Vault name or path (default: configured vault)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Limit number of results")
	cmd.Flags().BoolVarP(&caseSensitive, "case-sensitive", "c", false, "Case sensitive search")

	return cmd
}

// newDailyCmd gets or creates today's daily note
func newDailyCmd() *cobra.Command {
	var vault string
	var folder string
	var create bool
	var offset int

	cmd := &cobra.Command{
		Use:   "daily",
		Short: "Get or create today's daily note",
		Long:  `Get today's daily note. Use --create to create if it doesn't exist. Use --offset for other days (-1 for yesterday, 1 for tomorrow).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath, err := getVaultPath(vault)
			if err != nil {
				return output.PrintError("vault_error", err.Error(), nil)
			}

			// Calculate date with offset
			targetDate := time.Now().AddDate(0, 0, offset)
			dateFormat := getDailyFormat()
			noteName := targetDate.Format(dateFormat)

			// Build path
			notePath := noteName + ".md"
			if folder != "" {
				notePath = filepath.Join(folder, notePath)
			}

			fullPath := filepath.Join(vaultPath, notePath)

			// Check if file exists
			info, statErr := os.Stat(fullPath)
			exists := statErr == nil

			if !exists && !create {
				return output.Print(map[string]any{
					"exists":  false,
					"name":    noteName,
					"path":    notePath,
					"date":    targetDate.Format("2006-01-02"),
					"message": "Daily note does not exist. Use --create to create it.",
				})
			}

			if !exists && create {
				// Create the daily note
				parentDir := filepath.Dir(fullPath)
				if err := os.MkdirAll(parentDir, 0o755); err != nil {
					return output.PrintError("write_error", "Failed to create directory: "+err.Error(), nil)
				}

				// Create with default template
				template := fmt.Sprintf("# %s\n\n", targetDate.Format("Monday, January 2, 2006"))
				if err := os.WriteFile(fullPath, []byte(template), 0o600); err != nil {
					return output.PrintError("write_error", "Failed to create daily note: "+err.Error(), nil)
				}

				info, _ = os.Stat(fullPath)
				return output.Print(map[string]any{
					"exists":   true,
					"created":  true,
					"name":     noteName,
					"path":     notePath,
					"date":     targetDate.Format("2006-01-02"),
					"content":  template,
					"modified": info.ModTime(),
				})
			}

			// Read existing note
			content, err := os.ReadFile(fullPath)
			if err != nil {
				return output.PrintError("read_error", "Failed to read daily note: "+err.Error(), nil)
			}

			return output.Print(map[string]any{
				"exists":   true,
				"created":  false,
				"name":     noteName,
				"path":     notePath,
				"date":     targetDate.Format("2006-01-02"),
				"content":  string(content),
				"modified": info.ModTime(),
				"size":     info.Size(),
			})
		},
	}

	cmd.Flags().StringVar(&vault, "vault", "", "Vault name or path (default: configured vault)")
	cmd.Flags().StringVarP(&folder, "folder", "f", "", "Folder for daily notes (e.g., 'Daily Notes')")
	cmd.Flags().BoolVarP(&create, "create", "c", false, "Create if it doesn't exist")
	cmd.Flags().IntVar(&offset, "offset", 0, "Day offset (-1 for yesterday, 1 for tomorrow)")

	return cmd
}

// newRecentCmd lists recently modified notes
func newRecentCmd() *cobra.Command {
	var vault string
	var limit int
	var days int

	cmd := &cobra.Command{
		Use:   "recent",
		Short: "List recently modified notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath, err := getVaultPath(vault)
			if err != nil {
				return output.PrintError("vault_error", err.Error(), nil)
			}

			cutoff := time.Now().AddDate(0, 0, -days)
			var notes []NoteInfo

			err = filepath.WalkDir(vaultPath, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}

				// Skip hidden files and directories
				if strings.HasPrefix(d.Name(), ".") {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}

				// Only include markdown files
				if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
					info, err := d.Info()
					if err != nil {
						return nil
					}

					// Filter by date if days is set
					if days > 0 && info.ModTime().Before(cutoff) {
						return nil
					}

					relPath, _ := filepath.Rel(vaultPath, path)
					folderPath := filepath.Dir(relPath)
					if folderPath == "." {
						folderPath = ""
					}

					notes = append(notes, NoteInfo{
						Name:     strings.TrimSuffix(d.Name(), ".md"),
						Path:     relPath,
						Folder:   folderPath,
						Modified: info.ModTime(),
						Size:     info.Size(),
					})
				}
				return nil
			})

			if err != nil {
				return output.PrintError("walk_error", "Failed to list notes: "+err.Error(), nil)
			}

			// Sort by modification time (newest first)
			sort.Slice(notes, func(i, j int) bool {
				return notes[i].Modified.After(notes[j].Modified)
			})

			// Apply limit
			if limit > 0 && len(notes) > limit {
				notes = notes[:limit]
			}

			return output.Print(map[string]any{
				"notes": notes,
				"count": len(notes),
				"vault": vaultPath,
			})
		},
	}

	cmd.Flags().StringVar(&vault, "vault", "", "Vault name or path (default: configured vault)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Limit number of results")
	cmd.Flags().IntVarP(&days, "days", "d", 0, "Only show notes modified in last N days (0 = no limit)")

	return cmd
}
