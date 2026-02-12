package logseq

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

// Graph represents a Logseq graph
type Graph struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Format string `json:"format"`
}

// Page represents a page in the graph
type Page struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
}

// SearchResult represents a search result
type SearchResult struct {
	Page    string `json:"page"`
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

// NewCmd creates the logseq command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "logseq",
		Aliases: []string{"ls"},
		Short:   "Logseq graph commands",
	}

	cmd.AddCommand(newGraphsCmd())
	cmd.AddCommand(newPagesCmd())
	cmd.AddCommand(newReadCmd())
	cmd.AddCommand(newWriteCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newJournalCmd())
	cmd.AddCommand(newRecentCmd())

	return cmd
}

// getGraphPath returns the path to the default graph or specified graph
func getGraphPath(graphName string) (graphPath, format string, err error) {
	// If a specific graph is requested, look it up in graphs list
	if graphName != "" {
		graphsJSON, _ := config.Get("logseq_graphs")
		if graphsJSON != "" {
			var graphs []Graph
			if err := json.Unmarshal([]byte(graphsJSON), &graphs); err == nil {
				for _, g := range graphs {
					if g.Name == graphName || g.Path == graphName {
						return g.Path, g.Format, nil
					}
				}
			}
		}
		// If not in list, assume it's a path
		if _, err := os.Stat(graphName); err == nil {
			format, _ = config.Get("logseq_format")
			if format == "" {
				format = "md"
			}
			return graphName, format, nil
		}
		return "", "", fmt.Errorf("graph not found: %s", graphName)
	}

	// Use default graph
	graphPath, err = config.Get("logseq_graph")
	if err != nil {
		return "", "", err
	}
	if graphPath == "" {
		return "", "", fmt.Errorf("logseq_graph not configured (use: pocket config set logseq_graph /path/to/graph)")
	}

	format, _ = config.Get("logseq_format")
	if format == "" {
		format = "md"
	}

	return graphPath, format, nil
}

// getFileExtension returns the file extension for the format
func getFileExtension(format string) string {
	if format == "org" {
		return ".org"
	}
	return ".md"
}

// newGraphsCmd lists configured graphs
func newGraphsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "graphs",
		Short: "List configured graphs",
		RunE: func(cmd *cobra.Command, args []string) error {
			var graphs []Graph

			// Get default graph
			defaultGraph, _ := config.Get("logseq_graph")
			format, _ := config.Get("logseq_format")
			if format == "" {
				format = "md"
			}

			if defaultGraph != "" {
				graphs = append(graphs, Graph{
					Name:   filepath.Base(defaultGraph),
					Path:   defaultGraph,
					Format: format,
				})
			}

			// Get additional graphs
			graphsJSON, _ := config.Get("logseq_graphs")
			if graphsJSON != "" {
				var additionalGraphs []Graph
				if err := json.Unmarshal([]byte(graphsJSON), &additionalGraphs); err == nil {
					graphs = append(graphs, additionalGraphs...)
				}
			}

			if len(graphs) == 0 {
				return output.Print(map[string]any{
					"graphs":  []Graph{},
					"message": "No graphs configured. Set with: pocket config set logseq_graph /path/to/graph",
				})
			}

			return output.Print(map[string]any{
				"graphs":        graphs,
				"default_graph": defaultGraph,
			})
		},
	}

	return cmd
}

// newPagesCmd lists pages in the graph
func newPagesCmd() *cobra.Command {
	var graphName string
	var limit int

	cmd := &cobra.Command{
		Use:   "pages",
		Short: "List pages in graph",
		RunE: func(cmd *cobra.Command, args []string) error {
			graphPath, format, err := getGraphPath(graphName)
			if err != nil {
				return output.PrintError("config_error", err.Error(), nil)
			}

			pagesDir := filepath.Join(graphPath, "pages")
			ext := getFileExtension(format)

			pages, err := listPages(pagesDir, ext, limit)
			if err != nil {
				return output.PrintError("list_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"graph": graphPath,
				"pages": pages,
				"count": len(pages),
			})
		},
	}

	cmd.Flags().StringVarP(&graphName, "graph", "g", "", "Graph name or path")
	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Maximum number of pages to return")

	return cmd
}

// listPages returns a list of pages in the directory
func listPages(pagesDir, ext string, limit int) ([]Page, error) {
	entries, err := os.ReadDir(pagesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read pages directory: %w", err)
	}

	pages := make([]Page, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ext) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		pageName := strings.TrimSuffix(entry.Name(), ext)
		// Decode URL-encoded names (Logseq uses %2F for /)
		pageName = decodePageName(pageName)

		pages = append(pages, Page{
			Name:       pageName,
			Path:       filepath.Join(pagesDir, entry.Name()),
			ModifiedAt: info.ModTime(),
			Size:       info.Size(),
		})
	}

	// Sort by modification time (newest first)
	sort.Slice(pages, func(i, j int) bool {
		return pages[i].ModifiedAt.After(pages[j].ModifiedAt)
	})

	if limit > 0 && len(pages) > limit {
		pages = pages[:limit]
	}

	return pages, nil
}

// decodePageName decodes URL-encoded page names
func decodePageName(name string) string {
	// Common Logseq encodings
	name = strings.ReplaceAll(name, "%2F", "/")
	name = strings.ReplaceAll(name, "%3A", ":")
	name = strings.ReplaceAll(name, "%3F", "?")
	name = strings.ReplaceAll(name, "%23", "#")
	name = strings.ReplaceAll(name, "%26", "&")
	name = strings.ReplaceAll(name, "%25", "%")
	return name
}

// encodePageName encodes page names for file system
func encodePageName(name string) string {
	// Common Logseq encodings
	name = strings.ReplaceAll(name, "%", "%25")
	name = strings.ReplaceAll(name, "/", "%2F")
	name = strings.ReplaceAll(name, ":", "%3A")
	name = strings.ReplaceAll(name, "?", "%3F")
	name = strings.ReplaceAll(name, "#", "%23")
	name = strings.ReplaceAll(name, "&", "%26")
	return name
}

// newReadCmd reads a page's content
func newReadCmd() *cobra.Command {
	var graphName string

	cmd := &cobra.Command{
		Use:   "read [page]",
		Short: "Read a page's content",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageName := args[0]

			graphPath, format, err := getGraphPath(graphName)
			if err != nil {
				return output.PrintError("config_error", err.Error(), nil)
			}

			// Try to find the page
			pagePath, err := findPage(graphPath, pageName, format)
			if err != nil {
				return output.PrintError("not_found", err.Error(), nil)
			}

			content, err := os.ReadFile(pagePath)
			if err != nil {
				return output.PrintError("read_error", err.Error(), nil)
			}

			info, _ := os.Stat(pagePath)
			var modTime time.Time
			if info != nil {
				modTime = info.ModTime()
			}

			return output.Print(map[string]any{
				"page":        pageName,
				"path":        pagePath,
				"content":     string(content),
				"modified_at": modTime,
			})
		},
	}

	cmd.Flags().StringVarP(&graphName, "graph", "g", "", "Graph name or path")

	return cmd
}

// findPage finds a page file in the graph
func findPage(graphPath, pageName, format string) (string, error) {
	ext := getFileExtension(format)
	pagesDir := filepath.Join(graphPath, "pages")
	journalsDir := filepath.Join(graphPath, "journals")

	// Try exact match in pages
	encodedName := encodePageName(pageName)
	pagePath := filepath.Join(pagesDir, encodedName+ext)
	if _, err := os.Stat(pagePath); err == nil {
		return pagePath, nil
	}

	// Try without encoding
	pagePath = filepath.Join(pagesDir, pageName+ext)
	if _, err := os.Stat(pagePath); err == nil {
		return pagePath, nil
	}

	// Try in journals
	pagePath = filepath.Join(journalsDir, pageName+ext)
	if _, err := os.Stat(pagePath); err == nil {
		return pagePath, nil
	}

	// Try case-insensitive search in pages
	entries, err := os.ReadDir(pagesDir)
	if err == nil {
		lowerName := strings.ToLower(pageName)
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			fileName := strings.TrimSuffix(entry.Name(), ext)
			if strings.ToLower(decodePageName(fileName)) == lowerName {
				return filepath.Join(pagesDir, entry.Name()), nil
			}
		}
	}

	return "", fmt.Errorf("page not found: %s", pageName)
}

// newWriteCmd creates or updates a page
func newWriteCmd() *cobra.Command {
	var graphName string
	var appendMode bool

	cmd := &cobra.Command{
		Use:   "write [page] [content]",
		Short: "Create or update a page",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageName := args[0]
			content := args[1]

			graphPath, format, err := getGraphPath(graphName)
			if err != nil {
				return output.PrintError("config_error", err.Error(), nil)
			}

			pagesDir := filepath.Join(graphPath, "pages")
			ext := getFileExtension(format)

			// Ensure pages directory exists
			if err := os.MkdirAll(pagesDir, 0o755); err != nil {
				return output.PrintError("write_error", "failed to create pages directory", err.Error())
			}

			encodedName := encodePageName(pageName)
			pagePath := filepath.Join(pagesDir, encodedName+ext)

			var finalContent string
			if appendMode {
				// Read existing content
				existing, err := os.ReadFile(pagePath)
				if err != nil && !os.IsNotExist(err) {
					return output.PrintError("read_error", err.Error(), nil)
				}
				finalContent = string(existing) + "\n" + content
			} else {
				finalContent = content
			}

			if err := os.WriteFile(pagePath, []byte(finalContent), 0o600); err != nil {
				return output.PrintError("write_error", err.Error(), nil)
			}

			action := "created"
			if appendMode {
				action = "appended"
			}

			return output.Print(map[string]any{
				"success": true,
				"action":  action,
				"page":    pageName,
				"path":    pagePath,
			})
		},
	}

	cmd.Flags().StringVarP(&graphName, "graph", "g", "", "Graph name or path")
	cmd.Flags().BoolVarP(&appendMode, "append", "a", false, "Append to existing content")

	return cmd
}

// newSearchCmd searches pages by content
func newSearchCmd() *cobra.Command {
	var graphName string
	var limit int
	var caseSensitive bool

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search pages by content",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			graphPath, format, err := getGraphPath(graphName)
			if err != nil {
				return output.PrintError("config_error", err.Error(), nil)
			}

			results, err := searchPages(graphPath, query, format, limit, caseSensitive)
			if err != nil {
				return output.PrintError("search_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"query":   query,
				"graph":   graphPath,
				"results": results,
				"count":   len(results),
			})
		},
	}

	cmd.Flags().StringVarP(&graphName, "graph", "g", "", "Graph name or path")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum number of results")
	cmd.Flags().BoolVarP(&caseSensitive, "case-sensitive", "c", false, "Case-sensitive search")

	return cmd
}

// searchPages searches for content in pages
func searchPages(graphPath, query, format string, limit int, caseSensitive bool) ([]SearchResult, error) {
	var results []SearchResult
	ext := getFileExtension(format)

	searchQuery := query
	if !caseSensitive {
		searchQuery = strings.ToLower(query)
	}

	// Search in pages
	pagesDir := filepath.Join(graphPath, "pages")
	if err := searchInDir(pagesDir, ext, searchQuery, caseSensitive, &results, limit); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// Search in journals
	journalsDir := filepath.Join(graphPath, "journals")
	if len(results) < limit {
		if err := searchInDir(journalsDir, ext, searchQuery, caseSensitive, &results, limit); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}

	return results, nil
}

// searchInDir searches for content in a directory
func searchInDir(dir, ext, query string, caseSensitive bool, results *[]SearchResult, limit int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if len(*results) >= limit {
			break
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ext) {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		pageName := strings.TrimSuffix(entry.Name(), ext)
		pageName = decodePageName(pageName)

		for lineNum, line := range lines {
			if len(*results) >= limit {
				break
			}

			searchLine := line
			if !caseSensitive {
				searchLine = strings.ToLower(line)
			}

			if strings.Contains(searchLine, query) {
				*results = append(*results, SearchResult{
					Page:    pageName,
					Path:    filePath,
					Line:    lineNum + 1,
					Content: strings.TrimSpace(line),
				})
			}
		}
	}

	return nil
}

// newJournalCmd gets or creates today's journal entry
func newJournalCmd() *cobra.Command {
	var graphName string
	var dateStr string
	var content string

	cmd := &cobra.Command{
		Use:   "journal",
		Short: "Get or create journal entry",
		RunE: func(cmd *cobra.Command, args []string) error {
			graphPath, format, err := getGraphPath(graphName)
			if err != nil {
				return output.PrintError("config_error", err.Error(), nil)
			}

			// Parse date or use today
			var date time.Time
			if dateStr != "" {
				date, err = parseDate(dateStr)
				if err != nil {
					return output.PrintError("date_error", err.Error(), nil)
				}
			} else {
				date = time.Now()
			}

			journalsDir := filepath.Join(graphPath, "journals")
			ext := getFileExtension(format)

			// Logseq uses YYYY_MM_DD format for journal files
			fileName := date.Format("2006_01_02") + ext
			journalPath := filepath.Join(journalsDir, fileName)

			// Read existing content
			existingContent, err := os.ReadFile(journalPath)
			exists := err == nil

			// If content is provided, write to journal
			if content != "" {
				// Ensure journals directory exists
				if err := os.MkdirAll(journalsDir, 0o755); err != nil {
					return output.PrintError("write_error", "failed to create journals directory", err.Error())
				}

				var finalContent string
				if exists {
					finalContent = string(existingContent) + "\n" + content
				} else {
					finalContent = content
				}

				if err := os.WriteFile(journalPath, []byte(finalContent), 0o600); err != nil {
					return output.PrintError("write_error", err.Error(), nil)
				}

				return output.Print(map[string]any{
					"success": true,
					"action":  "updated",
					"date":    date.Format("2006-01-02"),
					"path":    journalPath,
				})
			}

			// Return journal content
			if !exists {
				return output.Print(map[string]any{
					"date":    date.Format("2006-01-02"),
					"path":    journalPath,
					"exists":  false,
					"content": "",
					"message": "Journal entry does not exist",
				})
			}

			info, _ := os.Stat(journalPath)
			var modTime time.Time
			if info != nil {
				modTime = info.ModTime()
			}

			return output.Print(map[string]any{
				"date":        date.Format("2006-01-02"),
				"path":        journalPath,
				"exists":      true,
				"content":     string(existingContent),
				"modified_at": modTime,
			})
		},
	}

	cmd.Flags().StringVarP(&graphName, "graph", "g", "", "Graph name or path")
	cmd.Flags().StringVarP(&dateStr, "date", "d", "", "Date (YYYY-MM-DD), defaults to today")
	cmd.Flags().StringVarP(&content, "content", "c", "", "Content to append to journal")

	return cmd
}

// parseDate parses various date formats
func parseDate(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"01-02-2006",
		"01/02/2006",
		"Jan 2, 2006",
		"January 2, 2006",
		"2 Jan 2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", s)
}

// newRecentCmd lists recently modified pages
func newRecentCmd() *cobra.Command {
	var graphName string
	var limit int
	var days int

	cmd := &cobra.Command{
		Use:   "recent",
		Short: "List recently modified pages",
		RunE: func(cmd *cobra.Command, args []string) error {
			graphPath, format, err := getGraphPath(graphName)
			if err != nil {
				return output.PrintError("config_error", err.Error(), nil)
			}

			ext := getFileExtension(format)
			cutoff := time.Now().AddDate(0, 0, -days)

			var allPages []Page

			// Get pages from pages directory
			pagesDir := filepath.Join(graphPath, "pages")
			pages, err := listPagesWithCutoff(pagesDir, ext, cutoff)
			if err == nil {
				allPages = append(allPages, pages...)
			}

			// Get pages from journals directory
			journalsDir := filepath.Join(graphPath, "journals")
			journals, err := listPagesWithCutoff(journalsDir, ext, cutoff)
			if err == nil {
				allPages = append(allPages, journals...)
			}

			// Sort by modification time (newest first)
			sort.Slice(allPages, func(i, j int) bool {
				return allPages[i].ModifiedAt.After(allPages[j].ModifiedAt)
			})

			if limit > 0 && len(allPages) > limit {
				allPages = allPages[:limit]
			}

			return output.Print(map[string]any{
				"graph": graphPath,
				"pages": allPages,
				"count": len(allPages),
				"days":  days,
			})
		},
	}

	cmd.Flags().StringVarP(&graphName, "graph", "g", "", "Graph name or path")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum number of pages to return")
	cmd.Flags().IntVarP(&days, "days", "d", 7, "Number of days to look back")

	return cmd
}

// listPagesWithCutoff returns pages modified after the cutoff time
func listPagesWithCutoff(dir, ext string, cutoff time.Time) ([]Page, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	pages := make([]Page, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ext) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			continue
		}

		pageName := strings.TrimSuffix(entry.Name(), ext)
		pageName = decodePageName(pageName)

		pages = append(pages, Page{
			Name:       pageName,
			Path:       filepath.Join(dir, entry.Name()),
			ModifiedAt: info.ModTime(),
			Size:       info.Size(),
		})
	}

	return pages, nil
}
