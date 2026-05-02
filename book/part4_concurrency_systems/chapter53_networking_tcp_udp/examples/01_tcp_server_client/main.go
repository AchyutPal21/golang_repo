// FILE: book/part4_concurrency_systems/chapter53_networking_tcp_udp/examples/01_tcp_server_client/main.go
// CHAPTER: 53 — Networking I: TCP/UDP
// TOPIC: TCP server/client — listen, accept, handle connection, read/write
//        with deadlines, graceful shutdown, concurrent connections.
//
// Run (from the chapter folder):
//   go run ./examples/01_tcp_server_client

package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// ECHO SERVER
// ─────────────────────────────────────────────────────────────────────────────

type EchoServer struct {
	ln      net.Listener
	wg      sync.WaitGroup
	once    sync.Once
	quit    chan struct{}
	addr    string
}

func NewEchoServer(addr string) (*EchoServer, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	s := &EchoServer{
		ln:   ln,
		quit: make(chan struct{}),
		addr: ln.Addr().String(),
	}
	return s, nil
}

func (s *EchoServer) Addr() string { return s.addr }

func (s *EchoServer) Serve() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			conn, err := s.ln.Accept()
			if err != nil {
				select {
				case <-s.quit:
					return // graceful shutdown
				default:
					fmt.Printf("  [server] accept error: %v\n", err)
					return
				}
			}
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				s.handleConn(conn)
			}()
		}
	}()
}

func (s *EchoServer) handleConn(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.ToLower(line) == "quit" {
			fmt.Fprintf(conn, "bye\n")
			return
		}
		// Set a write deadline to prevent slow-writer attacks.
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		fmt.Fprintf(conn, "echo: %s\n", line)
	}
}

func (s *EchoServer) Shutdown() {
	s.once.Do(func() {
		close(s.quit)
		s.ln.Close()
	})
	s.wg.Wait()
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 1: connect, send messages, read responses
// ─────────────────────────────────────────────────────────────────────────────

func demoEchoServer() {
	fmt.Println("=== TCP echo server/client ===")

	srv, err := NewEchoServer("127.0.0.1:0") // OS picks port
	if err != nil {
		fmt.Println("  listen failed:", err)
		return
	}
	srv.Serve()
	defer srv.Shutdown()

	// Client.
	conn, err := net.DialTimeout("tcp", srv.Addr(), 2*time.Second)
	if err != nil {
		fmt.Println("  dial failed:", err)
		return
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	messages := []string{"hello", "world", "quit"}
	reader := bufio.NewReader(conn)

	for _, msg := range messages {
		fmt.Fprintf(conn, "%s\n", msg)
		resp, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		fmt.Printf("  sent: %-10s  got: %s", msg, resp)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 2: concurrent clients, connection tracking
// ─────────────────────────────────────────────────────────────────────────────

func demoConcurrentClients() {
	fmt.Println()
	fmt.Println("=== Concurrent clients ===")

	srv, err := NewEchoServer("127.0.0.1:0")
	if err != nil {
		fmt.Println("  listen failed:", err)
		return
	}
	srv.Serve()
	defer srv.Shutdown()

	var wg sync.WaitGroup
	var mu sync.Mutex
	responses := make(map[int]string)

	for i := range 5 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			conn, err := net.DialTimeout("tcp", srv.Addr(), 2*time.Second)
			if err != nil {
				return
			}
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(2 * time.Second))

			fmt.Fprintf(conn, "client-%d\n", id)
			reader := bufio.NewReader(conn)
			resp, _ := reader.ReadString('\n')

			mu.Lock()
			responses[id] = strings.TrimSpace(resp)
			mu.Unlock()

			fmt.Fprintf(conn, "quit\n")
		}(i)
	}

	wg.Wait()
	for id, resp := range responses {
		fmt.Printf("  client %d: %s\n", id, resp)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 3: read/write deadline enforcement
// ─────────────────────────────────────────────────────────────────────────────

func demoDeadline() {
	fmt.Println()
	fmt.Println("=== Connection deadline ===")

	// Simple server that sends one message then sleeps.
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	go func() {
		conn, _ := ln.Accept()
		defer conn.Close()
		fmt.Fprintf(conn, "data\n")
		time.Sleep(500 * time.Millisecond) // stalls
	}()

	conn, _ := net.DialTimeout("tcp", ln.Addr().String(), 2*time.Second)
	defer conn.Close()

	// Overall deadline of 100ms — the server will stall after first message.
	conn.SetDeadline(time.Now().Add(100 * time.Millisecond))

	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				fmt.Printf("  deadline exceeded (expected): read timed out\n")
			} else {
				fmt.Printf("  read ended: %v\n", err)
			}
			break
		}
		fmt.Printf("  received: %s", line)
	}

	ln.Close()
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 4: context-controlled dialer
// ─────────────────────────────────────────────────────────────────────────────

func demoContextDial() {
	fmt.Println()
	fmt.Println("=== Context-controlled dial ===")

	// Dial a port with no listener — connection refused.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Use a listener on a random port, immediately close it so the dial sees
	// "connection refused" within the timeout.
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		fmt.Printf("  dial error (expected): %v\n", err)
		return
	}
	conn.Close()
}

func main() {
	demoEchoServer()
	demoConcurrentClients()
	demoDeadline()
	demoContextDial()
}
