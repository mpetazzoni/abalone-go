package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/mpetazzoni/abalone-go/game"
)

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Allow all origins for local development
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("WebSocket accept error: %v", err)
		return
	}
	defer conn.CloseNow()

	action := r.URL.Query().Get("action")
	code := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("code")))

	switch action {
	case "create":
		s.handleCreate(conn)
	case "join":
		if code == "" {
			sendError(conn, "Game code is required")
			return
		}
		s.handleJoin(conn, code)
	default:
		sendError(conn, "Invalid action. Use 'create' or 'join'.")
	}
}

func (s *Server) handleCreate(conn *websocket.Conn) {
	room := s.rooms.CreateRoom()
	player := &Player{Conn: conn, Color: game.Black}
	room.Players[0] = player

	log.Printf("Game created: %s", room.Code)

	// Tell the host their game code and color
	sendJSON(conn, ServerMessage{
		Type:  "waiting",
		Code:  room.Code,
		Color: "black",
	})

	// Wait for messages (the join will trigger the game start)
	s.readLoop(conn, room, player)
}

func (s *Server) handleJoin(conn *websocket.Conn, code string) {
	room := s.rooms.GetRoom(code)
	if room == nil {
		sendError(conn, fmt.Sprintf("Game '%s' not found", code))
		return
	}

	room.mu.Lock()
	if room.Players[1] != nil {
		room.mu.Unlock()
		sendError(conn, "Game is already full")
		return
	}

	player := &Player{Conn: conn, Color: game.White}
	room.Players[1] = player
	host := room.Players[0]
	room.mu.Unlock()

	log.Printf("Player joined game: %s", room.Code)

	// Tell joiner their color
	sendJSON(conn, ServerMessage{
		Type:  "joined",
		Color: "white",
	})

	// Tell host the opponent joined
	if host != nil && host.Conn != nil {
		sendJSON(host.Conn, ServerMessage{
			Type:  "joined",
			Color: "black",
		})
	}

	// Send initial board state to both
	room.mu.Lock()
	room.broadcastState() // unlocks room.mu internally

	// Enter read loop for this player
	s.readLoop(conn, room, player)
}

func (s *Server) readLoop(conn *websocket.Conn, room *Room, player *Player) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Heartbeat: ping every 30s to detect stale connections
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.Ping(ctx); err != nil {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			log.Printf("Player disconnected from %s: %v", room.Code, err)
			room.mu.Lock()
			// Clear this player's slot
			for i, p := range room.Players {
				if p == player {
					room.Players[i] = nil
					break
				}
			}
			// Notify remaining player
			for _, p := range room.Players {
				if p != nil && p.Conn != nil {
					sendJSON(p.Conn, ServerMessage{
						Type:    "opponent_disconnected",
						Message: "Your opponent has disconnected",
					})
				}
			}
			// Check if room is empty
			bothNil := room.Players[0] == nil && room.Players[1] == nil
			code := room.Code
			room.mu.Unlock()

			if bothNil {
				s.rooms.RemoveRoom(code)
				log.Printf("Room %s removed (empty)", code)
			}
			return
		}

		var msg ClientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			sendError(conn, "Invalid message format")
			continue
		}

		switch msg.Type {
		case "move":
			s.handleMove(room, player, msg)
		default:
			sendError(conn, fmt.Sprintf("Unknown message type: %s", msg.Type))
		}
	}
}

func (s *Server) handleMove(room *Room, player *Player, msg ClientMessage) {
	room.mu.Lock()

	// Verify it's this player's turn
	if room.Game.Turn != player.Color {
		room.mu.Unlock()
		sendError(player.Conn, "Not your turn")
		return
	}

	// Parse marbles
	marbles := make([]game.Hex, len(msg.Marbles))
	for i, m := range msg.Marbles {
		if len(m) != 2 {
			room.mu.Unlock()
			sendError(player.Conn, "Invalid marble coordinates")
			return
		}
		marbles[i] = game.Hex{Q: m[0], R: m[1]}
	}

	// Parse direction
	if len(msg.Dir) != 2 {
		room.mu.Unlock()
		sendError(player.Conn, "Invalid direction")
		return
	}
	dir := game.Hex{Q: msg.Dir[0], R: msg.Dir[1]}

	// Apply the move
	err := game.ApplyMove(room.Game, marbles, dir)
	if err != nil {
		room.mu.Unlock()
		sendError(player.Conn, err.Error())
		return
	}

	// broadcastState unlocks room.mu internally
	room.broadcastState()
}
