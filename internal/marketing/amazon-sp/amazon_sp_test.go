package amazonsp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/unstablemind/pocket/internal/common/config"
)

var testConfigPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "amzsp-test-*")
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
	if err := os.WriteFile(testConfigPath, data, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func clearTestConfig(t *testing.T) {
	t.Helper()
	os.Remove(testConfigPath)
}

func TestNewCmdSubcommands(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "amazon-sp" {
		t.Errorf("expected Use=amazon-sp, got %s", cmd.Use)
	}

	want := map[string]bool{
		"orders":        false,
		"order":         false,
		"order-items":   false,
		"inventory":     false,
		"report-create": false,
		"report-status": false,
	}

	for _, sub := range cmd.Commands() {
		use := sub.Use
		if idx := strings.IndexByte(use, ' '); idx != -1 {
			use = use[:idx]
		}
		if _, ok := want[use]; ok {
			want[use] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q not found in NewCmd()", name)
		}
	}
}

func TestEnsureAccessToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", ct)
		}

		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		if !strings.Contains(bodyStr, "grant_type=refresh_token") {
			t.Errorf("body missing grant_type=refresh_token: %s", bodyStr)
		}
		if !strings.Contains(bodyStr, "client_id=test_cid") {
			t.Errorf("body missing client_id: %s", bodyStr)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "new_access_token",
			"expires_in":   float64(3600),
		})
	}))
	defer ts.Close()

	origTokenURL := tokenURL
	tokenURL = ts.URL
	defer func() { tokenURL = origTokenURL }()

	clearTestConfig(t)

	c := &spClient{
		clientID:     "test_cid",
		clientSecret: "test_secret",
		refreshToken: "test_refresh",
		sellerID:     "test_seller",
	}

	if err := c.ensureAccessToken(); err != nil {
		t.Fatalf("ensureAccessToken failed: %v", err)
	}

	if c.accessToken != "new_access_token" {
		t.Errorf("accessToken = %q, want %q", c.accessToken, "new_access_token")
	}

	if c.tokenExpiry.IsZero() {
		t.Error("tokenExpiry should not be zero")
	}
}

func TestEnsureAccessTokenCached(t *testing.T) {
	c := &spClient{
		clientID:     "test_cid",
		clientSecret: "test_secret",
		refreshToken: "test_refresh",
		sellerID:     "test_seller",
		accessToken:  "cached_token",
		tokenExpiry:  time.Now().Add(1 * time.Hour),
	}

	if err := c.ensureAccessToken(); err != nil {
		t.Fatalf("ensureAccessToken failed: %v", err)
	}

	if c.accessToken != "cached_token" {
		t.Errorf("accessToken = %q, want %q (should use cached)", c.accessToken, "cached_token")
	}
}

func TestDoGetSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("x-amz-access-token")
		if token != "test_access_token" {
			t.Errorf("x-amz-access-token = %q, want %q", token, "test_access_token")
		}
		if r.Method != "GET" {
			t.Errorf("method = %s, want GET", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"payload": map[string]any{
				"Orders": []any{
					map[string]any{"AmazonOrderId": "111-222-333"},
				},
			},
		})
	}))
	defer ts.Close()

	c := &spClient{
		accessToken: "test_access_token",
		tokenExpiry: time.Now().Add(1 * time.Hour),
		apiBaseURL:  ts.URL,
	}

	result, err := c.doGet("/orders/v0/orders", nil)
	if err != nil {
		t.Fatalf("doGet failed: %v", err)
	}

	payload, _ := result["payload"].(map[string]any)
	if payload == nil {
		t.Fatal("expected payload in response")
	}
}

func TestDoGetHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []any{
				map[string]any{
					"code":    "InvalidInput",
					"message": "Invalid marketplace ID",
				},
			},
		})
	}))
	defer ts.Close()

	c := &spClient{
		accessToken: "tok",
		tokenExpiry: time.Now().Add(1 * time.Hour),
		apiBaseURL:  ts.URL,
	}

	_, err := c.doGet("/orders/v0/orders", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "InvalidInput") {
		t.Errorf("error %q should contain 'InvalidInput'", err.Error())
	}
	if !strings.Contains(err.Error(), "Invalid marketplace ID") {
		t.Errorf("error %q should contain 'Invalid marketplace ID'", err.Error())
	}
}

func TestDoPostSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("x-amz-access-token")
		if token != "post_token" {
			t.Errorf("x-amz-access-token = %q, want %q", token, "post_token")
		}
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "reportType") {
			t.Errorf("body %q should contain 'reportType'", string(body))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"reportId": "RPT-123"})
	}))
	defer ts.Close()

	c := &spClient{
		accessToken: "post_token",
		tokenExpiry: time.Now().Add(1 * time.Hour),
		apiBaseURL:  ts.URL,
	}

	result, err := c.doPost("/reports/2021-06-30/reports", map[string]any{
		"reportType":     "GET_FLAT_FILE_ALL_ORDERS_DATA_BY_ORDER_DATE_GENERAL",
		"marketplaceIds": []string{"ATVPDKIKX0DER"},
	})
	if err != nil {
		t.Fatalf("doPost failed: %v", err)
	}

	if getString(result, "reportId") != "RPT-123" {
		t.Errorf("reportId = %q, want %q", getString(result, "reportId"), "RPT-123")
	}
}

func TestNewClientMissingKeys(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]string
	}{
		{"missing client_id", map[string]string{}},
		{"missing client_secret", map[string]string{
			"amazon_sp_client_id": "cid",
		}},
		{"missing refresh_token", map[string]string{
			"amazon_sp_client_id":     "cid",
			"amazon_sp_client_secret": "secret",
		}},
		{"missing seller_id", map[string]string{
			"amazon_sp_client_id":     "cid",
			"amazon_sp_client_secret": "secret",
			"amazon_sp_refresh_token": "refresh",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeTestConfig(t, tt.config)
			_, err := newClient()
			if err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}
		})
	}
}

func TestRegionBaseURL(t *testing.T) {
	tests := []struct {
		region string
		want   string
	}{
		{"na", "https://sellingpartnerapi-na.amazon.com"},
		{"eu", "https://sellingpartnerapi-eu.amazon.com"},
		{"fe", "https://sellingpartnerapi-fe.amazon.com"},
	}

	for _, tt := range tests {
		got, ok := regionBaseURLs[tt.region]
		if !ok {
			t.Errorf("region %q not found in regionBaseURLs", tt.region)
			continue
		}
		if got != tt.want {
			t.Errorf("regionBaseURLs[%q] = %q, want %q", tt.region, got, tt.want)
		}
	}
}

func TestHelpers(t *testing.T) {
	m := map[string]any{
		"name":  "test",
		"count": float64(42),
		"OrderTotal": map[string]any{
			"CurrencyCode": "USD",
			"Amount":       "29.99",
		},
	}

	if got := getString(m, "name"); got != "test" {
		t.Errorf("getString(name) = %q, want %q", got, "test")
	}
	if got := getString(m, "missing"); got != "" {
		t.Errorf("getString(missing) = %q, want empty", got)
	}

	if got := getInt(m, "count"); got != 42 {
		t.Errorf("getInt(count) = %d, want 42", got)
	}
	if got := getInt(m, "missing"); got != 0 {
		t.Errorf("getInt(missing) = %d, want 0", got)
	}

	money := getMoneyField(m, "OrderTotal")
	if money == nil {
		t.Fatal("getMoneyField(OrderTotal) returned nil")
	}
	if money.CurrencyCode != "USD" {
		t.Errorf("money.CurrencyCode = %q, want %q", money.CurrencyCode, "USD")
	}
	if money.Amount != "29.99" {
		t.Errorf("money.Amount = %q, want %q", money.Amount, "29.99")
	}

	if got := getMoneyField(m, "missing"); got != nil {
		t.Errorf("getMoneyField(missing) = %v, want nil", got)
	}
}

func TestParseHTTPError(t *testing.T) {
	resp := map[string]any{
		"errors": []any{
			map[string]any{
				"code":    "InvalidInput",
				"message": "Bad request parameter",
			},
		},
	}

	err := parseHTTPError(resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	msg := err.Error()
	if !strings.Contains(msg, "InvalidInput") {
		t.Errorf("error %q should contain 'InvalidInput'", msg)
	}
	if !strings.Contains(msg, "Bad request parameter") {
		t.Errorf("error %q should contain 'Bad request parameter'", msg)
	}

	// No errors key
	noErr := map[string]any{"data": "ok"}
	if got := parseHTTPError(noErr); got != nil {
		t.Errorf("expected nil for no errors key, got %v", got)
	}

	// Empty errors array
	emptyErr := map[string]any{"errors": []any{}}
	if got := parseHTTPError(emptyErr); got != nil {
		t.Errorf("expected nil for empty errors array, got %v", got)
	}
}

func TestAmazonSPConfigKeys(t *testing.T) {
	clearTestConfig(t)

	keys := map[string]string{
		"amazon_sp_client_id":     "test_cid",
		"amazon_sp_client_secret": "test_secret",
		"amazon_sp_refresh_token": "test_refresh",
		"amazon_sp_seller_id":     "test_seller",
		"amazon_sp_region":        "na",
		"amazon_sp_access_token":  "test_access",
		"amazon_sp_token_expiry":  "2026-01-01T00:00:00Z",
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
}
