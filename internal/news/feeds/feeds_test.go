package feeds

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "feeds" {
		t.Errorf("expected Use 'feeds', got %q", cmd.Use)
	}
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	expectedSubs := []string{"fetch [url]", "list", "add [url]", "read [name]"}
	// Note: remove has "remove [name-or-url]" but also aliases "rm" and "delete"
	for _, name := range expectedSubs {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
	// Check for remove with its full signature
	if !subs["remove [name-or-url]"] {
		t.Error("missing subcommand 'remove [name-or-url]'")
	}
}

func TestFetch_RSSFeed(t *testing.T) {
	// Mock RSS feed server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
	<channel>
		<title>Test Feed</title>
		<link>https://example.com</link>
		<description>Test RSS feed</description>
		<item>
			<title>Test Article</title>
			<link>https://example.com/article1</link>
			<description>This is a test article</description>
			<pubDate>Mon, 01 Jan 2024 12:00:00 +0000</pubDate>
		</item>
	</channel>
</rss>`))
	}))
	defer srv.Close()

	cmd := newFetchCmd()
	cmd.SetArgs([]string{srv.URL, "--limit", "10"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestFetch_AtomFeed(t *testing.T) {
	// Mock Atom feed server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
	<title>Test Atom Feed</title>
	<link href="https://example.com"/>
	<updated>2024-01-01T12:00:00Z</updated>
	<entry>
		<title>Test Entry</title>
		<link href="https://example.com/entry1"/>
		<summary>Test summary</summary>
		<updated>2024-01-01T12:00:00Z</updated>
	</entry>
</feed>`))
	}))
	defer srv.Close()

	cmd := newFetchCmd()
	cmd.SetArgs([]string{srv.URL, "--limit", "5", "--summary", "100"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestFetch_InvalidFeed(t *testing.T) {
	// Mock server returning invalid XML
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not xml"))
	}))
	defer srv.Close()

	cmd := newFetchCmd()
	cmd.SetArgs([]string{srv.URL})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for invalid feed, got nil")
	}
}

func TestCleanHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<p>Hello World</p>", "Hello World"},
		{"<br>Line break<br/>", "Line break"},
		{"<a href='test'>Link</a>", "Link"},
		{"&lt;escaped&gt;", "<escaped>"},
		{"Multiple   spaces", "Multiple spaces"},
		{"  trim  ", "trim"},
	}

	for _, tt := range tests {
		result := cleanHTML(tt.input)
		if result != tt.expected {
			t.Errorf("cleanHTML(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly ten!!", 10, "exactly..."},
		{"this is a very long string that needs truncation", 20, "this is a very lo..."},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestTimeAgo(t *testing.T) {
	now := time.Now()
	tests := []struct {
		time     time.Time
		expected string
	}{
		{now.Add(time.Hour), "future"},
		{now.Add(-30 * time.Second), "now"},
		{now.Add(-5 * time.Minute), "5m"},
		{now.Add(-2 * time.Hour), "2h"},
		{now.Add(-25 * time.Hour), "1d"},
		{now.Add(-72 * time.Hour), "3d"},
	}

	for _, tt := range tests {
		result := timeAgo(tt.time)
		if result != tt.expected {
			t.Errorf("timeAgo(%v) = %q, want %q", tt.time, result, tt.expected)
		}
	}
}

func TestList_NoFeeds(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "feeds-*.json")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Override feedsFilePath by creating a temporary HOME
	tmpDir := os.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cmd := newListCmd()
	err = cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestAddRemove_SavedFeeds(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "feeds-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Mock feed server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`<?xml version="1.0"?>
<rss version="2.0">
	<channel>
		<title>My Test Feed</title>
		<item><title>Item 1</title></item>
	</channel>
</rss>`))
	}))
	defer srv.Close()

	// Add feed
	addCmd := newAddCmd()
	addCmd.SetArgs([]string{srv.URL})
	err = addCmd.Execute()
	if err != nil {
		t.Errorf("expected no error adding feed, got %v", err)
	}

	// List feeds
	listCmd := newListCmd()
	err = listCmd.Execute()
	if err != nil {
		t.Errorf("expected no error listing feeds, got %v", err)
	}

	// Remove feed
	rmCmd := newRemoveCmd()
	rmCmd.SetArgs([]string{srv.URL})
	err = rmCmd.Execute()
	if err != nil {
		t.Errorf("expected no error removing feed, got %v", err)
	}
}

func TestAdd_Duplicate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "feeds-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Mock feed server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`<?xml version="1.0"?>
<rss version="2.0"><channel><title>Test</title></channel></rss>`))
	}))
	defer srv.Close()

	// Add feed first time
	addCmd := newAddCmd()
	addCmd.SetArgs([]string{srv.URL, "--name", "TestFeed"})
	err = addCmd.Execute()
	if err != nil {
		t.Errorf("expected no error on first add, got %v", err)
	}

	// Try adding duplicate
	addCmd2 := newAddCmd()
	addCmd2.SetArgs([]string{srv.URL})
	err = addCmd2.Execute()
	if err == nil {
		t.Error("expected error for duplicate feed, got nil")
	}
}

func TestRemove_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "feeds-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cmd := newRemoveCmd()
	cmd.SetArgs([]string{"nonexistent"})
	err = cmd.Execute()
	if err == nil {
		t.Error("expected error for nonexistent feed, got nil")
	}
}

func TestRead_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "feeds-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cmd := newReadCmd()
	cmd.SetArgs([]string{"nonexistent"})
	err = cmd.Execute()
	if err == nil {
		t.Error("expected error for nonexistent feed, got nil")
	}
}
