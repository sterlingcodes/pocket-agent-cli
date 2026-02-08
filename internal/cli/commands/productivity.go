package commands

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/productivity/calendar"
	"github.com/unstablemind/pocket/internal/productivity/gdrive"
	"github.com/unstablemind/pocket/internal/productivity/gsheets"
	"github.com/unstablemind/pocket/internal/productivity/logseq"
	"github.com/unstablemind/pocket/internal/productivity/notion"
	"github.com/unstablemind/pocket/internal/productivity/obsidian"
	"github.com/unstablemind/pocket/internal/productivity/todoist"
	"github.com/unstablemind/pocket/internal/productivity/trello"
)

func NewProductivityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "productivity",
		Aliases: []string{"p", "prod"},
		Short:   "Productivity tool commands",
		Long:    `Interact with productivity tools: Calendar, Notion, Todoist, Trello, etc.`,
	}

	cmd.AddCommand(calendar.NewCmd())
	cmd.AddCommand(logseq.NewCmd())
	cmd.AddCommand(notion.NewCmd())
	cmd.AddCommand(obsidian.NewCmd())
	cmd.AddCommand(todoist.NewCmd())
	cmd.AddCommand(trello.NewCmd())
	cmd.AddCommand(gsheets.NewCmd())
	cmd.AddCommand(gdrive.NewCmd())

	return cmd
}
