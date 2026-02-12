package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	configPath string
	configOnce sync.Once
)

// Config holds all configuration
type Config struct {
	// Social
	XClientID          string `json:"x_client_id,omitempty"`
	XAccessToken       string `json:"x_access_token,omitempty"`
	XRefreshToken      string `json:"x_refresh_token,omitempty"`
	XTokenExpiry       string `json:"x_token_expiry,omitempty"`
	RedditClientID     string `json:"reddit_client_id,omitempty"`
	RedditAccessToken  string `json:"reddit_access_token,omitempty"`
	RedditRefreshToken string `json:"reddit_refresh_token,omitempty"`
	RedditTokenExpiry  string `json:"reddit_token_expiry,omitempty"`
	MastodonServer     string `json:"mastodon_server,omitempty"`
	MastodonToken      string `json:"mastodon_token,omitempty"`
	YouTubeAPIKey      string `json:"youtube_api_key,omitempty"`

	// Communication
	SlackToken    string `json:"slack_token,omitempty"`
	DiscordToken  string `json:"discord_token,omitempty"`
	TelegramToken string `json:"telegram_token,omitempty"`
	TwilioSID     string `json:"twilio_sid,omitempty"`
	TwilioToken   string `json:"twilio_token,omitempty"`
	TwilioPhone   string `json:"twilio_phone,omitempty"`

	// Email (IMAP/SMTP)
	EmailAddress  string `json:"email_address,omitempty"`
	EmailPassword string `json:"email_password,omitempty"`
	IMAPServer    string `json:"imap_server,omitempty"`
	IMAPPort      string `json:"imap_port,omitempty"`
	SMTPServer    string `json:"smtp_server,omitempty"`
	SMTPPort      string `json:"smtp_port,omitempty"`

	// Dev
	GitHubToken     string `json:"github_token,omitempty"`
	GitLabToken     string `json:"gitlab_token,omitempty"`
	GitLabURL       string `json:"gitlab_url,omitempty"`
	LinearToken     string `json:"linear_token,omitempty"`
	JiraURL         string `json:"jira_url,omitempty"`
	JiraEmail       string `json:"jira_email,omitempty"`
	JiraToken       string `json:"jira_token,omitempty"`
	VercelToken     string `json:"vercel_token,omitempty"`
	CloudflareToken string `json:"cloudflare_token,omitempty"`
	SentryAuthToken string `json:"sentry_auth_token,omitempty"`
	SentryOrg       string `json:"sentry_org,omitempty"`
	RedisURL        string `json:"redis_url,omitempty"`
	RedisPassword   string `json:"redis_password,omitempty"`
	PrometheusURL   string `json:"prometheus_url,omitempty"`
	PrometheusToken string `json:"prometheus_token,omitempty"`

	// Productivity
	NotionToken        string `json:"notion_token,omitempty"`
	TodoistToken       string `json:"todoist_token,omitempty"`
	TrelloKey          string `json:"trello_key,omitempty"`
	TrelloToken        string `json:"trello_token,omitempty"`
	GoogleCredPath     string `json:"google_cred_path,omitempty"`
	GoogleAPIKey       string `json:"google_api_key,omitempty"`
	GoogleClientID     string `json:"google_client_id,omitempty"`
	GoogleClientSecret string `json:"google_client_secret,omitempty"`
	GoogleRefreshToken string `json:"google_refresh_token,omitempty"`

	// AWS / S3
	AWSProfile string `json:"aws_profile,omitempty"`
	AWSRegion  string `json:"aws_region,omitempty"`

	// Spotify
	SpotifyClientID     string `json:"spotify_client_id,omitempty"`
	SpotifyClientSecret string `json:"spotify_client_secret,omitempty"`

	// News
	NewsAPIKey string `json:"newsapi_key,omitempty"`

	// Utility
	AlphaVantageKey string `json:"alphavantage_key,omitempty"`

	// Security
	VirusTotalAPIKey string `json:"virustotal_api_key,omitempty"`

	// Push Notifications
	PushoverToken string `json:"pushover_token,omitempty"`
	PushoverUser  string `json:"pushover_user,omitempty"`

	// Logseq
	LogseqGraph  string `json:"logseq_graph,omitempty"`
	LogseqGraphs string `json:"logseq_graphs,omitempty"`
	LogseqFormat string `json:"logseq_format,omitempty"`

	// Obsidian
	ObsidianVault       string `json:"obsidian_vault,omitempty"`
	ObsidianVaults      string `json:"obsidian_vaults,omitempty"`
	ObsidianDailyFormat string `json:"obsidian_daily_format,omitempty"`

	// Marketing
	FacebookAdsToken     string `json:"facebook_ads_token,omitempty"`
	FacebookAdsAccountID string `json:"facebook_ads_account_id,omitempty"`

	// Marketing - Amazon SP-API
	AmazonSPClientID     string `json:"amazon_sp_client_id,omitempty"`
	AmazonSPClientSecret string `json:"amazon_sp_client_secret,omitempty"`
	AmazonSPRefreshToken string `json:"amazon_sp_refresh_token,omitempty"`
	AmazonSPSellerID     string `json:"amazon_sp_seller_id,omitempty"`
	AmazonSPRegion       string `json:"amazon_sp_region,omitempty"`
	AmazonSPAccessToken  string `json:"amazon_sp_access_token,omitempty"`
	AmazonSPTokenExpiry  string `json:"amazon_sp_token_expiry,omitempty"`

	// Marketing - Shopify
	ShopifyStore string `json:"shopify_store,omitempty"`
	ShopifyToken string `json:"shopify_token,omitempty"`
}

// Path returns the config file path
func Path() string {
	configOnce.Do(func() {
		if p := os.Getenv("POCKET_CONFIG"); p != "" {
			configPath = p
			return
		}

		home, err := os.UserHomeDir()
		if err != nil {
			configPath = ".pocket.json"
			return
		}
		configPath = filepath.Join(home, ".config", "pocket", "config.json")
	})
	return configPath
}

// Load reads the config file
func Load() (*Config, error) {
	path := Path()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// Save writes the config file
func Save(cfg *Config) error {
	path := Path()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Set sets a config value by key
//
//nolint:gocyclo // complex but clear sequential logic
func Set(key, value string) error {
	cfg, err := Load()
	if err != nil {
		return err
	}

	key = normalizeKey(key)

	switch key {
	case "x_client_id":
		cfg.XClientID = value
	case "x_access_token":
		cfg.XAccessToken = value
	case "x_refresh_token":
		cfg.XRefreshToken = value
	case "x_token_expiry":
		cfg.XTokenExpiry = value
	case "reddit_client_id":
		cfg.RedditClientID = value
	case "reddit_access_token":
		cfg.RedditAccessToken = value
	case "reddit_refresh_token":
		cfg.RedditRefreshToken = value
	case "reddit_token_expiry":
		cfg.RedditTokenExpiry = value
	case "mastodon_server":
		cfg.MastodonServer = value
	case "mastodon_token":
		cfg.MastodonToken = value
	case "youtube_api_key":
		cfg.YouTubeAPIKey = value
	case "slack_token":
		cfg.SlackToken = value
	case "discord_token":
		cfg.DiscordToken = value
	case "telegram_token":
		cfg.TelegramToken = value
	case "twilio_sid":
		cfg.TwilioSID = value
	case "twilio_token":
		cfg.TwilioToken = value
	case "twilio_phone":
		cfg.TwilioPhone = value
	case "email_address":
		cfg.EmailAddress = value
	case "email_password":
		cfg.EmailPassword = value
	case "imap_server":
		cfg.IMAPServer = value
	case "imap_port":
		cfg.IMAPPort = value
	case "smtp_server":
		cfg.SMTPServer = value
	case "smtp_port":
		cfg.SMTPPort = value
	case "github_token":
		cfg.GitHubToken = value
	case "gitlab_token":
		cfg.GitLabToken = value
	case "gitlab_url":
		cfg.GitLabURL = value
	case "linear_token":
		cfg.LinearToken = value
	case "jira_url":
		cfg.JiraURL = value
	case "jira_email":
		cfg.JiraEmail = value
	case "jira_token":
		cfg.JiraToken = value
	case "vercel_token":
		cfg.VercelToken = value
	case "cloudflare_token":
		cfg.CloudflareToken = value
	case "sentry_auth_token":
		cfg.SentryAuthToken = value
	case "sentry_org":
		cfg.SentryOrg = value
	case "redis_url":
		cfg.RedisURL = value
	case "redis_password":
		cfg.RedisPassword = value
	case "prometheus_url":
		cfg.PrometheusURL = value
	case "prometheus_token":
		cfg.PrometheusToken = value
	case "notion_token":
		cfg.NotionToken = value
	case "todoist_token":
		cfg.TodoistToken = value
	case "trello_key":
		cfg.TrelloKey = value
	case "trello_token":
		cfg.TrelloToken = value
	case "google_cred_path":
		cfg.GoogleCredPath = value
	case "google_api_key":
		cfg.GoogleAPIKey = value
	case "google_client_id":
		cfg.GoogleClientID = value
	case "google_client_secret":
		cfg.GoogleClientSecret = value
	case "google_refresh_token":
		cfg.GoogleRefreshToken = value
	case "virustotal_api_key":
		cfg.VirusTotalAPIKey = value
	case "aws_profile":
		cfg.AWSProfile = value
	case "aws_region":
		cfg.AWSRegion = value
	case "spotify_client_id":
		cfg.SpotifyClientID = value
	case "spotify_client_secret":
		cfg.SpotifyClientSecret = value
	case "newsapi_key":
		cfg.NewsAPIKey = value
	case "alphavantage_key":
		cfg.AlphaVantageKey = value
	case "pushover_token":
		cfg.PushoverToken = value
	case "pushover_user":
		cfg.PushoverUser = value
	case "logseq_graph":
		cfg.LogseqGraph = value
	case "logseq_graphs":
		cfg.LogseqGraphs = value
	case "logseq_format":
		cfg.LogseqFormat = value
	case "obsidian_vault":
		cfg.ObsidianVault = value
	case "obsidian_vaults":
		cfg.ObsidianVaults = value
	case "obsidian_daily_format":
		cfg.ObsidianDailyFormat = value
	case "facebook_ads_token":
		cfg.FacebookAdsToken = value
	case "facebook_ads_account_id":
		cfg.FacebookAdsAccountID = value
	case "amazon_sp_client_id":
		cfg.AmazonSPClientID = value
	case "amazon_sp_client_secret":
		cfg.AmazonSPClientSecret = value
	case "amazon_sp_refresh_token":
		cfg.AmazonSPRefreshToken = value
	case "amazon_sp_seller_id":
		cfg.AmazonSPSellerID = value
	case "amazon_sp_region":
		cfg.AmazonSPRegion = value
	case "amazon_sp_access_token":
		cfg.AmazonSPAccessToken = value
	case "amazon_sp_token_expiry":
		cfg.AmazonSPTokenExpiry = value
	case "shopify_store":
		cfg.ShopifyStore = value
	case "shopify_token":
		cfg.ShopifyToken = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return Save(cfg)
}

// Get gets a config value by key
//
//nolint:gocyclo // complex but clear sequential logic
func Get(key string) (string, error) {
	cfg, err := Load()
	if err != nil {
		return "", err
	}

	key = normalizeKey(key)

	switch key {
	case "x_client_id":
		return cfg.XClientID, nil
	case "x_access_token":
		return cfg.XAccessToken, nil
	case "x_refresh_token":
		return cfg.XRefreshToken, nil
	case "x_token_expiry":
		return cfg.XTokenExpiry, nil
	case "reddit_client_id":
		return cfg.RedditClientID, nil
	case "reddit_access_token":
		return cfg.RedditAccessToken, nil
	case "reddit_refresh_token":
		return cfg.RedditRefreshToken, nil
	case "reddit_token_expiry":
		return cfg.RedditTokenExpiry, nil
	case "mastodon_server":
		return cfg.MastodonServer, nil
	case "mastodon_token":
		return cfg.MastodonToken, nil
	case "youtube_api_key":
		return cfg.YouTubeAPIKey, nil
	case "slack_token":
		return cfg.SlackToken, nil
	case "discord_token":
		return cfg.DiscordToken, nil
	case "telegram_token":
		return cfg.TelegramToken, nil
	case "twilio_sid":
		return cfg.TwilioSID, nil
	case "twilio_token":
		return cfg.TwilioToken, nil
	case "twilio_phone":
		return cfg.TwilioPhone, nil
	case "email_address":
		return cfg.EmailAddress, nil
	case "email_password":
		return cfg.EmailPassword, nil
	case "imap_server":
		return cfg.IMAPServer, nil
	case "imap_port":
		return cfg.IMAPPort, nil
	case "smtp_server":
		return cfg.SMTPServer, nil
	case "smtp_port":
		return cfg.SMTPPort, nil
	case "github_token":
		return cfg.GitHubToken, nil
	case "gitlab_token":
		return cfg.GitLabToken, nil
	case "gitlab_url":
		return cfg.GitLabURL, nil
	case "linear_token":
		return cfg.LinearToken, nil
	case "jira_url":
		return cfg.JiraURL, nil
	case "jira_email":
		return cfg.JiraEmail, nil
	case "jira_token":
		return cfg.JiraToken, nil
	case "vercel_token":
		return cfg.VercelToken, nil
	case "cloudflare_token":
		return cfg.CloudflareToken, nil
	case "sentry_auth_token":
		return cfg.SentryAuthToken, nil
	case "sentry_org":
		return cfg.SentryOrg, nil
	case "redis_url":
		return cfg.RedisURL, nil
	case "redis_password":
		return cfg.RedisPassword, nil
	case "prometheus_url":
		return cfg.PrometheusURL, nil
	case "prometheus_token":
		return cfg.PrometheusToken, nil
	case "notion_token":
		return cfg.NotionToken, nil
	case "todoist_token":
		return cfg.TodoistToken, nil
	case "trello_key":
		return cfg.TrelloKey, nil
	case "trello_token":
		return cfg.TrelloToken, nil
	case "google_cred_path":
		return cfg.GoogleCredPath, nil
	case "google_api_key":
		return cfg.GoogleAPIKey, nil
	case "google_client_id":
		return cfg.GoogleClientID, nil
	case "google_client_secret":
		return cfg.GoogleClientSecret, nil
	case "google_refresh_token":
		return cfg.GoogleRefreshToken, nil
	case "virustotal_api_key":
		return cfg.VirusTotalAPIKey, nil
	case "aws_profile":
		return cfg.AWSProfile, nil
	case "aws_region":
		return cfg.AWSRegion, nil
	case "spotify_client_id":
		return cfg.SpotifyClientID, nil
	case "spotify_client_secret":
		return cfg.SpotifyClientSecret, nil
	case "newsapi_key":
		return cfg.NewsAPIKey, nil
	case "alphavantage_key":
		return cfg.AlphaVantageKey, nil
	case "pushover_token":
		return cfg.PushoverToken, nil
	case "pushover_user":
		return cfg.PushoverUser, nil
	case "logseq_graph":
		return cfg.LogseqGraph, nil
	case "logseq_graphs":
		return cfg.LogseqGraphs, nil
	case "logseq_format":
		return cfg.LogseqFormat, nil
	case "obsidian_vault":
		return cfg.ObsidianVault, nil
	case "obsidian_vaults":
		return cfg.ObsidianVaults, nil
	case "obsidian_daily_format":
		return cfg.ObsidianDailyFormat, nil
	case "facebook_ads_token":
		return cfg.FacebookAdsToken, nil
	case "facebook_ads_account_id":
		return cfg.FacebookAdsAccountID, nil
	case "amazon_sp_client_id":
		return cfg.AmazonSPClientID, nil
	case "amazon_sp_client_secret":
		return cfg.AmazonSPClientSecret, nil
	case "amazon_sp_refresh_token":
		return cfg.AmazonSPRefreshToken, nil
	case "amazon_sp_seller_id":
		return cfg.AmazonSPSellerID, nil
	case "amazon_sp_region":
		return cfg.AmazonSPRegion, nil
	case "amazon_sp_access_token":
		return cfg.AmazonSPAccessToken, nil
	case "amazon_sp_token_expiry":
		return cfg.AmazonSPTokenExpiry, nil
	case "shopify_store":
		return cfg.ShopifyStore, nil
	case "shopify_token":
		return cfg.ShopifyToken, nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

// Redacted returns config with sensitive values masked
func (c *Config) Redacted() map[string]string {
	redact := func(s string) string {
		if s == "" {
			return "(not set)"
		}
		if len(s) <= 8 {
			return "****"
		}
		return s[:4] + "****" + s[len(s)-4:]
	}

	return map[string]string{
		"x_client_id":             redact(c.XClientID),
		"x_access_token":          redact(c.XAccessToken),
		"x_refresh_token":         redact(c.XRefreshToken),
		"x_token_expiry":          c.XTokenExpiry,
		"reddit_client_id":        redact(c.RedditClientID),
		"reddit_access_token":     redact(c.RedditAccessToken),
		"reddit_refresh_token":    redact(c.RedditRefreshToken),
		"reddit_token_expiry":     c.RedditTokenExpiry,
		"mastodon_server":         c.MastodonServer,
		"mastodon_token":          redact(c.MastodonToken),
		"youtube_api_key":         redact(c.YouTubeAPIKey),
		"slack_token":             redact(c.SlackToken),
		"discord_token":           redact(c.DiscordToken),
		"telegram_token":          redact(c.TelegramToken),
		"twilio_sid":              redact(c.TwilioSID),
		"twilio_token":            redact(c.TwilioToken),
		"twilio_phone":            c.TwilioPhone,
		"email_address":           c.EmailAddress,
		"email_password":          redact(c.EmailPassword),
		"imap_server":             c.IMAPServer,
		"imap_port":               c.IMAPPort,
		"smtp_server":             c.SMTPServer,
		"smtp_port":               c.SMTPPort,
		"github_token":            redact(c.GitHubToken),
		"gitlab_token":            redact(c.GitLabToken),
		"gitlab_url":              c.GitLabURL,
		"linear_token":            redact(c.LinearToken),
		"jira_url":                c.JiraURL,
		"jira_email":              c.JiraEmail,
		"jira_token":              redact(c.JiraToken),
		"vercel_token":            redact(c.VercelToken),
		"cloudflare_token":        redact(c.CloudflareToken),
		"sentry_auth_token":       redact(c.SentryAuthToken),
		"sentry_org":              c.SentryOrg,
		"redis_url":               c.RedisURL,
		"redis_password":          redact(c.RedisPassword),
		"prometheus_url":          c.PrometheusURL,
		"prometheus_token":        redact(c.PrometheusToken),
		"notion_token":            redact(c.NotionToken),
		"todoist_token":           redact(c.TodoistToken),
		"trello_key":              redact(c.TrelloKey),
		"trello_token":            redact(c.TrelloToken),
		"google_cred_path":        c.GoogleCredPath,
		"google_api_key":          redact(c.GoogleAPIKey),
		"google_client_id":        redact(c.GoogleClientID),
		"google_client_secret":    redact(c.GoogleClientSecret),
		"google_refresh_token":    redact(c.GoogleRefreshToken),
		"virustotal_api_key":      redact(c.VirusTotalAPIKey),
		"aws_profile":             c.AWSProfile,
		"aws_region":              c.AWSRegion,
		"spotify_client_id":       redact(c.SpotifyClientID),
		"spotify_client_secret":   redact(c.SpotifyClientSecret),
		"newsapi_key":             redact(c.NewsAPIKey),
		"alphavantage_key":        redact(c.AlphaVantageKey),
		"pushover_token":          redact(c.PushoverToken),
		"pushover_user":           redact(c.PushoverUser),
		"logseq_graph":            c.LogseqGraph,
		"logseq_graphs":           c.LogseqGraphs,
		"logseq_format":           c.LogseqFormat,
		"obsidian_vault":          c.ObsidianVault,
		"obsidian_vaults":         c.ObsidianVaults,
		"obsidian_daily_format":   c.ObsidianDailyFormat,
		"facebook_ads_token":      redact(c.FacebookAdsToken),
		"facebook_ads_account_id": c.FacebookAdsAccountID,
		"amazon_sp_client_id":     redact(c.AmazonSPClientID),
		"amazon_sp_client_secret": redact(c.AmazonSPClientSecret),
		"amazon_sp_refresh_token": redact(c.AmazonSPRefreshToken),
		"amazon_sp_seller_id":     c.AmazonSPSellerID,
		"amazon_sp_region":        c.AmazonSPRegion,
		"amazon_sp_access_token":  redact(c.AmazonSPAccessToken),
		"amazon_sp_token_expiry":  c.AmazonSPTokenExpiry,
		"shopify_store":           c.ShopifyStore,
		"shopify_token":           redact(c.ShopifyToken),
	}
}

// MustGet gets a config value or returns an error if not set
func MustGet(key string) (string, error) {
	val, err := Get(key)
	if err != nil {
		return "", err
	}
	if val == "" {
		return "", errors.New("config key not set: " + key + " (use: pocket config set " + key + " <value>)")
	}
	return val, nil
}

func normalizeKey(key string) string {
	return strings.ToLower(strings.ReplaceAll(key, "-", "_"))
}
