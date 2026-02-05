package commands

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

// Integration describes an available integration
type Integration struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Group       string   `json:"group"`
	Description string   `json:"desc"`
	AuthNeeded  bool     `json:"auth_needed"`
	Status      string   `json:"status"` // "ready", "needs_setup", "no_auth"
	Commands    []string `json:"commands"`
	SetupCmd    string   `json:"setup_cmd,omitempty"`
}

var allIntegrations = []Integration{
	// News - No Auth
	{
		ID:          "hackernews",
		Name:        "Hacker News",
		Group:       "news",
		Description: "Tech news, stories, and discussions from Hacker News",
		AuthNeeded:  false,
		Commands:    []string{"pocket news hn top", "pocket news hn new", "pocket news hn best", "pocket news hn ask", "pocket news hn show", "pocket news hn item [id]"},
	},
	{
		ID:          "rss",
		Name:        "RSS/Atom Feeds",
		Group:       "news",
		Description: "Fetch and manage RSS/Atom feeds from any source",
		AuthNeeded:  false,
		Commands:    []string{"pocket news feeds fetch [url]", "pocket news feeds list", "pocket news feeds add [url]", "pocket news feeds read [name]", "pocket news feeds remove [name]"},
	},
	{
		ID:          "newsapi",
		Name:        "NewsAPI",
		Group:       "news",
		Description: "Search news articles and get headlines from 80,000+ sources",
		AuthNeeded:  true,
		Commands:    []string{"pocket news newsapi headlines", "pocket news newsapi search [query]", "pocket news newsapi sources"},
		SetupCmd:    "pocket setup show newsapi",
	},

	// Knowledge - No Auth
	{
		ID:          "wikipedia",
		Name:        "Wikipedia",
		Group:       "knowledge",
		Description: "Search and read Wikipedia articles",
		AuthNeeded:  false,
		Commands:    []string{"pocket knowledge wiki search [query]", "pocket knowledge wiki summary [title]", "pocket knowledge wiki article [title]"},
	},
	{
		ID:          "stackexchange",
		Name:        "StackOverflow",
		Group:       "knowledge",
		Description: "Search programming Q&A from StackOverflow and StackExchange sites",
		AuthNeeded:  false,
		Commands:    []string{"pocket knowledge so search [query]", "pocket knowledge so question [id]", "pocket knowledge so answers [id]"},
	},
	{
		ID:          "dictionary",
		Name:        "Dictionary",
		Group:       "knowledge",
		Description: "Word definitions, synonyms, antonyms, and pronunciations",
		AuthNeeded:  false,
		Commands:    []string{"pocket knowledge dict define [word]", "pocket knowledge dict synonyms [word]", "pocket knowledge dict antonyms [word]"},
	},

	// Utility - No Auth
	{
		ID:          "weather",
		Name:        "Weather",
		Group:       "utility",
		Description: "Current weather and forecasts for any location",
		AuthNeeded:  false,
		Commands:    []string{"pocket utility weather now [location]", "pocket utility weather forecast [location]"},
	},
	{
		ID:          "crypto",
		Name:        "CoinGecko",
		Group:       "utility",
		Description: "Cryptocurrency prices, market data, and trending coins",
		AuthNeeded:  false,
		Commands:    []string{"pocket utility crypto price [coins...]", "pocket utility crypto info [coin]", "pocket utility crypto top", "pocket utility crypto trending", "pocket utility crypto search [query]"},
	},
	{
		ID:          "ipinfo",
		Name:        "IP Geolocation",
		Group:       "utility",
		Description: "IP address lookup with geolocation data",
		AuthNeeded:  false,
		Commands:    []string{"pocket utility ip me", "pocket utility ip lookup [ip]"},
	},
	{
		ID:          "domain",
		Name:        "DNS/WHOIS/SSL",
		Group:       "utility",
		Description: "DNS lookups, WHOIS domain info, and SSL certificate inspection",
		AuthNeeded:  false,
		Commands:    []string{"pocket utility domain dns [domain]", "pocket utility domain whois [domain]", "pocket utility domain ssl [domain]"},
	},
	{
		ID:          "currency",
		Name:        "Currency Exchange",
		Group:       "utility",
		Description: "Real-time currency exchange rates and conversion",
		AuthNeeded:  false,
		Commands:    []string{"pocket utility currency rate [from] [to]", "pocket utility currency convert [amount] [from] [to]", "pocket utility currency list"},
	},
	{
		ID:          "wayback",
		Name:        "Wayback Machine",
		Group:       "utility",
		Description: "Check archived versions of websites via Internet Archive",
		AuthNeeded:  false,
		Commands:    []string{"pocket utility wayback check [url]", "pocket utility wayback latest [url]", "pocket utility wayback snapshots [url]"},
	},
	{
		ID:          "holidays",
		Name:        "Public Holidays",
		Group:       "utility",
		Description: "Public holidays by country and year",
		AuthNeeded:  false,
		Commands:    []string{"pocket utility holidays list [country] [year]", "pocket utility holidays next [country]", "pocket utility holidays countries"},
	},
	{
		ID:          "translate",
		Name:        "Translation",
		Group:       "utility",
		Description: "Translate text between languages",
		AuthNeeded:  false,
		Commands:    []string{"pocket utility translate text [text] --from [lang] --to [lang]", "pocket utility translate languages"},
	},
	{
		ID:          "urlshort",
		Name:        "URL Shortener",
		Group:       "utility",
		Description: "Shorten and expand URLs",
		AuthNeeded:  false,
		Commands:    []string{"pocket utility url shorten [url]", "pocket utility url expand [short-url]"},
	},
	// Utility - Auth Required
	{
		ID:          "stocks",
		Name:        "Stock Market",
		Group:       "utility",
		Description: "Stock quotes, search, and company info via Alpha Vantage",
		AuthNeeded:  true,
		Commands:    []string{"pocket utility stocks quote [symbol]", "pocket utility stocks search [query]", "pocket utility stocks info [symbol]"},
		SetupCmd:    "pocket setup show alphavantage",
	},

	// Dev - No Auth
	{
		ID:          "npm",
		Name:        "npm Registry",
		Group:       "dev",
		Description: "Search npm packages, get info, versions, and dependencies",
		AuthNeeded:  false,
		Commands:    []string{"pocket dev npm search [query]", "pocket dev npm info [package]", "pocket dev npm versions [package]", "pocket dev npm deps [package]"},
	},
	{
		ID:          "pypi",
		Name:        "PyPI Registry",
		Group:       "dev",
		Description: "Search Python packages, get info, versions, and dependencies",
		AuthNeeded:  false,
		Commands:    []string{"pocket dev pypi search [query]", "pocket dev pypi info [package]", "pocket dev pypi versions [package]", "pocket dev pypi deps [package]"},
	},
	{
		ID:          "dockerhub",
		Name:        "Docker Hub",
		Group:       "dev",
		Description: "Search Docker images, get tags, and inspect manifests",
		AuthNeeded:  false,
		Commands:    []string{"pocket dev dockerhub search [query]", "pocket dev dockerhub image [name]", "pocket dev dockerhub tags [name]", "pocket dev dockerhub inspect [name:tag]"},
	},

	// Dev - Auth Required
	{
		ID:          "github",
		Name:        "GitHub",
		Group:       "dev",
		Description: "Repos, issues, PRs, notifications, and search on GitHub",
		AuthNeeded:  true,
		Commands:    []string{"pocket dev github repos", "pocket dev github repo [owner/name]", "pocket dev github issues", "pocket dev github issue [repo] [num]", "pocket dev github prs -r [repo]", "pocket dev github pr [repo] [num]", "pocket dev github notifications", "pocket dev github search [query]"},
		SetupCmd:    "pocket setup show github",
	},
	{
		ID:          "gitlab",
		Name:        "GitLab",
		Group:       "dev",
		Description: "Projects, issues, and merge requests on GitLab",
		AuthNeeded:  true,
		Commands:    []string{"pocket dev gitlab projects", "pocket dev gitlab issues", "pocket dev gitlab mrs"},
		SetupCmd:    "pocket setup show gitlab",
	},
	{
		ID:          "linear",
		Name:        "Linear",
		Group:       "dev",
		Description: "Issues and project management with Linear",
		AuthNeeded:  true,
		Commands:    []string{"pocket dev linear issues", "pocket dev linear teams", "pocket dev linear create [desc]"},
		SetupCmd:    "pocket setup show linear",
	},
	{
		ID:          "jira",
		Name:        "Jira",
		Group:       "dev",
		Description: "Issues, projects, and sprint management with Atlassian Jira",
		AuthNeeded:  true,
		Commands:    []string{"pocket dev jira issues", "pocket dev jira issue [key]", "pocket dev jira projects", "pocket dev jira create [summary]", "pocket dev jira transition [key] [status]"},
		SetupCmd:    "pocket setup show jira",
	},
	{
		ID:          "cloudflare",
		Name:        "Cloudflare",
		Group:       "dev",
		Description: "DNS, zones, cache purge, and analytics via Cloudflare",
		AuthNeeded:  true,
		Commands:    []string{"pocket dev cloudflare zones", "pocket dev cloudflare zone [id]", "pocket dev cloudflare dns [zone-id]", "pocket dev cloudflare purge [zone-id]", "pocket dev cloudflare analytics [zone-id]"},
		SetupCmd:    "pocket setup show cloudflare",
	},
	{
		ID:          "vercel",
		Name:        "Vercel",
		Group:       "dev",
		Description: "Projects, deployments, domains, and environment variables on Vercel",
		AuthNeeded:  true,
		Commands:    []string{"pocket dev vercel projects", "pocket dev vercel project [name]", "pocket dev vercel deployments [project]", "pocket dev vercel domains", "pocket dev vercel env [project]"},
		SetupCmd:    "pocket setup show vercel",
	},

	// Social - Auth Required
	{
		ID:          "twitter",
		Name:        "Twitter/X",
		Group:       "social",
		Description: "Post tweets, read timeline, search, and get user info",
		AuthNeeded:  true,
		Commands:    []string{"pocket social twitter timeline", "pocket social twitter post [msg]", "pocket social twitter search [query]", "pocket social twitter user [name]"},
		SetupCmd:    "pocket setup show twitter",
	},
	{
		ID:          "reddit",
		Name:        "Reddit",
		Group:       "social",
		Description: "Browse feeds, subreddits, search, and post",
		AuthNeeded:  true,
		Commands:    []string{"pocket social reddit feed", "pocket social reddit subreddit [name]", "pocket social reddit search [query]", "pocket social reddit post [content]"},
		SetupCmd:    "pocket setup show reddit",
	},
	{
		ID:          "mastodon",
		Name:        "Mastodon",
		Group:       "social",
		Description: "Fediverse: timelines, posting, and search",
		AuthNeeded:  true,
		Commands:    []string{"pocket social mastodon timeline", "pocket social mastodon post [content]", "pocket social mastodon search [query]"},
		SetupCmd:    "pocket setup show mastodon",
	},
	{
		ID:          "youtube",
		Name:        "YouTube",
		Group:       "social",
		Description: "Search videos, get channel info, video metrics, comments, and trending",
		AuthNeeded:  true,
		Commands:    []string{"pocket social youtube search [query]", "pocket social youtube video [id]", "pocket social youtube channel [id]", "pocket social youtube videos [channel-id]", "pocket social youtube comments [video-id]", "pocket social youtube trending"},
		SetupCmd:    "pocket setup show youtube",
	},

	// Communication - Auth Required
	{
		ID:          "email",
		Name:        "Email (IMAP/SMTP)",
		Group:       "comms",
		Description: "Read, search, send, and reply to emails via IMAP/SMTP (Gmail, Outlook, Yahoo, etc.)",
		AuthNeeded:  true,
		Commands:    []string{"pocket comms email list", "pocket comms email read [uid]", "pocket comms email send [body]", "pocket comms email reply [uid] [body]", "pocket comms email search [query]", "pocket comms email mailboxes"},
		SetupCmd:    "pocket setup show email",
	},
	{
		ID:          "slack",
		Name:        "Slack",
		Group:       "comms",
		Description: "Channels, messages, users, DMs, and search in Slack workspaces",
		AuthNeeded:  true,
		Commands:    []string{"pocket comms slack channels", "pocket comms slack messages [channel]", "pocket comms slack send [channel] [msg]", "pocket comms slack users", "pocket comms slack dm [user] [msg]", "pocket comms slack search [query]"},
		SetupCmd:    "pocket setup show slack",
	},
	{
		ID:          "discord",
		Name:        "Discord",
		Group:       "comms",
		Description: "Servers (guilds), channels, messages, and DMs in Discord",
		AuthNeeded:  true,
		Commands:    []string{"pocket comms discord guilds", "pocket comms discord channels [guild]", "pocket comms discord messages [channel]", "pocket comms discord send [channel] [msg]", "pocket comms discord dm [user] [msg]"},
		SetupCmd:    "pocket setup show discord",
	},
	{
		ID:          "telegram",
		Name:        "Telegram",
		Group:       "comms",
		Description: "Chats, messages, and forwarding via Telegram bot",
		AuthNeeded:  true,
		Commands:    []string{"pocket comms telegram me", "pocket comms telegram chats", "pocket comms telegram updates", "pocket comms telegram send [chat] [msg]", "pocket comms telegram forward [from] [to] [msg-id]"},
		SetupCmd:    "pocket setup show telegram",
	},
	{
		ID:          "twilio",
		Name:        "Twilio (SMS)",
		Group:       "comms",
		Description: "Send and manage SMS messages via Twilio",
		AuthNeeded:  true,
		Commands:    []string{"pocket comms twilio send [to] [msg]", "pocket comms twilio messages", "pocket comms twilio message [sid]", "pocket comms twilio account"},
		SetupCmd:    "pocket setup show twilio",
	},
	// Communication - No Auth
	{
		ID:          "notify",
		Name:        "Push Notifications",
		Group:       "comms",
		Description: "Send push notifications via ntfy.sh (no auth) or Pushover (auth)",
		AuthNeeded:  false,
		Commands:    []string{"pocket comms notify ntfy [topic] [msg]", "pocket comms notify pushover [msg]"},
	},
	{
		ID:          "webhook",
		Name:        "Webhooks",
		Group:       "comms",
		Description: "Send data to webhooks (generic, Slack, Discord)",
		AuthNeeded:  false,
		Commands:    []string{"pocket comms webhook send [url] [data]", "pocket comms webhook slack [url] [msg]", "pocket comms webhook discord [url] [msg]"},
	},

	// Productivity - Auth Required
	{
		ID:          "calendar",
		Name:        "Google Calendar",
		Group:       "productivity",
		Description: "View and create calendar events",
		AuthNeeded:  true,
		Commands:    []string{"pocket productivity calendar events", "pocket productivity calendar today", "pocket productivity calendar create"},
		SetupCmd:    "pocket setup show google",
	},
	{
		ID:          "notion",
		Name:        "Notion",
		Group:       "productivity",
		Description: "Search pages and query databases in Notion",
		AuthNeeded:  true,
		Commands:    []string{"pocket productivity notion search [query]", "pocket productivity notion page [id]", "pocket productivity notion database [id]"},
		SetupCmd:    "pocket setup show notion",
	},
	{
		ID:          "todoist",
		Name:        "Todoist",
		Group:       "productivity",
		Description: "Tasks and projects in Todoist",
		AuthNeeded:  true,
		Commands:    []string{"pocket productivity todoist tasks", "pocket productivity todoist projects", "pocket productivity todoist add [task]", "pocket productivity todoist complete [id]"},
		SetupCmd:    "pocket setup show todoist",
	},
	{
		ID:          "trello",
		Name:        "Trello",
		Group:       "productivity",
		Description: "Boards, lists, and cards in Trello",
		AuthNeeded:  true,
		Commands:    []string{"pocket productivity trello boards", "pocket productivity trello board [id]", "pocket productivity trello cards [board-id]", "pocket productivity trello card [id]", "pocket productivity trello create [name]"},
		SetupCmd:    "pocket setup show trello",
	},

	// AI - Auth Required
	{
		ID:          "openai",
		Name:        "OpenAI",
		Group:       "ai",
		Description: "Chat completions with GPT models",
		AuthNeeded:  true,
		Commands:    []string{"pocket ai openai chat [prompt]", "pocket ai openai models"},
		SetupCmd:    "pocket setup show openai",
	},
	{
		ID:          "anthropic",
		Name:        "Anthropic",
		Group:       "ai",
		Description: "Chat with Claude models",
		AuthNeeded:  true,
		Commands:    []string{"pocket ai anthropic chat [prompt]"},
		SetupCmd:    "pocket setup show anthropic",
	},
}

func NewIntegrationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "integrations",
		Aliases: []string{"int", "services"},
		Short:   "List all available integrations",
	}

	cmd.AddCommand(newIntListCmd())
	cmd.AddCommand(newIntReadyCmd())
	cmd.AddCommand(newIntGroupCmd())

	return cmd
}

func newIntListCmd() *cobra.Command {
	var noAuth bool
	var group string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all integrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			result := make([]Integration, 0)

			for _, integ := range allIntegrations {
				// Filter by auth requirement
				if noAuth && integ.AuthNeeded {
					continue
				}

				// Filter by group
				if group != "" && integ.Group != group {
					continue
				}

				// Set status
				integ.Status = getIntegrationStatus(cfg, integ)
				result = append(result, integ)
			}

			return output.Print(result)
		},
	}

	cmd.Flags().BoolVar(&noAuth, "no-auth", false, "Only show integrations that don't need authentication")
	cmd.Flags().StringVarP(&group, "group", "g", "", "Filter by group: news, knowledge, utility, dev, social, comms, productivity, ai")

	return cmd
}

func newIntReadyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ready",
		Short: "List integrations ready to use (configured or no auth needed)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			result := make([]Integration, 0)

			for _, integ := range allIntegrations {
				status := getIntegrationStatus(cfg, integ)
				if status == "ready" || status == "no_auth" {
					integ.Status = status
					result = append(result, integ)
				}
			}

			return output.Print(result)
		},
	}

	return cmd
}

func newIntGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "groups",
		Short: "List integration groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			groups := map[string]struct {
				Name  string `json:"name"`
				Desc  string `json:"desc"`
				Count int    `json:"count"`
			}{
				"news":         {Name: "News", Desc: "News feeds and articles", Count: 0},
				"knowledge":    {Name: "Knowledge", Desc: "Research and reference", Count: 0},
				"utility":      {Name: "Utility", Desc: "Weather, tools", Count: 0},
				"dev":          {Name: "Dev", Desc: "Developer tools and package registries", Count: 0},
				"social":       {Name: "Social", Desc: "Social media platforms", Count: 0},
				"comms":        {Name: "Comms", Desc: "Email and messaging", Count: 0},
				"productivity": {Name: "Productivity", Desc: "Calendar, tasks, notes", Count: 0},
				"ai":           {Name: "AI", Desc: "AI/LLM providers", Count: 0},
			}

			for _, integ := range allIntegrations {
				if g, ok := groups[integ.Group]; ok {
					g.Count++
					groups[integ.Group] = g
				}
			}

			type GroupInfo struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Desc  string `json:"desc"`
				Count int    `json:"count"`
			}

			result := []GroupInfo{
				{ID: "news", Name: groups["news"].Name, Desc: groups["news"].Desc, Count: groups["news"].Count},
				{ID: "knowledge", Name: groups["knowledge"].Name, Desc: groups["knowledge"].Desc, Count: groups["knowledge"].Count},
				{ID: "utility", Name: groups["utility"].Name, Desc: groups["utility"].Desc, Count: groups["utility"].Count},
				{ID: "dev", Name: groups["dev"].Name, Desc: groups["dev"].Desc, Count: groups["dev"].Count},
				{ID: "social", Name: groups["social"].Name, Desc: groups["social"].Desc, Count: groups["social"].Count},
				{ID: "comms", Name: groups["comms"].Name, Desc: groups["comms"].Desc, Count: groups["comms"].Count},
				{ID: "productivity", Name: groups["productivity"].Name, Desc: groups["productivity"].Desc, Count: groups["productivity"].Count},
				{ID: "ai", Name: groups["ai"].Name, Desc: groups["ai"].Desc, Count: groups["ai"].Count},
			}

			return output.Print(result)
		},
	}

	return cmd
}

func getIntegrationStatus(cfg *config.Config, integ Integration) string {
	if !integ.AuthNeeded {
		return "no_auth"
	}

	// Check if required keys are set
	switch integ.ID {
	case "github":
		if v, _ := config.Get("github_token"); v != "" {
			return "ready"
		}
	case "gitlab":
		if v, _ := config.Get("gitlab_token"); v != "" {
			return "ready"
		}
	case "linear":
		if v, _ := config.Get("linear_token"); v != "" {
			return "ready"
		}
	case "twitter":
		if v, _ := config.Get("twitter_api_key"); v != "" {
			return "ready"
		}
	case "reddit":
		if v, _ := config.Get("reddit_client_id"); v != "" {
			return "ready"
		}
	case "mastodon":
		if v, _ := config.Get("mastodon_token"); v != "" {
			return "ready"
		}
	case "youtube":
		if v, _ := config.Get("youtube_api_key"); v != "" {
			return "ready"
		}
	case "email":
		addr, _ := config.Get("email_address")
		pass, _ := config.Get("email_password")
		imap, _ := config.Get("imap_server")
		smtp, _ := config.Get("smtp_server")
		if addr != "" && pass != "" && imap != "" && smtp != "" {
			return "ready"
		}
	case "slack":
		if v, _ := config.Get("slack_token"); v != "" {
			return "ready"
		}
	case "discord":
		if v, _ := config.Get("discord_token"); v != "" {
			return "ready"
		}
	case "telegram":
		if v, _ := config.Get("telegram_token"); v != "" {
			return "ready"
		}
	case "twilio":
		sid, _ := config.Get("twilio_sid")
		token, _ := config.Get("twilio_token")
		phone, _ := config.Get("twilio_phone")
		if sid != "" && token != "" && phone != "" {
			return "ready"
		}
	case "calendar":
		if v, _ := config.Get("google_cred_path"); v != "" {
			return "ready"
		}
	case "notion":
		if v, _ := config.Get("notion_token"); v != "" {
			return "ready"
		}
	case "todoist":
		if v, _ := config.Get("todoist_token"); v != "" {
			return "ready"
		}
	case "openai":
		if v, _ := config.Get("openai_key"); v != "" {
			return "ready"
		}
	case "anthropic":
		if v, _ := config.Get("anthropic_key"); v != "" {
			return "ready"
		}
	case "newsapi":
		if v, _ := config.Get("newsapi_key"); v != "" {
			return "ready"
		}
	case "stocks":
		if v, _ := config.Get("alphavantage_key"); v != "" {
			return "ready"
		}
	case "jira":
		url, _ := config.Get("jira_url")
		email, _ := config.Get("jira_email")
		token, _ := config.Get("jira_token")
		if url != "" && email != "" && token != "" {
			return "ready"
		}
	case "cloudflare":
		if v, _ := config.Get("cloudflare_token"); v != "" {
			return "ready"
		}
	case "vercel":
		if v, _ := config.Get("vercel_token"); v != "" {
			return "ready"
		}
	case "trello":
		key, _ := config.Get("trello_key")
		token, _ := config.Get("trello_token")
		if key != "" && token != "" {
			return "ready"
		}
	}

	return "needs_setup"
}
