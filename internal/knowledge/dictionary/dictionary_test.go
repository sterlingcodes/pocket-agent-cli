package dictionary

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "dictionary" {
		t.Errorf("expected Use 'dictionary', got %q", cmd.Use)
	}

	// Check aliases
	expectedAliases := []string{"dict", "define"}
	if len(cmd.Aliases) != len(expectedAliases) {
		t.Errorf("expected %d aliases, got %d", len(expectedAliases), len(cmd.Aliases))
	}

	// Check subcommands exist
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"define [word]", "synonyms [word]", "antonyms [word]"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestDefineCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hello" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		entries := []apiEntry{
			{
				Word:     "hello",
				Phonetic: "/həˈloʊ/",
				Origin:   "alteration of hallo",
				Phonetics: []struct {
					Text  string `json:"text"`
					Audio string `json:"audio"`
				}{
					{Text: "/həˈloʊ/", Audio: "https://example.com/hello.mp3"},
				},
				Meanings: []struct {
					PartOfSpeech string   `json:"partOfSpeech"`
					Synonyms     []string `json:"synonyms"`
					Antonyms     []string `json:"antonyms"`
					Definitions  []struct {
						Definition string   `json:"definition"`
						Example    string   `json:"example"`
						Synonyms   []string `json:"synonyms"`
						Antonyms   []string `json:"antonyms"`
					} `json:"definitions"`
				}{
					{
						PartOfSpeech: "interjection",
						Synonyms:     []string{"greeting", "hi"},
						Antonyms:     []string{"goodbye"},
						Definitions: []struct {
							Definition string   `json:"definition"`
							Example    string   `json:"example"`
							Synonyms   []string `json:"synonyms"`
							Antonyms   []string `json:"antonyms"`
						}{
							{
								Definition: "used as a greeting",
								Example:    "hello, how are you?",
								Synonyms:   []string{"hey"},
								Antonyms:   []string{},
							},
						},
					},
				},
			},
		}

		json.NewEncoder(w).Encode(entries)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newDefineCmd()
	cmd.SetArgs([]string{"hello", "--limit", "2"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDefineNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"title":"No Definitions Found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newDefineCmd()
	cmd.SetArgs([]string{"xyzabc123"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for non-existent word")
	}
}

func TestSynonymsCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/happy" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		entries := []apiEntry{
			{
				Word: "happy",
				Meanings: []struct {
					PartOfSpeech string   `json:"partOfSpeech"`
					Synonyms     []string `json:"synonyms"`
					Antonyms     []string `json:"antonyms"`
					Definitions  []struct {
						Definition string   `json:"definition"`
						Example    string   `json:"example"`
						Synonyms   []string `json:"synonyms"`
						Antonyms   []string `json:"antonyms"`
					} `json:"definitions"`
				}{
					{
						PartOfSpeech: "adjective",
						Synonyms:     []string{"joyful", "cheerful", "content"},
						Definitions: []struct {
							Definition string   `json:"definition"`
							Example    string   `json:"example"`
							Synonyms   []string `json:"synonyms"`
							Antonyms   []string `json:"antonyms"`
						}{
							{
								Definition: "feeling pleasure",
								Synonyms:   []string{"glad", "pleased"},
							},
						},
					},
				},
			},
		}

		json.NewEncoder(w).Encode(entries)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newSynonymsCmd()
	cmd.SetArgs([]string{"happy"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAntonymsCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hot" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		entries := []apiEntry{
			{
				Word: "hot",
				Meanings: []struct {
					PartOfSpeech string   `json:"partOfSpeech"`
					Synonyms     []string `json:"synonyms"`
					Antonyms     []string `json:"antonyms"`
					Definitions  []struct {
						Definition string   `json:"definition"`
						Example    string   `json:"example"`
						Synonyms   []string `json:"synonyms"`
						Antonyms   []string `json:"antonyms"`
					} `json:"definitions"`
				}{
					{
						PartOfSpeech: "adjective",
						Antonyms:     []string{"cold", "cool"},
						Definitions: []struct {
							Definition string   `json:"definition"`
							Example    string   `json:"example"`
							Synonyms   []string `json:"synonyms"`
							Antonyms   []string `json:"antonyms"`
						}{
							{
								Definition: "having high temperature",
								Antonyms:   []string{"freezing", "chilly"},
							},
						},
					},
				},
			},
		}

		json.NewEncoder(w).Encode(entries)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newAntonymsCmd()
	cmd.SetArgs([]string{"hot"})

	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFetchWordError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newDefineCmd()
	cmd.SetArgs([]string{"test"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestFetchWordMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not valid json"))
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	cmd := newDefineCmd()
	cmd.SetArgs([]string{"test"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestLimitSlice(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		limit  int
		expect []string
	}{
		{
			name:   "slice shorter than limit",
			input:  []string{"a", "b", "c"},
			limit:  5,
			expect: []string{"a", "b", "c"},
		},
		{
			name:   "slice equal to limit",
			input:  []string{"a", "b", "c"},
			limit:  3,
			expect: []string{"a", "b", "c"},
		},
		{
			name:   "slice longer than limit",
			input:  []string{"a", "b", "c", "d", "e"},
			limit:  3,
			expect: []string{"a", "b", "c"},
		},
		{
			name:   "empty slice",
			input:  []string{},
			limit:  5,
			expect: []string{},
		},
		{
			name:   "zero limit",
			input:  []string{"a", "b"},
			limit:  0,
			expect: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := limitSlice(tt.input, tt.limit)
			if len(result) != len(tt.expect) {
				t.Errorf("limitSlice(%v, %d) length = %d, want %d", tt.input, tt.limit, len(result), len(tt.expect))
			}
			for i := range result {
				if result[i] != tt.expect[i] {
					t.Errorf("limitSlice(%v, %d)[%d] = %q, want %q", tt.input, tt.limit, i, result[i], tt.expect[i])
				}
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name   string
		slice  []string
		item   string
		expect bool
	}{
		{
			name:   "item present",
			slice:  []string{"apple", "banana", "cherry"},
			item:   "banana",
			expect: true,
		},
		{
			name:   "item not present",
			slice:  []string{"apple", "banana", "cherry"},
			item:   "grape",
			expect: false,
		},
		{
			name:   "empty slice",
			slice:  []string{},
			item:   "apple",
			expect: false,
		},
		{
			name:   "case sensitive",
			slice:  []string{"Apple", "Banana"},
			item:   "apple",
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			if result != tt.expect {
				t.Errorf("contains(%v, %q) = %v, want %v", tt.slice, tt.item, result, tt.expect)
			}
		})
	}
}
