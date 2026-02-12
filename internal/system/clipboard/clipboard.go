package clipboard

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/pkg/output"
)

// Content represents the content of the clipboard
type Content struct {
	Content   string `json:"content"`
	Length    int    `json:"length"`
	Lines     int    `json:"lines"`
	IsText    bool   `json:"is_text"`
	Truncated bool   `json:"truncated,omitempty"`
}

// NewCmd creates the clipboard command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clipboard",
		Aliases: []string{"clip", "cb", "pasteboard"},
		Short:   "Clipboard commands (macOS only)",
		Long:    `Interact with the system clipboard using pbcopy/pbpaste. Only available on macOS.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS != "darwin" {
				return output.PrintError("platform_unsupported",
					"Clipboard commands are only available on macOS",
					map[string]string{
						"current_platform": runtime.GOOS,
						"required":         "darwin (macOS)",
						"suggestion":       "On Linux, consider using xclip or xsel. On Windows, use clip.exe.",
					})
			}
			return nil
		},
	}

	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newSetCmd())
	cmd.AddCommand(newClearCmd())
	cmd.AddCommand(newCopyCmd())
	cmd.AddCommand(newHistoryCmd())

	return cmd
}

// newGetCmd gets the current clipboard content
func newGetCmd() *cobra.Command {
	var maxLength int
	var raw bool

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get current clipboard content",
		Long:  `Retrieve the current text content from the system clipboard using pbpaste.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := getClipboard()
			if err != nil {
				return output.PrintError("clipboard_read_error", err.Error(), nil)
			}

			// Check if content is valid UTF-8 text
			if !utf8.ValidString(content) {
				return output.Print(map[string]any{
					"is_text": false,
					"message": "Clipboard contains non-text data (binary content)",
					"hint":    "The clipboard may contain an image or other binary data",
				})
			}

			if content == "" {
				return output.Print(map[string]any{
					"content": "",
					"length":  0,
					"lines":   0,
					"is_text": true,
					"message": "Clipboard is empty",
				})
			}

			// If raw mode, just return the content as-is
			if raw {
				fmt.Print(content)
				return nil
			}

			// Calculate line count
			lines := strings.Count(content, "\n") + 1
			if strings.HasSuffix(content, "\n") {
				lines--
			}

			result := Content{
				Content: content,
				Length:  len(content),
				Lines:   lines,
				IsText:  true,
			}

			// Truncate if needed
			if maxLength > 0 && len(content) > maxLength {
				result.Content = content[:maxLength]
				result.Truncated = true
			}

			return output.Print(result)
		},
	}

	cmd.Flags().IntVarP(&maxLength, "max-length", "m", 0, "Maximum content length to return (0 = no limit)")
	cmd.Flags().BoolVarP(&raw, "raw", "r", false, "Output raw content without JSON wrapping")

	return cmd
}

// newSetCmd sets the clipboard content
func newSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set [text]",
		Short: "Set clipboard content",
		Long:  `Set the system clipboard content using pbcopy. Pass the text as an argument.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text := args[0]

			err := setClipboard(text)
			if err != nil {
				return output.PrintError("clipboard_write_error", err.Error(), nil)
			}

			// Calculate line count
			lines := strings.Count(text, "\n") + 1
			if strings.HasSuffix(text, "\n") {
				lines--
			}

			return output.Print(map[string]any{
				"success": true,
				"message": "Clipboard content set successfully",
				"length":  len(text),
				"lines":   lines,
			})
		},
	}

	return cmd
}

// newClearCmd clears the clipboard
func newClearCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear clipboard content",
		Long:  `Clear the system clipboard by setting it to an empty string.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := setClipboard("")
			if err != nil {
				return output.PrintError("clipboard_clear_error", err.Error(), nil)
			}

			return output.Print(map[string]any{
				"success": true,
				"message": "Clipboard cleared",
			})
		},
	}

	return cmd
}

// newCopyCmd copies file contents to clipboard
func newCopyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "copy [file]",
		Short: "Copy file contents to clipboard",
		Long:  `Read a file and copy its contents to the system clipboard.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]

			// Read the file
			content, err := os.ReadFile(filePath)
			if err != nil {
				if os.IsNotExist(err) {
					return output.PrintError("file_not_found",
						fmt.Sprintf("File not found: %s", filePath),
						map[string]string{"path": filePath})
				}
				return output.PrintError("file_read_error", err.Error(),
					map[string]string{"path": filePath})
			}

			// Check if content is valid UTF-8 text
			if !utf8.Valid(content) {
				return output.PrintError("binary_content",
					"File contains binary (non-text) data",
					map[string]any{
						"path": filePath,
						"size": len(content),
						"hint": "Only text files can be copied to the clipboard",
					})
			}

			text := string(content)
			err = setClipboard(text)
			if err != nil {
				return output.PrintError("clipboard_write_error", err.Error(), nil)
			}

			// Calculate line count
			lines := strings.Count(text, "\n") + 1
			if strings.HasSuffix(text, "\n") {
				lines--
			}

			return output.Print(map[string]any{
				"success": true,
				"message": "File contents copied to clipboard",
				"file":    filePath,
				"length":  len(text),
				"lines":   lines,
			})
		},
	}

	return cmd
}

// newHistoryCmd explains clipboard history limitations
func newHistoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Clipboard history information",
		Long:  `macOS does not provide native clipboard history. This command provides information about alternatives.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return output.Print(map[string]any{
				"available": false,
				"message":   "macOS does not have native clipboard history",
				"alternatives": []map[string]string{
					{
						"name":        "Paste",
						"description": "Commercial clipboard manager for macOS",
						"url":         "https://pasteapp.io",
					},
					{
						"name":        "Maccy",
						"description": "Free, lightweight clipboard manager",
						"url":         "https://maccy.app",
					},
					{
						"name":        "CopyClip",
						"description": "Free clipboard manager from the Mac App Store",
						"url":         "https://apps.apple.com/app/copyclip-clipboard-history/id595191960",
					},
					{
						"name":        "Alfred Powerpack",
						"description": "Alfred's clipboard history feature (requires Powerpack)",
						"url":         "https://www.alfredapp.com",
					},
				},
				"hint": "Install a third-party clipboard manager to access clipboard history",
			})
		},
	}

	return cmd
}

// getClipboard retrieves the current clipboard content using pbpaste
func getClipboard() (string, error) {
	cmd := exec.Command("pbpaste")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("pbpaste failed: %s", strings.TrimSpace(errMsg))
	}

	return stdout.String(), nil
}

// setClipboard sets the clipboard content using pbcopy
func setClipboard(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return fmt.Errorf("pbcopy failed: %s", strings.TrimSpace(errMsg))
	}

	return nil
}
