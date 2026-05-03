// FILE: book/part5_building_backends/chapter75_websockets/exercises/01_chat_hub/main.go
// CHAPTER: 75 — WebSockets, SSE, and Long-Lived Connections
// TOPIC: Chat hub — rooms, direct messages, presence, message history, reconnection.
//
// Run (from the chapter folder):
//   go run ./exercises/01_chat_hub

package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// MESSAGE
// ─────────────────────────────────────────────────────────────────────────────

type MsgType string

const (
	MsgChat    MsgType = "chat"
	MsgJoin    MsgType = "join"
	MsgLeave   MsgType = "leave"
	MsgDirect  MsgType = "dm"
	MsgPresence MsgType = "presence"
)

type ChatMessage struct {
	ID       int64
	Type     MsgType
	RoomID   string
	From     string
	To       string // non-empty for DM
	Text     string
	At       time.Time
}

func (m ChatMessage) String() string {
	switch m.Type {
	case MsgJoin:
		return fmt.Sprintf("[%s joined %s]", m.From, m.RoomID)
	case MsgLeave:
		return fmt.Sprintf("[%s left %s]", m.From, m.RoomID)
	case MsgDirect:
		return fmt.Sprintf("[DM %s→%s] %s", m.From, m.To, m.Text)
	default:
		return fmt.Sprintf("[%s/%s] %s", m.RoomID, m.From, m.Text)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MESSAGE HISTORY (ring buffer per room)
// ─────────────────────────────────────────────────────────────────────────────

type RoomHistory struct {
	mu       sync.RWMutex
	messages []*ChatMessage
	limit    int
}

func NewRoomHistory(limit int) *RoomHistory {
	return &RoomHistory{limit: limit}
}

func (h *RoomHistory) Add(m *ChatMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messages = append(h.messages, m)
	if len(h.messages) > h.limit {
		h.messages = h.messages[len(h.messages)-h.limit:]
	}
}

// Since returns messages with ID > lastID.
func (h *RoomHistory) Since(lastID int64) []*ChatMessage {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var out []*ChatMessage
	for _, m := range h.messages {
		if m.ID > lastID {
			out = append(out, m)
		}
	}
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// PRESENCE TRACKER
// ─────────────────────────────────────────────────────────────────────────────

type Presence struct {
	mu      sync.RWMutex
	online  map[string]time.Time // userID → last seen
	inRooms map[string]map[string]bool // roomID → set of userIDs
}

func NewPresence() *Presence {
	return &Presence{
		online:  make(map[string]time.Time),
		inRooms: make(map[string]map[string]bool),
	}
}

func (p *Presence) Join(userID, roomID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.online[userID] = time.Now()
	if p.inRooms[roomID] == nil {
		p.inRooms[roomID] = make(map[string]bool)
	}
	p.inRooms[roomID][userID] = true
}

func (p *Presence) Leave(userID, roomID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.online, userID)
	if p.inRooms[roomID] != nil {
		delete(p.inRooms[roomID], userID)
	}
}

func (p *Presence) OnlineInRoom(roomID string) []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var out []string
	for u := range p.inRooms[roomID] {
		out = append(out, u)
	}
	return out
}

func (p *Presence) IsOnline(userID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, ok := p.online[userID]
	return ok
}

// ─────────────────────────────────────────────────────────────────────────────
// CHAT CLIENT
// ─────────────────────────────────────────────────────────────────────────────

type ChatClient struct {
	UserID string
	RoomID string
	inbox  chan *ChatMessage
	closed chan struct{}
	once   sync.Once
}

func NewChatClient(userID, roomID string) *ChatClient {
	return &ChatClient{
		UserID: userID,
		RoomID: roomID,
		inbox:  make(chan *ChatMessage, 64),
		closed: make(chan struct{}),
	}
}

func (c *ChatClient) Deliver(m *ChatMessage) bool {
	select {
	case <-c.closed:
		return false
	case c.inbox <- m:
		return true
	default:
		return false // slow client
	}
}

func (c *ChatClient) Read(ctx context.Context) (*ChatMessage, bool) {
	select {
	case <-ctx.Done():
		return nil, false
	case <-c.closed:
		return nil, false
	case m := <-c.inbox:
		return m, true
	}
}

func (c *ChatClient) Close() {
	c.once.Do(func() { close(c.closed) })
}

// ─────────────────────────────────────────────────────────────────────────────
// CHAT HUB
// ─────────────────────────────────────────────────────────────────────────────

type ChatHub struct {
	mu       sync.RWMutex
	clients  map[string]*ChatClient   // userID → client
	rooms    map[string][]*ChatClient // roomID → clients
	history  map[string]*RoomHistory  // roomID → history
	presence *Presence
	msgSeq   atomic.Int64
	Metrics  struct {
		Sent    atomic.Int64
		Dropped atomic.Int64
	}
}

func NewChatHub() *ChatHub {
	return &ChatHub{
		clients:  make(map[string]*ChatClient),
		rooms:    make(map[string][]*ChatClient),
		history:  make(map[string]*RoomHistory),
		presence: NewPresence(),
	}
}

func (h *ChatHub) Join(client *ChatClient) []*ChatMessage {
	h.mu.Lock()
	if h.history[client.RoomID] == nil {
		h.history[client.RoomID] = NewRoomHistory(50)
	}
	h.clients[client.UserID] = client
	h.rooms[client.RoomID] = append(h.rooms[client.RoomID], client)
	hist := h.history[client.RoomID]
	h.mu.Unlock()

	h.presence.Join(client.UserID, client.RoomID)

	// Deliver join event to room.
	h.roomBroadcast(client.RoomID, &ChatMessage{
		Type: MsgJoin, RoomID: client.RoomID, From: client.UserID, At: time.Now(),
	})

	// Return history for reconnection.
	return hist.Since(0)
}

func (h *ChatHub) Leave(client *ChatClient) {
	h.mu.Lock()
	delete(h.clients, client.UserID)
	chs := h.rooms[client.RoomID]
	for i, c := range chs {
		if c.UserID == client.UserID {
			h.rooms[client.RoomID] = append(chs[:i], chs[i+1:]...)
			break
		}
	}
	h.mu.Unlock()

	h.presence.Leave(client.UserID, client.RoomID)
	client.Close()

	h.roomBroadcast(client.RoomID, &ChatMessage{
		Type: MsgLeave, RoomID: client.RoomID, From: client.UserID, At: time.Now(),
	})
}

func (h *ChatHub) Send(from *ChatClient, text string) {
	msg := &ChatMessage{
		ID:     h.msgSeq.Add(1),
		Type:   MsgChat,
		RoomID: from.RoomID,
		From:   from.UserID,
		Text:   text,
		At:     time.Now(),
	}
	h.mu.Lock()
	if h.history[from.RoomID] != nil {
		h.history[from.RoomID].Add(msg)
	}
	h.mu.Unlock()
	h.roomBroadcast(from.RoomID, msg)
}

func (h *ChatHub) DirectMessage(fromID, toID, text string) error {
	h.mu.RLock()
	to, ok := h.clients[toID]
	h.mu.RUnlock()
	if !ok {
		return fmt.Errorf("user %q not online", toID)
	}
	msg := &ChatMessage{
		ID: h.msgSeq.Add(1), Type: MsgDirect,
		From: fromID, To: toID, Text: text, At: time.Now(),
	}
	if to.Deliver(msg) {
		h.Metrics.Sent.Add(1)
	} else {
		h.Metrics.Dropped.Add(1)
	}
	return nil
}

func (h *ChatHub) roomBroadcast(roomID string, msg *ChatMessage) {
	h.mu.RLock()
	clients := make([]*ChatClient, len(h.rooms[roomID]))
	copy(clients, h.rooms[roomID])
	h.mu.RUnlock()

	for _, c := range clients {
		if c.Deliver(msg) {
			h.Metrics.Sent.Add(1)
		} else {
			h.Metrics.Dropped.Add(1)
		}
	}
}

func (h *ChatHub) ReconnectHistory(roomID string, lastID int64) []*ChatMessage {
	h.mu.RLock()
	hist := h.history[roomID]
	h.mu.RUnlock()
	if hist == nil {
		return nil
	}
	return hist.Since(lastID)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chat Hub Exercise ===")
	fmt.Println()

	hub := NewChatHub()
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// ── ROOM CHAT ─────────────────────────────────────────────────────────────
	fmt.Println("--- Room chat ---")
	alice := NewChatClient("alice", "general")
	bob := NewChatClient("bob", "general")
	carol := NewChatClient("carol", "general")

	// Collect messages concurrently.
	var mu sync.Mutex
	msgs := make(map[string][]*ChatMessage)
	var readWg sync.WaitGroup

	startRead := func(c *ChatClient, count int) {
		readWg.Add(1)
		go func() {
			defer readWg.Done()
			for i := 0; i < count; i++ {
				m, ok := c.Read(ctx)
				if !ok {
					return
				}
				mu.Lock()
				msgs[c.UserID] = append(msgs[c.UserID], m)
				mu.Unlock()
				fmt.Printf("  [%s] %s\n", c.UserID, m)
			}
		}()
	}

	// Join (each join broadcasts to room — 1 msg each join).
	hub.Join(alice)
	startRead(alice, 5) // join(bob), join(carol), 2 chats, leave(carol)
	hub.Join(bob)
	startRead(bob, 4)   // join(carol), 2 chats, leave(carol)
	hub.Join(carol)
	startRead(carol, 2) // 2 chat messages

	hub.Send(alice, "hello everyone!")
	hub.Send(bob, "hey alice!")
	hub.Leave(carol)

	readWg.Wait()

	// ── DIRECT MESSAGE ────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Direct messages ---")
	dave := NewChatClient("dave", "general")
	hub.Join(dave)
	startRead(dave, 2) // 2 DMs

	hub.DirectMessage("alice", "dave", "hey dave, welcome!")
	hub.DirectMessage("bob", "dave", "hi dave!")

	var dmWg sync.WaitGroup
	dmWg.Add(1)
	go func() {
		defer dmWg.Done()
		for i := 0; i < 2; i++ {
			m, ok := dave.Read(ctx)
			if !ok {
				return
			}
			fmt.Printf("  [dave inbox] %s\n", m)
		}
	}()
	dmWg.Wait()

	// DM to offline user.
	err := hub.DirectMessage("alice", "ghost", "are you there?")
	fmt.Printf("  DM to offline: %v\n", err)

	// ── PRESENCE ──────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Presence ---")
	fmt.Printf("  online in general: %v\n", hub.presence.OnlineInRoom("general"))
	fmt.Printf("  carol online: %v\n", hub.presence.IsOnline("carol"))
	fmt.Printf("  alice online: %v\n", hub.presence.IsOnline("alice"))

	// ── RECONNECTION WITH HISTORY ─────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Reconnection with message history ---")
	// eve reconnects and requests messages since lastID=2
	eve := NewChatClient("eve", "general")
	history := hub.Join(eve)
	fmt.Printf("  eve joins fresh: %d messages in history\n", len(history))
	for _, m := range history {
		fmt.Printf("    id=%d %s\n", m.ID, m)
	}

	// Simulate eve disconnecting then reconnecting (lastID=1).
	reconnectMsgs := hub.ReconnectHistory("general", 1)
	fmt.Printf("  eve reconnects with lastID=1: gets %d missed messages\n", len(reconnectMsgs))

	// ── METRICS ───────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Printf("  hub metrics: sent=%d dropped=%d\n",
		hub.Metrics.Sent.Load(), hub.Metrics.Dropped.Load())

	cancel()
}
