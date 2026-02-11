package facebookads

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

var baseURL = "https://graph.facebook.com/v24.0"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// fbClient holds credentials for Meta Marketing API calls.
type fbClient struct {
	token     string
	accountID string
}

// Account represents a Facebook Ads account.
type Account struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	AccountID       string `json:"account_id"`
	AccountStatus   int    `json:"account_status"`
	Currency        string `json:"currency"`
	TimezoneID      int    `json:"timezone_id"`
	TimezoneName    string `json:"timezone_name"`
	AmountSpent     string `json:"amount_spent"`
	Balance         string `json:"balance"`
	SpendCap        string `json:"spend_cap"`
	BusinessName    string `json:"business_name,omitempty"`
	BusinessCountry string `json:"business_country,omitempty"`
}

// Campaign represents a Facebook Ads campaign.
type Campaign struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Status         string `json:"status"`
	Objective      string `json:"objective"`
	DailyBudget    string `json:"daily_budget,omitempty"`
	LifetimeBudget string `json:"lifetime_budget,omitempty"`
	BudgetRemain   string `json:"budget_remaining,omitempty"`
	CreatedTime    string `json:"created_time"`
	UpdatedTime    string `json:"updated_time"`
}

// AdSet represents a Facebook Ads ad set.
type AdSet struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	CampaignID       string `json:"campaign_id"`
	Status           string `json:"status"`
	DailyBudget      string `json:"daily_budget,omitempty"`
	LifetimeBudget   string `json:"lifetime_budget,omitempty"`
	BillingEvent     string `json:"billing_event"`
	OptimizationGoal string `json:"optimization_goal"`
	CreatedTime      string `json:"created_time"`
	UpdatedTime      string `json:"updated_time"`
}

// Ad represents a Facebook Ad.
type Ad struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	AdSetID     string `json:"adset_id"`
	CampaignID  string `json:"campaign_id"`
	Status      string `json:"status"`
	CreatedTime string `json:"created_time"`
	UpdatedTime string `json:"updated_time"`
}

// Insight represents a Facebook Ads insight row.
type Insight struct {
	DateStart   string `json:"date_start"`
	DateStop    string `json:"date_stop"`
	Impressions string `json:"impressions"`
	Clicks      string `json:"clicks,omitempty"`
	Spend       string `json:"spend"`
	Reach       string `json:"reach,omitempty"`
	CTR         string `json:"ctr,omitempty"`
	CPC         string `json:"cpc,omitempty"`
	CPM         string `json:"cpm,omitempty"`
	Actions     []any  `json:"actions,omitempty"`
}

// NewCmd returns the facebook-ads parent command.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "facebook-ads",
		Aliases: []string{"fb", "meta-ads"},
		Short:   "Facebook Ads (Meta Marketing API) commands",
	}

	cmd.AddCommand(newAccountCmd())
	cmd.AddCommand(newCampaignsCmd())
	cmd.AddCommand(newCampaignCreateCmd())
	cmd.AddCommand(newCampaignUpdateCmd())
	cmd.AddCommand(newAdSetsCmd())
	cmd.AddCommand(newAdSetCreateCmd())
	cmd.AddCommand(newAdsCmd())
	cmd.AddCommand(newInsightsCmd())

	return cmd
}

func newClient() (*fbClient, error) {
	token, err := config.Get("facebook_ads_token")
	if err != nil {
		return nil, err
	}
	if token == "" {
		return nil, output.PrintError("missing_config", "Facebook Ads token not configured", map[string]string{
			"setup": "Run: pocket setup show facebook-ads",
		})
	}

	accountID, err := config.Get("facebook_ads_account_id")
	if err != nil {
		return nil, err
	}
	if accountID == "" {
		return nil, output.PrintError("missing_config", "Facebook Ads account ID not configured", map[string]string{
			"setup": "Run: pocket config set facebook_ads_account_id YOUR_AD_ACCOUNT_ID",
		})
	}

	return &fbClient{token: token, accountID: accountID}, nil
}

func (c *fbClient) actID() string {
	if strings.HasPrefix(c.accountID, "act_") {
		return c.accountID
	}
	return "act_" + c.accountID
}

func (c *fbClient) doGet(endpoint string, params url.Values) (map[string]any, error) {
	if params == nil {
		params = url.Values{}
	}

	reqURL := fmt.Sprintf("%s/%s?%s", baseURL, endpoint, params.Encode())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", "Pocket-CLI/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]any
		if decErr := json.NewDecoder(resp.Body).Decode(&errResp); decErr == nil {
			if apiErr := parseAPIError(errResp); apiErr != nil {
				return nil, apiErr
			}
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

func (c *fbClient) doPost(endpoint string, payload map[string]string) (map[string]any, error) {
	form := url.Values{}
	for k, v := range payload {
		form.Set(k, v)
	}

	reqURL := fmt.Sprintf("%s/%s", baseURL, endpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Pocket-CLI/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]any
		if decErr := json.NewDecoder(resp.Body).Decode(&errResp); decErr == nil {
			if apiErr := parseAPIError(errResp); apiErr != nil {
				return nil, apiErr
			}
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

func parseAPIError(resp map[string]any) error {
	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		return nil
	}
	msg := getString(errObj, "message")
	typ := getString(errObj, "type")
	code := getInt(errObj, "code")
	subcode := getInt(errObj, "error_subcode")
	traceID := getString(errObj, "fbtrace_id")

	errMsg := fmt.Sprintf("Meta API error (type=%s, code=%d", typ, code)
	if subcode != 0 {
		errMsg += fmt.Sprintf(", subcode=%d", subcode)
	}
	if traceID != "" {
		errMsg += fmt.Sprintf(", trace=%s", traceID)
	}
	errMsg += "): " + msg
	return fmt.Errorf("%s", errMsg)
}

// --- account ---

func newAccountCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "account",
		Short: "Get ad account details",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			params := url.Values{}
			params.Set("fields", "id,name,account_id,account_status,currency,timezone_id,timezone_name,amount_spent,balance,spend_cap,business_name,business_country_code")

			raw, err := c.doGet(c.actID(), params)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			acct := Account{
				ID:              getString(raw, "id"),
				Name:            getString(raw, "name"),
				AccountID:       getString(raw, "account_id"),
				AccountStatus:   getInt(raw, "account_status"),
				Currency:        getString(raw, "currency"),
				TimezoneID:      getInt(raw, "timezone_id"),
				TimezoneName:    getString(raw, "timezone_name"),
				AmountSpent:     getString(raw, "amount_spent"),
				Balance:         getString(raw, "balance"),
				SpendCap:        getString(raw, "spend_cap"),
				BusinessName:    getString(raw, "business_name"),
				BusinessCountry: getString(raw, "business_country_code"),
			}

			return output.Print(acct)
		},
	}
}

// --- campaigns ---

func newCampaignsCmd() *cobra.Command {
	var status string
	var limit int

	cmd := &cobra.Command{
		Use:   "campaigns",
		Short: "List campaigns",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			params := url.Values{}
			params.Set("fields", "id,name,status,objective,daily_budget,lifetime_budget,budget_remaining,created_time,updated_time")
			params.Set("limit", fmt.Sprintf("%d", limit))
			if status != "" {
				params.Set("effective_status", fmt.Sprintf(`["%s"]`, strings.ToUpper(status)))
			}

			raw, err := c.doGet(c.actID()+"/campaigns", params)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			data := getDataArray(raw)
			result := make([]Campaign, 0, len(data))
			for _, item := range data {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				result = append(result, Campaign{
					ID:             getString(m, "id"),
					Name:           getString(m, "name"),
					Status:         getString(m, "status"),
					Objective:      getString(m, "objective"),
					DailyBudget:    getString(m, "daily_budget"),
					LifetimeBudget: getString(m, "lifetime_budget"),
					BudgetRemain:   getString(m, "budget_remaining"),
					CreatedTime:    getString(m, "created_time"),
					UpdatedTime:    getString(m, "updated_time"),
				})
			}

			return output.Print(result)
		},
	}

	cmd.Flags().StringVarP(&status, "status", "s", "", "Filter by status: ACTIVE, PAUSED, ARCHIVED")
	cmd.Flags().IntVarP(&limit, "limit", "l", 25, "Number of campaigns to return")

	return cmd
}

func newCampaignCreateCmd() *cobra.Command {
	var name, objective, status, dailyBudget string
	var specialCategories []string

	cmd := &cobra.Command{
		Use:   "campaign-create",
		Short: "Create a new campaign",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			if name == "" {
				return output.PrintError("missing_flag", "--name is required", nil)
			}
			if objective == "" {
				return output.PrintError("missing_flag", "--objective is required (OUTCOME_AWARENESS, OUTCOME_TRAFFIC, OUTCOME_ENGAGEMENT, OUTCOME_LEADS, OUTCOME_APP_PROMOTION, OUTCOME_SALES)", nil)
			}

			payload := map[string]string{
				"name":      name,
				"objective": strings.ToUpper(objective),
				"status":    strings.ToUpper(status),
			}

			if dailyBudget != "" {
				payload["daily_budget"] = dailyBudget
			}

			catJSON, _ := json.Marshal(specialCategories)
			payload["special_ad_categories"] = string(catJSON)

			raw, err := c.doPost(c.actID()+"/campaigns", payload)
			if err != nil {
				return output.PrintError("create_failed", err.Error(), nil)
			}

			return output.Print(map[string]string{
				"id":     getString(raw, "id"),
				"status": "created",
			})
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Campaign name (required)")
	cmd.Flags().StringVar(&objective, "objective", "", "Campaign objective (required): OUTCOME_AWARENESS, OUTCOME_TRAFFIC, OUTCOME_ENGAGEMENT, OUTCOME_LEADS, OUTCOME_APP_PROMOTION, OUTCOME_SALES")
	cmd.Flags().StringVar(&status, "status", "PAUSED", "Initial status: ACTIVE or PAUSED")
	cmd.Flags().StringVar(&dailyBudget, "daily-budget", "", "Daily budget in cents (e.g. 1000 = $10.00)")
	cmd.Flags().StringSliceVar(&specialCategories, "special-categories", []string{}, "Special ad categories (EMPLOYMENT, HOUSING, CREDIT, ISSUES_ELECTIONS_POLITICS)")

	return cmd
}

func newCampaignUpdateCmd() *cobra.Command {
	var name, status, dailyBudget string

	cmd := &cobra.Command{
		Use:   "campaign-update [campaign-id]",
		Short: "Update a campaign",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			payload := map[string]string{}
			if name != "" {
				payload["name"] = name
			}
			if status != "" {
				payload["status"] = strings.ToUpper(status)
			}
			if dailyBudget != "" {
				payload["daily_budget"] = dailyBudget
			}

			if len(payload) == 0 {
				return output.PrintError("no_changes", "Provide at least one flag: --name, --status, or --daily-budget", nil)
			}

			raw, err := c.doPost(args[0], payload)
			if err != nil {
				return output.PrintError("update_failed", err.Error(), nil)
			}

			success := getBool(raw, "success")
			return output.Print(map[string]any{
				"id":      args[0],
				"success": success,
			})
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New campaign name")
	cmd.Flags().StringVar(&status, "status", "", "New status: ACTIVE, PAUSED, ARCHIVED")
	cmd.Flags().StringVar(&dailyBudget, "daily-budget", "", "New daily budget in cents")

	return cmd
}

// --- adsets ---

func newAdSetsCmd() *cobra.Command {
	var campaignID, status string
	var limit int

	cmd := &cobra.Command{
		Use:   "adsets",
		Short: "List ad sets",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			params := url.Values{}
			params.Set("fields", "id,name,campaign_id,status,daily_budget,lifetime_budget,billing_event,optimization_goal,created_time,updated_time")
			params.Set("limit", fmt.Sprintf("%d", limit))
			if status != "" {
				params.Set("effective_status", fmt.Sprintf(`["%s"]`, strings.ToUpper(status)))
			}

			endpoint := c.actID() + "/adsets"
			if campaignID != "" {
				endpoint = campaignID + "/adsets"
			}

			raw, err := c.doGet(endpoint, params)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			data := getDataArray(raw)
			result := make([]AdSet, 0, len(data))
			for _, item := range data {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				result = append(result, AdSet{
					ID:               getString(m, "id"),
					Name:             getString(m, "name"),
					CampaignID:       getString(m, "campaign_id"),
					Status:           getString(m, "status"),
					DailyBudget:      getString(m, "daily_budget"),
					LifetimeBudget:   getString(m, "lifetime_budget"),
					BillingEvent:     getString(m, "billing_event"),
					OptimizationGoal: getString(m, "optimization_goal"),
					CreatedTime:      getString(m, "created_time"),
					UpdatedTime:      getString(m, "updated_time"),
				})
			}

			return output.Print(result)
		},
	}

	cmd.Flags().StringVarP(&campaignID, "campaign-id", "c", "", "Filter by campaign ID")
	cmd.Flags().StringVarP(&status, "status", "s", "", "Filter by status: ACTIVE, PAUSED, ARCHIVED")
	cmd.Flags().IntVarP(&limit, "limit", "l", 25, "Number of ad sets to return")

	return cmd
}

func newAdSetCreateCmd() *cobra.Command {
	var name, campaignID, billingEvent, optimizationGoal, status string

	cmd := &cobra.Command{
		Use:   "adset-create",
		Short: "Create a new ad set",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			if name == "" {
				return output.PrintError("missing_flag", "--name is required", nil)
			}
			if campaignID == "" {
				return output.PrintError("missing_flag", "--campaign-id is required", nil)
			}
			if billingEvent == "" {
				return output.PrintError("missing_flag", "--billing-event is required (IMPRESSIONS, LINK_CLICKS, etc.)", nil)
			}
			if optimizationGoal == "" {
				return output.PrintError("missing_flag", "--optimization-goal is required (REACH, LINK_CLICKS, IMPRESSIONS, etc.)", nil)
			}

			payload := map[string]string{
				"name":              name,
				"campaign_id":       campaignID,
				"billing_event":     strings.ToUpper(billingEvent),
				"optimization_goal": strings.ToUpper(optimizationGoal),
				"status":            strings.ToUpper(status),
			}

			raw, err := c.doPost(c.actID()+"/adsets", payload)
			if err != nil {
				return output.PrintError("create_failed", err.Error(), nil)
			}

			return output.Print(map[string]string{
				"id":     getString(raw, "id"),
				"status": "created",
			})
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Ad set name (required)")
	cmd.Flags().StringVar(&campaignID, "campaign-id", "", "Campaign ID (required)")
	cmd.Flags().StringVar(&billingEvent, "billing-event", "", "Billing event: IMPRESSIONS, LINK_CLICKS, etc. (required)")
	cmd.Flags().StringVar(&optimizationGoal, "optimization-goal", "", "Optimization goal: REACH, LINK_CLICKS, IMPRESSIONS, etc. (required)")
	cmd.Flags().StringVar(&status, "status", "PAUSED", "Initial status: ACTIVE or PAUSED")

	return cmd
}

// --- ads ---

func newAdsCmd() *cobra.Command {
	var adsetID, status string
	var limit int

	cmd := &cobra.Command{
		Use:   "ads",
		Short: "List ads",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			params := url.Values{}
			params.Set("fields", "id,name,adset_id,campaign_id,status,created_time,updated_time")
			params.Set("limit", fmt.Sprintf("%d", limit))
			if status != "" {
				params.Set("effective_status", fmt.Sprintf(`["%s"]`, strings.ToUpper(status)))
			}

			endpoint := c.actID() + "/ads"
			if adsetID != "" {
				endpoint = adsetID + "/ads"
			}

			raw, err := c.doGet(endpoint, params)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			data := getDataArray(raw)
			result := make([]Ad, 0, len(data))
			for _, item := range data {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				result = append(result, Ad{
					ID:          getString(m, "id"),
					Name:        getString(m, "name"),
					AdSetID:     getString(m, "adset_id"),
					CampaignID:  getString(m, "campaign_id"),
					Status:      getString(m, "status"),
					CreatedTime: getString(m, "created_time"),
					UpdatedTime: getString(m, "updated_time"),
				})
			}

			return output.Print(result)
		},
	}

	cmd.Flags().StringVar(&adsetID, "adset-id", "", "Filter by ad set ID")
	cmd.Flags().StringVarP(&status, "status", "s", "", "Filter by status: ACTIVE, PAUSED, ARCHIVED")
	cmd.Flags().IntVarP(&limit, "limit", "l", 25, "Number of ads to return")

	return cmd
}

// --- insights ---

func newInsightsCmd() *cobra.Command {
	var objectID, dateStart, dateStop, level, fields string

	cmd := &cobra.Command{
		Use:   "insights",
		Short: "Get ad performance insights",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			endpoint := c.actID() + "/insights"
			if objectID != "" {
				endpoint = objectID + "/insights"
			}

			params := url.Values{}
			if fields != "" {
				params.Set("fields", fields)
			} else {
				params.Set("fields", "date_start,date_stop,impressions,clicks,spend,reach,ctr,cpc,cpm,actions")
			}
			if dateStart != "" {
				params.Set("time_range", fmt.Sprintf(`{"since":"%s","until":"%s"}`, dateStart, dateStop))
			}
			if level != "" {
				params.Set("level", level)
			}

			raw, err := c.doGet(endpoint, params)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			data := getDataArray(raw)
			result := make([]Insight, 0, len(data))
			for _, item := range data {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				insight := Insight{
					DateStart:   getString(m, "date_start"),
					DateStop:    getString(m, "date_stop"),
					Impressions: getString(m, "impressions"),
					Clicks:      getString(m, "clicks"),
					Spend:       getString(m, "spend"),
					Reach:       getString(m, "reach"),
					CTR:         getString(m, "ctr"),
					CPC:         getString(m, "cpc"),
					CPM:         getString(m, "cpm"),
				}
				if actions, ok := m["actions"].([]any); ok {
					insight.Actions = actions
				}
				result = append(result, insight)
			}

			return output.Print(result)
		},
	}

	cmd.Flags().StringVar(&objectID, "object-id", "", "Campaign, ad set, or ad ID (default: account level)")
	cmd.Flags().StringVar(&dateStart, "date-start", "", "Start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&dateStop, "date-stop", "", "End date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&level, "level", "", "Aggregation level: account, campaign, adset, ad")
	cmd.Flags().StringVarP(&fields, "fields", "f", "", "Comma-separated fields to return")

	return cmd
}

// --- helpers ---

func getDataArray(resp map[string]any) []any {
	if data, ok := resp["data"].([]any); ok {
		return data
	}
	return nil
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
