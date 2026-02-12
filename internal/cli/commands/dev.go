package commands

import (
	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/internal/dev/cloudflare"
	"github.com/unstablemind/pocket/internal/dev/database"
	"github.com/unstablemind/pocket/internal/dev/dockerhub"
	"github.com/unstablemind/pocket/internal/dev/gist"
	"github.com/unstablemind/pocket/internal/dev/github"
	"github.com/unstablemind/pocket/internal/dev/gitlab"
	"github.com/unstablemind/pocket/internal/dev/jira"
	"github.com/unstablemind/pocket/internal/dev/kubernetes"
	"github.com/unstablemind/pocket/internal/dev/linear"
	"github.com/unstablemind/pocket/internal/dev/npm"
	"github.com/unstablemind/pocket/internal/dev/prometheus"
	"github.com/unstablemind/pocket/internal/dev/pypi"
	"github.com/unstablemind/pocket/internal/dev/redis"
	"github.com/unstablemind/pocket/internal/dev/s3"
	"github.com/unstablemind/pocket/internal/dev/sentry"
	"github.com/unstablemind/pocket/internal/dev/vercel"
)

func NewDevCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dev",
		Aliases: []string{"d"},
		Short:   "Developer tool commands",
		Long:    `Interact with developer tools: GitHub, GitLab, Jira, Cloudflare, Vercel, Docker Hub, etc.`,
	}

	cmd.AddCommand(github.NewCmd())
	cmd.AddCommand(gitlab.NewCmd())
	cmd.AddCommand(linear.NewCmd())
	cmd.AddCommand(npm.NewCmd())
	cmd.AddCommand(pypi.NewCmd())
	cmd.AddCommand(jira.NewCmd())
	cmd.AddCommand(cloudflare.NewCmd())
	cmd.AddCommand(vercel.NewCmd())
	cmd.AddCommand(dockerhub.NewCmd())
	cmd.AddCommand(sentry.NewCmd())
	cmd.AddCommand(redis.NewCmd())
	cmd.AddCommand(prometheus.NewCmd())
	cmd.AddCommand(kubernetes.NewCmd())
	cmd.AddCommand(database.NewCmd())
	cmd.AddCommand(s3.NewCmd())
	cmd.AddCommand(gist.NewCmd())

	return cmd
}
