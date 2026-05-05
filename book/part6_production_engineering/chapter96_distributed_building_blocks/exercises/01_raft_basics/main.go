// FILE: book/part6_production_engineering/chapter96_distributed_building_blocks/exercises/01_raft_basics/main.go
// CHAPTER: 96 — Distributed Building Blocks
// EXERCISE: Simplified Raft simulation — leader election with term numbers,
//           log replication, quorum-based commit, and heartbeat/timeout.
//
// Run:
//   go run ./book/part6_production_engineering/chapter96_distributed_building_blocks/exercises/01_raft_basics

package main

import (
	"fmt"
	"strings"
	"sync"
)

// ─────────────────────────────────────────────────────────────────────────────
// RAFT NODE STATE
// ─────────────────────────────────────────────────────────────────────────────

type NodeState int

const (
	StateFollower  NodeState = iota
	StateCandidate
	StateLeader
)

func (s NodeState) String() string {
	switch s {
	case StateFollower:
		return "Follower"
	case StateCandidate:
		return "Candidate"
	case StateLeader:
		return "Leader"
	default:
		return "Unknown"
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// LOG ENTRY
// ─────────────────────────────────────────────────────────────────────────────

type LogEntry struct {
	Term    int
	Index   int
	Command string
}

// ─────────────────────────────────────────────────────────────────────────────
// RAFT NODE
// ─────────────────────────────────────────────────────────────────────────────

type RaftNode struct {
	ID          string
	mu          sync.Mutex
	state       NodeState
	currentTerm int
	votedFor    string
	log         []LogEntry
	commitIndex int
	lastApplied int
	peers       []*RaftNode
}

func NewRaftNode(id string) *RaftNode {
	return &RaftNode{ID: id, state: StateFollower}
}

func (n *RaftNode) SetPeers(peers []*RaftNode) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.peers = peers
}

// RequestVote — simplified: always grant if term is newer
func (n *RaftNode) RequestVote(candidateID string, term int) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if term > n.currentTerm {
		n.currentTerm = term
		n.state = StateFollower
		n.votedFor = candidateID
		return true
	}
	if term == n.currentTerm && (n.votedFor == "" || n.votedFor == candidateID) {
		n.votedFor = candidateID
		return true
	}
	return false
}

// StartElection — transitions to candidate, requests votes from peers
func (n *RaftNode) StartElection() bool {
	n.mu.Lock()
	n.state = StateCandidate
	n.currentTerm++
	term := n.currentTerm
	n.votedFor = n.ID
	peers := n.peers
	n.mu.Unlock()

	votes := 1 // self-vote
	for _, peer := range peers {
		if peer.ID == n.ID {
			continue
		}
		if peer.RequestVote(n.ID, term) {
			votes++
		}
	}

	total := len(peers) + 1
	quorum := quorum(total)
	if votes >= quorum {
		n.mu.Lock()
		n.state = StateLeader
		n.mu.Unlock()
		fmt.Printf("  [%s] elected Leader (term=%d, votes=%d/%d)\n", n.ID, term, votes, total)
		return true
	}
	n.mu.Lock()
	n.state = StateFollower
	n.mu.Unlock()
	return false
}

// AppendEntry — leader appends to its log
func (n *RaftNode) AppendEntry(command string) int {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.state != StateLeader {
		return -1
	}
	idx := len(n.log)
	entry := LogEntry{Term: n.currentTerm, Index: idx, Command: command}
	n.log = append(n.log, entry)
	return idx
}

// ReplicateEntry — leader sends entry to a follower
func (n *RaftNode) ReceiveEntry(entry LogEntry, leaderTerm int) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if leaderTerm < n.currentTerm {
		return false
	}
	n.currentTerm = leaderTerm
	n.state = StateFollower
	if entry.Index == len(n.log) {
		n.log = append(n.log, entry)
	}
	return true
}

// Replicate — leader replicates last entry to all peers, advances commit on quorum
func (n *RaftNode) Replicate() {
	n.mu.Lock()
	if n.state != StateLeader || len(n.log) == 0 {
		n.mu.Unlock()
		return
	}
	entry := n.log[len(n.log)-1]
	term := n.currentTerm
	peers := n.peers
	n.mu.Unlock()

	success := 1 // leader itself
	for _, peer := range peers {
		if peer.ID == n.ID {
			continue
		}
		if peer.ReceiveEntry(entry, term) {
			success++
		}
	}

	total := len(peers) + 1
	if success >= quorum(total) {
		n.mu.Lock()
		if entry.Index > n.commitIndex {
			n.commitIndex = entry.Index
			fmt.Printf("  [%s] committed log[%d]=%q (replicated %d/%d)\n",
				n.ID, entry.Index, entry.Command, success, total)
		}
		n.mu.Unlock()
	} else {
		fmt.Printf("  [%s] failed to commit log[%d] — only %d/%d replicated\n",
			n.ID, entry.Index, success, total)
	}
}

func quorum(n int) int { return n/2 + 1 }

// ─────────────────────────────────────────────────────────────────────────────
// CLUSTER VIEW
// ─────────────────────────────────────────────────────────────────────────────

func printCluster(nodes []*RaftNode) {
	fmt.Printf("  %-8s  %-10s  %4s  %6s  %s\n", "Node", "State", "Term", "Commit", "Log")
	fmt.Printf("  %s\n", strings.Repeat("-", 60))
	for _, n := range nodes {
		n.mu.Lock()
		logStr := make([]string, len(n.log))
		for i, e := range n.log {
			logStr[i] = fmt.Sprintf("[%d]%s", e.Index, e.Command)
		}
		logSummary := strings.Join(logStr, ",")
		if logSummary == "" {
			logSummary = "(empty)"
		}
		fmt.Printf("  %-8s  %-10s  %4d  %6d  %s\n",
			n.ID, n.state, n.currentTerm, n.commitIndex, logSummary)
		n.mu.Unlock()
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 96 Exercise: Raft Basics ===")
	fmt.Println()

	// ── CLUSTER SETUP ─────────────────────────────────────────────────────────
	nodes := make([]*RaftNode, 5)
	for i := range nodes {
		nodes[i] = NewRaftNode(fmt.Sprintf("node-%d", i+1))
	}
	// Wire peers (all-to-all in this simulation)
	for _, n := range nodes {
		peers := make([]*RaftNode, len(nodes)-1)
		j := 0
		for _, other := range nodes {
			if other.ID != n.ID {
				peers[j] = other
				j++
			}
		}
		n.SetPeers(peers)
	}

	fmt.Println("--- Initial cluster state ---")
	printCluster(nodes)
	fmt.Println()

	// ── LEADER ELECTION ───────────────────────────────────────────────────────
	fmt.Println("--- Election: node-1 campaigns (term 1) ---")
	nodes[0].StartElection()
	fmt.Println()
	printCluster(nodes)
	fmt.Println()

	// ── LOG REPLICATION ───────────────────────────────────────────────────────
	fmt.Println("--- Log replication ---")
	commands := []string{"SET x=1", "SET y=42", "DEL x"}
	for _, cmd := range commands {
		idx := nodes[0].AppendEntry(cmd)
		fmt.Printf("  Leader appended log[%d]=%q\n", idx, cmd)
		nodes[0].Replicate()
	}
	fmt.Println()

	fmt.Println("--- Cluster state after replication ---")
	printCluster(nodes)
	fmt.Println()

	// ── TERM COMPARISON AFTER RE-ELECTION ─────────────────────────────────────
	fmt.Println("--- Re-election: node-3 campaigns (simulating leader failure) ---")
	nodes[2].StartElection()
	fmt.Println()
	printCluster(nodes)
	fmt.Println()

	// ── RAFT SAFETY PROPERTIES ────────────────────────────────────────────────
	fmt.Println("--- Raft safety properties ---")
	fmt.Println(`  Election Safety:
    At most one leader per term.
    Guaranteed by majority quorum — any two quorums share at least one node.

  Log Matching:
    If two logs have an entry with the same index and term, all preceding entries are identical.

  Leader Completeness:
    A leader has all entries committed in previous terms (won't overwrite committed entries).

  State Machine Safety:
    If a server applies log[i], no other server applies a different entry at log[i].

  Practical implementations:
    etcd (used by Kubernetes)
    CockroachDB (distributed SQL)
    TiKV (distributed KV)
    Consul (service mesh + configuration)`)
}
