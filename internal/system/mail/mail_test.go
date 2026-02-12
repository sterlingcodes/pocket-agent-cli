package mail

import (
	"strings"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "mail" {
		t.Errorf("expected Use 'mail', got %q", cmd.Use)
	}

	// Check aliases
	aliases := map[string]bool{"applemail": true, "amail": true}
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
	for _, name := range []string{"accounts", "mailboxes", "list", "read [id]", "search [query]", "send", "unread", "count"} {
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
			name:  "email address",
			input: "test@example.com",
			want:  "test@example.com",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
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
			name:  "email signature",
			input: "<div>Best regards,<br/><strong>John Doe</strong></div>",
			want:  "Best regards,\nJohn Doe",
		},
		{
			name:  "multiple newlines collapse",
			input: "Line 1\n\n\n\nLine 2",
			want:  "Line 1\n\nLine 2",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
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

func TestBoolTrueConst(t *testing.T) {
	if boolTrue != "true" {
		t.Errorf("expected boolTrue = %q, got %q", "true", boolTrue)
	}
}

func TestParseAccounts(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   int // number of accounts expected
	}{
		{
			name:   "empty output",
			output: "",
			want:   0,
		},
		{
			name:   "single account",
			output: "Gmail\ttrue\tIMAP\ttest@gmail.com\n",
			want:   1,
		},
		{
			name:   "multiple accounts",
			output: "Gmail\ttrue\tIMAP\ttest@gmail.com\nWork\tfalse\tExchange\twork@company.com\n",
			want:   2,
		},
		{
			name:   "account without email",
			output: "iCloud\ttrue\tIMAP\t\n",
			want:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accounts := parseAccounts(tt.output)
			if len(accounts) != tt.want {
				t.Errorf("parseAccounts() returned %d accounts, want %d", len(accounts), tt.want)
			}

			// Check first account if present
			if len(accounts) > 0 {
				if accounts[0].Name == "" {
					t.Error("first account has empty name")
				}
				if tt.output != "" && !strings.Contains(tt.output, accounts[0].Name) {
					t.Errorf("account name %q not found in output", accounts[0].Name)
				}
			}
		})
	}
}

func TestParseMailboxes(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   int // number of mailboxes expected
	}{
		{
			name:   "empty output",
			output: "",
			want:   0,
		},
		{
			name:   "single mailbox",
			output: "INBOX\tGmail\t5\t100\n",
			want:   1,
		},
		{
			name:   "multiple mailboxes",
			output: "INBOX\tGmail\t5\t100\nSent\tGmail\t0\t50\nDrafts\tGmail\t2\t10\n",
			want:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mailboxes := parseMailboxes(tt.output)
			if len(mailboxes) != tt.want {
				t.Errorf("parseMailboxes() returned %d mailboxes, want %d", len(mailboxes), tt.want)
			}

			// Check first mailbox if present
			if len(mailboxes) > 0 {
				if mailboxes[0].Name == "" {
					t.Error("first mailbox has empty name")
				}
				if mailboxes[0].UnreadCount < 0 {
					t.Error("unread count should not be negative")
				}
			}
		})
	}
}

func TestParseMessageList(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   int // number of messages expected
	}{
		{
			name:   "empty output",
			output: "",
			want:   0,
		},
		{
			name:   "single message",
			output: "12345\tTest Subject\tsender@example.com\tTuesday, January 1, 2025 at 9:00:00 AM\ttrue\tfalse\n",
			want:   1,
		},
		{
			name:   "multiple messages",
			output: "12345\tSubject 1\tsender1@example.com\tDate 1\ttrue\tfalse\n12346\tSubject 2\tsender2@example.com\tDate 2\tfalse\ttrue\n",
			want:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := parseMessageList(tt.output)
			if len(messages) != tt.want {
				t.Errorf("parseMessageList() returned %d messages, want %d", len(messages), tt.want)
			}

			// Check first message if present
			if len(messages) > 0 {
				if messages[0].ID == "" {
					t.Error("first message has empty ID")
				}
				if messages[0].Subject == "" {
					t.Error("first message has empty subject")
				}
			}
		})
	}
}

func TestParseFullMessage(t *testing.T) {
	tests := []struct {
		name   string
		output string
		isNil  bool
	}{
		{
			name:   "valid message",
			output: "12345|||Test Subject|||sender@example.com|||recipient@example.com|||cc@example.com|||Date Sent|||Date Received|||true|||false|||<div>Message body</div>",
			isNil:  false,
		},
		{
			name:   "insufficient parts",
			output: "12345|||Subject",
			isNil:  true,
		},
		{
			name:   "empty output",
			output: "",
			isNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := parseFullMessage(tt.output)
			if (msg == nil) != tt.isNil {
				t.Errorf("parseFullMessage() nil = %v, want nil = %v", msg == nil, tt.isNil)
			}

			if msg != nil {
				if msg.ID == "" {
					t.Error("message has empty ID")
				}
				if msg.Subject == "" {
					t.Error("message has empty subject")
				}
				// Body should be plain text (HTML stripped)
				if strings.Contains(msg.Body, "<div>") || strings.Contains(msg.Body, "</div>") {
					t.Error("body should have HTML tags stripped")
				}
			}
		})
	}
}

func TestParseUnreadMessages(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   int // number of messages expected
	}{
		{
			name:   "empty output",
			output: "",
			want:   0,
		},
		{
			name:   "single unread message",
			output: "12345\tTest Subject\tsender@example.com\tDate\tINBOX\n",
			want:   1,
		},
		{
			name:   "multiple unread messages",
			output: "12345\tSubject 1\tsender1@example.com\tDate 1\tINBOX\n12346\tSubject 2\tsender2@example.com\tDate 2\tWork\n",
			want:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := parseUnreadMessages(tt.output)
			if len(messages) != tt.want {
				t.Errorf("parseUnreadMessages() returned %d messages, want %d", len(messages), tt.want)
			}

			// All messages should be unread
			for _, msg := range messages {
				if !msg.Unread {
					t.Error("expected all messages to be marked as unread")
				}
			}
		})
	}
}

func TestNewAccountsCmd(t *testing.T) {
	cmd := newAccountsCmd()
	if cmd.Use != "accounts" {
		t.Errorf("expected Use 'accounts', got %q", cmd.Use)
	}
}

func TestNewMailboxesCmd(t *testing.T) {
	cmd := newMailboxesCmd()
	if cmd.Use != "mailboxes" {
		t.Errorf("expected Use 'mailboxes', got %q", cmd.Use)
	}

	// Check flags
	accountFlag := cmd.Flags().Lookup("account")
	if accountFlag == nil {
		t.Error("expected 'account' flag")
	}
}

func TestNewListCmd(t *testing.T) {
	cmd := newListCmd()
	if cmd.Use != "list" {
		t.Errorf("expected Use 'list', got %q", cmd.Use)
	}

	// Check flags
	flags := []string{"mailbox", "account", "limit", "unread"}
	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected %q flag", flagName)
		}
	}
}

func TestNewSendCmd(t *testing.T) {
	cmd := newSendCmd()
	if cmd.Use != "send" {
		t.Errorf("expected Use 'send', got %q", cmd.Use)
	}

	// Check flags
	flags := []string{"to", "subject", "body", "cc", "bcc", "account"}
	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected %q flag", flagName)
		}
	}
}
