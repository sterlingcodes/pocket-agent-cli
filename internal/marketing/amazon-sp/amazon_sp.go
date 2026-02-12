package amazonsp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

var baseURL = "https://sellingpartnerapi-na.amazon.com"

var tokenURL = "https://api.amazon.com/auth/o2/token" //nolint:gosec // OAuth endpoint URL, not a credential

var httpClient = &http.Client{Timeout: 30 * time.Second}

var regionBaseURLs = map[string]string{
	"na": "https://sellingpartnerapi-na.amazon.com",
	"eu": "https://sellingpartnerapi-eu.amazon.com",
	"fe": "https://sellingpartnerapi-fe.amazon.com",
}

// spClient holds credentials for Amazon SP-API calls.
type spClient struct {
	clientID     string
	clientSecret string
	refreshToken string
	sellerID     string
	accessToken  string
	tokenExpiry  time.Time
	apiBaseURL   string
}

// Order represents an Amazon order.
type Order struct {
	AmazonOrderID    string `json:"amazon_order_id"`
	PurchaseDate     string `json:"purchase_date"`
	OrderStatus      string `json:"order_status"`
	OrderTotal       *Money `json:"order_total,omitempty"`
	NumberOfItems    int    `json:"number_of_items_shipped"`
	MarketplaceID    string `json:"marketplace_id"`
	FulfillmentChan  string `json:"fulfillment_channel,omitempty"`
	PaymentMethod    string `json:"payment_method,omitempty"`
	ShipServiceLevel string `json:"ship_service_level,omitempty"`
}

// Money represents a currency amount.
type Money struct {
	CurrencyCode string `json:"currency_code"`
	Amount       string `json:"amount"`
}

// OrderItem represents an item within an order.
type OrderItem struct {
	ASIN              string `json:"asin"`
	SellerSKU         string `json:"seller_sku,omitempty"`
	OrderItemID       string `json:"order_item_id"`
	Title             string `json:"title"`
	QuantityOrdered   int    `json:"quantity_ordered"`
	QuantityShipped   int    `json:"quantity_shipped"`
	ItemPrice         *Money `json:"item_price,omitempty"`
	PromotionDiscount *Money `json:"promotion_discount,omitempty"`
}

// InventorySummary represents FBA inventory.
type InventorySummary struct {
	ASIN              string `json:"asin"`
	FNSKU             string `json:"fn_sku"`
	SellerSKU         string `json:"seller_sku"`
	ProductName       string `json:"product_name"`
	TotalQuantity     int    `json:"total_quantity"`
	FulfillableQty    int    `json:"fulfillable_quantity"`
	InboundWorkingQty int    `json:"inbound_working_quantity"`
	InboundShippedQty int    `json:"inbound_shipped_quantity"`
}

// ReportInfo represents a report.
type ReportInfo struct {
	ReportID       string `json:"report_id"`
	ReportType     string `json:"report_type"`
	ProcessingStat string `json:"processing_status"`
	ReportDocID    string `json:"report_document_id,omitempty"`
	CreatedTime    string `json:"created_time"`
}

// NewCmd returns the amazon-sp parent command.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "amazon-sp",
		Aliases: []string{"amz", "sp-api"},
		Short:   "Amazon Selling Partner API commands",
	}

	cmd.AddCommand(newOrdersCmd())
	cmd.AddCommand(newOrderCmd())
	cmd.AddCommand(newOrderItemsCmd())
	cmd.AddCommand(newInventoryCmd())
	cmd.AddCommand(newReportCreateCmd())
	cmd.AddCommand(newReportStatusCmd())

	return cmd
}

func newClient() (*spClient, error) {
	clientID, err := config.Get("amazon_sp_client_id")
	if err != nil {
		return nil, err
	}
	if clientID == "" {
		return nil, output.PrintError("missing_config", "Amazon SP-API client ID not configured", map[string]string{
			"setup": "Run: pocket setup show amazon-sp",
		})
	}

	clientSecret, err := config.Get("amazon_sp_client_secret")
	if err != nil {
		return nil, err
	}
	if clientSecret == "" {
		return nil, output.PrintError("missing_config", "Amazon SP-API client secret not configured", map[string]string{
			"setup": "Run: pocket setup show amazon-sp",
		})
	}

	refreshToken, err := config.Get("amazon_sp_refresh_token")
	if err != nil {
		return nil, err
	}
	if refreshToken == "" {
		return nil, output.PrintError("missing_config", "Amazon SP-API refresh token not configured", map[string]string{
			"setup": "Run: pocket setup show amazon-sp",
		})
	}

	sellerID, err := config.Get("amazon_sp_seller_id")
	if err != nil {
		return nil, err
	}
	if sellerID == "" {
		return nil, output.PrintError("missing_config", "Amazon SP-API seller ID not configured", map[string]string{
			"setup": "Run: pocket setup show amazon-sp",
		})
	}

	// Resolve region-specific base URL without mutating package-level var
	resolvedURL := baseURL
	region, _ := config.Get("amazon_sp_region")
	if u, ok := regionBaseURLs[region]; ok {
		resolvedURL = u
	}

	c := &spClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		refreshToken: refreshToken,
		sellerID:     sellerID,
		apiBaseURL:   resolvedURL,
	}

	// Load cached token
	cachedToken, _ := config.Get("amazon_sp_access_token")
	expiryStr, _ := config.Get("amazon_sp_token_expiry")
	if cachedToken != "" && expiryStr != "" {
		expiry, parseErr := time.Parse(time.RFC3339, expiryStr)
		if parseErr == nil {
			c.accessToken = cachedToken
			c.tokenExpiry = expiry
		}
	}

	return c, nil
}

func (c *spClient) ensureAccessToken() error {
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return nil
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {c.refreshToken},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("token refresh failed (HTTP %d)", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	// Cache token
	_ = config.Set("amazon_sp_access_token", c.accessToken)
	_ = config.Set("amazon_sp_token_expiry", c.tokenExpiry.Format(time.RFC3339))

	return nil
}

func (c *spClient) doGet(endpoint string, params url.Values) (map[string]any, error) {
	if err := c.ensureAccessToken(); err != nil {
		return nil, err
	}

	if params == nil {
		params = url.Values{}
	}

	reqURL := fmt.Sprintf("%s%s?%s", c.apiBaseURL, endpoint, params.Encode())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-amz-access-token", c.accessToken)
	req.Header.Set("User-Agent", "Pocket-CLI/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]any
		if decErr := json.NewDecoder(resp.Body).Decode(&errResp); decErr == nil {
			if apiErr := parseHTTPError(errResp); apiErr != nil {
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

func (c *spClient) doPost(endpoint string, payload any) (map[string]any, error) {
	if err := c.ensureAccessToken(); err != nil {
		return nil, err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("%s%s", c.apiBaseURL, endpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-amz-access-token", c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Pocket-CLI/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]any
		if decErr := json.NewDecoder(resp.Body).Decode(&errResp); decErr == nil {
			if apiErr := parseHTTPError(errResp); apiErr != nil {
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

func parseHTTPError(resp map[string]any) error {
	// SP-API returns {"errors": [{"code": "...", "message": "..."}]}
	errArr, ok := resp["errors"].([]any)
	if !ok || len(errArr) == 0 {
		return nil
	}
	firstErr, ok := errArr[0].(map[string]any)
	if !ok {
		return nil
	}
	code := getString(firstErr, "code")
	msg := getString(firstErr, "message")
	return fmt.Errorf("SP-API error (code=%s): %s", code, msg)
}

// --- orders ---

func newOrdersCmd() *cobra.Command {
	var status, after, before, marketplace string
	var limit int

	cmd := &cobra.Command{
		Use:   "orders",
		Short: "List orders",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			params := url.Values{}
			params.Set("MarketplaceIds", marketplace)
			if status != "" {
				params.Set("OrderStatuses", status)
			}
			if after != "" {
				params.Set("CreatedAfter", after)
			}
			if before != "" {
				params.Set("CreatedBefore", before)
			}
			if limit > 0 {
				params.Set("MaxResultsPerPage", fmt.Sprintf("%d", limit))
			}

			raw, err := c.doGet("/orders/v0/orders", params)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			orders := extractOrders(raw)
			return output.Print(orders)
		},
	}

	cmd.Flags().StringVarP(&status, "status", "s", "", "Filter by status: Pending, Unshipped, Shipped, Canceled")
	cmd.Flags().StringVar(&after, "after", "", "Created after date (ISO 8601)")
	cmd.Flags().StringVar(&before, "before", "", "Created before date (ISO 8601)")
	cmd.Flags().StringVar(&marketplace, "marketplace", "ATVPDKIKX0DER", "Marketplace ID (default: US)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Max orders to return")

	return cmd
}

func newOrderCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "order [order-id]",
		Short: "Get order details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			raw, err := c.doGet("/orders/v0/orders/"+args[0], nil)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			payload, _ := raw["payload"].(map[string]any)
			if payload == nil {
				return output.Print(raw)
			}

			order := mapOrder(payload)
			return output.Print(order)
		},
	}
}

func newOrderItemsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "order-items [order-id]",
		Short: "Get items for an order",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			raw, err := c.doGet("/orders/v0/orders/"+args[0]+"/orderItems", nil)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			payload, _ := raw["payload"].(map[string]any)
			if payload == nil {
				return output.Print(raw)
			}

			items := extractOrderItems(payload)
			return output.Print(items)
		},
	}
}

// --- inventory ---

func newInventoryCmd() *cobra.Command {
	var sku, marketplace string
	var limit int

	cmd := &cobra.Command{
		Use:   "inventory",
		Short: "List FBA inventory summaries",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			params := url.Values{}
			params.Set("details", "true")
			params.Set("granularityType", "Marketplace")
			params.Set("granularityId", marketplace)
			params.Set("marketplaceIds", marketplace)
			if sku != "" {
				params.Set("sellerSkus", sku)
			}
			if limit > 0 {
				params.Set("maxResults", fmt.Sprintf("%d", limit))
			}

			raw, err := c.doGet("/fba/inventory/v1/summaries", params)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			summaries := extractInventory(raw)
			return output.Print(summaries)
		},
	}

	cmd.Flags().StringVar(&sku, "sku", "", "Filter by seller SKU")
	cmd.Flags().StringVar(&marketplace, "marketplace", "ATVPDKIKX0DER", "Marketplace ID (default: US)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Max items to return")

	return cmd
}

// --- reports ---

func newReportCreateCmd() *cobra.Command {
	var reportType, startTime, endTime, marketplace string

	cmd := &cobra.Command{
		Use:   "report-create",
		Short: "Create a report request",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			if reportType == "" {
				return output.PrintError("missing_flag", "--type is required (e.g., GET_FLAT_FILE_ALL_ORDERS_DATA_BY_ORDER_DATE_GENERAL)", nil)
			}

			payload := map[string]any{
				"reportType":     reportType,
				"marketplaceIds": []string{marketplace},
			}
			if startTime != "" {
				payload["dataStartTime"] = startTime
			}
			if endTime != "" {
				payload["dataEndTime"] = endTime
			}

			raw, err := c.doPost("/reports/2021-06-30/reports", payload)
			if err != nil {
				return output.PrintError("create_failed", err.Error(), nil)
			}

			return output.Print(map[string]string{
				"report_id": getString(raw, "reportId"),
				"status":    "created",
			})
		},
	}

	cmd.Flags().StringVar(&reportType, "type", "", "Report type (required)")
	cmd.Flags().StringVar(&startTime, "start", "", "Data start time (ISO 8601)")
	cmd.Flags().StringVar(&endTime, "end", "", "Data end time (ISO 8601)")
	cmd.Flags().StringVar(&marketplace, "marketplace", "ATVPDKIKX0DER", "Marketplace ID (default: US)")

	return cmd
}

func newReportStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "report-status [report-id]",
		Short: "Get report processing status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			raw, err := c.doGet("/reports/2021-06-30/reports/"+args[0], nil)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			report := ReportInfo{
				ReportID:       getString(raw, "reportId"),
				ReportType:     getString(raw, "reportType"),
				ProcessingStat: getString(raw, "processingStatus"),
				ReportDocID:    getString(raw, "reportDocumentId"),
				CreatedTime:    getString(raw, "createdTime"),
			}

			return output.Print(report)
		},
	}
}

// --- extraction helpers ---

func extractOrders(raw map[string]any) []Order {
	payload, _ := raw["payload"].(map[string]any)
	if payload == nil {
		return nil
	}
	ordersArr, _ := payload["Orders"].([]any)
	result := make([]Order, 0, len(ordersArr))
	for _, item := range ordersArr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, mapOrder(m))
	}
	return result
}

func mapOrder(m map[string]any) Order {
	o := Order{
		AmazonOrderID:    getString(m, "AmazonOrderId"),
		PurchaseDate:     getString(m, "PurchaseDate"),
		OrderStatus:      getString(m, "OrderStatus"),
		NumberOfItems:    getInt(m, "NumberOfItemsShipped"),
		MarketplaceID:    getString(m, "MarketplaceId"),
		FulfillmentChan:  getString(m, "FulfillmentChannel"),
		PaymentMethod:    getString(m, "PaymentMethod"),
		ShipServiceLevel: getString(m, "ShipServiceLevel"),
	}
	o.OrderTotal = getMoneyField(m, "OrderTotal")
	return o
}

func extractOrderItems(payload map[string]any) []OrderItem {
	itemsArr, _ := payload["OrderItems"].([]any)
	result := make([]OrderItem, 0, len(itemsArr))
	for _, item := range itemsArr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, OrderItem{
			ASIN:              getString(m, "ASIN"),
			SellerSKU:         getString(m, "SellerSKU"),
			OrderItemID:       getString(m, "OrderItemId"),
			Title:             getString(m, "Title"),
			QuantityOrdered:   getInt(m, "QuantityOrdered"),
			QuantityShipped:   getInt(m, "QuantityShipped"),
			ItemPrice:         getMoneyField(m, "ItemPrice"),
			PromotionDiscount: getMoneyField(m, "PromotionDiscount"),
		})
	}
	return result
}

func extractInventory(raw map[string]any) []InventorySummary {
	payload, _ := raw["payload"].(map[string]any)
	if payload == nil {
		return nil
	}
	summaries, _ := payload["inventorySummaries"].([]any)
	result := make([]InventorySummary, 0, len(summaries))
	for _, item := range summaries {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, InventorySummary{
			ASIN:              getString(m, "asin"),
			FNSKU:             getString(m, "fnSku"),
			SellerSKU:         getString(m, "sellerSku"),
			ProductName:       getString(m, "productName"),
			TotalQuantity:     getInt(m, "totalQuantity"),
			FulfillableQty:    getInt(m, "fulfillableQuantity"),
			InboundWorkingQty: getInt(m, "inboundWorkingQuantity"),
			InboundShippedQty: getInt(m, "inboundShippedQuantity"),
		})
	}
	return result
}

// --- helpers ---

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

func getMoneyField(m map[string]any, key string) *Money {
	obj, ok := m[key].(map[string]any)
	if !ok {
		return nil
	}
	return &Money{
		CurrencyCode: getString(obj, "CurrencyCode"),
		Amount:       getString(obj, "Amount"),
	}
}
