package paste

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "paste" {
		t.Errorf("expected Use 'paste', got %q", cmd.Use)
	}

	aliases := cmd.Aliases
	if len(aliases) != 2 {
		t.Errorf("expected 2 aliases, got %d", len(aliases))
	}

	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	for _, name := range []string{"create [content]", "get [url]"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestCreatePaste(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Location", "https://dpaste.com/ABC123")
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	oldURL := apiURL
	apiURL = srv.URL
	defer func() { apiURL = oldURL }()

	err := createPaste("test content", 7, "Test Title")
	if err != nil {
		t.Errorf("createPaste failed: %v", err)
	}
}

func TestCreatePasteBodyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("https://dpaste.com/XYZ789"))
	}))
	defer srv.Close()

	oldURL := apiURL
	apiURL = srv.URL
	defer func() { apiURL = oldURL }()

	err := createPaste("test content", 3, "")
	if err != nil {
		t.Errorf("createPaste with body response failed: %v", err)
	}
}

func TestCreatePasteHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid content"))
	}))
	defer srv.Close()

	oldURL := apiURL
	apiURL = srv.URL
	defer func() { apiURL = oldURL }()

	err := createPaste("", 7, "")
	if err == nil {
		t.Error("expected HTTP error, got nil")
	}
}

func TestGetPaste(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, ".txt") {
			t.Errorf("expected .txt suffix in URL, got %s", r.URL.Path)
		}
		w.Write([]byte("This is the paste content"))
	}))
	defer srv.Close()

	err := getPaste(srv.URL + "/ABC123")
	if err != nil {
		t.Errorf("getPaste failed: %v", err)
	}
}

func TestGetPasteAlreadyTxt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Content"))
	}))
	defer srv.Close()

	err := getPaste(srv.URL + "/ABC123.txt")
	if err != nil {
		t.Errorf("getPaste with .txt URL failed: %v", err)
	}
}

func TestGetPasteNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	err := getPaste(srv.URL + "/INVALID")
	if err == nil {
		t.Error("expected not found error, got nil")
	}
}

func TestGetPasteHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := getPaste(srv.URL + "/ABC123")
	if err == nil {
		t.Error("expected HTTP error, got nil")
	}
}

func TestCreateCmdWithArgs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		// URL-encoded content: "content=hello+world"
		if !bytes.Contains(body, []byte("content=hello+world")) && !bytes.Contains(body, []byte("content=hello%20world")) {
			t.Errorf("expected content 'hello world' (URL-encoded), got %s", body)
		}
		w.Header().Set("Location", "https://dpaste.com/TEST")
	}))
	defer srv.Close()

	oldURL := apiURL
	apiURL = srv.URL
	defer func() { apiURL = oldURL }()

	cmd := newCreateCmd()
	cmd.SetArgs([]string{"hello", "world"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("create command with args failed: %v", err)
	}
}

func TestGetCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("paste content"))
	}))
	defer srv.Close()

	cmd := newGetCmd()
	cmd.SetArgs([]string{srv.URL + "/TEST"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("get command failed: %v", err)
	}
}
