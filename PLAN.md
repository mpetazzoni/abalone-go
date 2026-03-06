# Abalone-go, an Abalone game in Golang

Build a playable version of the game Abalone, the hexagonal marble-pushing
strategy game. Web-based, playable by two players over the network. One
player (white) becomes the game's host, and the other player connects to it.

---

## Game Rules (Summary)

- **Board**: Hexagonal, 61 cells arranged in a hex grid (rows of 5-6-7-8-9-8-7-6-5).
- **Pieces**: 14 black marbles, 14 white marbles. Black moves first.
- **Objective**: Push 6 of your opponent's marbles off the board.
- **Moves**:
  - **In-line**: Move 1, 2, or 3 marbles along their shared axis into an adjacent free space.
  - **Broadside (side-step)**: Move 2 or 3 marbles sideways in unison to adjacent free spaces.
- **Sumito (pushing)**: When your in-line column faces a *smaller* number of opponent marbles on the same axis, you push them. 3v2, 3v1, 2v1 are legal. Pushing is only allowed in-line, never broadside.
- **Push-off**: If a pushed marble has no space behind it (board edge), it's removed from play.
- **Win**: First player to push off 6 opponent marbles wins.

---

## Architecture

### High-level Design

```
┌──────────────┐         WebSocket          ┌──────────────┐
│   Browser     │◄────────────────────────►  │   Go Server  │
│   (HTML/JS)   │    JSON messages           │              │
│               │                            │  Game Engine │
│  Canvas/SVG   │                            │  Room Mgmt   │
│  board render │                            │  WebSocket   │
└──────────────┘                            └──────────────┘
```

### Components

1. **Game Engine** (`game/`) — Pure game logic, no I/O
   - Board representation (axial hex coordinates)
   - Move validation (in-line, broadside, sumito)
   - Game state (whose turn, captured counts, win condition)
   - Move generation (for future AI, optional)

2. **Server** (`server/`) — HTTP + WebSocket
   - Room creation (host generates a game code)
   - Player join (second player enters code)
   - WebSocket upgrade and message routing
   - Game lifecycle (waiting → playing → finished)

3. **Frontend** (`web/`) — Single-page HTML/JS/CSS
   - Hexagonal board rendering (SVG or Canvas)
   - Click-to-select, click-to-move interaction
   - WebSocket client for real-time updates
   - Game status display (turn indicator, captured count)

4. **Main** (`main.go`) — Wires everything together, starts HTTP server

### Hex Coordinate System

Using **axial coordinates** (q, r) for the hex grid. The board is a hex
with radius 4 (center at 0,0), so valid cells satisfy: `|q| <= 4`,
`|r| <= 4`, and `|q + r| <= 4`. This gives us exactly 61 cells.

Six directions in axial coordinates:
```
E  = (+1,  0)    W  = (-1,  0)
NE = (+1, -1)    SW = (-1, +1)
NW = ( 0, -1)    SE = ( 0, +1)
```

### Message Protocol (WebSocket JSON)

**Client → Server:**
```json
{ "type": "move", "marbles": [[q,r], ...], "direction": [dq, dr] }
```

**Server → Client:**
```json
{ "type": "state", "board": {...}, "turn": "black", "captured": {"black": 0, "white": 0} }
{ "type": "error", "message": "Invalid move" }
{ "type": "game_over", "winner": "white" }
{ "type": "waiting", "code": "swift-volcano" }
{ "type": "joined", "color": "black" }
```

---

## Decisions

| # | Decision | Choice | Rationale |
|---|----------|--------|-----------|
| 1 | Hex coordinate system | Axial (q, r) | Simple, well-documented, easy neighbor math |
| 2 | Frontend rendering | SVG | Easier click handling than Canvas, scales well |
| 3 | WebSocket library | `github.com/coder/websocket` (nhooyr) | Modern, maintained, context-aware API |
| 4 | Frontend framework | Vanilla JS | No build step, keep it simple |
| 5 | Board storage | `map[[2]int]Cell` | Sparse map, natural for hex grids |
| 6 | Starting position | Standard (not Belgian Daisy) | Classic layout, simplest to implement first |
| 7 | Marble selection UX | Click marbles → click direction arrow | Explicit, easy to implement; revisit after testing |
| 8 | Frontend embedding | `//go:embed` | Single binary deployment, no external files |
| 9 | Game codes | Two random words (adjective+noun) | Fun, memorable ("angry-panda", "swift-volcano") |

---

## Todo

- [x] **Phase 1: Game Engine**
  - [x] Define board types and hex coordinate helpers
  - [x] Implement board initialization (standard starting position)
  - [x] Implement move validation (in-line moves)
  - [x] Implement move validation (broadside moves)
  - [x] Implement sumito (pushing) logic
  - [x] Implement push-off detection
  - [x] Implement win condition check
  - [x] Write comprehensive unit tests for all move types
- [x] **Phase 2: Server**
  - [x] Set up Go module and project structure
  - [x] Implement room/lobby system (create game, join game)
  - [x] WebSocket connection handling
  - [x] Wire game engine to WebSocket message handler
  - [x] Handle disconnection/reconnection gracefully
- [x] **Phase 3: Frontend**
  - [x] HTML page with SVG hex board
  - [x] Render board state from server
  - [x] Marble selection UI (click to select 1-3 marbles)
  - [x] Direction selection / move submission
  - [x] Turn indicator, captured marble display
  - [x] Game over screen
  - [x] Join/host lobby UI
- [x] **Phase 4: Polish**
  - [x] Move animation
  - [x] Sound effects (optional)
  - [x] Mobile-friendly layout
  - [x] Deployment (single binary serves everything)

---

## Resolved Questions

1. **WebSocket library** → `github.com/coder/websocket` (nhooyr). Modern,
   maintained, good `context.Context` integration.

2. **Marble selection UX** → Click marbles, then click direction arrow.
   Explicit and straightforward. We'll revisit after playtesting.

3. **Embedded frontend** → Yes, `//go:embed`. Single binary deployment.

4. **Game code format** → Two random words, adjective+noun
   (e.g. "angry-panda", "swift-volcano"). Small word lists (~100 each)
   give ~10K combos, plenty for concurrent games.

---

## Implementation Notes

### Game Code Word Lists

Embed two small word lists in the server:
- ~100 adjectives: swift, angry, lazy, clever, bold, fuzzy, ...
- ~100 nouns: panda, volcano, falcon, marble, comet, fortress, ...

Format: `adjective-noun` (lowercase, hyphenated).

### Directory Structure

```
abalone-go/
├── main.go              # Entry point, wires server
├── go.mod
├── go.sum
├── PLAN.md
├── game/
│   ├── board.go         # Board type, hex coords, cell state
│   ├── game.go          # Game state, turn management, win check
│   ├── move.go          # Move types, validation, execution
│   └── game_test.go     # Tests for all game logic
├── server/
│   ├── server.go        # HTTP routes, static files
│   ├── room.go          # Room/lobby management, game codes
│   ├── ws.go            # WebSocket upgrade, message handling
│   └── wordlist.go      # Adjective/noun lists for game codes
└── web/
    ├── index.html        # Single page app
    ├── style.css         # Board + UI styling
    └── app.js            # Game client, SVG rendering, WebSocket
```
