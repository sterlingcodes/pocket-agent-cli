package shopify

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

var apiVersion = "2026-01"

var baseURL = "" // computed from store name; empty for test override

var httpClient = &http.Client{Timeout: 30 * time.Second}

// shopClient holds credentials for Shopify Admin API calls.
type shopClient struct {
	store      string
	token      string
	apiBaseURL string
}

// Shop represents store info.
type Shop struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Domain       string `json:"domain"`
	MyshopifyDom string `json:"myshopify_domain"`
	PlanName     string `json:"plan_name"`
	Currency     string `json:"currency"`
	Timezone     string `json:"timezone"`
	Country      string `json:"country_name"`
	CreatedAt    string `json:"created_at"`
}

// ShopifyOrder represents an order.
type ShopifyOrder struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	TotalPrice      string `json:"total_price"`
	Currency        string `json:"currency"`
	FinancialStatus string `json:"financial_status"`
	FulfillStatus   string `json:"fulfillment_status"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	OrderNumber     int    `json:"order_number"`
}

// Product represents a product.
type Product struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Vendor      string `json:"vendor"`
	ProductType string `json:"product_type"`
	Status      string `json:"status"`
	Tags        string `json:"tags,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	VariantsCnt int    `json:"variants_count"`
}

// Customer represents a customer.
type Customer struct {
	ID           int64  `json:"id"`
	Email        string `json:"email"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	OrdersCount  int    `json:"orders_count"`
	TotalSpent   string `json:"total_spent"`
	State        string `json:"state"`
	CreatedAt    string `json:"created_at"`
	VerifiedMail bool   `json:"verified_email"`
}

// InventoryLevel represents inventory at a location.
type InventoryLevel struct {
	InventoryItemID int64  `json:"inventory_item_id"`
	LocationID      int64  `json:"location_id"`
	Available       int    `json:"available"`
	UpdatedAt       string `json:"updated_at"`
}

// NewCmd returns the shopify parent command.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "shopify",
		Aliases: []string{"shop"},
		Short:   "Shopify Admin API commands",
	}

	cmd.AddCommand(newShopCmd())
	cmd.AddCommand(newOrdersCmd())
	cmd.AddCommand(newOrderCmd())
	cmd.AddCommand(newProductsCmd())
	cmd.AddCommand(newProductCmd())
	cmd.AddCommand(newCustomersCmd())
	cmd.AddCommand(newCustomerSearchCmd())
	cmd.AddCommand(newInventoryCmd())
	cmd.AddCommand(newInventorySetCmd())

	return cmd
}

func newClient() (*shopClient, error) {
	store, err := config.Get("shopify_store")
	if err != nil {
		return nil, err
	}
	if store == "" {
		return nil, output.PrintError("missing_config", "Shopify store not configured", map[string]string{
			"setup": "Run: pocket setup show shopify",
		})
	}

	token, err := config.Get("shopify_token")
	if err != nil {
		return nil, err
	}
	if token == "" {
		return nil, output.PrintError("missing_config", "Shopify access token not configured", map[string]string{
			"setup": "Run: pocket setup show shopify",
		})
	}

	// Resolve base URL without mutating package-level var
	resolvedURL := baseURL
	if resolvedURL == "" {
		resolvedURL = fmt.Sprintf("https://%s.myshopify.com/admin/api/%s", store, apiVersion)
	}

	return &shopClient{store: store, token: token, apiBaseURL: resolvedURL}, nil
}

func (c *shopClient) doGet(endpoint string, params url.Values) (map[string]any, error) {
	if params == nil {
		params = url.Values{}
	}

	reqURL := fmt.Sprintf("%s/%s?%s", c.apiBaseURL, endpoint, params.Encode())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Shopify-Access-Token", c.token)
	req.Header.Set("User-Agent", "Pocket-CLI/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]any
		if decErr := json.NewDecoder(resp.Body).Decode(&errResp); decErr == nil {
			if apiErr := parseShopifyError(errResp, resp.StatusCode); apiErr != nil {
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

func (c *shopClient) doPost(endpoint string, payload any) (map[string]any, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("%s/%s", c.apiBaseURL, endpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Shopify-Access-Token", c.token)
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
			if apiErr := parseShopifyError(errResp, resp.StatusCode); apiErr != nil {
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

func (c *shopClient) doPut(endpoint string, payload any) (map[string]any, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("%s/%s", c.apiBaseURL, endpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "PUT", reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Shopify-Access-Token", c.token)
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
			if apiErr := parseShopifyError(errResp, resp.StatusCode); apiErr != nil {
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

func (c *shopClient) doDelete(endpoint string) error {
	reqURL := fmt.Sprintf("%s/%s", c.apiBaseURL, endpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "DELETE", reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Shopify-Access-Token", c.token)
	req.Header.Set("User-Agent", "Pocket-CLI/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]any
		if decErr := json.NewDecoder(resp.Body).Decode(&errResp); decErr == nil {
			if apiErr := parseShopifyError(errResp, resp.StatusCode); apiErr != nil {
				return apiErr
			}
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return nil
}

func parseShopifyError(resp map[string]any, statusCode int) error {
	// Shopify errors can be string or object
	if errStr, ok := resp["errors"].(string); ok {
		return fmt.Errorf("Shopify error (HTTP %d): %s", statusCode, errStr)
	}
	if errObj, ok := resp["errors"].(map[string]any); ok {
		data, _ := json.Marshal(errObj)
		return fmt.Errorf("Shopify error (HTTP %d): %s", statusCode, string(data))
	}
	return nil
}

// --- shop ---

func newShopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "shop",
		Short: "Get store info",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			raw, err := c.doGet("shop.json", nil)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			shopData, _ := raw["shop"].(map[string]any)
			if shopData == nil {
				return output.Print(raw)
			}

			shop := Shop{
				ID:           getInt64(shopData, "id"),
				Name:         getString(shopData, "name"),
				Email:        getString(shopData, "email"),
				Domain:       getString(shopData, "domain"),
				MyshopifyDom: getString(shopData, "myshopify_domain"),
				PlanName:     getString(shopData, "plan_name"),
				Currency:     getString(shopData, "currency"),
				Timezone:     getString(shopData, "timezone"),
				Country:      getString(shopData, "country_name"),
				CreatedAt:    getString(shopData, "created_at"),
			}

			return output.Print(shop)
		},
	}
}

// --- orders ---

func newOrdersCmd() *cobra.Command {
	var limit int
	var status, since, financial string

	cmd := &cobra.Command{
		Use:   "orders",
		Short: "List orders",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			params := url.Values{}
			params.Set("limit", fmt.Sprintf("%d", limit))
			if status != "" {
				params.Set("status", status)
			}
			if since != "" {
				params.Set("created_at_min", since)
			}
			if financial != "" {
				params.Set("financial_status", financial)
			}

			raw, err := c.doGet("orders.json", params)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			orders := extractOrders(raw)
			return output.Print(orders)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Number of orders to return")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status: open, closed, cancelled, any")
	cmd.Flags().StringVar(&since, "since", "", "Created after date (ISO 8601)")
	cmd.Flags().StringVar(&financial, "financial", "", "Filter: paid, pending, refunded, etc.")

	return cmd
}

func newOrderCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "order [id]",
		Short: "Get order details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			raw, err := c.doGet("orders/"+args[0]+".json", nil)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			orderData, _ := raw["order"].(map[string]any)
			if orderData == nil {
				return output.Print(raw)
			}

			order := mapOrder(orderData)
			return output.Print(order)
		},
	}
}

// --- products ---

func newProductsCmd() *cobra.Command {
	var limit int
	var status, vendor, collection string

	cmd := &cobra.Command{
		Use:   "products",
		Short: "List products",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			params := url.Values{}
			params.Set("limit", fmt.Sprintf("%d", limit))
			if status != "" {
				params.Set("status", status)
			}
			if vendor != "" {
				params.Set("vendor", vendor)
			}
			if collection != "" {
				params.Set("collection_id", collection)
			}

			raw, err := c.doGet("products.json", params)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			products := extractProducts(raw)
			return output.Print(products)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Number of products to return")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status: active, archived, draft")
	cmd.Flags().StringVar(&vendor, "vendor", "", "Filter by vendor name")
	cmd.Flags().StringVar(&collection, "collection", "", "Filter by collection ID")

	return cmd
}

func newProductCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "product [id]",
		Short: "Get product details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			raw, err := c.doGet("products/"+args[0]+".json", nil)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			prodData, _ := raw["product"].(map[string]any)
			if prodData == nil {
				return output.Print(raw)
			}

			product := mapProduct(prodData)
			return output.Print(product)
		},
	}
}

// --- customers ---

func newCustomersCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "customers",
		Short: "List customers",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			params := url.Values{}
			params.Set("limit", fmt.Sprintf("%d", limit))

			raw, err := c.doGet("customers.json", params)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			customers := extractCustomers(raw)
			return output.Print(customers)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Number of customers to return")

	return cmd
}

func newCustomerSearchCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "customer-search [query]",
		Short: "Search customers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			params := url.Values{}
			params.Set("query", args[0])
			params.Set("limit", fmt.Sprintf("%d", limit))

			raw, err := c.doGet("customers/search.json", params)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			customers := extractCustomers(raw)
			return output.Print(customers)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Number of results to return")

	return cmd
}

// --- inventory ---

func newInventoryCmd() *cobra.Command {
	var locationID string
	var limit int

	cmd := &cobra.Command{
		Use:   "inventory",
		Short: "List inventory levels at a location",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			if locationID == "" {
				return output.PrintError("missing_flag", "--location is required", nil)
			}

			params := url.Values{}
			params.Set("location_ids", locationID)
			params.Set("limit", fmt.Sprintf("%d", limit))

			raw, err := c.doGet("inventory_levels.json", params)
			if err != nil {
				return output.PrintError("fetch_failed", err.Error(), nil)
			}

			levels := extractInventoryLevels(raw)
			return output.Print(levels)
		},
	}

	cmd.Flags().StringVar(&locationID, "location", "", "Location ID (required)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Number of items to return")

	return cmd
}

func newInventorySetCmd() *cobra.Command {
	var itemID, locationID string
	var available int

	cmd := &cobra.Command{
		Use:   "inventory-set",
		Short: "Set inventory level for an item at a location",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			if itemID == "" {
				return output.PrintError("missing_flag", "--item is required (inventory_item_id)", nil)
			}
			if locationID == "" {
				return output.PrintError("missing_flag", "--location is required (location_id)", nil)
			}
			if !cmd.Flags().Changed("available") {
				return output.PrintError("missing_flag", "--available is required", nil)
			}

			payload := map[string]any{
				"location_id":       locationID,
				"inventory_item_id": itemID,
				"available":         available,
			}

			raw, err := c.doPost("inventory_levels/set.json", payload)
			if err != nil {
				return output.PrintError("set_failed", err.Error(), nil)
			}

			level, _ := raw["inventory_level"].(map[string]any)
			if level == nil {
				return output.Print(raw)
			}

			return output.Print(InventoryLevel{
				InventoryItemID: getInt64(level, "inventory_item_id"),
				LocationID:      getInt64(level, "location_id"),
				Available:       getInt(level, "available"),
				UpdatedAt:       getString(level, "updated_at"),
			})
		},
	}

	cmd.Flags().StringVar(&itemID, "item", "", "Inventory item ID (required)")
	cmd.Flags().StringVar(&locationID, "location", "", "Location ID (required)")
	cmd.Flags().IntVar(&available, "available", 0, "Available quantity (required)")

	return cmd
}

// --- extraction helpers ---

func extractOrders(raw map[string]any) []ShopifyOrder {
	ordersArr, _ := raw["orders"].([]any)
	result := make([]ShopifyOrder, 0, len(ordersArr))
	for _, item := range ordersArr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, mapOrder(m))
	}
	return result
}

func mapOrder(m map[string]any) ShopifyOrder {
	return ShopifyOrder{
		ID:              getInt64(m, "id"),
		Name:            getString(m, "name"),
		Email:           getString(m, "email"),
		TotalPrice:      getString(m, "total_price"),
		Currency:        getString(m, "currency"),
		FinancialStatus: getString(m, "financial_status"),
		FulfillStatus:   getString(m, "fulfillment_status"),
		CreatedAt:       getString(m, "created_at"),
		UpdatedAt:       getString(m, "updated_at"),
		OrderNumber:     getInt(m, "order_number"),
	}
}

func extractProducts(raw map[string]any) []Product {
	prodsArr, _ := raw["products"].([]any)
	result := make([]Product, 0, len(prodsArr))
	for _, item := range prodsArr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, mapProduct(m))
	}
	return result
}

func mapProduct(m map[string]any) Product {
	p := Product{
		ID:          getInt64(m, "id"),
		Title:       getString(m, "title"),
		Vendor:      getString(m, "vendor"),
		ProductType: getString(m, "product_type"),
		Status:      getString(m, "status"),
		Tags:        getString(m, "tags"),
		CreatedAt:   getString(m, "created_at"),
		UpdatedAt:   getString(m, "updated_at"),
	}
	if variants, ok := m["variants"].([]any); ok {
		p.VariantsCnt = len(variants)
	}
	return p
}

func extractCustomers(raw map[string]any) []Customer {
	custArr, _ := raw["customers"].([]any)
	result := make([]Customer, 0, len(custArr))
	for _, item := range custArr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, Customer{
			ID:           getInt64(m, "id"),
			Email:        getString(m, "email"),
			FirstName:    getString(m, "first_name"),
			LastName:     getString(m, "last_name"),
			OrdersCount:  getInt(m, "orders_count"),
			TotalSpent:   getString(m, "total_spent"),
			State:        getString(m, "state"),
			CreatedAt:    getString(m, "created_at"),
			VerifiedMail: getBool(m, "verified_email"),
		})
	}
	return result
}

func extractInventoryLevels(raw map[string]any) []InventoryLevel {
	levelsArr, _ := raw["inventory_levels"].([]any)
	result := make([]InventoryLevel, 0, len(levelsArr))
	for _, item := range levelsArr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, InventoryLevel{
			InventoryItemID: getInt64(m, "inventory_item_id"),
			LocationID:      getInt64(m, "location_id"),
			Available:       getInt(m, "available"),
			UpdatedAt:       getString(m, "updated_at"),
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

func getInt64(m map[string]any, key string) int64 {
	if v, ok := m[key].(float64); ok {
		return int64(v)
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
