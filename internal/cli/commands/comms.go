package commands

import (
	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/internal/communication/discord"
	"github.com/unstablemind/pocket/internal/communication/email"
	"github.com/unstablemind/pocket/internal/communication/notify"
	"github.com/unstablemind/pocket/internal/communication/slack"
	"github.com/unstablemind/pocket/internal/communication/telegram"
	"github.com/unstablemind/pocket/internal/communication/twilio"
	"github.com/unstablemind/pocket/internal/communication/webhook"
)

func NewCommsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "comms",
		Aliases: []string{"c", "comm"},
		Short:   "Communication commands",
		Long:    `Interact with communication platforms: Email, Slack, Discord, Telegram, etc.`,
	}

	cmd.AddCommand(email.NewCmd())
	cmd.AddCommand(slack.NewCmd())
	cmd.AddCommand(discord.NewCmd())
	cmd.AddCommand(telegram.NewCmd())
	cmd.AddCommand(twilio.NewCmd())
	cmd.AddCommand(webhook.NewCmd())
	cmd.AddCommand(notify.NewCmd())

	return cmd
}
