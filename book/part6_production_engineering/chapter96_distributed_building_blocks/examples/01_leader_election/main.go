// FILE: book/part6_production_engineering/chapter96_distributed_building_blocks/examples/01_leader_election/main.go
// CHAPTER: 96 — Distributed Building Blocks
// TOPIC: Leader election — heartbeat-based detection, epoch fencing,
//        split-brain prevention, and graceful handoff.
//
// Run:
//   go run ./book/part6_production_engineering/chapter96_distributed_building_blocks/examples/01_leader_election

package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// NODE ROLES
// ─────────────────────────────────────────────────────────────────────────────

type Role int

const (
	Follower  Role = iota
	Candidate
	Leader
)

func (r Role) String() string {
	switch r {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	default:
		return "Unknown"
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// NODE
// ─────────────────────────────────────────────────────────────────────────────

type Node struct {
	ID              string
	mu              sync.Mutex
	role            Role
	epoch           int64  // monotonically increasing term
	leaderID        string
	lastHeartbeat   time.Time
	heartbeatTimeout time.Duration
}

func NewNode(id string, timeout time.Duration) *Node {
	return &Node{
		ID:               id,
		role:             Follower,
		lastHeartbeat:    time.Now(),
		heartbeatTimeout: timeout,
	}
}

func (n *Node) ReceiveHeartbeat(leaderID string, epoch int64) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if epoch < n.epoch {
		// Stale leader — reject
		return false
	}
	n.epoch = epoch
	n.leaderID = leaderID
	n.role = Follower
	n.lastHeartbeat = time.Now()
	return true
}

func (n *Node) CheckTimeout() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.role != Leader && time.Since(n.lastHeartbeat) > n.heartbeatTimeout
}

func (n *Node) StartElection(clusterEpoch *atomic.Int64, peers []*Node) bool {
	n.mu.Lock()
	n.role = Candidate
	n.epoch = clusterEpoch.Add(1)
	epoch := n.epoch
	n.mu.Unlock()

	fmt.Printf("  [%s] Starting election (epoch=%d)\n", n.ID, epoch)

	votes := 1 // vote for self
	for _, peer := range peers {
		if peer.ID == n.ID {
			continue
		}
		if peer.VoteFor(n.ID, epoch) {
			votes++
		}
	}

	total := len(peers) + 1
	quorum := total/2 + 1
	if votes >= quorum {
		n.mu.Lock()
		n.role = Leader
		n.leaderID = n.ID
		n.mu.Unlock()
		fmt.Printf("  [%s] Won election (votes=%d/%d, epoch=%d) → LEADER\n",
			n.ID, votes, total, epoch)
		return true
	}
	n.mu.Lock()
	n.role = Follower
	n.mu.Unlock()
	fmt.Printf("  [%s] Lost election (votes=%d/%d)\n", n.ID, votes, total)
	return false
}

func (n *Node) VoteFor(candidateID string, epoch int64) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if epoch > n.epoch {
		n.epoch = epoch
		n.role = Follower
		fmt.Printf("  [%s] Voted for %s (epoch=%d)\n", n.ID, candidateID, epoch)
		return true
	}
	return false
}

func (n *Node) SendHeartbeat(peers []*Node) {
	n.mu.Lock()
	epoch := n.epoch
	id := n.ID
	role := n.role
	n.mu.Unlock()

	if role != Leader {
		return
	}
	for _, peer := range peers {
		if peer.ID != id {
			peer.ReceiveHeartbeat(id, epoch)
		}
	}
}

func (n *Node) Status() string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return fmt.Sprintf("[%s] role=%-10s epoch=%d leader=%s",
		n.ID, n.role, n.epoch, n.leaderID)
}

// ─────────────────────────────────────────────────────────────────────────────
// FENCING TOKEN DEMO
// ─────────────────────────────────────────────────────────────────────────────

type FencingStore struct {
	mu           sync.Mutex
	currentToken int64
	data         string
}

func (s *FencingStore) Write(token int64, data string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if token < s.currentToken {
		return fmt.Errorf("stale token %d (current=%d): write rejected", token, s.currentToken)
	}
	s.currentToken = token
	s.data = data
	return nil
}

func (s *FencingStore) Read() (string, int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data, s.currentToken
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 96: Leader Election ===")
	fmt.Println()

	// ── ELECTION SIMULATION ───────────────────────────────────────────────────
	fmt.Println("--- Leader election among 5 nodes ---")
	nodes := []*Node{
		NewNode("node-1", 500*time.Millisecond),
		NewNode("node-2", 500*time.Millisecond),
		NewNode("node-3", 500*time.Millisecond),
		NewNode("node-4", 500*time.Millisecond),
		NewNode("node-5", 500*time.Millisecond),
	}
	var clusterEpoch atomic.Int64

	// node-1 starts an election
	peers := nodes[1:]
	nodes[0].StartElection(&clusterEpoch, peers)
	fmt.Println()

	fmt.Println("Node status after first election:")
	for _, n := range nodes {
		fmt.Printf("  %s\n", n.Status())
	}
	fmt.Println()

	// ── HEARTBEATS ────────────────────────────────────────────────────────────
	fmt.Println("--- Leader sends heartbeats ---")
	nodes[0].SendHeartbeat(nodes)
	fmt.Println("  All followers reset their heartbeat timers.")
	fmt.Println()

	// ── LEADER FAILURE + RE-ELECTION ──────────────────────────────────────────
	fmt.Println("--- Leader (node-1) crashes; node-3 detects timeout and campaigns ---")
	// Stop heartbeats from node-1 (simulate crash)
	time.Sleep(10 * time.Millisecond)
	// node-3 campaigns
	peers3 := append(nodes[:2], nodes[3:]...)
	nodes[2].StartElection(&clusterEpoch, peers3)
	fmt.Println()

	fmt.Println("Node status after re-election:")
	for _, n := range nodes {
		fmt.Printf("  %s\n", n.Status())
	}
	fmt.Println()

	// ── STALE LEADER FENCING ──────────────────────────────────────────────────
	fmt.Println("--- Fencing token: old leader write rejected ---")
	store := &FencingStore{}
	token1 := int64(1) // old leader's token
	token2 := int64(2) // new leader's token

	if err := store.Write(token2, "written by new leader"); err != nil {
		fmt.Printf("  token=%d: ERROR: %v\n", token2, err)
	} else {
		fmt.Printf("  token=%d: accepted — %q\n", token2, "written by new leader")
	}

	if err := store.Write(token1, "stale write from old leader"); err != nil {
		fmt.Printf("  token=%d: REJECTED — %v\n", token1, err)
	}

	data, tok := store.Read()
	fmt.Printf("  Store: token=%d data=%q\n", tok, data)
	fmt.Println()

	// ── SPLIT-BRAIN PREVENTION NOTES ─────────────────────────────────────────
	fmt.Println("--- Split-brain prevention ---")
	fmt.Println(`  Requirements:
    1. Majority quorum: leader needs >N/2 votes (5-node cluster: 3 votes)
    2. Term/epoch: nodes reject messages with a lower epoch than they've seen
    3. Fencing token: storage rejects writes from old leaders

  What NOT to do:
    - Timeout-based leader: clocks drift; two nodes can both think they won
    - Ping-based: network partition can fool both sides into thinking the other is dead
    - No fencing: even with proper election, a slow network packet from an old
      leader can arrive late and corrupt state`)
}
