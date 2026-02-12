package trello

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "trello" {
		t.Errorf("expected Use 'trello', got %q", cmd.Use)
	}
	// Check that command has subcommands
	if len(cmd.Commands()) < 4 {
		t.Errorf("expected at least 4 subcommands, got %d", len(cmd.Commands()))
	}
}

func TestClientRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check query params
		params := r.URL.Query()
		if params.Get("key") != "test-key" {
			t.Errorf("expected key 'test-key', got %q", params.Get("key"))
		}
		if params.Get("token") != "test-token" {
			t.Errorf("expected token 'test-token', got %q", params.Get("token"))
		}

		// Return mock boards
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":             "board1",
				"name":           "Test Board",
				"desc":           "Test Description",
				"url":            "https://trello.com/b/board1",
				"shortUrl":       "https://trello.com/b/short1",
				"closed":         false,
				"idOrganization": "org1",
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	client := NewClient("test-key", "test-token")
	data, err := client.request("GET", "/boards/me", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	var boards []Board
	if err := json.Unmarshal(data, &boards); err != nil {
		t.Fatalf("failed to unmarshal boards: %v", err)
	}

	if len(boards) != 1 {
		t.Fatalf("expected 1 board, got %d", len(boards))
	}

	if boards[0].ID != "board1" {
		t.Errorf("expected board ID 'board1', got %q", boards[0].ID)
	}
	if boards[0].Name != "Test Board" {
		t.Errorf("expected board name 'Test Board', got %q", boards[0].Name)
	}
}

func TestClientRequestWithParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check custom params
		params := r.URL.Query()
		if params.Get("filter") != "open" {
			t.Errorf("expected filter 'open', got %q", params.Get("filter"))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	client := NewClient("test-key", "test-token")
	params := url.Values{}
	params.Set("filter", "open")

	_, err := client.request("GET", "/boards", params)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
}

func TestClientRequestError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid key"}`))
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	client := NewClient("bad-key", "bad-token")
	_, err := client.request("GET", "/boards/me", nil)
	if err == nil {
		t.Error("expected error for 401 response, got nil")
	}
}

func TestGetBoards(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/members/me/boards" {
			t.Errorf("expected path '/members/me/boards', got %q", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":     "board1",
				"name":   "Board 1",
				"closed": false,
			},
			{
				"id":     "board2",
				"name":   "Board 2",
				"closed": false,
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	client := NewClient("test-key", "test-token")
	boards, err := client.GetBoards()
	if err != nil {
		t.Fatalf("GetBoards failed: %v", err)
	}

	if len(boards) != 2 {
		t.Fatalf("expected 2 boards, got %d", len(boards))
	}

	if boards[0].Name != "Board 1" {
		t.Errorf("expected board name 'Board 1', got %q", boards[0].Name)
	}
}

func TestGetBoard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boards/board123" {
			t.Errorf("expected path '/boards/board123', got %q", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "board123",
			"name": "Single Board",
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	client := NewClient("test-key", "test-token")
	board, err := client.GetBoard("board123")
	if err != nil {
		t.Fatalf("GetBoard failed: %v", err)
	}

	if board.ID != "board123" {
		t.Errorf("expected board ID 'board123', got %q", board.ID)
	}
	if board.Name != "Single Board" {
		t.Errorf("expected board name 'Single Board', got %q", board.Name)
	}
}

func TestGetBoardLists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boards/board123/lists" {
			t.Errorf("expected path '/boards/board123/lists', got %q", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":      "list1",
				"name":    "To Do",
				"closed":  false,
				"idBoard": "board123",
				"pos":     1.0,
			},
			{
				"id":      "list2",
				"name":    "Done",
				"closed":  false,
				"idBoard": "board123",
				"pos":     2.0,
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	client := NewClient("test-key", "test-token")
	lists, err := client.GetBoardLists("board123")
	if err != nil {
		t.Fatalf("GetBoardLists failed: %v", err)
	}

	if len(lists) != 2 {
		t.Fatalf("expected 2 lists, got %d", len(lists))
	}

	if lists[0].Name != "To Do" {
		t.Errorf("expected list name 'To Do', got %q", lists[0].Name)
	}
}

func TestGetBoardCards(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boards/board123/cards" {
			t.Errorf("expected path '/boards/board123/cards', got %q", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":      "card1",
				"name":    "Card 1",
				"desc":    "Description",
				"closed":  false,
				"idBoard": "board123",
				"idList":  "list1",
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	client := NewClient("test-key", "test-token")
	cards, err := client.GetBoardCards("board123")
	if err != nil {
		t.Fatalf("GetBoardCards failed: %v", err)
	}

	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}

	if cards[0].Name != "Card 1" {
		t.Errorf("expected card name 'Card 1', got %q", cards[0].Name)
	}
}

func TestGetListCards(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lists/list123/cards" {
			t.Errorf("expected path '/lists/list123/cards', got %q", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":     "card1",
				"name":   "List Card",
				"idList": "list123",
			},
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	client := NewClient("test-key", "test-token")
	cards, err := client.GetListCards("list123")
	if err != nil {
		t.Fatalf("GetListCards failed: %v", err)
	}

	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}
}

func TestGetCard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cards/card123" {
			t.Errorf("expected path '/cards/card123', got %q", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "card123",
			"name": "Single Card",
			"desc": "Card description",
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	client := NewClient("test-key", "test-token")
	card, err := client.GetCard("card123")
	if err != nil {
		t.Fatalf("GetCard failed: %v", err)
	}

	if card.ID != "card123" {
		t.Errorf("expected card ID 'card123', got %q", card.ID)
	}
	if card.Name != "Single Card" {
		t.Errorf("expected card name 'Single Card', got %q", card.Name)
	}
}

func TestCreateCard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/cards" {
			t.Errorf("expected path '/cards', got %q", r.URL.Path)
		}

		// Check params
		params := r.URL.Query()
		if params.Get("name") != "New Card" {
			t.Errorf("expected name 'New Card', got %q", params.Get("name"))
		}
		if params.Get("idList") != "list123" {
			t.Errorf("expected idList 'list123', got %q", params.Get("idList"))
		}
		if params.Get("desc") != "Card description" {
			t.Errorf("expected desc 'Card description', got %q", params.Get("desc"))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "new123",
			"name":   params.Get("name"),
			"idList": params.Get("idList"),
			"desc":   params.Get("desc"),
		})
	}))
	defer srv.Close()

	oldURL := baseURL
	baseURL = srv.URL
	defer func() { baseURL = oldURL }()

	client := NewClient("test-key", "test-token")
	card, err := client.CreateCard("New Card", "list123", "Card description")
	if err != nil {
		t.Fatalf("CreateCard failed: %v", err)
	}

	if card.ID != "new123" {
		t.Errorf("expected card ID 'new123', got %q", card.ID)
	}
	if card.Name != "New Card" {
		t.Errorf("expected card name 'New Card', got %q", card.Name)
	}
}
