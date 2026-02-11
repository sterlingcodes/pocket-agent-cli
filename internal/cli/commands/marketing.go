package commands

import (
	"github.com/spf13/cobra"
	amazonsp "github.com/unstablemind/pocket/internal/marketing/amazon-sp"
	facebookads "github.com/unstablemind/pocket/internal/marketing/facebook-ads"
	"github.com/unstablemind/pocket/internal/marketing/shopify"
)

func NewMarketingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "marketing",
		Aliases: []string{"mkt"},
		Short:   "Marketing commands",
		Long:    "Marketing tools: Facebook Ads, Amazon SP-API, and Shopify.",
	}

	cmd.AddCommand(facebookads.NewCmd())
	cmd.AddCommand(amazonsp.NewCmd())
	cmd.AddCommand(shopify.NewCmd())

	return cmd
}
