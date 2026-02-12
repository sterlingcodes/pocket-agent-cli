package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/internal/common/config"
)

// testConfigPath is set once in TestMain and reused by all tests.
var testConfigPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "integ-test-*")
	if err != nil {
		panic(err)
	}
	testConfigPath = filepath.Join(dir, "config.json")
	os.Setenv("POCKET_CONFIG", testConfigPath)

	code := m.Run()

	os.Unsetenv("POCKET_CONFIG")
	os.RemoveAll(dir)
	os.Exit(code)
}

func writeTestConfig(t *testing.T, values map[string]string) {
	t.Helper()
	data, err := json.Marshal(values)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(testConfigPath, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func clearTestConfig(t *testing.T) {
	t.Helper()
	os.Remove(testConfigPath)
}

// configKeysForIntegration returns the config keys needed for an integration
// to be considered "ready", based on the getIntegrationStatus switch.
var configKeysForIntegration = map[string][]string{
	"github":       {"github_token"},
	"gitlab":       {"gitlab_token"},
	"linear":       {"linear_token"},
	"twitter":      {"x_client_id"},
	"reddit":       {"reddit_client_id"},
	"mastodon":     {"mastodon_token"},
	"youtube":      {"youtube_api_key"},
	"email":        {"email_address", "email_password", "imap_server", "smtp_server"},
	"slack":        {"slack_token"},
	"discord":      {"discord_token"},
	"telegram":     {"telegram_token"},
	"twilio":       {"twilio_sid", "twilio_token", "twilio_phone"},
	"calendar":     {"google_cred_path"},
	"notion":       {"notion_token"},
	"todoist":      {"todoist_token"},
	"newsapi":      {"newsapi_key"},
	"stocks":       {"alphavantage_key"},
	"jira":         {"jira_url", "jira_email", "jira_token"},
	"cloudflare":   {"cloudflare_token"},
	"vercel":       {"vercel_token"},
	"trello":       {"trello_key", "trello_token"},
	"logseq":       {"logseq_graph"},
	"obsidian":     {"obsidian_vault"},
	"facebook-ads": {"facebook_ads_token", "facebook_ads_account_id"},
	"amazon-sp":    {"amazon_sp_client_id", "amazon_sp_client_secret", "amazon_sp_refresh_token", "amazon_sp_seller_id"},
	"shopify":      {"shopify_store", "shopify_token"},
	"spotify":      {"spotify_client_id", "spotify_client_secret"},
	"sentry":       {"sentry_auth_token"},
	"s3":           {"aws_profile", "aws_region"},
	"redis":        {"redis_url"},
	"prometheus":   {"prometheus_url"},
	"virustotal":   {"virustotal_api_key"},
	"gdrive":       {"google_api_key"},
	"gsheets":      {"google_api_key"},
}

// TestAllAuthIntegrationsHaveStatusCheck verifies that every integration with
// AuthNeeded==true has a corresponding case in getIntegrationStatus that
// returns "needs_setup" (not falling through to the default).
func TestAllAuthIntegrationsHaveStatusCheck(t *testing.T) {
	clearTestConfig(t)
	_, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	for _, integ := range allIntegrations {
		if !integ.AuthNeeded {
			continue
		}

		status := getIntegrationStatus(integ)
		if status != "needs_setup" {
			t.Errorf("integration %q: expected 'needs_setup' with empty config, got %q", integ.ID, status)
		}
	}
}

// TestAllAuthIntegrationsReturnReady verifies that when the correct config
// keys are set, getIntegrationStatus returns "ready" for each auth integration.
func TestAllAuthIntegrationsReturnReady(t *testing.T) {
	for _, integ := range allIntegrations {
		if !integ.AuthNeeded {
			continue
		}

		keys, ok := configKeysForIntegration[integ.ID]
		if !ok {
			t.Errorf("integration %q: no config keys mapping defined in test (add to configKeysForIntegration)", integ.ID)
			continue
		}

		// Write config with all required keys set
		cfgValues := make(map[string]string, len(keys))
		for _, k := range keys {
			cfgValues[k] = "test_value"
		}
		writeTestConfig(t, cfgValues)

		_, err := config.Load()
		if err != nil {
			t.Fatalf("load config for %q: %v", integ.ID, err)
		}

		status := getIntegrationStatus(integ)
		if status != "ready" {
			t.Errorf("integration %q: expected 'ready' with keys %v set, got %q", integ.ID, keys, status)
		}
	}

	clearTestConfig(t)
}

// TestNoAuthIntegrationsReturnNoAuth verifies that integrations with
// AuthNeeded==false return "no_auth" from getIntegrationStatus.
func TestNoAuthIntegrationsReturnNoAuth(t *testing.T) {
	clearTestConfig(t)
	_, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	for _, integ := range allIntegrations {
		if integ.AuthNeeded {
			continue
		}

		status := getIntegrationStatus(integ)
		if status != "no_auth" {
			t.Errorf("integration %q (AuthNeeded=false): expected 'no_auth', got %q", integ.ID, status)
		}
	}
}

// TestAuthIntegrationsWithSetupCmdHaveServiceEntry verifies that every
// integration referencing "pocket setup show X" has a corresponding entry
// in the services map in setup.go.
func TestAuthIntegrationsWithSetupCmdHaveServiceEntry(t *testing.T) {
	for _, integ := range allIntegrations {
		if integ.SetupCmd == "" {
			continue
		}

		// Extract service name from "pocket setup show <name>"
		const prefix = "pocket setup show "
		if !strings.HasPrefix(integ.SetupCmd, prefix) {
			t.Errorf("integration %q: SetupCmd %q doesn't match expected format %q...", integ.ID, integ.SetupCmd, prefix)
			continue
		}
		serviceName := strings.TrimPrefix(integ.SetupCmd, prefix)

		if _, ok := services[serviceName]; !ok {
			t.Errorf("integration %q: SetupCmd references service %q, but it's not in the services map", integ.ID, serviceName)
		}
	}
}

// TestSetupServiceKeysAreValidConfigKeys verifies that every KeyInfo.Key
// in the services map can be round-tripped through config.Set/Get.
func TestSetupServiceKeysAreValidConfigKeys(t *testing.T) {
	clearTestConfig(t)

	for svcName, svc := range services {
		for _, keyInfo := range svc.Keys {
			testVal := "test_" + keyInfo.Key

			if err := config.Set(keyInfo.Key, testVal); err != nil {
				t.Errorf("service %q: config.Set(%q) failed: %v", svcName, keyInfo.Key, err)
				continue
			}

			got, err := config.Get(keyInfo.Key)
			if err != nil {
				t.Errorf("service %q: config.Get(%q) failed: %v", svcName, keyInfo.Key, err)
				continue
			}

			if got != testVal {
				t.Errorf("service %q: config round-trip for %q: got %q, want %q", svcName, keyInfo.Key, got, testVal)
			}
		}
	}

	clearTestConfig(t)
}

// TestAllCommandGroupsRegistered verifies that each domain group command
// constructor returns a command with the expected Use name.
func TestAllCommandGroupsRegistered(t *testing.T) {
	// Each group constructor must return a command with the expected Use name.
	// This proves all group commands exist and can be registered on root.
	wantGroups := map[string]func() *cobra.Command{
		"social":       NewSocialCmd,
		"comms":        NewCommsCmd,
		"dev":          NewDevCmd,
		"productivity": NewProductivityCmd,
		"news":         NewNewsCmd,
		"knowledge":    NewKnowledgeCmd,
		"utility":      NewUtilityCmd,
		"system":       NewSystemCmd,
		"security":     NewSecurityCmd,
		"marketing":    NewMarketingCmd,
	}

	for name, constructor := range wantGroups {
		cmd := constructor()
		if cmd == nil {
			t.Errorf("group %q: constructor returned nil", name)
			continue
		}
		// Strip any args/flags from Use for comparison
		use := cmd.Use
		if idx := strings.IndexByte(use, ' '); idx != -1 {
			use = use[:idx]
		}
		if use != name {
			t.Errorf("group %q: expected Use=%q, got %q", name, name, use)
		}
	}
}

// TestFacebookAdsConfigKeys verifies the Facebook Ads config keys round-trip.
func TestFacebookAdsConfigKeys(t *testing.T) {
	clearTestConfig(t)

	keys := map[string]string{
		"facebook_ads_token":      "test_fb_token",
		"facebook_ads_account_id": "987654",
	}

	for k, v := range keys {
		if err := config.Set(k, v); err != nil {
			t.Fatalf("config.Set(%q) failed: %v", k, err)
		}
	}

	for k, want := range keys {
		got, err := config.Get(k)
		if err != nil {
			t.Fatalf("config.Get(%q) failed: %v", k, err)
		}
		if got != want {
			t.Errorf("config.Get(%q) = %q, want %q", k, got, want)
		}
	}

	clearTestConfig(t)
}

// TestIntegrationIDsAreUnique verifies no duplicate integration IDs exist.
func TestIntegrationIDsAreUnique(t *testing.T) {
	seen := make(map[string]bool, len(allIntegrations))
	for _, integ := range allIntegrations {
		if seen[integ.ID] {
			t.Errorf("duplicate integration ID: %q", integ.ID)
		}
		seen[integ.ID] = true
	}
}

// TestAllAuthIntegrationsHaveSetupCmd verifies that every auth-required
// integration has a SetupCmd pointing to setup instructions.
func TestAllAuthIntegrationsHaveSetupCmd(t *testing.T) {
	for _, integ := range allIntegrations {
		if !integ.AuthNeeded {
			continue
		}
		if integ.SetupCmd == "" {
			t.Errorf("integration %q requires auth but has no SetupCmd", integ.ID)
		}
	}
}
