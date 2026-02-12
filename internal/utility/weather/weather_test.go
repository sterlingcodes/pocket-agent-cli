package weather

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "weather" {
		t.Errorf("expected Use 'weather', got %q", cmd.Use)
	}

	aliases := cmd.Aliases
	if len(aliases) != 1 || aliases[0] != "wttr" {
		t.Errorf("expected Aliases ['wttr'], got %v", aliases)
	}

	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"now [location]", "forecast [location]"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestAtoi(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"42", 42},
		{"0", 0},
		{"-10", -10},
		{"invalid", 0},
		{"", 0},
		{"123.45", 123},
	}

	for _, tt := range tests {
		got := atoi(tt.input)
		if got != tt.want {
			t.Errorf("atoi(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestFetchWeatherNow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"current_condition": []map[string]any{
				{
					"temp_C":         "20",
					"temp_F":         "68",
					"FeelsLikeC":     "19",
					"FeelsLikeF":     "66",
					"humidity":       "50",
					"windspeedKmph":  "10",
					"windspeedMiles": "6",
					"winddir16Point": "N",
					"visibility":     "10",
					"uvIndex":        "3",
					"weatherDesc":    []map[string]string{{"value": "Sunny"}},
				},
			},
			"nearest_area": []map[string]any{
				{
					"areaName": []map[string]string{{"value": "San Francisco"}},
					"country":  []map[string]string{{"value": "USA"}},
				},
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	err := fetchWeather("test", 0)
	if err != nil {
		t.Errorf("fetchWeather failed: %v", err)
	}
}

func TestFetchWeatherForecast(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"current_condition": []map[string]any{
				{
					"temp_C":         "20",
					"temp_F":         "68",
					"FeelsLikeC":     "19",
					"FeelsLikeF":     "66",
					"humidity":       "50",
					"windspeedKmph":  "10",
					"windspeedMiles": "6",
					"winddir16Point": "N",
					"visibility":     "10",
					"uvIndex":        "3",
					"weatherDesc":    []map[string]string{{"value": "Sunny"}},
				},
			},
			"nearest_area": []map[string]any{
				{
					"areaName": []map[string]string{{"value": "London"}},
					"country":  []map[string]string{{"value": "UK"}},
				},
			},
			"weather": []map[string]any{
				{
					"date":     "2024-01-01",
					"maxtempC": "22",
					"mintempC": "15",
					"maxtempF": "72",
					"mintempF": "59",
					"hourly": []map[string]any{
						{
							"chanceofrain": "20",
							"humidity":     "60",
							"weatherDesc":  []map[string]string{{"value": "Partly Cloudy"}},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	err := fetchWeather("test", 1)
	if err != nil {
		t.Errorf("fetchWeather with forecast failed: %v", err)
	}
}

func TestFetchWeatherNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"current_condition": []map[string]any{},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	err := fetchWeather("invalid", 0)
	if err == nil {
		t.Error("expected error for not found location, got nil")
	}
}

func TestFetchWeatherHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	err := fetchWeather("test", 0)
	if err == nil {
		t.Error("expected error for HTTP 500, got nil")
	}
}

func TestFetchWeatherParseError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid json"))
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	err := fetchWeather("test", 0)
	if err == nil {
		t.Error("expected parse error, got nil")
	}
}
