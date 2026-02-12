package reminders

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/pkg/output"
)

const boolTrue = "true"

// Reminder represents a single reminder item
type Reminder struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	List        string  `json:"list"`
	DueDate     *string `json:"due_date,omitempty"`
	Notes       *string `json:"notes,omitempty"`
	Completed   bool    `json:"completed"`
	Priority    int     `json:"priority"`
	CreatedDate string  `json:"created_date,omitempty"`
}

// ReminderList represents a reminder list
type ReminderList struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reminders",
		Aliases: []string{"remind", "rem"},
		Short:   "Apple Reminders commands (macOS only)",
		Long:    `Interact with Apple Reminders via AppleScript. Only available on macOS.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS != "darwin" {
				return output.PrintError("platform_unsupported",
					"Apple Reminders is only available on macOS",
					map[string]string{
						"current_platform": runtime.GOOS,
						"required":         "darwin (macOS)",
					})
			}
			return nil
		},
	}

	cmd.AddCommand(newListsCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newCompleteCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newTodayCmd())
	cmd.AddCommand(newOverdueCmd())

	return cmd
}

// newListsCmd lists all reminder lists
func newListsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lists",
		Short: "List all reminder lists",
		RunE: func(cmd *cobra.Command, args []string) error {
			script := `
tell application "Reminders"
	set output to ""
	repeat with reminderList in lists
		set listName to name of reminderList
		set listID to id of reminderList
		set reminderCount to count of (reminders of reminderList whose completed is false)
		set output to output & listID & "	" & listName & "	" & reminderCount & linefeed
	end repeat
	return output
end tell
`
			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("applescript_error", err.Error(), nil)
			}

			lists := parseReminderLists(result)
			return output.Print(map[string]any{
				"lists": lists,
				"count": len(lists),
			})
		},
	}

	return cmd
}

// newListCmd lists reminders in a specific list or all lists
func newListCmd() *cobra.Command {
	var showCompleted bool

	cmd := &cobra.Command{
		Use:   "list [name]",
		Short: "List reminders in a specific list (or all if no name given)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			listName := ""
			if len(args) > 0 {
				listName = args[0]
			}

			var script string
			if listName != "" {
				completedFilter := "false"
				if showCompleted {
					completedFilter = boolTrue
				}
				script = fmt.Sprintf(`
tell application "Reminders"
	try
		set targetList to list "%s"
		set output to ""
		if %s then
			set reminderItems to reminders of targetList
		else
			set reminderItems to (reminders of targetList whose completed is false)
		end if
		repeat with r in reminderItems
			set rID to id of r
			set rName to name of r
			set rCompleted to completed of r
			set rPriority to priority of r
			set rNotes to ""
			try
				set rNotes to body of r
			end try
			set rDue to ""
			try
				set rDue to (due date of r) as string
			end try
			set output to output & rID & "	" & rName & "	" & rCompleted & "	" & rPriority & "	" & rDue & "	" & rNotes & linefeed
		end repeat
		return output
	on error errMsg
		return "ERROR: " & errMsg
	end try
end tell
`, escapeAppleScriptString(listName), completedFilter)
			} else {
				completedFilter := "false"
				if showCompleted {
					completedFilter = boolTrue
				}
				script = fmt.Sprintf(`
tell application "Reminders"
	set output to ""
	repeat with reminderList in lists
		set listName to name of reminderList
		if %s then
			set reminderItems to reminders of reminderList
		else
			set reminderItems to (reminders of reminderList whose completed is false)
		end if
		repeat with r in reminderItems
			set rID to id of r
			set rName to name of r
			set rCompleted to completed of r
			set rPriority to priority of r
			set rNotes to ""
			try
				set rNotes to body of r
			end try
			set rDue to ""
			try
				set rDue to (due date of r) as string
			end try
			set output to output & rID & "	" & rName & "	" & listName & "	" & rCompleted & "	" & rPriority & "	" & rDue & "	" & rNotes & linefeed
		end repeat
	end repeat
	return output
end tell
`, completedFilter)
			}

			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("applescript_error", err.Error(), nil)
			}

			if strings.HasPrefix(result, "ERROR:") {
				return output.PrintError("reminders_error", strings.TrimPrefix(result, "ERROR: "), nil)
			}

			reminders := parseReminders(result, listName)
			return output.Print(map[string]any{
				"reminders": reminders,
				"count":     len(reminders),
				"list":      listName,
			})
		},
	}

	cmd.Flags().BoolVar(&showCompleted, "completed", false, "Include completed reminders")

	return cmd
}

// newAddCmd adds a new reminder
func newAddCmd() *cobra.Command {
	var listName string
	var dueDate string
	var notes string
	var priority int

	cmd := &cobra.Command{
		Use:   "add [title]",
		Short: "Add a new reminder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := args[0]

			// Default to "Reminders" list if not specified
			if listName == "" {
				listName = "Reminders"
			}

			// Build the AppleScript
			var scriptBuilder strings.Builder
			scriptBuilder.WriteString(`
tell application "Reminders"
	try
		set targetList to list "`)
			scriptBuilder.WriteString(escapeAppleScriptString(listName))
			scriptBuilder.WriteString(`"
		set newReminder to make new reminder at end of targetList with properties {name:"`)
			scriptBuilder.WriteString(escapeAppleScriptString(title))
			scriptBuilder.WriteString(`"`)

			// Add priority if specified (1=high, 5=medium, 9=low, 0=none)
			if priority > 0 && priority <= 9 {
				scriptBuilder.WriteString(fmt.Sprintf(`, priority:%d`, priority))
			}

			// Add notes if specified
			if notes != "" {
				scriptBuilder.WriteString(fmt.Sprintf(`, body:"%s"`, escapeAppleScriptString(notes))) //nolint:gocritic // AppleScript syntax requires this format
			}

			scriptBuilder.WriteString(`}
`)

			// Add due date if specified
			if dueDate != "" {
				parsedDate, err := parseFlexibleDate(dueDate)
				if err != nil {
					return output.PrintError("invalid_date", err.Error(), map[string]string{
						"input":   dueDate,
						"formats": "YYYY-MM-DD, YYYY-MM-DD HH:MM, today, tomorrow, next week",
					})
				}
				scriptBuilder.WriteString(fmt.Sprintf(`		set due date of newReminder to date "%s"
`, parsedDate.Format("January 2, 2006 3:04:05 PM")))
			}

			scriptBuilder.WriteString(`		set rID to id of newReminder
		return rID
	on error errMsg
		return "ERROR: " & errMsg
	end try
end tell
`)

			result, err := runAppleScript(scriptBuilder.String())
			if err != nil {
				return output.PrintError("applescript_error", err.Error(), nil)
			}

			if strings.HasPrefix(result, "ERROR:") {
				return output.PrintError("reminders_error", strings.TrimPrefix(result, "ERROR: "), nil)
			}

			response := map[string]any{
				"success": true,
				"message": "Reminder created",
				"id":      strings.TrimSpace(result),
				"title":   title,
				"list":    listName,
			}
			if dueDate != "" {
				response["due"] = dueDate
			}
			if notes != "" {
				response["notes"] = notes
			}
			if priority > 0 {
				response["priority"] = priority
			}

			return output.Print(response)
		},
	}

	cmd.Flags().StringVarP(&listName, "list", "l", "", "List name (default: Reminders)")
	cmd.Flags().StringVarP(&dueDate, "due", "d", "", "Due date (YYYY-MM-DD, YYYY-MM-DD HH:MM, today, tomorrow)")
	cmd.Flags().StringVarP(&notes, "notes", "n", "", "Notes/description")
	cmd.Flags().IntVarP(&priority, "priority", "p", 0, "Priority (1=high, 5=medium, 9=low)")

	return cmd
}

// newCompleteCmd marks a reminder as complete
func newCompleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "complete [id or name]",
		Short: "Mark a reminder as complete",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier := args[0]

			// Try to find by ID first, then by name
			script := fmt.Sprintf(`
tell application "Reminders"
	try
		-- Try to find by ID first
		set foundReminder to missing value
		repeat with reminderList in lists
			repeat with r in (reminders of reminderList whose completed is false)
				if id of r is "%s" then
					set foundReminder to r
					exit repeat
				end if
			end repeat
			if foundReminder is not missing value then exit repeat
		end repeat

		-- If not found by ID, try by name
		if foundReminder is missing value then
			repeat with reminderList in lists
				repeat with r in (reminders of reminderList whose completed is false)
					if name of r is "%s" then
						set foundReminder to r
						exit repeat
					end if
				end repeat
				if foundReminder is not missing value then exit repeat
			end repeat
		end if

		if foundReminder is missing value then
			return "ERROR: Reminder not found"
		end if

		set completed of foundReminder to true
		return "SUCCESS: " & (name of foundReminder)
	on error errMsg
		return "ERROR: " & errMsg
	end try
end tell
`, escapeAppleScriptString(identifier), escapeAppleScriptString(identifier))

			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("applescript_error", err.Error(), nil)
			}

			if strings.HasPrefix(result, "ERROR:") {
				return output.PrintError("reminders_error", strings.TrimPrefix(result, "ERROR: "), nil)
			}

			completedName := strings.TrimPrefix(strings.TrimSpace(result), "SUCCESS: ")
			return output.Print(map[string]any{
				"success":   true,
				"message":   "Reminder marked as complete",
				"name":      completedName,
				"completed": true,
			})
		},
	}

	return cmd
}

// newDeleteCmd deletes a reminder
func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [id or name]",
		Short: "Delete a reminder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier := args[0]

			script := fmt.Sprintf(`
tell application "Reminders"
	try
		-- Try to find by ID first
		set foundReminder to missing value
		set foundList to missing value
		repeat with reminderList in lists
			repeat with r in reminders of reminderList
				if id of r is "%s" then
					set foundReminder to r
					set foundList to reminderList
					exit repeat
				end if
			end repeat
			if foundReminder is not missing value then exit repeat
		end repeat

		-- If not found by ID, try by name
		if foundReminder is missing value then
			repeat with reminderList in lists
				repeat with r in reminders of reminderList
					if name of r is "%s" then
						set foundReminder to r
						set foundList to reminderList
						exit repeat
					end if
				end repeat
				if foundReminder is not missing value then exit repeat
			end repeat
		end if

		if foundReminder is missing value then
			return "ERROR: Reminder not found"
		end if

		set reminderName to name of foundReminder
		delete foundReminder
		return "SUCCESS: " & reminderName
	on error errMsg
		return "ERROR: " & errMsg
	end try
end tell
`, escapeAppleScriptString(identifier), escapeAppleScriptString(identifier))

			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("applescript_error", err.Error(), nil)
			}

			if strings.HasPrefix(result, "ERROR:") {
				return output.PrintError("reminders_error", strings.TrimPrefix(result, "ERROR: "), nil)
			}

			deletedName := strings.TrimPrefix(strings.TrimSpace(result), "SUCCESS: ")
			return output.Print(map[string]any{
				"success": true,
				"message": "Reminder deleted",
				"name":    deletedName,
			})
		},
	}

	return cmd
}

// newTodayCmd shows reminders due today
func newTodayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "today",
		Short: "Show reminders due today",
		RunE: func(cmd *cobra.Command, args []string) error {
			script := `
tell application "Reminders"
	set output to ""
	set todayStart to current date
	set hours of todayStart to 0
	set minutes of todayStart to 0
	set seconds of todayStart to 0
	set todayEnd to todayStart + (24 * 60 * 60)

	repeat with reminderList in lists
		set listName to name of reminderList
		repeat with r in (reminders of reminderList whose completed is false)
			try
				set rDue to due date of r
				if rDue >= todayStart and rDue < todayEnd then
					set rID to id of r
					set rName to name of r
					set rPriority to priority of r
					set rNotes to ""
					try
						set rNotes to body of r
					end try
					set output to output & rID & "	" & rName & "	" & listName & "	false	" & rPriority & "	" & (rDue as string) & "	" & rNotes & linefeed
				end if
			end try
		end repeat
	end repeat
	return output
end tell
`
			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("applescript_error", err.Error(), nil)
			}

			reminders := parseReminders(result, "")
			return output.Print(map[string]any{
				"reminders": reminders,
				"count":     len(reminders),
				"filter":    "today",
				"date":      time.Now().Format("2006-01-02"),
			})
		},
	}

	return cmd
}

// newOverdueCmd shows overdue reminders
func newOverdueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "overdue",
		Short: "Show overdue reminders",
		RunE: func(cmd *cobra.Command, args []string) error {
			script := `
tell application "Reminders"
	set output to ""
	set todayStart to current date
	set hours of todayStart to 0
	set minutes of todayStart to 0
	set seconds of todayStart to 0

	repeat with reminderList in lists
		set listName to name of reminderList
		repeat with r in (reminders of reminderList whose completed is false)
			try
				set rDue to due date of r
				if rDue < todayStart then
					set rID to id of r
					set rName to name of r
					set rPriority to priority of r
					set rNotes to ""
					try
						set rNotes to body of r
					end try
					set output to output & rID & "	" & rName & "	" & listName & "	false	" & rPriority & "	" & (rDue as string) & "	" & rNotes & linefeed
				end if
			end try
		end repeat
	end repeat
	return output
end tell
`
			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("applescript_error", err.Error(), nil)
			}

			reminders := parseReminders(result, "")
			return output.Print(map[string]any{
				"reminders": reminders,
				"count":     len(reminders),
				"filter":    "overdue",
			})
		},
	}

	return cmd
}

// runAppleScript executes an AppleScript and returns the result
func runAppleScript(script string) (string, error) {
	cmd := exec.Command("osascript", "-e", script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%s: %s", err.Error(), stderr.String())
		}
		return "", err
	}

	return strings.TrimSpace(stdout.String()), nil
}

// escapeAppleScriptString escapes special characters for AppleScript strings
func escapeAppleScriptString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// parseReminderLists parses the tab-separated list output
func parseReminderLists(output string) []ReminderList {
	var lists []ReminderList
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) >= 3 {
			count := 0
			_, _ = fmt.Sscanf(parts[2], "%d", &count)
			lists = append(lists, ReminderList{
				ID:    parts[0],
				Name:  parts[1],
				Count: count,
			})
		}
	}

	return lists
}

// parseReminders parses the tab-separated reminder output
func parseReminders(output, defaultList string) []Reminder {
	lines := strings.Split(output, "\n")
	reminders := make([]Reminder, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")

		var reminder Reminder

		// Handle different formats based on whether list name is included
		switch {
		case defaultList != "" && len(parts) >= 6:
			// Format: ID, Name, Completed, Priority, DueDate, Notes
			reminder.ID = parts[0]
			reminder.Name = parts[1]
			reminder.List = defaultList
			reminder.Completed = parts[2] == boolTrue
			_, _ = fmt.Sscanf(parts[3], "%d", &reminder.Priority)
			if parts[4] != "" {
				dueStr := parts[4]
				reminder.DueDate = &dueStr
			}
			if len(parts) > 5 && parts[5] != "" {
				notes := parts[5]
				reminder.Notes = &notes
			}
		case len(parts) >= 7:
			// Format: ID, Name, ListName, Completed, Priority, DueDate, Notes
			reminder.ID = parts[0]
			reminder.Name = parts[1]
			reminder.List = parts[2]
			reminder.Completed = parts[3] == boolTrue
			_, _ = fmt.Sscanf(parts[4], "%d", &reminder.Priority)
			if parts[5] != "" {
				dueStr := parts[5]
				reminder.DueDate = &dueStr
			}
			if len(parts) > 6 && parts[6] != "" {
				notes := parts[6]
				reminder.Notes = &notes
			}
		default:
			continue
		}

		reminders = append(reminders, reminder)
	}

	return reminders
}

// parseFlexibleDate parses various date formats
func parseFlexibleDate(input string) (time.Time, error) {
	input = strings.ToLower(strings.TrimSpace(input))
	now := time.Now()

	switch input {
	case "today":
		return time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location()), nil
	case "tomorrow":
		tomorrow := now.AddDate(0, 0, 1)
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 9, 0, 0, 0, now.Location()), nil
	case "next week":
		nextWeek := now.AddDate(0, 0, 7)
		return time.Date(nextWeek.Year(), nextWeek.Month(), nextWeek.Day(), 9, 0, 0, 0, now.Location()), nil
	}

	// Try various date formats
	formats := []string{
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"01/02/2006 15:04",
		"01/02/2006",
		"Jan 2, 2006 15:04",
		"Jan 2, 2006",
		"January 2, 2006 15:04",
		"January 2, 2006",
	}

	for _, format := range formats {
		if t, err := time.ParseInLocation(format, input, now.Location()); err == nil {
			// If no time was specified, default to 9:00 AM
			if !strings.Contains(format, "15:04") {
				t = time.Date(t.Year(), t.Month(), t.Day(), 9, 0, 0, 0, now.Location())
			}
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unrecognized date format: %s", input)
}
