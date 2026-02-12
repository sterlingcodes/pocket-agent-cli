package hibp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "hibp" {
		t.Errorf("expected Use 'hibp', got %q", cmd.Use)
	}
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Use] = true
	}
	expectedSubs := []string{"password [password]", "breaches"}
	for _, name := range expectedSubs {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestPasswordCheck_Compromised(t *testing.T) {
	// Mock server returning k-anonymity response with matching hash
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/range/5BAA6" {
			t.Errorf("expected path /range/5BAA6, got %s", r.URL.Path)
		}
		// Return response with matching suffix (for password "password" -> SHA1 5BAA61E4C9B93F3F0682250B6CF8331B7EE68FD8)
		w.Write([]byte("1E4C9B93F3F0682250B6CF8331B7EE68FD8:3861493\n"))
	}))
	defer srv.Close()

	oldPasswordURL := passwordBaseURL
	passwordBaseURL = srv.URL
	defer func() { passwordBaseURL = oldPasswordURL }()

	cmd := newPasswordCmd()
	cmd.SetArgs([]string{"password"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestPasswordCheck_NotCompromised(t *testing.T) {
	// Mock server returning k-anonymity response without matching hash
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return response with non-matching suffixes
		w.Write([]byte("000000000000000000000000000000AAAAA:123\nBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB:456\n"))
	}))
	defer srv.Close()

	oldPasswordURL := passwordBaseURL
	passwordBaseURL = srv.URL
	defer func() { passwordBaseURL = oldPasswordURL }()

	cmd := newPasswordCmd()
	cmd.SetArgs([]string{"somesecurepassword123"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestPasswordCheck_APIError(t *testing.T) {
	// Mock server returning error
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer srv.Close()

	oldPasswordURL := passwordBaseURL
	passwordBaseURL = srv.URL
	defer func() { passwordBaseURL = oldPasswordURL }()

	cmd := newPasswordCmd()
	cmd.SetArgs([]string{"testpassword"})

	err := cmd.Execute()
	// Expect PrintedError (which is not nil)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestBreaches_Success(t *testing.T) {
	// Mock server returning breaches
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/breaches" {
			t.Errorf("expected path /breaches, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{
				"Name": "Adobe",
				"Title": "Adobe",
				"Domain": "adobe.com",
				"BreachDate": "2013-10-04",
				"PwnCount": 152445165,
				"Description": "Adobe breach",
				"DataClasses": ["Email addresses", "Passwords"],
				"IsVerified": true
			}
		]`))
	}))
	defer srv.Close()

	oldBreachesURL := breachesBaseURL
	breachesBaseURL = srv.URL
	defer func() { breachesBaseURL = oldBreachesURL }()

	cmd := newBreachesCmd()
	cmd.SetArgs([]string{"--limit", "10"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestBreaches_APIError(t *testing.T) {
	// Mock server returning error
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Service unavailable"))
	}))
	defer srv.Close()

	oldBreachesURL := breachesBaseURL
	breachesBaseURL = srv.URL
	defer func() { breachesBaseURL = oldBreachesURL }()

	cmd := newBreachesCmd()

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestBreaches_InvalidJSON(t *testing.T) {
	// Mock server returning invalid JSON
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	oldBreachesURL := breachesBaseURL
	breachesBaseURL = srv.URL
	defer func() { breachesBaseURL = oldBreachesURL }()

	cmd := newBreachesCmd()

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error, got nil")
	}
}
