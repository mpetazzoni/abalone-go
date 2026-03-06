# Contributing

Thanks for your interest in contributing to Abalone!

## Prerequisites

- **Go 1.25+**
- **[pre-commit](https://pre-commit.com/)** for git hooks

## Getting Started

```sh
git clone https://github.com/mpetazzoni/abalone-go.git
cd abalone-go
pre-commit install
make build
```

## Development Workflow

```sh
make run          # build and start the server on :8080
make test         # run all tests
make test-v       # run tests with verbose output
make lint         # check formatting + go vet
make fmt          # auto-format Go files
make help         # show all make targets
```

## Project Layout

| Directory | What lives there |
|-----------|-----------------|
| `game/` | Pure game logic — board, moves, validation. No I/O. |
| `server/` | HTTP server, WebSocket handling, room management. |
| `web/` | Frontend — vanilla HTML/CSS/JS, SVG board. Embedded via `go:embed`. |

## Running Tests

The project has two test suites:

- **`game/game_test.go`** — unit tests for move validation, board logic, win conditions
- **`server/server_test.go`** — end-to-end WebSocket tests simulating real game flows

```sh
make test
```

## Code Style

- Standard `gofmt` formatting (enforced by pre-commit)
- `go vet` for static analysis (enforced by pre-commit)
- Keep the game engine (`game/`) free of I/O and external dependencies
- Frontend is vanilla JS — no build step, no frameworks

## Commits

Follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` new features
- `fix:` bug fixes
- `test:` adding or updating tests
- `docs:` documentation changes
- `refactor:` code changes that don't add features or fix bugs

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
