# Sync

A real-time TCP chat server and client written in Go from scratch.

## Project Overview

Sync is a multi-client chat system that runs over raw TCP. It supports username-based identity, session tokens for reconnection, message history with WAL persistence, direct messages, and user listings. The project is currently in early development and **not yet compilable** (see [Known Issues](#known-issues)).

### Current Status

| Aspect | Status |
|---|---|
| Server | Mostly implemented, 1 bug prevents compilation |
| Client | Functional entry point |
| Persistence | WAL + snapshot implemented |
| Tests | None |
| Build | **Broken** (`handleCommand` missing) |
| Docker | Not yet available |

## Features

- **Multi-client TCP chat** — concurrent clients via goroutines
- **Username registration** — with `Guest<random>` fallback
- **Direct messages** — `/msg <user> <message>`
- **User listing** — `/users` with idle status
- **Message history** — `/history [N]` (last N messages)
- **Session tokens** — reconnect with `reconnect:<user>:<token>`
- **WAL persistence** — messages appended to `messages.wal`
- **Snapshot persistence** — periodic (5 min) JSON snapshots to `snapshot.json`
- **Crash recovery** — restores state from WAL/snapshot on startup
- **Graceful shutdown** — SIGINT/SIGTERM handling
- **Inactive client cleanup** — 30s tick, 5 min timeout
- **Read timeouts** — 30s for username, 5 min for messages
- **Panic recovery** — per-goroutine `recover()` guards

## Getting Started

### Prerequisites

- Go 1.25 or later

### Build

```bash
# Build both server and client
go build ./...

# Build individually
go build ./cmd/server
go build ./cmd/client
```

### Run

```bash
# Start the server (listens on :9000)
./server

# Connect as a client (in another terminal)
./client
```

## Usage

### Server

Start the server with no arguments:

```bash
./server
```

The server listens on `:9000` by default and stores chat data in `./chatdata/`.

### Client

Connect to the server:

```bash
./client
```

Once connected, you'll be prompted for a username. The following commands are available in the chat:

| Command | Description |
|---|---|
| `/msg <user> <message>` | Send a direct message |
| `/users` | List connected users |
| `/history [N]` | Show last N messages |
| Any other text | Broadcast to all users |

### Reconnecting

If you disconnect, you can reconnect with your previous identity:

```
reconnect:<username>:<token>
```

Your session token is displayed when you first connect.

## Project Structure

```
├── cmd/
│   ├── server/main.go       # Server entry point
│   └── client/main.go       # Client entry point
├── internal/
│   └── room/
│       ├── types.go         # Core types (Message, Client, Room, etc.)
│       ├── server.go        # Room init, Run() event loop, shutdown
│       ├── handler.go       # Message handling (broadcast, DM, history)
│       ├── io.go            # TCP I/O, command parsing
│       ├── client.go        # Interactive TCP client
│       ├── session.go       # Session creation, reconnection tokens
│       ├── persist.go       # WAL + snapshot persistence
│       └── startServer.go   # StartServer() wrapper
├── pkg/
│   └── token/
│       └── token.go         # Crypto-random hex token generator
├── go.mod
├── plan.md                  # Future roadmap
└── README.md
```

## Architecture

### Event Loop

The `Room.Run()` method runs a central event loop that processes messages from connected clients and handles channel operations sequentially in a single goroutine, avoiding concurrency issues in message processing.

### Persistence

Messages are written to a Write-Ahead Log (`messages.wal`) as JSON lines. Every 100 messages (or at shutdown), a snapshot is saved to `snapshot.json`. On startup, the server loads the most recent snapshot and replays any remaining WAL entries.

### Sessions

Each client gets a cryptographically random 32-character hex token on connect. Sessions have a 1-hour TTL and are tracked with activity timestamps.

## Known Issues

The project currently **does not compile** and has several known bugs:

- **Missing `handleCommand`** — `io.go:169` calls a function that was never defined
- **`sendHistory` infinite loop** — `handler.go:106` uses `1` instead of `i` in loop condition
- **Wrong mutex in `sendHistory`** — locks `room.mu` but unlocks `room.messageMu`
- **Duplicate username not rejected** — execution continues after writing error
- **`findUserbyUsername` assignment bug** — `=` used instead of `==` in comparison
- **`cmd/server/main.go` calls `StartClient`** — should call `StartServer()`
- **Typo inconsistencies** — `Recieved`, `persistance`, `actvity`, etc. throughout

See `plan.md` for the full roadmap, which includes fixes for all these issues in Phase 1.

## Roadmap

The project has a detailed 10-phase roadmap in `plan.md`. Key milestones:

1. **Phase 1** — Fix compilation, add config, tests, Docker
2. **Phase 2** — Multiple transports (WebSocket, SSE)
3. **Phase 3** — Channel-based pub/sub
4. **Phase 4** — Client SDKs (JS/TS, Go)
5. **Phase 5** — Resilience & recovery
6. **Phase 6** — Horizontal scaling with Redis
7. **Phase 7** — Observability & operations
8. **Phase 8** — E2E encryption, WASM plugins, WebRTC, federation
9. **Phase 9** — CLI tool, CI/CD, documentation site
10. **Phase 10** — Performance, enterprise features

## Dependencies

**Zero external dependencies.** The project uses only the Go standard library (`net`, `bufio`, `sync`, `crypto/rand`, `encoding/json`, `os/signal`, etc.).

## License

This project is not yet licensed. See `plan.md` for future plans.
