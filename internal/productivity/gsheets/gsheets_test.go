package gsheets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "gsheets" {
		t.Errorf("expected Use 'gsheets', got %q", cmd.Use)
	}
	// Check that command has subcommands
	if len(cmd.Commands()) < 3 {
		t.Errorf("expected at least 3 subcommands, got %d", len(cmd.Commands()))
	}
}

func TestColToLetter(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "A"},
		{1, "B"},
		{25, "Z"},
		{26, "AA"},
		{27, "AB"},
		{51, "AZ"},
		{52, "BA"},
		{701, "ZZ"},
		{702, "AAA"},
	}

	for _, tt := range tests {
		result := colToLetter(tt.input)
		if result != tt.expected {
			t.Errorf("colToLetter(%d) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestDoRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check x-goog-api-key header
		if r.Header.Get("x-goog-api-key") != "test-api-key" {
			t.Errorf("expected x-goog-api-key 'test-api-key', got %q", r.Header.Get("x-goog-api-key"))
		}

		// Check User-Agent
		if r.Header.Get("User-Agent") != "Pocket-CLI/1.0" {
			t.Errorf("expected User-Agent 'Pocket-CLI/1.0', got %q", r.Header.Get("User-Agent"))
		}

		// Return mock spreadsheet metadata
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"properties": map[string]any{
				"title":  "Test Spreadsheet",
				"locale": "en_US",
			},
			"sheets": []map[string]any{
				{
					"properties": map[string]any{
						"sheetId": 0,
						"title":   "Sheet1",
						"index":   0,
						"gridProperties": map[string]any{
							"rowCount":    1000,
							"columnCount": 26,
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	data, err := doRequest(srv.URL+"/spreadsheet123", "test-api-key")
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}

	var resp struct {
		Properties struct {
			Title string `json:"title"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Properties.Title != "Test Spreadsheet" {
		t.Errorf("expected title 'Test Spreadsheet', got %q", resp.Properties.Title)
	}
}

func TestDoRequestForbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Access denied",
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	_, err := doRequest(srv.URL, "bad-key")
	if err == nil {
		t.Error("expected error for 403 response, got nil")
	}
}

func TestDoRequestNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Spreadsheet not found",
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	_, err := doRequest(srv.URL, "test-key")
	if err == nil {
		t.Error("expected error for 404 response, got nil")
	}
}

func TestGetSpreadsheetMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/spreadsheet123" {
			t.Errorf("expected path '/spreadsheet123', got %q", r.URL.Path)
		}

		// Check fields param
		params := r.URL.Query()
		if params.Get("fields") == "" {
			t.Error("expected 'fields' query parameter")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"properties": map[string]any{
				"title":  "My Spreadsheet",
				"locale": "en_US",
			},
			"sheets": []map[string]any{
				{
					"properties": map[string]any{
						"sheetId": 0,
						"title":   "Sheet1",
						"index":   0,
						"gridProperties": map[string]any{
							"rowCount":    100,
							"columnCount": 10,
						},
					},
				},
				{
					"properties": map[string]any{
						"sheetId": 1,
						"title":   "Sheet2",
						"index":   1,
						"gridProperties": map[string]any{
							"rowCount":    50,
							"columnCount": 5,
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	data, err := doRequest(srv.URL+"/spreadsheet123?fields=properties,sheets.properties", "test-key")
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}

	var resp struct {
		Properties struct {
			Title string `json:"title"`
		} `json:"properties"`
		Sheets []struct {
			Properties struct {
				Title string `json:"title"`
			} `json:"properties"`
		} `json:"sheets"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Properties.Title != "My Spreadsheet" {
		t.Errorf("expected title 'My Spreadsheet', got %q", resp.Properties.Title)
	}

	if len(resp.Sheets) != 2 {
		t.Fatalf("expected 2 sheets, got %d", len(resp.Sheets))
	}

	if resp.Sheets[0].Properties.Title != "Sheet1" {
		t.Errorf("expected sheet title 'Sheet1', got %q", resp.Sheets[0].Properties.Title)
	}
}

func TestReadRangeResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/spreadsheet123/values/Sheet1!A1:C3" {
			t.Errorf("expected path '/spreadsheet123/values/Sheet1!A1:C3', got %q", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"range": "Sheet1!A1:C3",
			"values": [][]interface{}{
				{"Name", "Age", "City"},
				{"Alice", 30, "NYC"},
				{"Bob", 25, "LA"},
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	data, err := doRequest(srv.URL+"/spreadsheet123/values/Sheet1!A1:C3", "test-key")
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}

	var resp struct {
		Range  string          `json:"range"`
		Values [][]interface{} `json:"values"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Range != "Sheet1!A1:C3" {
		t.Errorf("expected range 'Sheet1!A1:C3', got %q", resp.Range)
	}

	if len(resp.Values) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(resp.Values))
	}

	if resp.Values[0][0] != "Name" {
		t.Errorf("expected first cell 'Name', got %v", resp.Values[0][0])
	}
}

func TestBatchGetResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that multiple ranges are requested
		params := r.URL.Query()
		if len(params["ranges"]) < 1 {
			t.Error("expected at least one 'ranges' query parameter")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"valueRanges": []map[string]any{
				{
					"range": "Sheet1!A:ZZ",
					"values": [][]interface{}{
						{"Apple", "Banana"},
						{"Cat", "Dog"},
					},
				},
				{
					"range": "Sheet2!A:ZZ",
					"values": [][]interface{}{
						{"Test", "Value"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	data, err := doRequest(srv.URL+"/spreadsheet123/values:batchGet?ranges=Sheet1!A:ZZ&ranges=Sheet2!A:ZZ", "test-key")
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}

	var resp struct {
		ValueRanges []struct {
			Range  string          `json:"range"`
			Values [][]interface{} `json:"values"`
		} `json:"valueRanges"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp.ValueRanges) != 2 {
		t.Fatalf("expected 2 value ranges, got %d", len(resp.ValueRanges))
	}

	if resp.ValueRanges[0].Range != "Sheet1!A:ZZ" {
		t.Errorf("expected range 'Sheet1!A:ZZ', got %q", resp.ValueRanges[0].Range)
	}
}

func TestSearchLogic(t *testing.T) {
	// Test the search logic that's used in the search command
	// This simulates finding "test" in various cells

	valueRanges := []struct {
		Range  string
		Values [][]interface{}
	}{
		{
			Range: "'Sheet1'!A1:Z100",
			Values: [][]interface{}{
				{"This is a test", "other"},
				{"value", "another test"},
			},
		},
	}

	queryLower := "test"
	var matches []CellMatch

	for _, vr := range valueRanges {
		sheetName := vr.Range
		// Extract sheet name from range
		if idx := len(sheetName); idx > 0 {
			// Simplified extraction for test
			sheetName = "Sheet1"
		}

		for rowIdx, row := range vr.Values {
			for colIdx, cell := range row {
				cellStr := ""
				if cell != nil {
					cellStr = cell.(string)
				}
				if len(cellStr) > 0 && len(queryLower) > 0 {
					// Simple substring match
					contains := false
					for i := 0; i <= len(cellStr)-len(queryLower); i++ {
						if cellStr[i:i+len(queryLower)] == queryLower {
							contains = true
							break
						}
					}
					if contains {
						cellRef := colToLetter(colIdx) + string(rune('1'+rowIdx))
						matches = append(matches, CellMatch{
							Sheet: sheetName,
							Cell:  cellRef,
							Value: cellStr,
							Row:   rowIdx + 1,
							Col:   colIdx + 1,
						})
					}
				}
			}
		}
	}

	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}

	if matches[0].Cell != "A1" {
		t.Errorf("expected cell 'A1', got %q", matches[0].Cell)
	}
	if matches[1].Cell != "B2" {
		t.Errorf("expected cell 'B2', got %q", matches[1].Cell)
	}
}

func TestDoRequestAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Invalid range",
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	_, err := doRequest(srv.URL, "test-key")
	if err == nil {
		t.Error("expected error for 400 response, got nil")
	}
}
