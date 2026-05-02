// FILE: book/part4_concurrency_systems/chapter53_networking_tcp_udp/exercises/01_echo_server/main.go
// CHAPTER: 53 — Networking I: TCP/UDP
// EXERCISE: Production-ready echo server with:
//   - Per-connection context cancellation (idle timeout)
//   - Connection rate limiting (max concurrent connections)
//   - Graceful shutdown with in-flight connection draining
//   - Stats: total accepted, active, bytes echoed
//   - A test harness that exercises all paths
//
// Run (from the chapter folder):
//   go run ./exercises/01_echo_server

package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// SERVER
// ─────────────────────────────────────────────────────────────────────────────

type ServerConfig struct {
	Addr        string
	MaxConns    int           // 0 = unlimited
	IdleTimeout time.Duration // per-connection read deadline reset
}

type ServerStats struct {
	Accepted atomic.Int64
	Active   atomic.Int64
	Bytes    atomic.Int64
}

type Server struct {
	cfg   ServerConfig
	ln    net.Listener
	stats ServerStats
	wg    sync.WaitGroup
	once  sync.Once
	quit  chan struct{}
	sem   chan struct{} // connection semaphore
}

func NewServer(cfg ServerConfig) (*Server, error) {
	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return nil, err
	}
	s := &Server{
		cfg:  cfg,
		ln:   ln,
		quit: make(chan struct{}),
	}
	if cfg.MaxConns > 0 {
		s.sem = make(chan struct{}, cfg.MaxConns)
	}
	return s, nil
}

func (s *Server) Addr() string { return s.ln.Addr().String() }

func (s *Server) Serve(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			conn, err := s.ln.Accept()
			if err != nil {
				select {
				case <-s.quit:
					return
				default:
					return
				}
			}
			s.stats.Accepted.Add(1)

			// Acquire semaphore slot.
			if s.sem != nil {
				select {
				case s.sem <- struct{}{}:
				default:
					conn.Close()
					fmt.Println("  [server] max connections reached — rejected")
					continue
				}
			}

			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				if s.sem != nil {
					defer func() { <-s.sem }()
				}
				s.handleConn(ctx, conn)
			}()
		}
	}()
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	s.stats.Active.Add(1)
	defer s.stats.Active.Add(-1)

	// Use a per-connection context derived from the server context.
	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Watch for server shutdown in a goroutine — close conn to unblock reads.
	go func() {
		<-connCtx.Done()
		conn.Close()
	}()

	idle := s.cfg.IdleTimeout
	if idle == 0 {
		idle = 30 * time.Second
	}

	scanner := bufio.NewScanner(conn)
	for {
		conn.SetReadDeadline(time.Now().Add(idle))
		if !scanner.Scan() {
			return
		}
		line := scanner.Text()
		if strings.ToLower(line) == "quit" {
			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			fmt.Fprintf(conn, "bye\n")
			return
		}
		resp := "echo: " + line + "\n"
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		n, err := fmt.Fprint(conn, resp)
		if err != nil {
			return
		}
		s.stats.Bytes.Add(int64(n))
	}
}

func (s *Server) Shutdown() {
	s.once.Do(func() {
		close(s.quit)
		s.ln.Close()
	})
	s.wg.Wait()
}

func (s *Server) Stats() (accepted, active, bytes int64) {
	return s.stats.Accepted.Load(), s.stats.Active.Load(), s.stats.Bytes.Load()
}

// ─────────────────────────────────────────────────────────────────────────────
// CLIENT HELPER
// ─────────────────────────────────────────────────────────────────────────────

func send(addr string, messages []string) ([]string, error) {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second))

	reader := bufio.NewReader(conn)
	var responses []string

	for _, msg := range messages {
		fmt.Fprintf(conn, "%s\n", msg)
		resp, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		responses = append(responses, strings.TrimSpace(resp))
	}
	return responses, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SCENARIOS
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	cfg := ServerConfig{
		Addr:        "127.0.0.1:0",
		MaxConns:    3,
		IdleTimeout: 200 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	srv, err := NewServer(cfg)
	if err != nil {
		fmt.Println("failed to start server:", err)
		return
	}
	srv.Serve(ctx)

	fmt.Println("=== Echo server: functional test ===")
	fmt.Println()

	// Scenario 1: normal echo.
	fmt.Println("--- Scenario 1: normal echo ---")
	resps, _ := send(srv.Addr(), []string{"hello", "world", "quit"})
	for _, r := range resps {
		fmt.Printf("  %s\n", r)
	}

	// Scenario 2: idle timeout.
	fmt.Println()
	fmt.Println("--- Scenario 2: idle timeout (200ms) ---")
	conn, _ := net.DialTimeout("tcp", srv.Addr(), 2*time.Second)
	fmt.Fprintf(conn, "start\n")
	r := bufio.NewReader(conn)
	resp, _ := r.ReadString('\n')
	fmt.Printf("  first response: %s", resp)
	time.Sleep(300 * time.Millisecond) // exceed idle timeout
	_, err2 := fmt.Fprintf(conn, "second\n")
	fmt.Printf("  after idle timeout, write error: %v\n", err2)
	conn.Close()

	// Scenario 3: max connections (3 held open, 4th rejected).
	fmt.Println()
	fmt.Println("--- Scenario 3: max connections (limit=3) ---")
	holders := make([]net.Conn, 3)
	for i := range 3 {
		c, _ := net.DialTimeout("tcp", srv.Addr(), 2*time.Second)
		holders[i] = c
	}
	time.Sleep(10 * time.Millisecond)
	_, err4 := net.DialTimeout("tcp", srv.Addr(), 500*time.Millisecond)
	if err4 != nil {
		fmt.Printf("  4th connection failed (expected, rejected): %v\n", err4)
	} else {
		fmt.Println("  4th connection accepted (semaphore slot may have freed)")
	}
	for _, c := range holders {
		c.Close()
	}
	time.Sleep(50 * time.Millisecond)

	// Scenario 4: graceful shutdown.
	fmt.Println()
	fmt.Println("--- Scenario 4: graceful shutdown ---")
	conn2, _ := net.DialTimeout("tcp", srv.Addr(), 2*time.Second)
	reader2 := bufio.NewReader(conn2)
	fmt.Fprintf(conn2, "before-shutdown\n")
	resp2, _ := reader2.ReadString('\n')
	fmt.Printf("  in-flight response: %s", resp2)

	cancel()         // signal context cancel
	srv.Shutdown() // wait for all connections to drain

	_, err3 := net.DialTimeout("tcp", srv.Addr(), 200*time.Millisecond)
	fmt.Printf("  dial after shutdown: error=%v\n", err3 != nil)

	conn2.Close()

	accepted, active, bytes := srv.Stats()
	fmt.Printf("\nStats: accepted=%d active=%d bytes-echoed=%d\n", accepted, active, bytes)
}
