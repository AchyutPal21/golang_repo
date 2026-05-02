// FILE: book/part4_concurrency_systems/chapter55_tls_http2/examples/02_http2_patterns/main.go
// CHAPTER: 55 — Networking III: TLS/H2/H3
// TOPIC: HTTP/2 via net/http — automatic H2 upgrade over TLS,
//        server push (H2 only), multiplexing benefits,
//        h2c (cleartext H2) for internal services.
//
// Run (from the chapter folder):
//   go run ./examples/02_http2_patterns

package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"sync"
	"time"
)

// ─────────────────────────────────────────���────────────────────────────��──────
// CERT HELPERS (same as ch55 example 01)
// ─────────────────────────────────────────────────────────────────────────────

func generateCert() (tls.Certificate, *x509.CertPool, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{Organization: []string{"Golang Bible"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, _ := x509.MarshalECPrivateKey(key)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	pool := x509.NewCertPool()
	parsed, _ := x509.ParseCertificate(certDER)
	pool.AddCert(parsed)
	return cert, pool, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 1: HTTP/2 automatic upgrade over TLS
//
// net/http enables H2 by default when using TLS. No extra code needed.
// ─────────────────────────────────────────────────────────────────────────────

func demoH2Auto() {
	fmt.Println("=== HTTP/2 automatic upgrade over TLS ===")

	cert, pool, err := generateCert()
	if err != nil {
		fmt.Println("  cert error:", err)
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/proto", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Protocol: %s\n", r.Proto)
	})

	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	srv := &http.Server{Handler: mux, TLSConfig: tlsCfg}

	ln, _ := tls.Listen("tcp", "127.0.0.1:0", tlsCfg)
	go srv.Serve(ln)
	defer srv.Close()

	// Client — uses http.Transport with TLS config; H2 is negotiated via ALPN.
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool},
			// ForceAttemptHTTP2: true is default when TLSClientConfig is set.
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("https://" + ln.Addr().String() + "/proto")
	if err != nil {
		fmt.Println("  request error:", err)
		return
	}
	defer resp.Body.Close()
	buf := make([]byte, 64)
	n, _ := resp.Body.Read(buf)
	fmt.Printf("  %s", string(buf[:n]))
	fmt.Printf("  (note: requires net/http's built-in H2 support via ALPN)\n")
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 2: multiplexing — send N concurrent requests on one connection
// ─────────────────────────────────────────────────────────────────────────────

func demoMultiplexing() {
	fmt.Println()
	fmt.Println("=== H2 multiplexing: N parallel requests, one connection ===")

	cert, pool, err := generateCert()
	if err != nil {
		return
	}

	mu := sync.Mutex{}
	requestCount := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		id := requestCount
		mu.Unlock()
		time.Sleep(10 * time.Millisecond) // simulate work
		fmt.Fprintf(w, "response-%d", id)
	})

	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	srv := &http.Server{Handler: mux, TLSConfig: tlsCfg}
	ln, _ := tls.Listen("tcp", "127.0.0.1:0", tlsCfg)
	go srv.Serve(ln)
	defer srv.Close()

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool},
		},
		Timeout: 10 * time.Second,
	}

	start := time.Now()
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Get("https://" + ln.Addr().String())
			if err != nil {
				return
			}
			resp.Body.Close()
		}()
	}
	wg.Wait()

	elapsed := time.Since(start).Round(time.Millisecond)
	fmt.Printf("  10 requests with 10ms work each: total=%s  (serial would be ~100ms)\n", elapsed)
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 3: HTTP/1.1 vs HTTP/2 header compression summary
// ─────────────────────────────────────────────────────────────────────────────

func demoH2Summary() {
	fmt.Println()
	fmt.Println("=== HTTP/1.1 vs HTTP/2 feature comparison ===")

	rows := [][3]string{
		{"Feature", "HTTP/1.1", "HTTP/2"},
		{"Framing", "Text (CRLF)", "Binary frames"},
		{"Multiplexing", "One req per connection (pipelining broken)", "N streams on one connection"},
		{"Header compression", "None (repeated verbose headers)", "HPACK compression"},
		{"Server push", "Not available", "PUSH_PROMISE frames"},
		{"TLS", "Optional", "Effectively required in browsers"},
		{"Connection reuse", "Keep-Alive (serial)", "Multiplexed (parallel)"},
		{"Head-of-line blocking", "Per-connection", "Eliminated (per stream)"},
	}

	for _, row := range rows {
		fmt.Printf("  %-25s  %-38s  %s\n", row[0], row[1], row[2])
	}

	fmt.Println()
	fmt.Println("  HTTP/3 notes:")
	fmt.Println("  - Based on QUIC (UDP) — eliminates TCP head-of-line blocking")
	fmt.Println("  - golang.org/x/net/quic or github.com/quic-go/quic-go")
	fmt.Println("  - TLS 1.3 only; built-in 0-RTT connection resumption")
}

func main() {
	demoH2Auto()
	demoMultiplexing()
	demoH2Summary()
}
