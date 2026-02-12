package wikipedia

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "wikipedia" {
		t.Errorf("expected Use 'wikipedia', got %q", cmd.Use)
	}

	// Check aliases
	expectedAliases := []string{"wiki", "wp"}
	if len(cmd.Aliases) != len(expectedAliases) {
		t.Errorf("expected %d aliases, got %d", len(expectedAliases), len(cmd.Aliases))
	}

	// Check subcommands exist
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"search [query]", "summary [title]", "article [title]"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestSearchCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("action") != "query" {
			http.Error(w, "bad action", http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("list") != "search" {
			http.Error(w, "bad list param", http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"query": map[string]any{
				"search": []map[string]any{
					{
						"title":   "Go (programming language)",
						"snippet": "Go is a <span class=\"searchmatch\">statically typed</span> language",
						"pageid":  12345,
					},
					{
						"title":   "Golang Tutorial",
						"snippet": "Learn the &quot;Go&quot; programming language &amp; more",
						"pageid":  67890,
					},
				},
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newSearchCmd()
	cmd.SetArgs([]string{"golang", "--limit", "2"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSummaryCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("action") != "query" {
			http.Error(w, "bad action", http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"query": map[string]any{
				"pages": map[string]any{
					"12345": map[string]any{
						"pageid":  12345,
						"title":   "Go (programming language)",
						"extract": "Go is a statically typed, compiled programming language designed at Google.",
					},
				},
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newSummaryCmd()
	cmd.SetArgs([]string{"Go", "--sentences", "3"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSummaryNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"query": map[string]any{
				"pages": map[string]any{
					"-1": map[string]any{
						"pageid": 0,
						"title":  "NonExistentArticle",
					},
				},
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newSummaryCmd()
	cmd.SetArgs([]string{"NonExistentArticle"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for non-existent article")
	}
}

func TestArticleCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("action") != "query" {
			http.Error(w, "bad action", http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"query": map[string]any{
				"pages": map[string]any{
					"54321": map[string]any{
						"pageid":  54321,
						"title":   "Python (programming language)",
						"extract": "Python is an interpreted, high-level programming language. It was created by Guido van Rossum.",
					},
				},
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newArticleCmd()
	cmd.SetArgs([]string{"Python", "--chars", "1000"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWikiGetError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newSearchCmd()
	cmd.SetArgs([]string{"test"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestWikiGetMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not valid json"))
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newSearchCmd()
	cmd.SetArgs([]string{"test"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestCleanSnippet(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "remove searchmatch spans",
			input:    `<span class="searchmatch">hello</span> world`,
			expected: "hello world",
		},
		{
			name:     "decode HTML entities",
			input:    `&quot;quoted&quot; &amp; &lt;tag&gt;`,
			expected: `"quoted" & <tag>`,
		},
		{
			name:     "mixed HTML and entities",
			input:    `<span class="searchmatch">Go</span> &amp; &quot;Python&quot;`,
			expected: `Go & "Python"`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no HTML",
			input:    "plain text",
			expected: "plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanSnippet(tt.input)
			if result != tt.expected {
				t.Errorf("cleanSnippet(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
