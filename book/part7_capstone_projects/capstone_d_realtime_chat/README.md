# Capstone D — Real-Time Chat System

## What It Builds

A fully concurrent, in-process real-time chat engine that demonstrates every
pattern you need before wiring real WebSockets to a production hub:

- **Chat rooms** — isolated broadcast domains; users subscribe and unsubscribe
  independently per room.
- **WebSocket hub** (simulated with channels) — a central `Hub` that owns all
  rooms, routes messages, and enforces lifecycle rules.
- **Message broadcast** — every message sent to a room is delivered concurrently
  to every current subscriber via per-user goroutines.
- **User presence** — a `PresenceTracker` records online/offline state with
  timestamps; any component can query `IsOnline(userID)`.
- **Message history** — each room retains the last 50 messages; late joiners can
  call `History(roomID)` to catch up.
- **Reconnection backoff** (simulated) — exponential backoff helper shown in the
  simulation to model client reconnect logic.

---

## Architecture Diagram

```
 ┌────────────────────────────────────────────────────────────┐
 │                          Hub                               │
 │                                                            │
 │   JoinRoom(userID, roomID)   LeaveRoom(userID, roomID)    │
 │   Send(userID, roomID, body) RoomMembers(roomID)          │
 │                                                            │
 │  ┌──────────────────────┐  ┌──────────────────────┐       │
 │  │       Room A         │  │       Room B         │       │
 │  │  ┌───────────────┐   │  │  ┌───────────────┐   │       │
 │  │  │  subscribers  │   │  │  │  subscribers  │   │       │
 │  │  │  user1 ──► ch │   │  │  │  user2 ──► ch │   │       │
 │  │  │  user2 ──► ch │   │  │  │  user3 ──► ch │   │       │
 │  │  └───────────────┘   │  │  └───────────────┘   │       │
 │  │  history [50 msgs]   │  │  history [50 msgs]   │       │
 │  └──────────────────────┘  └──────────────────────┘       │
 │                                                            │
 │  PresenceTracker                                           │
 │  ┌─────────────────────────────────────────┐              │
 │  │  user1: online  user2: online           │              │
 │  │  user3: offline                         │              │
 │  └─────────────────────────────────────────┘              │
 └────────────────────────────────────────────────────────────┘
          │                │                │
       user1            user2            user3
    goroutine         goroutine        goroutine
   (subscriber)     (subscriber)    (unsubscribed)
```

---

## Key Components

| Component          | Responsibility                                          | Book Chapter(s)        |
|--------------------|---------------------------------------------------------|------------------------|
| `Message`          | Immutable value type for a single chat event            | Ch 10 — Structs        |
| `Room`             | Broadcast domain; manages subscriber channels          | Ch 30 — Channels       |
| `Hub`              | Multi-room router; single source of truth for rooms    | Ch 31 — Select / Fan   |
| `PresenceTracker`  | Concurrent online/offline map with `sync.RWMutex`      | Ch 32 — sync primitives|
| History ring       | Fixed-size circular append (last 50 messages)          | Ch 12 — Slices         |
| Reconnect backoff  | Exponential sleep sequence for reconnect simulation    | Ch 33 — Timers         |
| Goroutine delivery | Per-subscriber goroutine; non-blocking send with select | Ch 29 — Goroutines     |

---

## Running

```bash
# From the repo root
cd /home/achyut-pal/Desktop/upskill-go/book
go run ./part7_capstone_projects/capstone_d_realtime_chat/

# Build only
go build ./part7_capstone_projects/capstone_d_realtime_chat/...
```

Expected output is a structured simulation log showing users joining rooms,
messages being delivered, presence changes, and history replay.

---

## What This Capstone Tests

| Skill                        | How It Is Exercised                                                   |
|------------------------------|-----------------------------------------------------------------------|
| Goroutine fan-out            | `Broadcast` spawns one goroutine per subscriber for concurrent send  |
| Channel lifecycle            | `Subscribe` creates a buffered channel; `Unsubscribe` closes it      |
| `sync.RWMutex` discipline    | All map mutations under write lock; reads under read lock            |
| Struct embedding / ownership | `Hub` owns `Room` map; `Room` owns subscriber + history state        |
| Non-blocking channel send    | Slow subscribers are skipped via `select { default: }` pattern       |
| Presence bookkeeping         | `PresenceTracker` is the single authoritative online/offline store   |
| History with bounded buffer  | Slice appended and sliced to cap at `maxHistory` (50)                |
| Exponential backoff          | `ReconnectBackoff` doubles delay on each retry, caps at 30 s         |
| Race-free concurrent test    | Run with `go run -race` to verify no data races                      |
