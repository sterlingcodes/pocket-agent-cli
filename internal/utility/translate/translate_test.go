package translate

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "translate" {
		t.Errorf("expected Use 'translate', got %q", cmd.Use)
	}

	aliases := cmd.Aliases
	if len(aliases) != 2 {
		t.Errorf("expected 2 aliases, got %d", len(aliases))
	}

	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"text [text]", "languages"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestTextCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"responseStatus": 200,
			"responseData": map[string]any{
				"translatedText": "Hola mundo",
				"match":          1.0,
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newTextCmd()
	cmd.SetArgs([]string{"Hello world", "--from", "en", "--to", "es"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("text command failed: %v", err)
	}
}

func TestTextCmdMultipleWords(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"responseStatus": 200,
			"responseData": map[string]any{
				"translatedText": "Bonjour le monde",
				"match":          0.95,
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newTextCmd()
	cmd.SetArgs([]string{"Hello", "world", "--from", "en", "--to", "fr"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("text command with multiple words failed: %v", err)
	}
}

func TestTextCmdAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"responseStatus":  403,
			"responseDetails": "Invalid language pair",
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newTextCmd()
	cmd.SetArgs([]string{"Hello", "--from", "invalid", "--to", "xyz"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected API error, got nil")
	}
}

func TestLanguagesCmd(t *testing.T) {
	cmd := newLanguagesCmd()
	err := cmd.Execute()
	if err != nil {
		t.Errorf("languages command failed: %v", err)
	}
}

func TestRateLimitHandling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": "rate limited"})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newTextCmd()
	cmd.SetArgs([]string{"Hello"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected rate limit error, got nil")
	}
}

func TestHTTPErrorHandling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newTextCmd()
	cmd.SetArgs([]string{"Hello"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected HTTP error, got nil")
	}
}

func TestParseError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid json"))
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newTextCmd()
	cmd.SetArgs([]string{"Hello"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected parse error, got nil")
	}
}
