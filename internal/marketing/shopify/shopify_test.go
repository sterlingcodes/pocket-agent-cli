package shopify

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstablemind/pocket/internal/common/config"
)

var testConfigPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "shopify-test-*")
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
	if cmd.Use != "shopify" {
		t.Errorf("expected Use=shopify, got %s", cmd.Use)
	}

	want := map[string]bool{
		"shop":            false,
		"orders":          false,
		"order":           false,
		"products":        false,
		"product":         false,
		"customers":       false,
		"customer-search": false,
		"inventory":       false,
		"inventory-set":   false,
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

func TestDoGetSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Shopify-Access-Token")
		if token != "test_shop_token" {
			t.Errorf("X-Shopify-Access-Token = %q, want %q", token, "test_shop_token")
		}
		if r.Method != "GET" {
			t.Errorf("method = %s, want GET", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"shop": map[string]any{
				"id":   float64(12345),
				"name": "Test Store",
			},
		})
	}))
	defer ts.Close()

	c := &shopClient{store: "test-store", token: "test_shop_token", apiBaseURL: ts.URL}
	result, err := c.doGet("shop.json", nil)
	if err != nil {
		t.Fatalf("doGet failed: %v", err)
	}

	shopData, _ := result["shop"].(map[string]any)
	if shopData == nil {
		t.Fatal("expected shop in response")
	}
	if getString(shopData, "name") != "Test Store" {
		t.Errorf("name = %q, want %q", getString(shopData, "name"), "Test Store")
	}
}

func TestDoGetHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": "Not authorized",
		})
	}))
	defer ts.Close()

	c := &shopClient{store: "test", token: "bad_token", apiBaseURL: ts.URL}
	_, err := c.doGet("shop.json", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "Not authorized") {
		t.Errorf("error %q should contain 'Not authorized'", err.Error())
	}
}

func TestDoPostSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Shopify-Access-Token")
		if token != "post_token" {
			t.Errorf("X-Shopify-Access-Token = %q, want %q", token, "post_token")
		}
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "inventory_item_id") {
			t.Errorf("body %q should contain 'inventory_item_id'", string(body))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"inventory_level": map[string]any{
				"inventory_item_id": float64(808950810),
				"location_id":       float64(905684977),
				"available":         float64(42),
			},
		})
	}))
	defer ts.Close()

	c := &shopClient{store: "test", token: "post_token", apiBaseURL: ts.URL}
	result, err := c.doPost("inventory_levels/set.json", map[string]any{
		"inventory_item_id": 808950810,
		"location_id":       905684977,
		"available":         42,
	})
	if err != nil {
		t.Fatalf("doPost failed: %v", err)
	}

	level, _ := result["inventory_level"].(map[string]any)
	if level == nil {
		t.Fatal("expected inventory_level in response")
	}
}

func TestDoPutSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Shopify-Access-Token")
		if token != "put_token" {
			t.Errorf("X-Shopify-Access-Token = %q, want %q", token, "put_token")
		}
		if r.Method != "PUT" {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"product": map[string]any{
				"id":    float64(123),
				"title": "Updated Product",
			},
		})
	}))
	defer ts.Close()

	c := &shopClient{store: "test", token: "put_token", apiBaseURL: ts.URL}
	result, err := c.doPut("products/123.json", map[string]any{
		"product": map[string]any{"title": "Updated Product"},
	})
	if err != nil {
		t.Fatalf("doPut failed: %v", err)
	}

	prod, _ := result["product"].(map[string]any)
	if prod == nil {
		t.Fatal("expected product in response")
	}
	if getString(prod, "title") != "Updated Product" {
		t.Errorf("title = %q, want %q", getString(prod, "title"), "Updated Product")
	}
}

func TestDoDeleteSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Shopify-Access-Token")
		if token != "del_token" {
			t.Errorf("X-Shopify-Access-Token = %q, want %q", token, "del_token")
		}
		if r.Method != "DELETE" {
			t.Errorf("method = %s, want DELETE", r.Method)
		}

		w.WriteHeader(200)
		w.Write([]byte("{}"))
	}))
	defer ts.Close()

	c := &shopClient{store: "test", token: "del_token", apiBaseURL: ts.URL}
	err := c.doDelete("products/123.json")
	if err != nil {
		t.Fatalf("doDelete failed: %v", err)
	}
}

func TestNewClientMissingStore(t *testing.T) {
	// Reset baseURL so newClient will try to compute it
	origURL := baseURL
	baseURL = ""
	defer func() { baseURL = origURL }()

	writeTestConfig(t, map[string]string{
		"shopify_token": "tok",
	})

	_, err := newClient()
	if err == nil {
		t.Fatal("expected error for missing store")
	}
}

func TestNewClientMissingToken(t *testing.T) {
	origURL := baseURL
	baseURL = ""
	defer func() { baseURL = origURL }()

	writeTestConfig(t, map[string]string{
		"shopify_store": "my-store",
	})

	_, err := newClient()
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestHelpers(t *testing.T) {
	m := map[string]any{
		"name":    "test",
		"count":   float64(42),
		"id":      float64(9876543210),
		"active":  true,
		"missing": nil,
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

	if got := getInt64(m, "id"); got != 9876543210 {
		t.Errorf("getInt64(id) = %d, want 9876543210", got)
	}
	if got := getInt64(m, "missing"); got != 0 {
		t.Errorf("getInt64(missing) = %d, want 0", got)
	}

	if got := getBool(m, "active"); !got {
		t.Errorf("getBool(active) = %v, want true", got)
	}
	if got := getBool(m, "missing"); got {
		t.Errorf("getBool(missing) = %v, want false", got)
	}
}

func TestParseShopifyError(t *testing.T) {
	// String error
	strResp := map[string]any{"errors": "Not found"}
	err := parseShopifyError(strResp, 404)
	if err == nil {
		t.Fatal("expected error for string errors")
	}
	if !strings.Contains(err.Error(), "Not found") {
		t.Errorf("error %q should contain 'Not found'", err.Error())
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error %q should contain '404'", err.Error())
	}

	// Object error
	objResp := map[string]any{
		"errors": map[string]any{
			"title": []any{"can't be blank"},
		},
	}
	err = parseShopifyError(objResp, 422)
	if err == nil {
		t.Fatal("expected error for object errors")
	}
	if !strings.Contains(err.Error(), "422") {
		t.Errorf("error %q should contain '422'", err.Error())
	}

	// No error key
	noErr := map[string]any{"data": "ok"}
	if got := parseShopifyError(noErr, 200); got != nil {
		t.Errorf("expected nil for no errors key, got %v", got)
	}
}

func TestShopifyConfigKeys(t *testing.T) {
	clearTestConfig(t)

	keys := map[string]string{
		"shopify_store": "my-test-store",
		"shopify_token": "shpat_xxxxxxxxxxxx",
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
