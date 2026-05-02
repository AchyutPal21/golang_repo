// FILE: book/part4_concurrency_systems/chapter53_networking_tcp_udp/examples/02_udp_patterns/main.go
// CHAPTER: 53 — Networking I: TCP/UDP
// TOPIC: UDP server/client — connectionless vs connected UDP, unreliable
//        delivery model, broadcast, and choosing TCP vs UDP.
//
// Run (from the chapter folder):
//   go run ./examples/02_udp_patterns

package main

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 1: basic UDP echo (connectionless)
//
// UDP server uses ReadFromUDP/WriteToUDP — it does not maintain per-client state.
// ─────────────────────────────────────────────────────────────────────────────

func demoUDPEcho() {
	fmt.Println("=== UDP echo (connectionless) ===")

	// Server.
	serverConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("  listen failed:", err)
		return
	}
	addr := serverConn.LocalAddr().String()

	var svrWg sync.WaitGroup
	svrWg.Add(1)
	go func() {
		defer svrWg.Done()
		buf := make([]byte, 1024)
		for {
			serverConn.SetDeadline(time.Now().Add(500 * time.Millisecond))
			n, clientAddr, err := serverConn.ReadFrom(buf)
			if err != nil {
				return // deadline or close
			}
			msg := string(buf[:n])
			serverConn.WriteTo([]byte("echo: "+msg), clientAddr)
		}
	}()

	// Client — "connected" UDP: Dial records the remote addr, no need to pass it each write.
	client, err := net.Dial("udp", addr)
	if err != nil {
		fmt.Println("  dial failed:", err)
		return
	}
	defer client.Close()
	client.SetDeadline(time.Now().Add(2 * time.Second))

	messages := []string{"ping", "hello", "world"}
	buf := make([]byte, 1024)

	for _, msg := range messages {
		client.Write([]byte(msg))
		n, _ := client.Read(buf)
		fmt.Printf("  sent: %-8s  got: %s\n", msg, string(buf[:n]))
	}

	serverConn.Close()
	svrWg.Wait()
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 2: UDP is unreliable — demonstrate dropped packets
//
// We don't actually drop packets in loopback, but we simulate the pattern
// of "fire and forget" by not waiting for responses.
// ─────────────────────────────────────────────────────────────────────────────

func demoUDPFireAndForget() {
	fmt.Println()
	fmt.Println("=== UDP fire-and-forget (no ACK) ===")

	serverConn, _ := net.ListenPacket("udp", "127.0.0.1:0")
	addr := serverConn.LocalAddr().String()

	received := make(chan string, 20)
	go func() {
		buf := make([]byte, 256)
		for {
			serverConn.SetDeadline(time.Now().Add(200 * time.Millisecond))
			n, _, err := serverConn.ReadFrom(buf)
			if err != nil {
				close(received)
				return
			}
			received <- string(buf[:n])
		}
	}()

	client, _ := net.Dial("udp", addr)
	defer client.Close()

	// Send 10 packets, no waiting for ACK.
	for i := range 10 {
		client.Write([]byte(fmt.Sprintf("pkt-%d", i)))
	}

	serverConn.Close()

	count := 0
	for range received {
		count++
	}
	fmt.Printf("  sent 10 packets, server received %d\n", count)
	fmt.Println("  (loopback rarely drops; real networks may lose packets)")
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 3: multiple clients talking to one UDP server
// ─────────────────────────────────────────────────────────────────────────────

func demoUDPMultiClient() {
	fmt.Println()
	fmt.Println("=== UDP: multiple clients, single server ===")

	serverConn, _ := net.ListenPacket("udp", "127.0.0.1:0")
	addr := serverConn.LocalAddr().String()

	var svrWg sync.WaitGroup
	svrWg.Add(1)
	go func() {
		defer svrWg.Done()
		buf := make([]byte, 256)
		served := 0
		for served < 5 {
			serverConn.SetDeadline(time.Now().Add(500 * time.Millisecond))
			n, clientAddr, err := serverConn.ReadFrom(buf)
			if err != nil {
				return
			}
			resp := fmt.Sprintf("ACK:%s", string(buf[:n]))
			serverConn.WriteTo([]byte(resp), clientAddr)
			served++
		}
	}()

	var wg sync.WaitGroup
	for i := range 5 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			c, _ := net.Dial("udp", addr)
			defer c.Close()
			c.SetDeadline(time.Now().Add(2 * time.Second))
			c.Write([]byte(fmt.Sprintf("client-%d", id)))
			buf := make([]byte, 256)
			n, _ := c.Read(buf)
			fmt.Printf("  client %d: %s\n", id, string(buf[:n]))
		}(i)
	}

	wg.Wait()
	serverConn.Close()
	svrWg.Wait()
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 4: TCP vs UDP summary
// ─────────────────────────────────────────────────────────────────────────────

func demoComparison() {
	fmt.Println()
	fmt.Println("=== TCP vs UDP cheat-sheet ===")
	table := [][3]string{
		{"Feature", "TCP", "UDP"},
		{"Connection", "3-way handshake required", "Connectionless"},
		{"Delivery", "Guaranteed, ordered", "Best-effort, unordered"},
		{"Flow control", "Yes (sliding window)", "None"},
		{"Overhead", "Higher (headers + retransmit)", "Lower (8-byte header)"},
		{"Use cases", "HTTP, databases, file transfer", "DNS, video, gaming, telemetry"},
	}
	for _, row := range table {
		fmt.Printf("  %-18s  %-30s  %s\n", row[0], row[1], row[2])
	}
}

func main() {
	demoUDPEcho()
	demoUDPFireAndForget()
	demoUDPMultiClient()
	demoComparison()
}
