package stackexchange

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "stackexchange" {
		t.Errorf("expected Use 'stackexchange', got %q", cmd.Use)
	}

	// Check aliases
	expectedAliases := []string{"se", "stackoverflow", "so"}
	if len(cmd.Aliases) != len(expectedAliases) {
		t.Errorf("expected %d aliases, got %d", len(expectedAliases), len(cmd.Aliases))
	}

	// Check subcommands exist
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"search [query]", "question [id]", "answers [question-id]"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func gzipEncode(data []byte) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write(data)
	gz.Close()
	return buf.Bytes()
}

func TestSearchCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/advanced" {
			http.Error(w, "wrong path", http.StatusNotFound)
			return
		}

		if r.URL.Query().Get("site") != "stackoverflow" {
			http.Error(w, "bad site param", http.StatusBadRequest)
			return
		}

		resp := seResponse{
			Items: []seItem{
				{
					QuestionID:   12345,
					Title:        "How to use Go channels?",
					Score:        42,
					AnswerCount:  5,
					ViewCount:    1000,
					Tags:         []string{"go", "channels"},
					Link:         "https://stackoverflow.com/q/12345",
					CreationDate: time.Now().Unix() - 86400,
					Owner: struct {
						DisplayName string `json:"display_name"`
					}{DisplayName: "testuser"},
				},
			},
		}

		jsonData, _ := json.Marshal(resp)
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(gzipEncode(jsonData))
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newSearchCmd()
	cmd.SetArgs([]string{"golang channels", "--limit", "10", "--site", "stackoverflow"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestQuestionCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/questions/12345" {
			http.Error(w, "wrong path", http.StatusNotFound)
			return
		}

		resp := seResponse{
			Items: []seItem{
				{
					QuestionID:       12345,
					Title:            "Test Question",
					Body:             "<p>This is a test question with <code>code</code> and <br/>newlines</p>",
					Score:            100,
					AnswerCount:      10,
					AcceptedAnswerID: 54321,
					ViewCount:        5000,
					Tags:             []string{"go", "testing"},
					Link:             "https://stackoverflow.com/q/12345",
					CreationDate:     time.Now().Unix() - 3600,
					Owner: struct {
						DisplayName string `json:"display_name"`
					}{DisplayName: "questioner"},
				},
			},
		}

		jsonData, _ := json.Marshal(resp)
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(gzipEncode(jsonData))
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newQuestionCmd()
	cmd.SetArgs([]string{"12345", "--site", "stackoverflow"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestQuestionNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := seResponse{Items: []seItem{}}
		jsonData, _ := json.Marshal(resp)
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(gzipEncode(jsonData))
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newQuestionCmd()
	cmd.SetArgs([]string{"99999"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for non-existent question")
	}
}

func TestAnswersCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/questions/12345/answers" {
			http.Error(w, "wrong path", http.StatusNotFound)
			return
		}

		resp := seResponse{
			Items: []seItem{
				{
					AnswerID:     54321,
					Body:         "<p>This is an answer with <pre>code block</pre> and <li>list items</li></p>",
					Score:        25,
					IsAccepted:   true,
					CreationDate: time.Now().Unix() - 1800,
					Owner: struct {
						DisplayName string `json:"display_name"`
					}{DisplayName: "answerer"},
				},
			},
		}

		jsonData, _ := json.Marshal(resp)
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(gzipEncode(jsonData))
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newAnswersCmd()
	cmd.SetArgs([]string{"12345", "--limit", "5"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSeGetError(t *testing.T) {
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

func TestSeGetUncompressed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := seResponse{
			Items: []seItem{
				{
					QuestionID: 111,
					Title:      "Uncompressed Test",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newSearchCmd()
	cmd.SetArgs([]string{"test"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCleanHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "paragraph tags",
			input:    "<p>Hello</p><p>World</p>",
			expected: "Hello\n\nWorld",
		},
		{
			name:     "code tags",
			input:    "Use <code>func</code> for functions",
			expected: "Use `func` for functions",
		},
		{
			name:     "pre tags",
			input:    "<pre>code block</pre>",
			expected: "```\ncode block\n```",
		},
		{
			name:     "list items",
			input:    "<li>Item 1</li><li>Item 2</li>",
			expected: "- Item 1\n- Item 2",
		},
		{
			name:     "br tags",
			input:    "Line 1<br>Line 2<br/>Line 3<br />Line 4",
			expected: "Line 1\nLine 2\nLine 3\nLine 4",
		},
		{
			name:     "HTML entities",
			input:    "&lt;tag&gt; &amp; &quot;quoted&quot;",
			expected: "<tag> & \"quoted\"",
		},
		{
			name:     "mixed content",
			input:    "<p>Use <code>fmt.Println</code> to print<br/>like this:</p><pre>fmt.Println(&quot;Hello&quot;)</pre>",
			expected: "Use `fmt.Println` to print\nlike this:\n\n```\nfmt.Println(\"Hello\")\n```",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanHTML(tt.input)
			if result != tt.expected {
				t.Errorf("cleanHTML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "now",
			time:     now,
			expected: "now",
		},
		{
			name:     "5 minutes ago",
			time:     now.Add(-5 * time.Minute),
			expected: "5m",
		},
		{
			name:     "2 hours ago",
			time:     now.Add(-2 * time.Hour),
			expected: "2h",
		},
		{
			name:     "5 days ago",
			time:     now.Add(-5 * 24 * time.Hour),
			expected: "5d",
		},
		{
			name:     "2 months ago",
			time:     now.Add(-60 * 24 * time.Hour),
			expected: "2mo",
		},
		{
			name:     "1 year ago",
			time:     now.Add(-365 * 24 * time.Hour),
			expected: "1y",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := timeAgo(tt.time)
			if result != tt.expected {
				t.Errorf("timeAgo(%v) = %q, want %q", tt.time, result, tt.expected)
			}
		})
	}
}
