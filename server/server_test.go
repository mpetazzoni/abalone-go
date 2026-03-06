package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// --- Helpers ---

func setupServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(NewServer(http.Dir(t.TempDir())))
	t.Cleanup(srv.Close)
	return srv
}

func wsConnect(t *testing.T, srv *httptest.Server, action, code string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	url := "ws" + srv.URL[len("http"):] + "/ws?action=" + action
	if code != "" {
		url += "&code=" + code
	}

	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("wsConnect: dial failed: %v", err)
	}
	t.Cleanup(func() { conn.CloseNow() })
	return conn
}

func readMsg(t *testing.T, conn *websocket.Conn) ServerMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("readMsg: %v", err)
	}

	var msg ServerMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("readMsg: unmarshal: %v", err)
	}
	return msg
}

func sendMove(t *testing.T, conn *websocket.Conn, marbles [][]int, dir []int) {
	t.Helper()
	msg := ClientMessage{Type: "move", Marbles: marbles, Dir: dir}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("sendMove: marshal: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatalf("sendMove: write: %v", err)
	}
}

// createAndJoin sets up a full two-player game and drains the initial
// handshake messages (waiting, joined x2, state x2), returning both
// connections ready for move tests.
func createAndJoin(t *testing.T, srv *httptest.Server) (host, joiner *websocket.Conn, code string) {
	t.Helper()

	// Host creates a game
	host = wsConnect(t, srv, "create", "")
	waiting := readMsg(t, host)
	if waiting.Type != "waiting" {
		t.Fatalf("createAndJoin: expected 'waiting', got %q", waiting.Type)
	}
	code = waiting.Code

	// Joiner joins
	joiner = wsConnect(t, srv, "join", code)

	// Joiner gets "joined"
	joinerJoined := readMsg(t, joiner)
	if joinerJoined.Type != "joined" {
		t.Fatalf("createAndJoin: joiner expected 'joined', got %q", joinerJoined.Type)
	}

	// Host gets "joined"
	hostJoined := readMsg(t, host)
	if hostJoined.Type != "joined" {
		t.Fatalf("createAndJoin: host expected 'joined', got %q", hostJoined.Type)
	}

	// Both get initial "state"
	joinerState := readMsg(t, joiner)
	if joinerState.Type != "state" {
		t.Fatalf("createAndJoin: joiner expected 'state', got %q", joinerState.Type)
	}
	hostState := readMsg(t, host)
	if hostState.Type != "state" {
		t.Fatalf("createAndJoin: host expected 'state', got %q", hostState.Type)
	}

	return host, joiner, code
}

// --- Tests ---

func TestCreateGame(t *testing.T) {
	srv := setupServer(t)
	conn := wsConnect(t, srv, "create", "")

	msg := readMsg(t, conn)
	if msg.Type != "waiting" {
		t.Errorf("expected type 'waiting', got %q", msg.Type)
	}
	if msg.Code == "" {
		t.Error("expected non-empty game code")
	}
	if msg.Color != "black" {
		t.Errorf("expected color 'black', got %q", msg.Color)
	}
}

func TestJoinGame(t *testing.T) {
	srv := setupServer(t)

	// Host creates
	host := wsConnect(t, srv, "create", "")
	waiting := readMsg(t, host)
	code := waiting.Code

	// Joiner joins
	joiner := wsConnect(t, srv, "join", code)

	// Joiner gets "joined" with color "white"
	joinerJoined := readMsg(t, joiner)
	if joinerJoined.Type != "joined" {
		t.Errorf("joiner: expected type 'joined', got %q", joinerJoined.Type)
	}
	if joinerJoined.Color != "white" {
		t.Errorf("joiner: expected color 'white', got %q", joinerJoined.Color)
	}

	// Host gets "joined" with color "black"
	hostJoined := readMsg(t, host)
	if hostJoined.Type != "joined" {
		t.Errorf("host: expected type 'joined', got %q", hostJoined.Type)
	}
	if hostJoined.Color != "black" {
		t.Errorf("host: expected color 'black', got %q", hostJoined.Color)
	}

	// Both receive "state" with turn "black" and 28 cells
	joinerState := readMsg(t, joiner)
	if joinerState.Type != "state" {
		t.Errorf("joiner: expected type 'state', got %q", joinerState.Type)
	}
	if joinerState.Turn != "black" {
		t.Errorf("joiner: expected turn 'black', got %q", joinerState.Turn)
	}
	if len(joinerState.Board) != 28 {
		t.Errorf("joiner: expected 28 board cells, got %d", len(joinerState.Board))
	}

	hostState := readMsg(t, host)
	if hostState.Type != "state" {
		t.Errorf("host: expected type 'state', got %q", hostState.Type)
	}
	if hostState.Turn != "black" {
		t.Errorf("host: expected turn 'black', got %q", hostState.Turn)
	}
	if len(hostState.Board) != 28 {
		t.Errorf("host: expected 28 board cells, got %d", len(hostState.Board))
	}
}

func TestJoinInvalidCode(t *testing.T) {
	srv := setupServer(t)
	conn := wsConnect(t, srv, "join", "nonexistent")

	msg := readMsg(t, conn)
	if msg.Type != "error" {
		t.Errorf("expected type 'error', got %q", msg.Type)
	}
	if msg.Error == "" {
		t.Error("expected non-empty error message")
	}
}

func TestJoinFullGame(t *testing.T) {
	srv := setupServer(t)
	_, _, code := createAndJoin(t, srv)

	// Third player tries to join the same code
	third := wsConnect(t, srv, "join", code)
	msg := readMsg(t, third)
	if msg.Type != "error" {
		t.Errorf("expected type 'error', got %q", msg.Type)
	}
	if msg.Error == "" {
		t.Error("expected non-empty error message")
	}
}

func TestInvalidAction(t *testing.T) {
	srv := setupServer(t)
	conn := wsConnect(t, srv, "bogus", "")

	msg := readMsg(t, conn)
	if msg.Type != "error" {
		t.Errorf("expected type 'error', got %q", msg.Type)
	}
	if msg.Error == "" {
		t.Error("expected non-empty error message")
	}
}

func TestValidMove(t *testing.T) {
	srv := setupServer(t)
	host, joiner, _ := createAndJoin(t, srv)

	// Black (host) moves marble at (0,-2) in direction SE (0,+1)
	sendMove(t, host, [][]int{{0, -2}}, []int{0, 1})

	// Both receive new state with turn "white"
	hostState := readMsg(t, host)
	if hostState.Type != "state" {
		t.Errorf("host: expected type 'state', got %q", hostState.Type)
	}
	if hostState.Turn != "white" {
		t.Errorf("host: expected turn 'white', got %q", hostState.Turn)
	}

	joinerState := readMsg(t, joiner)
	if joinerState.Type != "state" {
		t.Errorf("joiner: expected type 'state', got %q", joinerState.Type)
	}
	if joinerState.Turn != "white" {
		t.Errorf("joiner: expected turn 'white', got %q", joinerState.Turn)
	}
}

func TestWrongTurnMove(t *testing.T) {
	srv := setupServer(t)
	_, joiner, _ := createAndJoin(t, srv)

	// White (joiner) tries to move first — it's Black's turn
	sendMove(t, joiner, [][]int{{0, 2}}, []int{0, -1})

	msg := readMsg(t, joiner)
	if msg.Type != "error" {
		t.Errorf("expected type 'error', got %q", msg.Type)
	}
	if msg.Error == "" {
		t.Error("expected non-empty error about turn")
	}
}

func TestInvalidMove(t *testing.T) {
	srv := setupServer(t)
	host, _, _ := createAndJoin(t, srv)

	// Black tries to move an empty cell (0,0 is empty at start)
	sendMove(t, host, [][]int{{0, 0}}, []int{1, 0})

	msg := readMsg(t, host)
	if msg.Type != "error" {
		t.Errorf("expected type 'error', got %q", msg.Type)
	}
	if msg.Error == "" {
		t.Error("expected non-empty error message")
	}
}

func TestDisconnectNotification(t *testing.T) {
	srv := setupServer(t)
	host, joiner, _ := createAndJoin(t, srv)

	// Joiner disconnects
	joiner.Close(websocket.StatusNormalClosure, "bye")

	// Host should receive opponent_disconnected
	msg := readMsg(t, host)
	if msg.Type != "opponent_disconnected" {
		t.Errorf("expected type 'opponent_disconnected', got %q", msg.Type)
	}
	if msg.Message == "" {
		t.Error("expected non-empty disconnect message")
	}
}
