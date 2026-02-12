package commands

import (
	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/internal/system/battery"
	"github.com/unstablemind/pocket/internal/system/calendar"
	"github.com/unstablemind/pocket/internal/system/cleanup"
	"github.com/unstablemind/pocket/internal/system/clipboard"
	"github.com/unstablemind/pocket/internal/system/contacts"
	"github.com/unstablemind/pocket/internal/system/diskhealth"
	"github.com/unstablemind/pocket/internal/system/finder"
	"github.com/unstablemind/pocket/internal/system/imessage"
	"github.com/unstablemind/pocket/internal/system/mail"
	"github.com/unstablemind/pocket/internal/system/notes"
	"github.com/unstablemind/pocket/internal/system/reminders"
	"github.com/unstablemind/pocket/internal/system/safari"
	"github.com/unstablemind/pocket/internal/system/sysinfo"
)

func NewSystemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "system",
		Aliases: []string{"sys"},
		Short:   "System commands",
		Long:    `System-level integrations: Apple Notes, Calendar, Reminders, Contacts, Finder, Safari, Mail, Clipboard, iMessage (macOS only).`,
	}

	cmd.AddCommand(calendar.NewCmd())
	cmd.AddCommand(clipboard.NewCmd())
	cmd.AddCommand(contacts.NewCmd())
	cmd.AddCommand(finder.NewCmd())
	cmd.AddCommand(imessage.NewCmd())
	cmd.AddCommand(mail.NewCmd())
	cmd.AddCommand(notes.NewCmd())
	cmd.AddCommand(reminders.NewCmd())
	cmd.AddCommand(safari.NewCmd())
	cmd.AddCommand(sysinfo.NewCmd())
	cmd.AddCommand(battery.NewCmd())
	cmd.AddCommand(diskhealth.NewCmd())
	cmd.AddCommand(cleanup.NewCmd())

	return cmd
}
