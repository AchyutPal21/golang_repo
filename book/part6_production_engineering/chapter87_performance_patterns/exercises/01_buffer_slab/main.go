// FILE: book/part6_production_engineering/chapter87_performance_patterns/exercises/01_buffer_slab/main.go
// CHAPTER: 87 — Performance Patterns
// EXERCISE: Build a slab allocator — a pool of fixed-size byte arrays that
//   eliminates per-request allocation for network packet processing.
//   The slab holds N slots of exactly SlabSize bytes.
//   Workers acquire a slot, write data, then release it back.
//
// Run:
//   go run ./part6_production_engineering/chapter87_performance_patterns/exercises/01_buffer_slab

package main

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// SLAB ALLOCATOR
// ─────────────────────────────────────────────────────────────────────────────

const SlabSize = 4096 // bytes per slot
const SlabCount = 64  // number of slots in the slab

// Slab is a fixed-pool of SlabCount byte arrays of SlabSize each.
// Acquisition is O(1) lock-free via a channel of free indices.
type Slab struct {
	data  [SlabCount][SlabSize]byte
	free  chan int
	stats slabStats
}

type slabStats struct {
	acquired atomic.Int64
	released atomic.Int64
	misses   atomic.Int64 // acquire with no slot available
}

func NewSlab() *Slab {
	s := &Slab{
		free: make(chan int, SlabCount),
	}
	for i := 0; i < SlabCount; i++ {
		s.free <- i
	}
	return s
}

var ErrSlabFull = errors.New("slab: no free slots")

// Acquire returns a slot index and the backing byte slice.
// Returns ErrSlabFull if all slots are in use.
func (s *Slab) Acquire() (int, []byte, error) {
	select {
	case idx := <-s.free:
		s.stats.acquired.Add(1)
		// Return a zero-length slice backed by the slot's array.
		return idx, s.data[idx][:0], nil
	default:
		s.stats.misses.Add(1)
		return -1, nil, ErrSlabFull
	}
}

// Release returns slot idx back to the free pool.
func (s *Slab) Release(idx int) {
	if idx < 0 || idx >= SlabCount {
		return
	}
	// Zero the slot to avoid data leakage.
	for i := range s.data[idx] {
		s.data[idx][i] = 0
	}
	s.stats.released.Add(1)
	s.free <- idx
}

// FreeSlots returns the current number of available slots.
func (s *Slab) FreeSlots() int { return len(s.free) }

// ─────────────────────────────────────────────────────────────────────────────
// PACKET PROCESSOR — simulates network packet handling using the slab
// ─────────────────────────────────────────────────────────────────────────────

type PacketProcessor struct {
	slab *Slab
	mu   sync.Mutex
	log  []string
}

func NewPacketProcessor(slab *Slab) *PacketProcessor {
	return &PacketProcessor{slab: slab}
}

// Process simulates receiving and handling a packet.
func (pp *PacketProcessor) Process(packetData []byte) error {
	idx, buf, err := pp.slab.Acquire()
	if err != nil {
		return fmt.Errorf("processor: %w", err)
	}
	defer pp.slab.Release(idx)

	// Copy packet data into slab slot.
	n := copy(pp.slab.data[idx][:cap(buf)], packetData)
	buf = pp.slab.data[idx][:n]

	// Simulate processing: checksum the first byte.
	checksum := byte(0)
	for _, b := range buf {
		checksum ^= b
	}

	pp.mu.Lock()
	pp.log = append(pp.log, fmt.Sprintf("slot=%d len=%d checksum=0x%02x", idx, n, checksum))
	pp.mu.Unlock()

	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// ALLOCATION COMPARISON
// ─────────────────────────────────────────────────────────────────────────────

type memsnap struct{ allocs, bytes uint64 }

func snapMem() memsnap {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return memsnap{m.Mallocs, m.TotalAlloc}
}
func (a memsnap) diff(b memsnap) memsnap { return memsnap{b.allocs - a.allocs, b.bytes - a.bytes} }

const packetCount = 10_000

func withoutSlab() {
	packet := []byte("GET /api/v1/users HTTP/1.1\r\nHost: example.com\r\n\r\n")
	for i := 0; i < packetCount; i++ {
		buf := make([]byte, SlabSize) // per-call allocation
		n := copy(buf, packet)
		checksum := byte(0)
		for _, b := range buf[:n] {
			checksum ^= b
		}
		_ = checksum
	}
}

func withSlab(slab *Slab) {
	packet := []byte("GET /api/v1/users HTTP/1.1\r\nHost: example.com\r\n\r\n")
	for i := 0; i < packetCount; i++ {
		idx, _, err := slab.Acquire()
		if err != nil {
			continue
		}
		n := copy(slab.data[idx][:], packet)
		checksum := byte(0)
		for _, b := range slab.data[idx][:n] {
			checksum ^= b
		}
		_ = checksum
		slab.Release(idx)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// CONCURRENT STRESS TEST
// ─────────────────────────────────────────────────────────────────────────────

func stressTest(slab *Slab) {
	pp := NewPacketProcessor(slab)
	var wg sync.WaitGroup
	packets := [][]byte{
		[]byte("PING"),
		[]byte("GET /health HTTP/1.1\r\n\r\n"),
		[]byte("POST /events HTTP/1.1\r\nContent-Length: 8\r\n\r\n{\"e\":\"x\"}"),
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			pkt := packets[id%len(packets)]
			if err := pp.Process(pkt); err != nil {
				// slab full — expected under heavy load
			}
		}(i)
	}
	wg.Wait()
	fmt.Printf("  Processed %d packets, log entries: %d\n", 50, len(pp.log))
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 87 Exercise: Slab Allocator ===")
	fmt.Println()

	slab := NewSlab()

	fmt.Printf("Slab: %d slots × %d bytes = %d KB total\n",
		SlabCount, SlabSize, SlabCount*SlabSize/1024)
	fmt.Printf("Free slots at start: %d\n", slab.FreeSlots())
	fmt.Println()

	// ── BASIC ACQUIRE / RELEASE ───────────────────────────────────────────────
	fmt.Println("--- Basic acquire/release ---")
	idx1, buf1, err := slab.Acquire()
	fmt.Printf("  Acquired slot %d, err=%v, free=%d\n", idx1, err, slab.FreeSlots())
	buf1 = append(buf1, []byte("hello slab")...)
	_ = buf1
	slab.Release(idx1)
	fmt.Printf("  Released slot %d, free=%d\n", idx1, slab.FreeSlots())
	fmt.Println()

	// ── STRESS TEST ───────────────────────────────────────────────────────────
	fmt.Println("--- Concurrent stress test (50 goroutines) ---")
	stressTest(slab)
	fmt.Printf("  Slab stats: acquired=%d released=%d misses=%d\n",
		slab.stats.acquired.Load(), slab.stats.released.Load(), slab.stats.misses.Load())
	fmt.Println()

	// ── ALLOCATION COMPARISON ─────────────────────────────────────────────────
	fmt.Println("--- Allocation comparison ---")
	runtime.GC()
	a1 := snapMem()
	t1 := time.Now()
	withoutSlab()
	d1 := time.Since(t1)
	st1 := a1.diff(snapMem())

	slab2 := NewSlab()
	runtime.GC()
	a2 := snapMem()
	t2 := time.Now()
	withSlab(slab2)
	d2 := time.Since(t2)
	st2 := a2.diff(snapMem())

	fmt.Printf("  Without slab: %6v  allocs=%d  bytes=%d\n", d1.Round(time.Microsecond), st1.allocs, st1.bytes)
	fmt.Printf("  With slab:    %6v  allocs=%d  bytes=%d\n", d2.Round(time.Microsecond), st2.allocs, st2.bytes)
	if st1.allocs > 0 {
		fmt.Printf("  Allocation reduction: %.1f%%\n", 100*(1-float64(st2.allocs)/float64(st1.allocs)))
	}
	fmt.Println()

	// ── SLAB FULL SCENARIO ────────────────────────────────────────────────────
	fmt.Println("--- Slab-full scenario ---")
	slab3 := NewSlab()
	slots := make([]int, SlabCount)
	for i := range slots {
		idx, _, _ := slab3.Acquire()
		slots[i] = idx
	}
	_, _, err = slab3.Acquire()
	fmt.Printf("  All %d slots acquired. Next acquire error: %v\n", SlabCount, err)
	for _, idx := range slots {
		slab3.Release(idx)
	}
	fmt.Printf("  After full release: free=%d\n", slab3.FreeSlots())
}
