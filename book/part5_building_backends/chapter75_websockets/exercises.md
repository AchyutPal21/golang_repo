# Chapter 75 Exercises — WebSockets, SSE, and Long-Lived Connections

## Exercise 1 — Chat Hub (`exercises/01_chat_hub`)

Build a full-featured chat hub with rooms, direct messages, presence, message history for reconnection, and delivery metrics.

### Message types

```go
type MsgType string
const (
    MsgChat    MsgType = "chat"
    MsgJoin    MsgType = "join"
    MsgLeave   MsgType = "leave"
    MsgDirect  MsgType = "dm"
)

type ChatMessage struct {
    ID     int64
    Type   MsgType
    RoomID string
    From   string
    To     string   // non-empty for DMs
    Text   string
    At     time.Time
}
```

### Components to implement

**`RoomHistory`** — ring buffer per room:
```go
func (h *RoomHistory) Add(m *ChatMessage)
func (h *RoomHistory) Since(lastID int64) []*ChatMessage  // returns messages with ID > lastID
```

**`Presence`** — online user tracking:
```go
func (p *Presence) Join(userID, roomID string)
func (p *Presence) Leave(userID, roomID string)
func (p *Presence) OnlineInRoom(roomID string) []string
func (p *Presence) IsOnline(userID string) bool
```

**`ChatClient`** — buffered inbox:
```go
func (c *ChatClient) Deliver(m *ChatMessage) bool   // false if closed or buffer full
func (c *ChatClient) Read(ctx context.Context) (*ChatMessage, bool)
func (c *ChatClient) Close()
```

**`ChatHub`** — central coordinator:
```go
func (h *ChatHub) Join(client *ChatClient) []*ChatMessage  // returns history for replay
func (h *ChatHub) Leave(client *ChatClient)
func (h *ChatHub) Send(from *ChatClient, text string)
func (h *ChatHub) DirectMessage(fromID, toID, text string) error
func (h *ChatHub) ReconnectHistory(roomID string, lastID int64) []*ChatMessage
```

### Behaviour rules

- `Join` broadcasts a `MsgJoin` event to the room, then returns the full history
- `Leave` broadcasts a `MsgLeave` event and marks the client closed
- `Send` persists the message to `RoomHistory` and broadcasts to the room
- `DirectMessage` returns an error if the target user is not online
- `RoomHistory` keeps at most 50 messages (ring buffer)
- Presence is updated on Join/Leave; `Leave` removes the user from the online set

### Demonstration

1. Three users join "general"; each sees join events for users after them
2. Two chat messages are sent; all active members receive them
3. A user leaves; leave event goes to remaining members
4. Two direct messages to a new user; verify inbox
5. DM to an offline user returns error
6. Presence check: online list and `IsOnline` for departed user
7. Reconnect scenario: new user joins, gets history; `ReconnectHistory(room, lastID=1)` returns 1 missed message

### Hints

- Assign monotonically increasing IDs using `atomic.Int64`
- The hub does NOT need a separate goroutine — use `sync.RWMutex` for thread safety since the client inboxes are buffered channels
- `Deliver` should be non-blocking: use `select` with a `default` branch to drop if the buffer is full
- `RoomHistory.limit` enforces the ring: `messages = messages[len-limit:]` when over limit
- Collect client read goroutines with a `sync.WaitGroup` to avoid racing against the test output
