package mail

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/pkg/output"
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

const boolTrue = "true"

// htmlToPlaintext converts HTML email body to readable plaintext
// preserving line breaks, list items, and paragraph structure.
func htmlToPlaintext(html string) string {
	s := html
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	for _, tag := range []string{"div", "p", "br", "tr", "h1", "h2", "h3", "h4", "h5", "h6"} {
		s = regexp.MustCompile(`(?i)</`+tag+`\s*>`).ReplaceAllString(s, "\n")
		s = regexp.MustCompile(`(?i)<`+tag+`[^>]*/>`).ReplaceAllString(s, "\n")
		s = regexp.MustCompile(`(?i)<`+tag+`[^>]*>`).ReplaceAllString(s, "")
	}
	s = regexp.MustCompile(`(?i)<br\s*/?\s*>`).ReplaceAllString(s, "\n")
	s = regexp.MustCompile(`(?i)<li[^>]*>`).ReplaceAllString(s, "• ")
	s = regexp.MustCompile(`(?i)</li\s*>`).ReplaceAllString(s, "\n")
	s = htmlTagRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&amp", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&apos;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = regexp.MustCompile(`\n{3,}`).ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

// Account represents an Apple Mail account
type Account struct {
	Name    string `json:"name"`
	ID      string `json:"id,omitempty"`
	Email   string `json:"email,omitempty"`
	Enabled bool   `json:"enabled"`
	Type    string `json:"type,omitempty"`
}

// Mailbox represents a mailbox/folder in Apple Mail
type Mailbox struct {
	Name         string `json:"name"`
	Account      string `json:"account,omitempty"`
	UnreadCount  int    `json:"unread_count"`
	MessageCount int    `json:"message_count"`
}

// Message represents an email message
type Message struct {
	ID           string `json:"id"`
	Subject      string `json:"subject"`
	From         string `json:"from"`
	To           string `json:"to,omitempty"`
	CC           string `json:"cc,omitempty"`
	Date         string `json:"date"`
	DateReceived string `json:"date_received,omitempty"`
	Preview      string `json:"preview,omitempty"`
	Body         string `json:"body,omitempty"`
	Unread       bool   `json:"unread"`
	Flagged      bool   `json:"flagged"`
	Mailbox      string `json:"mailbox,omitempty"`
	Account      string `json:"account,omitempty"`
}

// NewCmd creates the Apple Mail command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "mail",
		Aliases: []string{"applemail", "amail"},
		Short:   "Apple Mail commands (macOS only)",
		Long:    `Interact with Apple Mail via AppleScript. Only available on macOS.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS != "darwin" {
				return output.PrintError("platform_unsupported",
					"Apple Mail is only available on macOS",
					map[string]string{
						"current_platform": runtime.GOOS,
						"required":         "darwin (macOS)",
						"suggestion":       "Use 'pocket comms email' for cross-platform IMAP/SMTP email",
					})
			}
			return nil
		},
	}

	cmd.AddCommand(newAccountsCmd())
	cmd.AddCommand(newMailboxesCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newReadCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newSendCmd())
	cmd.AddCommand(newUnreadCmd())
	cmd.AddCommand(newCountCmd())

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
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		// Check for Mail app not running
		if strings.Contains(errMsg, "Application isn't running") ||
			strings.Contains(errMsg, "not running") {
			return "", fmt.Errorf("apple Mail is not running, please open Mail.app first")
		}
		return "", fmt.Errorf("%s", strings.TrimSpace(errMsg))
	}

	return strings.TrimSpace(stdout.String()), nil
}

// escapeAppleScript escapes special characters for AppleScript strings
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// newAccountsCmd lists all mail accounts
func newAccountsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "List all mail accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			script := `
tell application "Mail"
	set output to ""
	repeat with acct in accounts
		set acctName to name of acct
		set acctEnabled to enabled of acct
		set acctType to account type of acct as string
		set emailAddr to ""
		try
			set emailAddr to email addresses of acct as string
		end try
		set output to output & acctName & "	" & acctEnabled & "	" & acctType & "	" & emailAddr & linefeed
	end repeat
	return output
end tell
`
			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("applescript_error", err.Error(), map[string]string{
					"hint": "Make sure Mail.app is running",
				})
			}

			accounts := parseAccounts(result)
			return output.Print(map[string]any{
				"accounts": accounts,
				"count":    len(accounts),
			})
		},
	}

	return cmd
}

// newMailboxesCmd lists all mailboxes/folders
func newMailboxesCmd() *cobra.Command {
	var accountName string

	cmd := &cobra.Command{
		Use:   "mailboxes",
		Short: "List mailboxes/folders",
		RunE: func(cmd *cobra.Command, args []string) error {
			var script string
			if accountName != "" {
				script = fmt.Sprintf(`
tell application "Mail"
	set output to ""
	try
		set targetAccount to account "%s"
		repeat with mbox in mailboxes of targetAccount
			set mboxName to name of mbox
			set unreadCount to unread count of mbox
			set msgCount to count of messages of mbox
			set output to output & mboxName & "	" & "%s" & "	" & unreadCount & "	" & msgCount & linefeed
		end repeat
	on error errMsg
		return "ERROR: " & errMsg
	end try
	return output
end tell
`, escapeAppleScript(accountName), escapeAppleScript(accountName))
			} else {
				script = `
tell application "Mail"
	set output to ""
	repeat with acct in accounts
		set acctName to name of acct
		repeat with mbox in mailboxes of acct
			set mboxName to name of mbox
			set unreadCount to unread count of mbox
			set msgCount to count of messages of mbox
			set output to output & mboxName & "	" & acctName & "	" & unreadCount & "	" & msgCount & linefeed
		end repeat
	end repeat
	-- Also include standard mailboxes
	repeat with mbox in {inbox, sent mailbox, drafts mailbox, trash mailbox, junk mailbox}
		try
			set mboxName to name of mbox
			set unreadCount to unread count of mbox
			set msgCount to count of messages of mbox
			set output to output & mboxName & "	(All Accounts)	" & unreadCount & "	" & msgCount & linefeed
		end try
	end repeat
	return output
end tell
`
			}

			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("applescript_error", err.Error(), nil)
			}

			if strings.HasPrefix(result, "ERROR:") {
				return output.PrintError("mail_error", strings.TrimPrefix(result, "ERROR: "), nil)
			}

			mailboxes := parseMailboxes(result)
			return output.Print(map[string]any{
				"mailboxes": mailboxes,
				"count":     len(mailboxes),
				"account":   accountName,
			})
		},
	}

	cmd.Flags().StringVarP(&accountName, "account", "a", "", "Filter by account name")

	return cmd
}

// newListCmd lists recent messages
func newListCmd() *cobra.Command {
	var mailbox string
	var accountName string
	var limit int
	var unreadOnly bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default to inbox
			if mailbox == "" {
				mailbox = "INBOX"
			}

			var script string
			unreadFilter := ""
			if unreadOnly {
				unreadFilter = "whose read status is false"
			}

			if accountName != "" {
				script = fmt.Sprintf(`
tell application "Mail"
	set output to ""
	try
		set targetAccount to account "%s"
		set targetMailbox to mailbox "%s" of targetAccount
		set msgs to messages of targetMailbox %s
		set msgCount to 0
		repeat with msg in msgs
			if msgCount >= %d then exit repeat
			set msgID to id of msg as string
			set msgSubject to subject of msg
			set msgFrom to sender of msg
			set msgDate to date received of msg as string
			set msgUnread to (read status of msg is false)
			set msgFlagged to flagged status of msg
			set output to output & msgID & "	" & msgSubject & "	" & msgFrom & "	" & msgDate & "	" & msgUnread & "	" & msgFlagged & linefeed
			set msgCount to msgCount + 1
		end repeat
	on error errMsg
		return "ERROR: " & errMsg
	end try
	return output
end tell
`, escapeAppleScript(accountName), escapeAppleScript(mailbox), unreadFilter, limit)
			} else {
				// Use the unified inbox
				script = fmt.Sprintf(`
tell application "Mail"
	set output to ""
	try
		set msgs to messages of inbox %s
		set msgCount to 0
		repeat with msg in msgs
			if msgCount >= %d then exit repeat
			set msgID to id of msg as string
			set msgSubject to subject of msg
			set msgFrom to sender of msg
			set msgDate to date received of msg as string
			set msgUnread to (read status of msg is false)
			set msgFlagged to flagged status of msg
			set output to output & msgID & "	" & msgSubject & "	" & msgFrom & "	" & msgDate & "	" & msgUnread & "	" & msgFlagged & linefeed
			set msgCount to msgCount + 1
		end repeat
	on error errMsg
		return "ERROR: " & errMsg
	end try
	return output
end tell
`, unreadFilter, limit)
			}

			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("applescript_error", err.Error(), nil)
			}

			if strings.HasPrefix(result, "ERROR:") {
				return output.PrintError("mail_error", strings.TrimPrefix(result, "ERROR: "), nil)
			}

			messages := parseMessageList(result)
			return output.Print(map[string]any{
				"messages": messages,
				"count":    len(messages),
				"mailbox":  mailbox,
				"account":  accountName,
			})
		},
	}

	cmd.Flags().StringVarP(&mailbox, "mailbox", "m", "", "Mailbox name (default: INBOX)")
	cmd.Flags().StringVarP(&accountName, "account", "a", "", "Account name")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum number of messages")
	cmd.Flags().BoolVarP(&unreadOnly, "unread", "u", false, "Show only unread messages")

	return cmd
}

// newReadCmd reads a message by ID
func newReadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read [id]",
		Short: "Read a message by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			msgID := args[0]

			script := fmt.Sprintf(`
tell application "Mail"
	set output to ""
	try
		-- Search for message by ID across all accounts
		set foundMsg to missing value
		repeat with acct in accounts
			repeat with mbox in mailboxes of acct
				try
					set msgs to (messages of mbox whose id is %s)
					if (count of msgs) > 0 then
						set foundMsg to item 1 of msgs
						exit repeat
					end if
				end try
			end repeat
			if foundMsg is not missing value then exit repeat
		end repeat

		-- Also check unified mailboxes
		if foundMsg is missing value then
			repeat with mbox in {inbox, sent mailbox, drafts mailbox, trash mailbox}
				try
					set msgs to (messages of mbox whose id is %s)
					if (count of msgs) > 0 then
						set foundMsg to item 1 of msgs
						exit repeat
					end if
				end try
			end repeat
		end if

		if foundMsg is missing value then
			return "ERROR: Message not found"
		end if

		set msgID to id of foundMsg as string
		set msgSubject to subject of foundMsg
		set msgFrom to sender of foundMsg
		set msgTo to ""
		try
			set msgTo to address of to recipient 1 of foundMsg
		end try
		set msgCC to ""
		try
			set ccList to cc recipients of foundMsg
			if (count of ccList) > 0 then
				set msgCC to address of item 1 of ccList
			end if
		end try
		set msgDate to date sent of foundMsg as string
		set msgDateReceived to date received of foundMsg as string
		set msgUnread to (read status of foundMsg is false)
		set msgFlagged to flagged status of foundMsg
		set msgContent to content of foundMsg

		return msgID & "|||" & msgSubject & "|||" & msgFrom & "|||" & msgTo & "|||" & msgCC & "|||" & msgDate & "|||" & msgDateReceived & "|||" & msgUnread & "|||" & msgFlagged & "|||" & msgContent
	on error errMsg
		return "ERROR: " & errMsg
	end try
end tell
`, msgID, msgID)

			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("applescript_error", err.Error(), nil)
			}

			if strings.HasPrefix(result, "ERROR:") {
				errMsg := strings.TrimPrefix(result, "ERROR: ")
				if strings.Contains(errMsg, "not found") {
					return output.PrintError("not_found", "Message not found", map[string]string{
						"id": msgID,
					})
				}
				return output.PrintError("mail_error", errMsg, nil)
			}

			message := parseFullMessage(result)
			if message == nil {
				return output.PrintError("parse_error", "Failed to parse message", nil)
			}

			return output.Print(message)
		},
	}

	return cmd
}

// newSearchCmd searches messages
func newSearchCmd() *cobra.Command {
	var limit int
	var mailbox string
	var accountName string

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search messages",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			var script string
			if accountName != "" && mailbox != "" {
				script = fmt.Sprintf(`
tell application "Mail"
	set output to ""
	try
		set targetAccount to account "%s"
		set targetMailbox to mailbox "%s" of targetAccount
		set searchQuery to "%s"
		set msgs to messages of targetMailbox whose subject contains searchQuery or sender contains searchQuery or content contains searchQuery
		set msgCount to 0
		repeat with msg in msgs
			if msgCount >= %d then exit repeat
			set msgID to id of msg as string
			set msgSubject to subject of msg
			set msgFrom to sender of msg
			set msgDate to date received of msg as string
			set msgUnread to (read status of msg is false)
			set msgFlagged to flagged status of msg
			set output to output & msgID & "	" & msgSubject & "	" & msgFrom & "	" & msgDate & "	" & msgUnread & "	" & msgFlagged & linefeed
			set msgCount to msgCount + 1
		end repeat
	on error errMsg
		return "ERROR: " & errMsg
	end try
	return output
end tell
`, escapeAppleScript(accountName), escapeAppleScript(mailbox), escapeAppleScript(query), limit)
			} else {
				// Search in inbox
				script = fmt.Sprintf(`
tell application "Mail"
	set output to ""
	try
		set searchQuery to "%s"
		set msgs to messages of inbox whose subject contains searchQuery or sender contains searchQuery or content contains searchQuery
		set msgCount to 0
		repeat with msg in msgs
			if msgCount >= %d then exit repeat
			set msgID to id of msg as string
			set msgSubject to subject of msg
			set msgFrom to sender of msg
			set msgDate to date received of msg as string
			set msgUnread to (read status of msg is false)
			set msgFlagged to flagged status of msg
			set output to output & msgID & "	" & msgSubject & "	" & msgFrom & "	" & msgDate & "	" & msgUnread & "	" & msgFlagged & linefeed
			set msgCount to msgCount + 1
		end repeat
	on error errMsg
		return "ERROR: " & errMsg
	end try
	return output
end tell
`, escapeAppleScript(query), limit)
			}

			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("applescript_error", err.Error(), nil)
			}

			if strings.HasPrefix(result, "ERROR:") {
				return output.PrintError("mail_error", strings.TrimPrefix(result, "ERROR: "), nil)
			}

			messages := parseMessageList(result)
			return output.Print(map[string]any{
				"messages": messages,
				"count":    len(messages),
				"query":    query,
				"mailbox":  mailbox,
				"account":  accountName,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum results")
	cmd.Flags().StringVarP(&mailbox, "mailbox", "m", "", "Mailbox to search")
	cmd.Flags().StringVarP(&accountName, "account", "a", "", "Account name")

	return cmd
}

// newSendCmd sends an email
func newSendCmd() *cobra.Command {
	var to string
	var subject string
	var body string
	var cc string
	var bcc string
	var accountName string

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send an email",
		RunE: func(cmd *cobra.Command, args []string) error {
			if to == "" {
				return output.PrintError("missing_recipient", "Recipient (--to) is required", nil)
			}
			if subject == "" {
				return output.PrintError("missing_subject", "Subject (--subject) is required", nil)
			}
			if body == "" {
				return output.PrintError("missing_body", "Body (--body) is required", nil)
			}

			// Build recipient parts
			toRecipients := strings.Split(to, ",")
			var toScript strings.Builder
			for _, recipient := range toRecipients {
				recipient = strings.TrimSpace(recipient)
				if recipient != "" {
					toScript.WriteString(fmt.Sprintf(`make new to recipient at end of to recipients with properties {address:"%s"}
				`, escapeAppleScript(recipient)))
				}
			}

			var ccScript strings.Builder
			if cc != "" {
				ccRecipients := strings.Split(cc, ",")
				for _, recipient := range ccRecipients {
					recipient = strings.TrimSpace(recipient)
					if recipient != "" {
						ccScript.WriteString(fmt.Sprintf(`make new cc recipient at end of cc recipients with properties {address:"%s"}
				`, escapeAppleScript(recipient)))
					}
				}
			}

			var bccScript strings.Builder
			if bcc != "" {
				bccRecipients := strings.Split(bcc, ",")
				for _, recipient := range bccRecipients {
					recipient = strings.TrimSpace(recipient)
					if recipient != "" {
						bccScript.WriteString(fmt.Sprintf(`make new bcc recipient at end of bcc recipients with properties {address:"%s"}
				`, escapeAppleScript(recipient)))
					}
				}
			}

			// Account selection
			accountPart := ""
			if accountName != "" {
				accountPart = fmt.Sprintf(`, sender:"%s"`, escapeAppleScript(accountName)) //nolint:gocritic // AppleScript syntax requires this format
			}

			script := fmt.Sprintf(`
tell application "Mail"
	try
		set newMessage to make new outgoing message with properties {subject:"%s", content:"%s", visible:false%s}
		tell newMessage
			%s
			%s
			%s
		end tell
		send newMessage
		return "SUCCESS"
	on error errMsg
		return "ERROR: " & errMsg
	end try
end tell
`, escapeAppleScript(subject), escapeAppleScript(body), accountPart, toScript.String(), ccScript.String(), bccScript.String())

			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("applescript_error", err.Error(), nil)
			}

			if strings.HasPrefix(result, "ERROR:") {
				return output.PrintError("send_failed", strings.TrimPrefix(result, "ERROR: "), nil)
			}

			return output.Print(map[string]any{
				"success": true,
				"message": "Email sent successfully",
				"to":      to,
				"cc":      cc,
				"bcc":     bcc,
				"subject": subject,
			})
		},
	}

	cmd.Flags().StringVar(&to, "to", "", "Recipient(s), comma-separated (required)")
	cmd.Flags().StringVar(&subject, "subject", "", "Subject (required)")
	cmd.Flags().StringVar(&body, "body", "", "Email body (required)")
	cmd.Flags().StringVar(&cc, "cc", "", "CC recipient(s), comma-separated")
	cmd.Flags().StringVar(&bcc, "bcc", "", "BCC recipient(s), comma-separated")
	cmd.Flags().StringVarP(&accountName, "account", "a", "", "Send from specific account")

	return cmd
}

// newUnreadCmd lists unread messages
func newUnreadCmd() *cobra.Command {
	var limit int
	var accountName string

	cmd := &cobra.Command{
		Use:   "unread",
		Short: "List unread messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			var script string
			if accountName != "" {
				script = fmt.Sprintf(`
tell application "Mail"
	set output to ""
	try
		set targetAccount to account "%s"
		set msgCount to 0
		repeat with mbox in mailboxes of targetAccount
			set msgs to (messages of mbox whose read status is false)
			repeat with msg in msgs
				if msgCount >= %d then exit repeat
				set msgID to id of msg as string
				set msgSubject to subject of msg
				set msgFrom to sender of msg
				set msgDate to date received of msg as string
				set mboxName to name of mbox
				set output to output & msgID & "	" & msgSubject & "	" & msgFrom & "	" & msgDate & "	" & mboxName & linefeed
				set msgCount to msgCount + 1
			end repeat
			if msgCount >= %d then exit repeat
		end repeat
	on error errMsg
		return "ERROR: " & errMsg
	end try
	return output
end tell
`, escapeAppleScript(accountName), limit, limit)
			} else {
				script = fmt.Sprintf(`
tell application "Mail"
	set output to ""
	try
		set msgs to (messages of inbox whose read status is false)
		set msgCount to 0
		repeat with msg in msgs
			if msgCount >= %d then exit repeat
			set msgID to id of msg as string
			set msgSubject to subject of msg
			set msgFrom to sender of msg
			set msgDate to date received of msg as string
			set output to output & msgID & "	" & msgSubject & "	" & msgFrom & "	" & msgDate & "	INBOX" & linefeed
			set msgCount to msgCount + 1
		end repeat
	on error errMsg
		return "ERROR: " & errMsg
	end try
	return output
end tell
`, limit)
			}

			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("applescript_error", err.Error(), nil)
			}

			if strings.HasPrefix(result, "ERROR:") {
				return output.PrintError("mail_error", strings.TrimPrefix(result, "ERROR: "), nil)
			}

			messages := parseUnreadMessages(result)
			return output.Print(map[string]any{
				"messages": messages,
				"count":    len(messages),
				"account":  accountName,
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum number of messages")
	cmd.Flags().StringVarP(&accountName, "account", "a", "", "Filter by account name")

	return cmd
}

// newCountCmd gets unread message count
func newCountCmd() *cobra.Command {
	var accountName string

	cmd := &cobra.Command{
		Use:   "count",
		Short: "Get unread message count",
		RunE: func(cmd *cobra.Command, args []string) error {
			var script string
			if accountName != "" {
				script = fmt.Sprintf(`
tell application "Mail"
	try
		set targetAccount to account "%s"
		set totalUnread to 0
		repeat with mbox in mailboxes of targetAccount
			set totalUnread to totalUnread + (unread count of mbox)
		end repeat
		return totalUnread as string
	on error errMsg
		return "ERROR: " & errMsg
	end try
end tell
`, escapeAppleScript(accountName))
			} else {
				script = `
tell application "Mail"
	try
		return (unread count of inbox) as string
	on error errMsg
		return "ERROR: " & errMsg
	end try
end tell
`
			}

			result, err := runAppleScript(script)
			if err != nil {
				return output.PrintError("applescript_error", err.Error(), nil)
			}

			if strings.HasPrefix(result, "ERROR:") {
				return output.PrintError("mail_error", strings.TrimPrefix(result, "ERROR: "), nil)
			}

			count, _ := strconv.Atoi(result)
			return output.Print(map[string]any{
				"unread_count": count,
				"account":      accountName,
			})
		},
	}

	cmd.Flags().StringVarP(&accountName, "account", "a", "", "Filter by account name")

	return cmd
}

// parseAccounts parses the tab-separated account output
func parseAccounts(output string) []Account {
	var accounts []Account
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) >= 3 {
			account := Account{
				Name:    parts[0],
				Enabled: parts[1] == boolTrue,
				Type:    parts[2],
			}
			if len(parts) >= 4 {
				account.Email = parts[3]
			}
			accounts = append(accounts, account)
		}
	}

	return accounts
}

// parseMailboxes parses the tab-separated mailbox output
func parseMailboxes(output string) []Mailbox {
	var mailboxes []Mailbox
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) >= 4 {
			unread, _ := strconv.Atoi(parts[2])
			msgCount, _ := strconv.Atoi(parts[3])
			mailboxes = append(mailboxes, Mailbox{
				Name:         parts[0],
				Account:      parts[1],
				UnreadCount:  unread,
				MessageCount: msgCount,
			})
		}
	}

	return mailboxes
}

// parseMessageList parses the tab-separated message list output
func parseMessageList(output string) []Message {
	var messages []Message
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) >= 6 {
			messages = append(messages, Message{
				ID:      parts[0],
				Subject: parts[1],
				From:    parts[2],
				Date:    parts[3],
				Unread:  parts[4] == boolTrue,
				Flagged: parts[5] == boolTrue,
			})
		}
	}

	return messages
}

// parseFullMessage parses a full message with body
func parseFullMessage(output string) *Message {
	// Use SplitN with limit 10 so "|||" inside the body doesn't break parsing
	parts := strings.SplitN(output, "|||", 10)
	if len(parts) < 10 {
		return nil
	}

	return &Message{
		ID:           parts[0],
		Subject:      parts[1],
		From:         parts[2],
		To:           parts[3],
		CC:           parts[4],
		Date:         parts[5],
		DateReceived: parts[6],
		Unread:       parts[7] == boolTrue,
		Flagged:      parts[8] == boolTrue,
		Body:         htmlToPlaintext(parts[9]),
	}
}

// parseUnreadMessages parses the tab-separated unread message output
func parseUnreadMessages(output string) []Message {
	var messages []Message
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) >= 5 {
			messages = append(messages, Message{
				ID:      parts[0],
				Subject: parts[1],
				From:    parts[2],
				Date:    parts[3],
				Mailbox: parts[4],
				Unread:  true,
			})
		}
	}

	return messages
}
