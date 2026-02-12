package prometheus

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "prometheus" {
		t.Errorf("expected Use 'prometheus', got %q", cmd.Use)
	}
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Name()] = true
	}
	for _, name := range []string{"query", "range", "alerts", "targets"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestGetString(t *testing.T) {
	tests := []struct {
		m        map[string]any
		key      string
		expected string
	}{
		{map[string]any{"status": "success"}, "status", "success"},
		{map[string]any{}, "status", ""},
		{map[string]any{"status": 123}, "status", ""},
	}
	for _, tt := range tests {
		result := getString(tt.m, tt.key)
		if result != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, result)
		}
	}
}

func TestToStringMap(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected map[string]string
	}{
		{
			"valid map",
			map[string]any{"key1": "value1", "key2": "value2"},
			map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			"mixed types",
			map[string]any{"key1": "value1", "key2": 123},
			map[string]string{"key1": "value1"},
		},
		{
			"not a map",
			"not a map",
			map[string]string{},
		},
		{
			"nil",
			nil,
			map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toStringMap(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d entries, got %d", len(tt.expected), len(result))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("expected %q for key %q, got %q", v, k, result[k])
				}
			}
		})
	}
}

func TestPromGetSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"status": "success",
			"data": map[string]any{
				"resultType": "vector",
				"result":     []any{},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	var result map[string]any
	err := promGet(srv.URL+"/api/v1/query?query=up", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result["status"] != "success" {
		t.Errorf("expected status 'success', got %v", result["status"])
	}
}

func TestPromGetError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"401", 401},
		{"404", 404},
		{"500", 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(map[string]any{"error": "Error"})
			}))
			defer srv.Close()

			var result map[string]any
			err := promGet(srv.URL+"/test", &result)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestPrometheusQueryResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"status": "success",
			"data": map[string]any{
				"resultType": "vector",
				"result": []any{
					map[string]any{
						"metric": map[string]any{
							"__name__": "up",
							"job":      "prometheus",
							"instance": "localhost:9090",
						},
						"value": []any{
							float64(1705320000),
							"1",
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	var result map[string]any
	err := promGet(srv.URL+"/api/v1/query", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	status := getString(result, "status")
	if status != "success" {
		t.Errorf("expected status 'success', got %q", status)
	}

	data, _ := result["data"].(map[string]any)
	resultType := getString(data, "resultType")
	if resultType != "vector" {
		t.Errorf("expected resultType 'vector', got %q", resultType)
	}
}

func TestPrometheusAlertsResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"status": "success",
			"data": map[string]any{
				"alerts": []any{
					map[string]any{
						"state": "firing",
						"labels": map[string]any{
							"alertname": "HighErrorRate",
							"severity":  "critical",
						},
						"annotations": map[string]any{
							"summary": "High error rate detected",
						},
						"activeAt": "2024-01-15T10:30:00Z",
						"value":    "0.95",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	var result map[string]any
	err := promGet(srv.URL+"/api/v1/alerts", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	status := getString(result, "status")
	if status != "success" {
		t.Errorf("expected status 'success', got %q", status)
	}

	data, _ := result["data"].(map[string]any)
	alerts, _ := data["alerts"].([]any)
	if len(alerts) != 1 {
		t.Errorf("expected 1 alert, got %d", len(alerts))
	}
}

func TestPrometheusTargetsResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"status": "success",
			"data": map[string]any{
				"activeTargets": []any{
					map[string]any{
						"labels": map[string]any{
							"instance": "localhost:9090",
							"job":      "prometheus",
						},
						"health":         "up",
						"lastScrape":     "2024-01-15T10:30:00Z",
						"scrapeInterval": "15s",
						"scrapeUrl":      "http://localhost:9090/metrics",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	var result map[string]any
	err := promGet(srv.URL+"/api/v1/targets", &result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	data, _ := result["data"].(map[string]any)
	targets, _ := data["activeTargets"].([]any)
	if len(targets) != 1 {
		t.Errorf("expected 1 target, got %d", len(targets))
	}
}
