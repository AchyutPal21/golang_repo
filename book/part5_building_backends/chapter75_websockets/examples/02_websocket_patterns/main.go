// FILE: book/part5_building_backends/chapter75_websockets/examples/02_websocket_patterns/main.go
// CHAPTER: 75 — WebSockets, SSE, and Long-Lived Connections
// TOPIC: Hub pattern — broadcast, rooms/channels, reconnection with
//        Last-Event-ID, and scaling sticky sessions across instances.
//
// Run (from the chapter folder):
//   go run ./examples/02_websocket_patterns

package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// CLIENT (represents one connected WebSocket/SSE consumer)
// ─────────────────────────────────────────────────────────────────────────────

type Client struct {
	ID     string
	RoomID string
	send   chan []byte
	closed chan struct{}
	once   sync.Once
}

func NewClient(id, room string) *Client {
	return &Client{
		ID:     id,
		RoomID: room,
		send:   make(chan []byte, 32),
		closed: make(chan struct{}),
	}
}

func (c *Client) Send(msg []byte) bool {
	select {
	case <-c.closed:
		return false
	case c.send <- msg:
		return true
	default:
		// Slow client: drop message (could also disconnect).
		return false
	}
}

func (c *Client) ReadMessages(ctx context.Context, onMsg func([]byte)) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.closed:
			return
		case msg := <-c.send:
			onMsg(msg)
		}
	}
}

func (c *Client) Close() {
	c.once.Do(func() { close(c.closed) })
}

// ─────────────────────────────────────────────────────────────────────────────
// HUB — central registry for connections and rooms
// ─────────────────────────────────────────────────────────────────────────────

type Hub struct {
	mu       sync.RWMutex
	clients  map[string]*Client   // clientID → client
	rooms    map[string][]*Client // roomID → clients

	register   chan *Client
	unregister chan *Client
	broadcast  chan broadcastMsg
	roomcast   chan broadcastMsg

	Broadcasts atomic.Int64
	Roomcasts  atomic.Int64
	Delivered  atomic.Int64
	Dropped    atomic.Int64
}

type broadcastMsg struct {
	RoomID  string // empty = all clients
	Payload []byte
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		rooms:      make(map[string][]*Client),
		register:   make(chan *Client, 8),
		unregister: make(chan *Client, 8),
		broadcast:  make(chan broadcastMsg, 64),
		roomcast:   make(chan broadcastMsg, 64),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c.ID] = c
			h.rooms[c.RoomID] = append(h.rooms[c.RoomID], c)
			h.mu.Unlock()
			fmt.Printf("  [hub] registered %s room=%s (total=%d)\n", c.ID, c.RoomID, len(h.clients))

		case c := <-h.unregister:
			h.mu.Lock()
			delete(h.clients, c.ID)
			chs := h.rooms[c.RoomID]
			for i, rc := range chs {
				if rc.ID == c.ID {
					h.rooms[c.RoomID] = append(chs[:i], chs[i+1:]...)
					break
				}
			}
			h.mu.Unlock()
			c.Close()
			fmt.Printf("  [hub] unregistered %s\n", c.ID)

		case msg := <-h.broadcast:
			h.Broadcasts.Add(1)
			h.mu.RLock()
			for _, c := range h.clients {
				if c.Send(msg.Payload) {
					h.Delivered.Add(1)
				} else {
					h.Dropped.Add(1)
				}
			}
			h.mu.RUnlock()

		case msg := <-h.roomcast:
			h.Roomcasts.Add(1)
			h.mu.RLock()
			for _, c := range h.rooms[msg.RoomID] {
				if c.Send(msg.Payload) {
					h.Delivered.Add(1)
				} else {
					h.Dropped.Add(1)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Register(c *Client)   { h.register <- c }
func (h *Hub) Unregister(c *Client) { h.unregister <- c }
func (h *Hub) Broadcast(msg []byte) { h.broadcast <- broadcastMsg{Payload: msg} }
func (h *Hub) RoomBroadcast(roomID string, msg []byte) {
	h.roomcast <- broadcastMsg{RoomID: roomID, Payload: msg}
}

// ─────────────────────────────────────────────────────────────────────────────
// RECONNECTION WITH LAST-EVENT-ID (SSE style)
// ─────────────────────────────────────────────────────────────────────────────

type EventLog struct {
	mu     sync.RWMutex
	events []struct {
		ID  int64
		Msg string
	}
	nextID atomic.Int64
}

func (el *EventLog) Add(msg string) int64 {
	id := el.nextID.Add(1)
	el.mu.Lock()
	el.events = append(el.events, struct {
		ID  int64
		Msg string
	}{id, msg})
	el.mu.Unlock()
	return id
}

// Since returns all events with ID > lastID.
func (el *EventLog) Since(lastID int64) []struct {
	ID  int64
	Msg string
} {
	el.mu.RLock()
	defer el.mu.RUnlock()
	var out []struct {
		ID  int64
		Msg string
	}
	for _, e := range el.events {
		if e.ID > lastID {
			out = append(out, e)
		}
	}
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== WebSocket Patterns ===")
	fmt.Println()

	// ── HUB WITH BROADCAST ────────────────────────────────────────────────────
	fmt.Println("--- Hub: broadcast to all clients ---")

	hub := NewHub()
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	var hubWg sync.WaitGroup
	hubWg.Add(1)
	go func() {
		defer hubWg.Done()
		hub.Run(ctx)
	}()

	// Register 3 clients.
	clients := []*Client{
		NewClient("c-1", "room-a"),
		NewClient("c-2", "room-a"),
		NewClient("c-3", "room-b"),
	}
	for _, c := range clients {
		hub.Register(c)
	}
	time.Sleep(10 * time.Millisecond) // let hub process registrations

	// Collect received messages per client.
	var mu sync.Mutex
	received := make(map[string][]string)
	var readWg sync.WaitGroup
	for _, c := range clients {
		c := c
		readWg.Add(1)
		go func() {
			defer readWg.Done()
			c.ReadMessages(ctx, func(msg []byte) {
				mu.Lock()
				received[c.ID] = append(received[c.ID], string(msg))
				mu.Unlock()
			})
		}()
	}

	hub.Broadcast([]byte("system: server maintenance in 5m"))
	time.Sleep(10 * time.Millisecond)

	// ── ROOM BROADCAST ────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Hub: room broadcast ---")
	hub.RoomBroadcast("room-a", []byte("room-a: welcome!"))
	hub.RoomBroadcast("room-b", []byte("room-b: hello!"))
	time.Sleep(10 * time.Millisecond)

	// ── UNREGISTER ────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Unregister client ---")
	hub.Unregister(clients[0])
	time.Sleep(10 * time.Millisecond)

	// Broadcast after c-1 is gone.
	hub.Broadcast([]byte("system: c-1 left, only 2 remain"))
	time.Sleep(10 * time.Millisecond)

	cancel()
	hubWg.Wait()
	readWg.Wait()

	fmt.Println()
	for id, msgs := range received {
		fmt.Printf("  %s received %d messages: %v\n", id, len(msgs), msgs)
	}
	fmt.Printf("  hub stats: broadcasts=%d roomcasts=%d delivered=%d dropped=%d\n",
		hub.Broadcasts.Load(), hub.Roomcasts.Load(), hub.Delivered.Load(), hub.Dropped.Load())

	// ── RECONNECTION WITH LAST-EVENT-ID ───────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Reconnection with Last-Event-ID ---")

	log := &EventLog{}
	log.Add("order placed: ord-1")
	log.Add("payment processed: ord-1")
	log.Add("order shipped: ord-1")
	log.Add("delivery update: ord-1")

	// Client connects fresh — gets all events.
	fmt.Println("  client connects fresh (lastID=0):")
	for _, e := range log.Since(0) {
		fmt.Printf("    id=%d data=%s\n", e.ID, e.Msg)
	}

	// Client reconnects after seeing event 2.
	fmt.Println("  client reconnects after lastID=2:")
	for _, e := range log.Since(2) {
		fmt.Printf("    id=%d data=%s\n", e.ID, e.Msg)
	}

	// ── SCALING ACROSS INSTANCES ──────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Scaling WebSocket across instances (conceptual) ---")
	fmt.Println(`
  Problem: user A connects to instance-1, user B connects to instance-2.
           A message to B goes to instance-1's hub which has no connection to B.

  Solution: Pub/Sub relay via Redis or a message broker:
    1. Each instance subscribes to a shared Redis channel
    2. When hub needs to broadcast, it PUBLISHES to Redis
    3. All instances receive the message and forward to their local clients

  instance-1 hub → redis.Publish("hub:broadcast", msg)
                       ↓
  instance-2 hub ← redis.Subscribe("hub:broadcast") → deliver to local clients

  For rooms: one Redis channel per room; instances subscribe only to rooms
  with active local connections (minimises traffic).`)
}
