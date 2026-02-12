package hackernews

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "hackernews" {
		t.Errorf("expected Use 'hackernews', got %q", cmd.Use)
	}

	// Check aliases
	expectedAliases := []string{"hn"}
	if len(cmd.Aliases) != len(expectedAliases) {
		t.Errorf("expected %d aliases, got %d", len(expectedAliases), len(cmd.Aliases))
	}

	// Check subcommands exist
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"top", "new", "best", "ask", "show", "item [id]"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestTopCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/topstories.json":
			json.NewEncoder(w).Encode([]int{12345, 67890})
		case "/item/12345.json":
			json.NewEncoder(w).Encode(Item{
				ID:          12345,
				Type:        "story",
				By:          "testuser",
				Time:        time.Now().Unix() - 3600,
				Title:       "Test Story 1",
				URL:         "https://example.com/1",
				Score:       100,
				Descendants: 50,
			})
		case "/item/67890.json":
			json.NewEncoder(w).Encode(Item{
				ID:          67890,
				Type:        "story",
				By:          "testuser2",
				Time:        time.Now().Unix() - 7200,
				Title:       "Test Story 2",
				URL:         "https://example.com/2",
				Score:       75,
				Descendants: 30,
			})
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newTopCmd()
	cmd.SetArgs([]string{"--limit", "2"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewCmd_Stories(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/newstories.json":
			json.NewEncoder(w).Encode([]int{111})
		case "/item/111.json":
			json.NewEncoder(w).Encode(Item{
				ID:    111,
				Type:  "story",
				Title: "New Story",
			})
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newNewCmd()
	cmd.SetArgs([]string{"--limit", "1"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBestCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/beststories.json":
			json.NewEncoder(w).Encode([]int{222})
		case "/item/222.json":
			json.NewEncoder(w).Encode(Item{
				ID:    222,
				Type:  "story",
				Title: "Best Story",
			})
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newBestCmd()
	cmd.SetArgs([]string{"--limit", "1"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAskCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/askstories.json":
			json.NewEncoder(w).Encode([]int{333})
		case "/item/333.json":
			json.NewEncoder(w).Encode(Item{
				ID:    333,
				Type:  "story",
				Title: "Ask HN: Test Question?",
			})
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newAskCmd()
	cmd.SetArgs([]string{"--limit", "1"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestShowCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/showstories.json":
			json.NewEncoder(w).Encode([]int{444})
		case "/item/444.json":
			json.NewEncoder(w).Encode(Item{
				ID:    444,
				Type:  "story",
				Title: "Show HN: My Project",
			})
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newShowCmd()
	cmd.SetArgs([]string{"--limit", "1"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestItemCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/item/999.json":
			json.NewEncoder(w).Encode(Item{
				ID:          999,
				Type:        "story",
				By:          "author",
				Time:        time.Now().Unix() - 86400,
				Title:       "Story with comments",
				URL:         "https://example.com",
				Score:       200,
				Descendants: 10,
				Kids:        []int{1000, 1001},
			})
		case "/item/1000.json":
			json.NewEncoder(w).Encode(Item{
				ID:     1000,
				Type:   "comment",
				By:     "commenter1",
				Time:   time.Now().Unix() - 3600,
				Text:   "<p>This is a comment with <p>paragraphs</p></p>",
				Parent: 999,
			})
		case "/item/1001.json":
			json.NewEncoder(w).Encode(Item{
				ID:     1001,
				Type:   "comment",
				By:     "commenter2",
				Time:   time.Now().Unix() - 1800,
				Text:   "Another comment",
				Parent: 999,
			})
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newItemCmd()
	cmd.SetArgs([]string{"999", "--comments", "5"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestItemCmdInvalidID(t *testing.T) {
	cmd := newItemCmd()
	cmd.SetArgs([]string{"not-a-number"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for non-numeric ID")
	}
}

func TestItemCmdComment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/item/5555.json" {
			json.NewEncoder(w).Encode(Item{
				ID:     5555,
				Type:   "comment",
				By:     "user",
				Time:   time.Now().Unix(),
				Text:   "A standalone comment",
				Parent: 4444,
			})
		}
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newItemCmd()
	cmd.SetArgs([]string{"5555"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetStoryIDsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newTopCmd()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for 500 response")
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
			input:    "<p>First paragraph</p><p>Second paragraph</p>",
			expected: "First paragraph\n\nSecond paragraph",
		},
		{
			name:     "HTML entities",
			input:    "&lt;html&gt; &amp; &quot;quotes&quot;",
			expected: "<html> & \"quotes\"",
		},
		{
			name:     "mixed content",
			input:    "<p>Text with &lt;tags&gt;</p>",
			expected: "Text with <tags>",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "plain text",
			input:    "No HTML here",
			expected: "No HTML here",
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
		unix     int64
		expected string
	}{
		{
			name:     "now",
			unix:     now.Unix(),
			expected: "now",
		},
		{
			name:     "30 seconds ago",
			unix:     now.Add(-30 * time.Second).Unix(),
			expected: "now",
		},
		{
			name:     "10 minutes ago",
			unix:     now.Add(-10 * time.Minute).Unix(),
			expected: "10m",
		},
		{
			name:     "3 hours ago",
			unix:     now.Add(-3 * time.Hour).Unix(),
			expected: "3h",
		},
		{
			name:     "7 days ago",
			unix:     now.Add(-7 * 24 * time.Hour).Unix(),
			expected: "7d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := timeAgo(tt.unix)
			if result != tt.expected {
				t.Errorf("timeAgo(%d) = %q, want %q", tt.unix, result, tt.expected)
			}
		})
	}
}
