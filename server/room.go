package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/coder/websocket"
	"github.com/mpetazzoni/abalone-go/game"
)

// PlayerColor maps to game.Cell
type PlayerColor int

const (
	ColorBlack PlayerColor = PlayerColor(game.Black)
	ColorWhite PlayerColor = PlayerColor(game.White)
)

// Player represents a connected player
type Player struct {
	Conn  *websocket.Conn
	Color game.Cell
}

// Room represents an active game room
type Room struct {
	Code    string
	Game    *game.Game
	Players [2]*Player // [0] = host (Black), [1] = joiner (White)
	mu      sync.Mutex
}

// RoomManager manages all active game rooms
type RoomManager struct {
	rooms map[string]*Room
	mu    sync.RWMutex
}

// NewRoomManager creates a new room manager
func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*Room),
	}
}

// CreateRoom creates a new game room with a unique code
func (rm *RoomManager) CreateRoom() *Room {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Generate a unique code
	var code string
	for {
		code = generateGameCode()
		if _, exists := rm.rooms[code]; !exists {
			break
		}
	}

	room := &Room{
		Code: code,
		Game: game.NewGame(),
	}
	rm.rooms[code] = room
	return room
}

// GetRoom returns the room with the given code, or nil if not found
func (rm *RoomManager) GetRoom(code string) *Room {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.rooms[code]
}

// RemoveRoom removes a room from the manager
func (rm *RoomManager) RemoveRoom(code string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.rooms, code)
}

// ClientMessage represents a message from the client
type ClientMessage struct {
	Type    string  `json:"type"`
	Marbles [][]int `json:"marbles,omitempty"` // [[q,r], ...]
	Dir     []int   `json:"direction,omitempty"`
}

// ServerMessage represents a message from the server
type ServerMessage struct {
	Type     string         `json:"type"`
	Board    map[string]int `json:"board,omitempty"`
	Turn     string         `json:"turn,omitempty"`
	Captured map[string]int `json:"captured,omitempty"`
	Winner   string         `json:"winner,omitempty"`
	Code     string         `json:"code,omitempty"`
	Color    string         `json:"color,omitempty"`
	Message  string         `json:"message,omitempty"`
	Error    string         `json:"error,omitempty"`
}

func colorName(c game.Cell) string {
	switch c {
	case game.Black:
		return "black"
	case game.White:
		return "white"
	default:
		return ""
	}
}

// broadcastState prepares and sends the current game state to all connected players.
// The caller must hold room.mu, which will be released before network I/O.
func (r *Room) broadcastState() {
	board := make(map[string]int)
	for hex, cell := range r.Game.Board.Cells {
		if cell != game.Empty {
			key := fmt.Sprintf("%d,%d", hex.Q, hex.R)
			board[key] = int(cell)
		}
	}

	msg := ServerMessage{
		Type:  "state",
		Board: board,
		Turn:  colorName(r.Game.Turn),
		Captured: map[string]int{
			"black": r.Game.CapturedBlack,
			"white": r.Game.CapturedWhite,
		},
	}

	if r.Game.IsOver() {
		msg.Type = "game_over"
		msg.Winner = colorName(r.Game.Winner)
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling state: %v", err)
		r.mu.Unlock()
		return
	}

	// Collect connections under lock
	var conns []*websocket.Conn
	for _, p := range r.Players {
		if p != nil && p.Conn != nil {
			conns = append(conns, p.Conn)
		}
	}

	// Release lock before network I/O
	r.mu.Unlock()
	for _, c := range conns {
		_ = c.Write(context.Background(), websocket.MessageText, data)
	}
}

func sendJSON(conn *websocket.Conn, msg ServerMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}
	_ = conn.Write(context.Background(), websocket.MessageText, data)
}

func sendError(conn *websocket.Conn, errMsg string) {
	sendJSON(conn, ServerMessage{Type: "error", Error: errMsg})
}
