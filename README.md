# GoChatTUI

A real-time, terminal-based 1-on-1 chat application written in Go: a WebSocket server that brokers messages between clients, and a terminal client with a hand-rolled ANSI UI — no TUI framework, no external renderer. Includes a load-testing harness that drives hundreds of concurrent simulated users against the server.

```
┌─────────────────────────────────────────┐
│  GoChatTUI (v1.0) - Logged in as: yash  │
├─────────────────────────────────────────┤
│  [ Unread ]   Online     Offline        │
├─────────────────────────────────────────┤
│ ▶ alex          (3)                     │
│   sam           (1)                     │
│                                         │
├─────────────────────────────────────────┤
│ ← → Switch tabs   ↑ ↓ Move   Enter: Open│
└─────────────────────────────────────────┘
```

## Why this project

Most terminal chat demos lean on a TUI framework (Bubble Tea, tcell) to get rendering and input handling for free. This one doesn't: the screen renderer, raw-mode keyboard input (including parsing multi-byte ANSI escape sequences for arrow keys), and terminal resize detection are all built directly on `golang.org/x/term` raw mode. The goal was to understand — and own — the full path from a keypress to a rendered frame, and to build a server that handles concurrent connections correctly using nothing but goroutines and channels (no mutexes).

## Features

- **Direct messaging** — 1:1 chats between users, no rooms or groups
- **Presence tracking** — server broadcasts the connected-user list on every join/leave; client splits contacts into Unread / Online / Offline tabs
- **Unread counts** — per-user unread badges, cleared on opening a conversation
- **Scrollable message history** — viewport scrolling sized to the current terminal height, recalculated live on resize
- **Local persistence** — each client caches its chat history as JSON under `~/.goChatTUIClient/<username>`, so history survives restarts (the server itself is a stateless, in-memory relay — nothing is stored server-side)
- **Alt-screen, raw-mode terminal UI** — clean enter/exit without trashing the user's shell
- **Cross-platform client builds** — Linux, macOS (amd64 + arm64), and Windows, all cross-compiled from a single build script

## Architecture

```
                       ┌────────────────────┐
                       │   Server (Broker)  │
                       │  single-goroutine  │
   ws://.../ws         │  actor over        │
  ┌──────────────►     │  buffered channels │
  │                    └─────────┬──────────┘
  │  Client A                    │  Client B
  │  (TUI, raw mode)             │  (TUI, raw mode)
  └──────────────────────────────┘
```

**Server** (`internal/server`) — a single-goroutine "actor" (`Broker`) owns a `map[string]User` and serializes every mutation through a `select` over four channels: join requests, inbound messages, kick-outs, and stop. This avoids locks entirely — the classic Go "share memory by communicating" pattern. Each connected user gets two dedicated goroutines: one blocking on `conn.ReadJSON` to feed the broker, one draining a buffered per-user outbox and writing to the socket with a write deadline, so one slow/dead client can't stall the others.

**Client** (`internal/client`) — an event loop selecting over three channels fed by background goroutines: raw keypress events (parsed byte-by-byte, including a short read deadline to distinguish a lone `ESC` from an arrow-key escape sequence), incoming WebSocket messages, and terminal-resize polling. All UI state mutation happens on a single goroutine, so the renderer never races with the network.

**Wire protocol** (`internal/protocol`) — JSON over WebSocket, three message shapes: `ClaimUsernameRequest` (must arrive within 5s of connecting or the server disconnects you), `ForwardMessageRequest` (client → server, "send this to X"), and `Message` (server → client), which carries either a `CHAT` payload or a `BROADCAST` presence update.

## Load testing

`loadtest.sh` spins up a configurable ring of simulated users (`testclient`), each sending timed messages to the next user in the ring, staggered by a ramp-up delay to avoid a thundering herd:

```bash
./loadtest.sh <users> <duration_s> <interval_ms> <ramp_ms> <log_dir>
./loadtest.sh 100 60 200 20 logs
```

Each simulated client tracks sent/received/error counts with `sync/atomic` and writes a per-client report; logs land in `logs/<user>.log`. It's a real mechanism for exercising the broker's concurrent connection handling and message routing under load, not just a toy script.

## Getting started

**Requires:** Go 1.25+

### Run it locally

```bash
# terminal 1 — start the server (listens on localhost:8123)
go run cmd/server/main.go

# terminal 2 — client A
go run cmd/client/main.go localhost:8123 alex

# terminal 3 — client B
go run cmd/client/main.go localhost:8123 sam
```

Open a chat from the main screen (arrow keys + Enter), type a message, press Enter to send. `Ctrl+C` backs out of a chat, and quits from the main screen.

### Build everything

```bash
./build.sh
```

Produces native `server` and `testclient` binaries plus cross-compiled `client` binaries for `linux-amd64`, `darwin-amd64`, `darwin-arm64`, and `windows-amd64` in `build/`.

## Project layout

```
cmd/
├── server/       # entry point: HTTP → WebSocket upgrade, wires up the Broker
├── client/       # entry point: raw-mode TUI client
└── testclient/   # entry point: headless load-test client (flags: -user -to -duration -interval)
internal/
├── protocol/     # shared wire types: ClaimUsernameRequest, ForwardMessageRequest, Message
├── server/       # Broker (actor/channel router) + WebSocket handlers
└── client/       # WebSocket dialer, UI state machine, ANSI renderer, raw keyboard input,
                   # terminal setup/resize, local JSON persistence
```

## Known limitations

- No authentication beyond claiming a username (first-come, first-served; no passwords)
- No TLS — `ws://` only
- No server-side persistence — message history lives only on clients
- No automated test suite yet; correctness under load is currently verified via `loadtest.sh`
