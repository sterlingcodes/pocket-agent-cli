package commands

import (
	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

const (
	statusNoAuth = "no_auth"
	statusReady  = "ready"
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
	{
		ID:          "sentry",
		Name:        "Sentry",
		Group:       "dev",
		Description: "Error tracking: projects, issues, and events from Sentry",
		AuthNeeded:  true,
		Commands:    []string{"pocket dev sentry projects", "pocket dev sentry issues [project-slug]", "pocket dev sentry issue [issue-id]", "pocket dev sentry events [issue-id]"},
		SetupCmd:    "pocket setup show sentry",
	},
	{
		ID:          "s3",
		Name:        "AWS S3",
		Group:       "dev",
		Description: "List buckets, browse objects, upload/download, and generate presigned URLs",
		AuthNeeded:  true,
		Commands:    []string{"pocket dev s3 buckets", "pocket dev s3 ls [s3-path]", "pocket dev s3 get [s3-path] [local-path]", "pocket dev s3 put [local-path] [s3-path]", "pocket dev s3 presign [s3-path]"},
		SetupCmd:    "pocket setup show s3",
	},
	{
		ID:          "redis",
		Name:        "Redis",
		Group:       "dev",
		Description: "Get/set keys, list keys, and view server info on Redis",
		AuthNeeded:  true,
		Commands:    []string{"pocket dev redis get [key]", "pocket dev redis set [key] [value]", "pocket dev redis del [key...]", "pocket dev redis keys [pattern]", "pocket dev redis info"},
		SetupCmd:    "pocket setup show redis",
	},
	{
		ID:          "prometheus",
		Name:        "Prometheus",
		Group:       "dev",
		Description: "PromQL queries, alerts, and scrape targets from Prometheus",
		AuthNeeded:  true,
		Commands:    []string{"pocket dev prometheus query [promql]", "pocket dev prometheus range [promql]", "pocket dev prometheus alerts", "pocket dev prometheus targets"},
		SetupCmd:    "pocket setup show prometheus",
	},
	// Dev - No Auth
	{
		ID:          "gist",
		Name:        "GitHub Gists",
		Group:       "dev",
		Description: "Create, list, and read GitHub Gists",
		AuthNeeded:  false,
		Commands:    []string{"pocket dev gist list", "pocket dev gist get [id]", "pocket dev gist create [content]"},
	},
	{
		ID:          "kubernetes",
		Name:        "Kubernetes",
		Group:       "dev",
		Description: "Pods, logs, deployments, services, and resource descriptions via kubectl",
		AuthNeeded:  false,
		Commands:    []string{"pocket dev kube pods", "pocket dev kube logs [pod]", "pocket dev kube deployments", "pocket dev kube services", "pocket dev kube describe [resource] [name]"},
	},
	{
		ID:          "database",
		Name:        "Database (SQLite)",
		Group:       "dev",
		Description: "Query, inspect schema, and list tables in SQLite databases",
		AuthNeeded:  false,
		Commands:    []string{"pocket dev db query [db-path] [sql]", "pocket dev db schema [db-path]", "pocket dev db tables [db-path]"},
	},

	// Social - Auth Required
	{
		ID:          "twitter",
		Name:        "Twitter/X",
		Group:       "social",
		Description: "Post tweets, delete tweets, get account info (free tier: 1,500 posts/month)",
		AuthNeeded:  true,
		Commands:    []string{"pocket social twitter post [msg]", "pocket social twitter delete [id]", "pocket social twitter me", "pocket social twitter --reply-to [id] [msg]"},
		SetupCmd:    "pocket setup show twitter",
	},
	{
		ID:          "reddit",
		Name:        "Reddit",
		Group:       "social",
		Description: "Browse feeds, subreddits, search, users, and comments (free tier: 100 req/min)",
		AuthNeeded:  true,
		Commands:    []string{"pocket social reddit feed", "pocket social reddit subreddit [name]", "pocket social reddit search [query]", "pocket social reddit user [name]", "pocket social reddit comments [post-id]"},
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
	{
		ID:          "spotify",
		Name:        "Spotify",
		Group:       "social",
		Description: "Search tracks, artists, and albums on Spotify",
		AuthNeeded:  true,
		Commands:    []string{"pocket social spotify search [query]", "pocket social spotify track [id]", "pocket social spotify artist [id]", "pocket social spotify album [id]"},
		SetupCmd:    "pocket setup show spotify",
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
		SetupCmd:    "pocket setup show calendar",
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
	// Productivity - Local (Path Required)
	{
		ID:          "logseq",
		Name:        "Logseq",
		Group:       "productivity",
		Description: "Local Logseq graphs - read/write pages, search, journals",
		AuthNeeded:  true,
		Commands:    []string{"pocket productivity logseq graphs", "pocket productivity logseq pages", "pocket productivity logseq read [page]", "pocket productivity logseq write [page] [content]", "pocket productivity logseq search [query]", "pocket productivity logseq journal", "pocket productivity logseq recent"},
		SetupCmd:    "pocket setup show logseq",
	},

	// Productivity - Local (Path Required)
	{
		ID:          "obsidian",
		Name:        "Obsidian",
		Group:       "productivity",
		Description: "Local Obsidian vaults - read/write notes, search, daily notes",
		AuthNeeded:  true,
		Commands:    []string{"pocket productivity obsidian vaults", "pocket productivity obsidian notes", "pocket productivity obsidian read [note]", "pocket productivity obsidian write [note] [content]", "pocket productivity obsidian search [query]", "pocket productivity obsidian daily", "pocket productivity obsidian recent"},
		SetupCmd:    "pocket setup show obsidian",
	},

	// Marketing - Auth Required
	{
		ID:          "facebook-ads",
		Name:        "Facebook Ads (Meta)",
		Group:       "marketing",
		Description: "Manage Facebook/Meta ad campaigns, ad sets, ads, and view performance insights",
		AuthNeeded:  true,
		Commands:    []string{"pocket marketing facebook-ads account", "pocket marketing facebook-ads campaigns", "pocket marketing facebook-ads campaign-create", "pocket marketing facebook-ads adsets", "pocket marketing facebook-ads ads", "pocket marketing facebook-ads insights"},
		SetupCmd:    "pocket setup show facebook-ads",
	},
	{
		ID:          "amazon-sp",
		Name:        "Amazon Selling Partner",
		Group:       "marketing",
		Description: "Manage Amazon seller orders, inventory, and reports via SP-API",
		AuthNeeded:  true,
		Commands:    []string{"pocket marketing amazon-sp orders", "pocket marketing amazon-sp order [id]", "pocket marketing amazon-sp order-items [id]", "pocket marketing amazon-sp inventory", "pocket marketing amazon-sp report-create", "pocket marketing amazon-sp report-status [id]"},
		SetupCmd:    "pocket setup show amazon-sp",
	},
	{
		ID:          "shopify",
		Name:        "Shopify",
		Group:       "marketing",
		Description: "Manage Shopify store: orders, products, customers, and inventory",
		AuthNeeded:  true,
		Commands:    []string{"pocket marketing shopify shop", "pocket marketing shopify orders", "pocket marketing shopify order [id]", "pocket marketing shopify products", "pocket marketing shopify product [id]", "pocket marketing shopify customers", "pocket marketing shopify customer-search [query]", "pocket marketing shopify inventory", "pocket marketing shopify inventory-set"},
		SetupCmd:    "pocket setup show shopify",
	},

	// System - macOS Only (No Auth)
	{
		ID:          "reminders",
		Name:        "Apple Reminders",
		Group:       "system",
		Description: "Manage Apple Reminders via AppleScript (macOS only)",
		AuthNeeded:  false,
		Commands:    []string{"pocket system reminders lists", "pocket system reminders list [name]", "pocket system reminders add [title]", "pocket system reminders complete [id]", "pocket system reminders delete [id]", "pocket system reminders today", "pocket system reminders overdue"},
	},
	{
		ID:          "notes",
		Name:        "Apple Notes",
		Group:       "system",
		Description: "Read and manage Apple Notes via AppleScript (macOS only)",
		AuthNeeded:  false,
		Commands:    []string{"pocket system notes folders", "pocket system notes list", "pocket system notes read [name]", "pocket system notes search [query]", "pocket system notes create [name] [body]", "pocket system notes append [name] [text]"},
	},
	{
		ID:          "apple-calendar",
		Name:        "Apple Calendar",
		Group:       "system",
		Description: "Manage Apple Calendar events via AppleScript (macOS only)",
		AuthNeeded:  false,
		Commands:    []string{"pocket system calendar calendars", "pocket system calendar today", "pocket system calendar events", "pocket system calendar event [title]", "pocket system calendar create [title]", "pocket system calendar upcoming", "pocket system calendar week"},
	},
	{
		ID:          "contacts",
		Name:        "Apple Contacts",
		Group:       "system",
		Description: "Search and manage Apple Contacts via AppleScript (macOS only)",
		AuthNeeded:  false,
		Commands:    []string{"pocket system contacts list", "pocket system contacts search [query]", "pocket system contacts get [name]", "pocket system contacts groups", "pocket system contacts group [name]", "pocket system contacts create [name]"},
	},
	{
		ID:          "finder",
		Name:        "Finder",
		Group:       "system",
		Description: "Finder operations, file info, tags, Spotlight search (macOS only)",
		AuthNeeded:  false,
		Commands:    []string{"pocket system finder open [path]", "pocket system finder reveal [path]", "pocket system finder info [path]", "pocket system finder list [path]", "pocket system finder tags [path]", "pocket system finder tag [path] [tag]", "pocket system finder trash [path]", "pocket system finder search [query]"},
	},
	{
		ID:          "safari",
		Name:        "Safari",
		Group:       "system",
		Description: "Safari tabs, bookmarks, reading list, history (macOS only)",
		AuthNeeded:  false,
		Commands:    []string{"pocket system safari tabs", "pocket system safari url", "pocket system safari open [url]", "pocket system safari bookmarks", "pocket system safari reading-list", "pocket system safari add-reading [url]", "pocket system safari history"},
	},
	{
		ID:          "clipboard",
		Name:        "Clipboard",
		Group:       "system",
		Description: "Get/set macOS clipboard content (macOS only)",
		AuthNeeded:  false,
		Commands:    []string{"pocket system clipboard get", "pocket system clipboard set [text]", "pocket system clipboard clear", "pocket system clipboard copy [file]"},
	},
	{
		ID:          "imessage",
		Name:        "iMessage",
		Group:       "system",
		Description: "Send and read iMessages via Messages.app (macOS only)",
		AuthNeeded:  false,
		Commands:    []string{"pocket system imessage send [recipient] [message]", "pocket system imessage chats", "pocket system imessage read [contact]", "pocket system imessage search [query]", "pocket system imessage unread"},
	},
	{
		ID:          "apple-mail",
		Name:        "Apple Mail",
		Group:       "system",
		Description: "Read and send emails via Apple Mail (macOS only)",
		AuthNeeded:  false,
		Commands:    []string{"pocket system mail accounts", "pocket system mail mailboxes", "pocket system mail list", "pocket system mail read [id]", "pocket system mail search [query]", "pocket system mail send", "pocket system mail unread", "pocket system mail count"},
	},

	// Security - Auth Required
	{
		ID:          "virustotal",
		Name:        "VirusTotal",
		Group:       "security",
		Description: "Scan URLs, domains, IPs, and file hashes for threats via VirusTotal",
		AuthNeeded:  true,
		Commands:    []string{"pocket security virustotal url [url]", "pocket security virustotal domain [domain]", "pocket security virustotal ip [ip]", "pocket security virustotal hash [hash]"},
		SetupCmd:    "pocket setup show virustotal",
	},
	// Security - No Auth
	{
		ID:          "shodan",
		Name:        "Shodan",
		Group:       "security",
		Description: "IP lookup for open ports and vulnerabilities via Shodan",
		AuthNeeded:  false,
		Commands:    []string{"pocket security shodan lookup [ip]"},
	},
	{
		ID:          "crtsh",
		Name:        "crt.sh",
		Group:       "security",
		Description: "Certificate transparency log lookups via crt.sh",
		AuthNeeded:  false,
		Commands:    []string{"pocket security crtsh lookup [domain]"},
	},
	{
		ID:          "hibp",
		Name:        "Have I Been Pwned",
		Group:       "security",
		Description: "Check passwords against breaches and list public data breaches",
		AuthNeeded:  false,
		Commands:    []string{"pocket security hibp password [password]", "pocket security hibp breaches"},
	},

	// Productivity - Auth Required (Google API)
	{
		ID:          "gdrive",
		Name:        "Google Drive",
		Group:       "productivity",
		Description: "Search and get file metadata from Google Drive",
		AuthNeeded:  true,
		Commands:    []string{"pocket productivity gdrive search [query]", "pocket productivity gdrive info [file-id]"},
		SetupCmd:    "pocket setup show google-api",
	},
	{
		ID:          "gsheets",
		Name:        "Google Sheets",
		Group:       "productivity",
		Description: "Read spreadsheets and search cell values in Google Sheets",
		AuthNeeded:  true,
		Commands:    []string{"pocket productivity gsheets get [spreadsheet-id]", "pocket productivity gsheets read [spreadsheet-id] [range]", "pocket productivity gsheets search [spreadsheet-id] [query]"},
		SetupCmd:    "pocket setup show google-api",
	},

	// Utility - No Auth (additional)
	{
		ID:          "geocoding",
		Name:        "Geocoding",
		Group:       "utility",
		Description: "Forward and reverse geocoding (address to coordinates and back)",
		AuthNeeded:  false,
		Commands:    []string{"pocket utility geocode forward [address]", "pocket utility geocode reverse [lat] [lon]"},
	},
	{
		ID:          "timezone",
		Name:        "Timezone",
		Group:       "utility",
		Description: "Get time in timezones, lookup timezone by IP, list all timezones",
		AuthNeeded:  false,
		Commands:    []string{"pocket utility timezone get [timezone]", "pocket utility timezone ip [ip]", "pocket utility timezone list"},
	},
	{
		ID:          "paste",
		Name:        "Paste",
		Group:       "utility",
		Description: "Create and fetch text pastes (pastebin-like)",
		AuthNeeded:  false,
		Commands:    []string{"pocket utility paste create [content]", "pocket utility paste get [url]"},
	},
	{
		ID:          "netdiag",
		Name:        "Network Diagnostics",
		Group:       "utility",
		Description: "HTTP headers, port scanning, and DNS/ping diagnostics",
		AuthNeeded:  false,
		Commands:    []string{"pocket utility netdiag headers [url]", "pocket utility netdiag ports [host]", "pocket utility netdiag ping [host]"},
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
			result := make([]Integration, 0)

			for i := range allIntegrations {
				integ := allIntegrations[i]
				// Filter by auth requirement
				if noAuth && integ.AuthNeeded {
					continue
				}

				// Filter by group
				if group != "" && integ.Group != group {
					continue
				}

				// Set status
				integ.Status = getIntegrationStatus(integ)
				result = append(result, integ)
			}

			return output.Print(result)
		},
	}

	cmd.Flags().BoolVar(&noAuth, "no-auth", false, "Only show integrations that don't need authentication")
	cmd.Flags().StringVarP(&group, "group", "g", "", "Filter by group: news, knowledge, utility, dev, social, comms, productivity, system, security, marketing")

	return cmd
}

func newIntReadyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ready",
		Short: "List integrations ready to use (configured or no auth needed)",
		RunE: func(cmd *cobra.Command, args []string) error {
			result := make([]Integration, 0)

			for i := range allIntegrations {
				integ := allIntegrations[i]
				status := getIntegrationStatus(integ)
				if status == statusReady || status == statusNoAuth {
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
				"system":       {Name: "System", Desc: "macOS system integrations", Count: 0},
				"security":     {Name: "Security", Desc: "Security scanning and threat intelligence", Count: 0},
				"marketing":    {Name: "Marketing", Desc: "Ad platforms and marketing tools", Count: 0},
			}

			for i := range allIntegrations {
				integ := allIntegrations[i]
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
				{ID: "system", Name: groups["system"].Name, Desc: groups["system"].Desc, Count: groups["system"].Count},
				{ID: "security", Name: groups["security"].Name, Desc: groups["security"].Desc, Count: groups["security"].Count},
				{ID: "marketing", Name: groups["marketing"].Name, Desc: groups["marketing"].Desc, Count: groups["marketing"].Count},
			}

			return output.Print(result)
		},
	}

	return cmd
}

//nolint:gocyclo,gocritic // complex but clear sequential logic; Integration is read-only value type
func getIntegrationStatus(integ Integration) string {
	if !integ.AuthNeeded {
		return statusNoAuth
	}

	// Check if required keys are set
	switch integ.ID {
	case "github":
		if v, _ := config.Get("github_token"); v != "" {
			return statusReady
		}
	case "gitlab":
		if v, _ := config.Get("gitlab_token"); v != "" {
			return statusReady
		}
	case "linear":
		if v, _ := config.Get("linear_token"); v != "" {
			return statusReady
		}
	case "twitter":
		if v, _ := config.Get("x_client_id"); v != "" {
			return statusReady
		}
	case "reddit":
		if v, _ := config.Get("reddit_client_id"); v != "" {
			return statusReady
		}
	case "mastodon":
		if v, _ := config.Get("mastodon_token"); v != "" {
			return statusReady
		}
	case "youtube":
		if v, _ := config.Get("youtube_api_key"); v != "" {
			return statusReady
		}
	case "email":
		addr, _ := config.Get("email_address")
		pass, _ := config.Get("email_password")
		imap, _ := config.Get("imap_server")
		smtp, _ := config.Get("smtp_server")
		if addr != "" && pass != "" && imap != "" && smtp != "" {
			return statusReady
		}
	case "slack":
		if v, _ := config.Get("slack_token"); v != "" {
			return statusReady
		}
	case "discord":
		if v, _ := config.Get("discord_token"); v != "" {
			return statusReady
		}
	case "telegram":
		if v, _ := config.Get("telegram_token"); v != "" {
			return statusReady
		}
	case "twilio":
		sid, _ := config.Get("twilio_sid")
		token, _ := config.Get("twilio_token")
		phone, _ := config.Get("twilio_phone")
		if sid != "" && token != "" && phone != "" {
			return statusReady
		}
	case "calendar":
		if v, _ := config.Get("google_cred_path"); v != "" {
			return statusReady
		}
	case "notion":
		if v, _ := config.Get("notion_token"); v != "" {
			return statusReady
		}
	case "todoist":
		if v, _ := config.Get("todoist_token"); v != "" {
			return statusReady
		}
	case "newsapi":
		if v, _ := config.Get("newsapi_key"); v != "" {
			return statusReady
		}
	case "stocks":
		if v, _ := config.Get("alphavantage_key"); v != "" {
			return statusReady
		}
	case "jira":
		url, _ := config.Get("jira_url")
		email, _ := config.Get("jira_email")
		token, _ := config.Get("jira_token")
		if url != "" && email != "" && token != "" {
			return statusReady
		}
	case "cloudflare":
		if v, _ := config.Get("cloudflare_token"); v != "" {
			return statusReady
		}
	case "vercel":
		if v, _ := config.Get("vercel_token"); v != "" {
			return statusReady
		}
	case "trello":
		key, _ := config.Get("trello_key")
		token, _ := config.Get("trello_token")
		if key != "" && token != "" {
			return statusReady
		}
	case "logseq":
		if v, _ := config.Get("logseq_graph"); v != "" {
			return statusReady
		}
	case "obsidian":
		if v, _ := config.Get("obsidian_vault"); v != "" {
			return statusReady
		}
	case "facebook-ads":
		token, _ := config.Get("facebook_ads_token")
		acctID, _ := config.Get("facebook_ads_account_id")
		if token != "" && acctID != "" {
			return statusReady
		}
	case "amazon-sp":
		cid, _ := config.Get("amazon_sp_client_id")
		secret, _ := config.Get("amazon_sp_client_secret")
		refresh, _ := config.Get("amazon_sp_refresh_token")
		seller, _ := config.Get("amazon_sp_seller_id")
		if cid != "" && secret != "" && refresh != "" && seller != "" {
			return statusReady
		}
	case "shopify":
		store, _ := config.Get("shopify_store")
		token, _ := config.Get("shopify_token")
		if store != "" && token != "" {
			return statusReady
		}
	case "spotify":
		cid, _ := config.Get("spotify_client_id")
		secret, _ := config.Get("spotify_client_secret")
		if cid != "" && secret != "" {
			return statusReady
		}
	case "sentry":
		if v, _ := config.Get("sentry_auth_token"); v != "" {
			return statusReady
		}
	case "s3":
		profile, _ := config.Get("aws_profile")
		region, _ := config.Get("aws_region")
		if profile != "" && region != "" {
			return statusReady
		}
	case "redis":
		if v, _ := config.Get("redis_url"); v != "" {
			return statusReady
		}
	case "prometheus":
		if v, _ := config.Get("prometheus_url"); v != "" {
			return statusReady
		}
	case "virustotal":
		if v, _ := config.Get("virustotal_api_key"); v != "" {
			return statusReady
		}
	case "gdrive":
		if v, _ := config.Get("google_api_key"); v != "" {
			return statusReady
		}
	case "gsheets":
		if v, _ := config.Get("google_api_key"); v != "" {
			return statusReady
		}
	}

	return "needs_setup"
}
