// CAPSTONE D — Real-Time Chat
//
// Self-contained simulation of a real-time chat hub using only the standard
// library.  WebSocket I/O is replaced by buffered channels so the entire
// system — Room, Hub, PresenceTracker, history, reconnect backoff — can be
// exercised without a network listener.
//
// Run:  go run ./part7_capstone_projects/capstone_d_realtime_chat/
// Race: go run -race ./part7_capstone_projects/capstone_d_realtime_chat/

package main

import (
	"fmt"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Domain types
// ---------------------------------------------------------------------------

// Message is an immutable value that travels through the chat system.
type Message struct {
	ID        string
	RoomID    string
	UserID    string
	Body      string
	Timestamp time.Time
}

func (m Message) String() string {
	return fmt.Sprintf("[%s] %s@%s: %s",
		m.Timestamp.Format("15:04:05"), m.UserID, m.RoomID, m.Body)
}

// ---------------------------------------------------------------------------
// Room
// ---------------------------------------------------------------------------

const (
	maxHistory    = 50
	subscriberBuf = 32 // buffer size per subscriber channel
)

// Room is a single broadcast domain.  All exported methods are thread-safe.
type Room struct {
	id string

	mu          sync.RWMutex
	subscribers map[string]chan Message // userID -> delivery channel
	history     []Message
}

func newRoom(id string) *Room {
	return &Room{
		id:          id,
		subscribers: make(map[string]chan Message),
	}
}

// Subscribe registers userID in this room and returns the channel on which
// the user will receive messages.  Calling Subscribe for an already-subscribed
// user returns the existing channel.
func (r *Room) Subscribe(userID string) <-chan Message {
	r.mu.Lock()
	defer r.mu.Unlock()

	if ch, ok := r.subscribers[userID]; ok {
		return ch
	}
	ch := make(chan Message, subscriberBuf)
	r.subscribers[userID] = ch
	return ch
}

// Unsubscribe removes userID from the room and closes their delivery channel.
func (r *Room) Unsubscribe(userID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if ch, ok := r.subscribers[userID]; ok {
		delete(r.subscribers, userID)
		close(ch)
	}
}

// Broadcast delivers msg to every current subscriber concurrently.
// Slow subscribers are skipped (non-blocking send) so one lagging client
// cannot stall the entire room.
func (r *Room) Broadcast(msg Message) {
	r.mu.Lock()
	// Record history before releasing the lock so ordering is consistent.
	r.history = appendHistory(r.history, msg)
	// Copy the subscriber list so we can release the lock before delivering.
	targets := make([]chan Message, 0, len(r.subscribers))
	for _, ch := range r.subscribers {
		targets = append(targets, ch)
	}
	r.mu.Unlock()

	var wg sync.WaitGroup
	for _, ch := range targets {
		wg.Add(1)
		go func(c chan Message) {
			defer wg.Done()
			select {
			case c <- msg:
			default:
				// Subscriber buffer full — skip this delivery cycle.
			}
		}(ch)
	}
	wg.Wait()
}

// Members returns a snapshot of currently subscribed user IDs.
func (r *Room) Members() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]string, 0, len(r.subscribers))
	for id := range r.subscribers {
		out = append(out, id)
	}
	return out
}

// appendHistory appends msg to history and trims to maxHistory length.
func appendHistory(h []Message, msg Message) []Message {
	h = append(h, msg)
	if len(h) > maxHistory {
		h = h[len(h)-maxHistory:]
	}
	return h
}

// ---------------------------------------------------------------------------
// Hub
// ---------------------------------------------------------------------------

// Hub manages all rooms and is the single entry point for application code.
type Hub struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

func NewHub() *Hub {
	return &Hub{rooms: make(map[string]*Room)}
}

// room returns (or lazily creates) the Room for roomID.
func (h *Hub) room(roomID string) *Room {
	// Fast path — room already exists.
	h.mu.RLock()
	r, ok := h.rooms[roomID]
	h.mu.RUnlock()
	if ok {
		return r
	}

	// Slow path — create under write lock (double-check pattern).
	h.mu.Lock()
	defer h.mu.Unlock()
	if r, ok = h.rooms[roomID]; ok {
		return r
	}
	r = newRoom(roomID)
	h.rooms[roomID] = r
	return r
}

// JoinRoom subscribes userID to roomID and returns their message channel.
func (h *Hub) JoinRoom(userID, roomID string) <-chan Message {
	return h.room(roomID).Subscribe(userID)
}

// LeaveRoom unsubscribes userID from roomID.
func (h *Hub) LeaveRoom(userID, roomID string) {
	h.mu.RLock()
	r, ok := h.rooms[roomID]
	h.mu.RUnlock()
	if ok {
		r.Unsubscribe(userID)
	}
}

// Send broadcasts a message from userID into roomID.
func (h *Hub) Send(userID, roomID, body string) Message {
	msg := Message{
		ID:        fmt.Sprintf("%s-%d", userID, time.Now().UnixNano()),
		RoomID:    roomID,
		UserID:    userID,
		Body:      body,
		Timestamp: time.Now(),
	}
	h.room(roomID).Broadcast(msg)
	return msg
}

// RoomMembers returns the current member list for roomID.
func (h *Hub) RoomMembers(roomID string) []string {
	h.mu.RLock()
	r, ok := h.rooms[roomID]
	h.mu.RUnlock()
	if !ok {
		return nil
	}
	return r.Members()
}

// History returns the stored message history for roomID (up to maxHistory).
func (h *Hub) History(roomID string) []Message {
	h.mu.RLock()
	r, ok := h.rooms[roomID]
	h.mu.RUnlock()
	if !ok {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Message, len(r.history))
	copy(out, r.history)
	return out
}

// ---------------------------------------------------------------------------
// PresenceTracker
// ---------------------------------------------------------------------------

// presenceEntry records whether a user is currently connected.
type presenceEntry struct {
	online    bool
	updatedAt time.Time
}

// PresenceTracker is the authoritative source of user online/offline state.
type PresenceTracker struct {
	mu      sync.RWMutex
	entries map[string]presenceEntry
}

func NewPresenceTracker() *PresenceTracker {
	return &PresenceTracker{entries: make(map[string]presenceEntry)}
}

func (pt *PresenceTracker) SetOnline(userID string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.entries[userID] = presenceEntry{online: true, updatedAt: time.Now()}
}

func (pt *PresenceTracker) SetOffline(userID string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.entries[userID] = presenceEntry{online: false, updatedAt: time.Now()}
}

func (pt *PresenceTracker) IsOnline(userID string) bool {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.entries[userID].online
}

// OnlineUsers returns a snapshot of all currently online user IDs.
func (pt *PresenceTracker) OnlineUsers() []string {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	out := make([]string, 0, len(pt.entries))
	for id, e := range pt.entries {
		if e.online {
			out = append(out, id)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Reconnect backoff helper
// ---------------------------------------------------------------------------

// ReconnectBackoff returns a channel that fires each successive reconnect
// delay (1s, 2s, 4s … up to maxDelay) and then closes.
func ReconnectBackoff(attempts int, maxDelay time.Duration) <-chan time.Duration {
	ch := make(chan time.Duration, attempts)
	go func() {
		defer close(ch)
		delay := time.Second
		for i := 0; i < attempts; i++ {
			ch <- delay
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}()
	return ch
}

// ---------------------------------------------------------------------------
// Simulation helpers
// ---------------------------------------------------------------------------

// drain reads up to n messages from ch with a short timeout, returning them.
func drain(ch <-chan Message, n int) []Message {
	var out []Message
	timeout := time.After(200 * time.Millisecond)
	for i := 0; i < n; i++ {
		select {
		case msg, ok := <-ch:
			if !ok {
				return out
			}
			out = append(out, msg)
		case <-timeout:
			return out
		}
	}
	return out
}

// section prints a visible section header.
func section(title string) {
	fmt.Printf("\n=== %s ===\n", title)
}

// ---------------------------------------------------------------------------
// main — simulation
// ---------------------------------------------------------------------------

func main() {
	hub := NewHub()
	presence := NewPresenceTracker()

	// -----------------------------------------------------------------------
	// 1. Three users come online
	// -----------------------------------------------------------------------
	section("1. Users connect")
	users := []string{"alice", "bob", "carol"}
	for _, u := range users {
		presence.SetOnline(u)
		fmt.Printf("  %s is now online\n", u)
	}
	fmt.Printf("  Online: %v\n", presence.OnlineUsers())

	// -----------------------------------------------------------------------
	// 2. Users join rooms
	//    alice + bob  -> room-general
	//    bob + carol  -> room-go
	// -----------------------------------------------------------------------
	section("2. Users join rooms")
	aliceGen := hub.JoinRoom("alice", "room-general")
	bobGen := hub.JoinRoom("bob", "room-general")
	bobGo := hub.JoinRoom("bob", "room-go")
	carolGo := hub.JoinRoom("carol", "room-go")

	fmt.Printf("  room-general members: %v\n", hub.RoomMembers("room-general"))
	fmt.Printf("  room-go    members: %v\n", hub.RoomMembers("room-go"))

	// -----------------------------------------------------------------------
	// 3. Alice sends messages to room-general; bob receives them
	// -----------------------------------------------------------------------
	section("3. Broadcast in room-general")
	hub.Send("alice", "room-general", "Hey everyone!")
	hub.Send("alice", "room-general", "Anyone working on goroutines today?")
	hub.Send("bob", "room-general", "I am! Just finished the fan-out pattern.")

	// Give goroutines a moment to deliver.
	time.Sleep(50 * time.Millisecond)

	aliceMsgs := drain(aliceGen, 5)
	bobMsgs := drain(bobGen, 5)
	fmt.Printf("  alice received %d messages in room-general:\n", len(aliceMsgs))
	for _, m := range aliceMsgs {
		fmt.Printf("    %s\n", m)
	}
	fmt.Printf("  bob received %d messages in room-general:\n", len(bobMsgs))
	for _, m := range bobMsgs {
		fmt.Printf("    %s\n", m)
	}

	// -----------------------------------------------------------------------
	// 4. Bob and Carol exchange messages in room-go
	// -----------------------------------------------------------------------
	section("4. Broadcast in room-go")
	hub.Send("carol", "room-go", "Starting the channel exercise now.")
	hub.Send("bob", "room-go", "Use a done channel to signal completion!")
	hub.Send("carol", "room-go", "Good call. Done channels are elegant.")

	time.Sleep(50 * time.Millisecond)

	bobGoMsgs := drain(bobGo, 5)
	carolGoMsgs := drain(carolGo, 5)
	fmt.Printf("  bob received %d messages in room-go:\n", len(bobGoMsgs))
	for _, m := range bobGoMsgs {
		fmt.Printf("    %s\n", m)
	}
	fmt.Printf("  carol received %d messages in room-go:\n", len(carolGoMsgs))
	for _, m := range carolGoMsgs {
		fmt.Printf("    %s\n", m)
	}

	// -----------------------------------------------------------------------
	// 5. Carol leaves room-go; confirm her channel is closed
	// -----------------------------------------------------------------------
	section("5. Carol leaves room-go")
	hub.LeaveRoom("carol", "room-go")
	presence.SetOffline("carol")
	fmt.Printf("  room-go members after Carol leaves: %v\n", hub.RoomMembers("room-go"))
	fmt.Printf("  carol online: %v\n", presence.IsOnline("carol"))
	fmt.Printf("  bob   online: %v\n", presence.IsOnline("bob"))

	// Attempting to receive from carol's now-closed channel should return zero
	// value and ok==false.
	_, carolStillOpen := <-carolGo
	fmt.Printf("  carol's channel still open: %v\n", carolStillOpen)

	// -----------------------------------------------------------------------
	// 6. Bob sends to room-go (only he is subscribed now)
	// -----------------------------------------------------------------------
	section("6. Solo message in room-go after Carol left")
	hub.Send("bob", "room-go", "Quiet in here now...")
	time.Sleep(50 * time.Millisecond)
	soloMsgs := drain(bobGo, 2)
	fmt.Printf("  bob received %d message(s): %v\n", len(soloMsgs), soloMsgs[0].Body)

	// -----------------------------------------------------------------------
	// 7. History replay — late joiner dave joins room-general
	// -----------------------------------------------------------------------
	section("7. History replay for late joiner dave")
	history := hub.History("room-general")
	fmt.Printf("  room-general has %d message(s) in history\n", len(history))
	_ = hub.JoinRoom("dave", "room-general") // subscribe after the fact
	presence.SetOnline("dave")
	fmt.Printf("  dave replays history:\n")
	for _, m := range history {
		fmt.Printf("    %s\n", m)
	}

	// -----------------------------------------------------------------------
	// 8. Presence summary
	// -----------------------------------------------------------------------
	section("8. Presence summary")
	for _, u := range append(users, "dave") {
		fmt.Printf("  %-8s online=%v\n", u, presence.IsOnline(u))
	}

	// -----------------------------------------------------------------------
	// 9. Reconnect backoff demo
	// -----------------------------------------------------------------------
	section("9. Reconnect backoff (simulated — not sleeping)")
	fmt.Println("  Backoff schedule for up to 5 attempts (max 30s):")
	for d := range ReconnectBackoff(5, 30*time.Second) {
		fmt.Printf("    retry in %s\n", d)
	}

	section("Simulation complete")
}
