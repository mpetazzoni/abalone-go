# Abalone

A web-based implementation of the classic hexagonal marble-pushing strategy board game, playable by two players over the network.

## What is Abalone?

[Abalone](https://en.wikipedia.org/wiki/Abalone_(board_game)) is a two-player strategy game played on a hexagonal board with 61 cells. Each player controls 14 marbles and takes turns pushing them around the board. The goal: push 6 of your opponent's marbles off the edge. Larger groups can push smaller ones (3v2, 3v1, 2v1), so positioning and coordination are everything.

## Features

- **Two-player network play** — one player hosts, the other joins with a code
- **Real-time WebSocket communication** — instant move updates, no polling
- **Fun game codes** — memorable words like `swift-volcano` or `angry-panda`
- **SVG hex board** — clean, scalable rendering with click-to-select interaction
- **Single binary deployment** — frontend is embedded; just run the binary

## 🤖 AI-Built

This project is entirely AI/vibe-coded. Contributions are welcome — and will also be reviewed by AI.

## Quick Start

```sh
make build
make run               # starts on :8080
./abalone -addr :3000  # custom port
```

Open `http://localhost:8080` in your browser. One player clicks **Host Game** to create a room and get a game code. The other player enters the code and clicks **Join Game**. That's it — you're playing.

### Development

```sh
make test              # run all tests
make lint              # check formatting + go vet
make help              # show all make targets
```

## How to Play

1. **Select 1–3 of your marbles** by clicking them
2. **Click a direction arrow** to move or push
3. Moves can be **in-line** (along the axis of your group) or **broadside** (sideways)
4. Push opponent marbles off the board when you outnumber them on an axis
5. First to push off **6 opponent marbles** wins

## Tech Stack

- **Go** — game engine, server, room management
- **Vanilla JS** — frontend with SVG board rendering
- [`github.com/coder/websocket`](https://github.com/coder/websocket) — WebSocket library
- `go:embed` — frontend files embedded in the binary

## Project Structure

```
abalone-go/
├── main.go              # Entry point, embeds frontend, starts server
├── game/
│   ├── board.go         # Board types, hex coordinates, initial position
│   ├── game.go          # Game state, turns, win condition
│   ├── move.go          # Move validation and execution
│   └── game_test.go     # Tests for all game logic
├── server/
│   ├── server.go        # HTTP routes, static file serving
│   ├── room.go          # Room/lobby management, game codes
│   ├── ws.go            # WebSocket upgrade and message handling
│   ├── wordlist.go      # Word lists for game code generation
│   └── server_test.go   # End-to-end WebSocket tests
└── web/
    ├── index.html       # Single page app
    ├── style.css        # Board and UI styling
    └── app.js           # Game client, SVG rendering, WebSocket
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

## License

[MIT](LICENSE)
