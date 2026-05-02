// FILE: book/part4_concurrency_systems/chapter55_tls_http2/examples/01_tls_server/main.go
// CHAPTER: 55 — Networking III: TLS/H2/H3
// TOPIC: TLS server and client — generating self-signed certificates in memory,
//        configuring tls.Config, HTTPS server, verifying TLS state, minimum
//        TLS version, cipher suite selection.
//
// Run (from the chapter folder):
//   go run ./examples/01_tls_server

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
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// SELF-SIGNED CERTIFICATE (in-memory, no files)
// ─────────────────────────────────────────────────────────────────────────────

func generateSelfSigned() (tls.Certificate, *x509.CertPool, error) {
	// Generate ECDSA P-256 key.
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	// Create certificate template.
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"Golang Bible"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:     []string{"localhost"},
	}

	// Self-sign: issuer == subject.
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	// Encode to PEM.
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	// Build CA pool from the self-signed cert.
	pool := x509.NewCertPool()
	parsed, _ := x509.ParseCertificate(certDER)
	pool.AddCert(parsed)

	return cert, pool, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 1: HTTPS server + TLS-aware client
// ─────────────────────────────────────────────────────────────────────────────

func demoHTTPS() {
	fmt.Println("=== HTTPS with self-signed certificate ===")

	cert, pool, err := generateSelfSigned()
	if err != nil {
		fmt.Println("  cert gen failed:", err)
		return
	}

	// Server TLS config — TLS 1.2+ only.
	serverTLS := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		state := r.TLS
		fmt.Fprintf(w, "TLS version: 0x%04x  cipher: 0x%04x\n",
			state.Version, state.CipherSuite)
	})

	srv := &http.Server{Handler: mux, TLSConfig: serverTLS}

	ln, _ := tls.Listen("tcp", "127.0.0.1:0", serverTLS)
	go srv.Serve(ln)
	defer srv.Close()

	// Client with custom CA pool.
	clientTLS := &tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS12,
	}
	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: clientTLS},
		Timeout:   5 * time.Second,
	}

	resp, err := client.Get("https://" + ln.Addr().String())
	if err != nil {
		fmt.Println("  request failed:", err)
		return
	}
	defer resp.Body.Close()

	buf := make([]byte, 256)
	n, _ := resp.Body.Read(buf)
	fmt.Printf("  status: %d\n  %s", resp.StatusCode, string(buf[:n]))
	fmt.Printf("  TLS version names: TLS1.2=0x%04x  TLS1.3=0x%04x\n",
		tls.VersionTLS12, tls.VersionTLS13)
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 2: raw TLS connection — inspect handshake state
// ─────────────────────────────────────────────────────────────────────────────

func demoRawTLS() {
	fmt.Println()
	fmt.Println("=== Raw TLS connection: inspect handshake ===")

	cert, pool, err := generateSelfSigned()
	if err != nil {
		return
	}

	ln, _ := tls.Listen("tcp", "127.0.0.1:0",
		&tls.Config{Certificates: []tls.Certificate{cert}})
	defer ln.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		tlsConn := conn.(*tls.Conn)
		if err := tlsConn.Handshake(); err != nil {
			fmt.Println("  server handshake error:", err)
			return
		}
		state := tlsConn.ConnectionState()
		fmt.Printf("  [server] TLS version: 0x%04x  negotiated protocol: %q\n",
			state.Version, state.NegotiatedProtocol)
		// Echo one line.
		buf := make([]byte, 64)
		n, _ := tlsConn.Read(buf)
		tlsConn.Write(buf[:n])
	}()

	conn, _ := tls.Dial("tcp", ln.Addr().String(),
		&tls.Config{RootCAs: pool})
	defer conn.Close()

	conn.Write([]byte("hello\n"))
	buf := make([]byte, 64)
	n, _ := conn.Read(buf)
	fmt.Printf("  [client] echo: %q\n", string(buf[:n]))

	state := conn.ConnectionState()
	fmt.Printf("  [client] TLS version: 0x%04x  server name: %q\n",
		state.Version, state.ServerName)

	<-done
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 3: TLS config best practices summary
// ─────────────────────────────────────────────────────────────────────────────

func demoTLSBestPractices() {
	fmt.Println()
	fmt.Println("=== TLS best-practices checklist ===")

	checks := []struct {
		item   string
		config string
	}{
		{"Minimum TLS version", "MinVersion: tls.VersionTLS12 (or TLS13)"},
		{"No SSLv3/TLS1.0/1.1", "Enforced by MinVersion >= TLS12"},
		{"Certificate rotation", "GetCertificate func(*tls.ClientHelloInfo) (*tls.Certificate, error)"},
		{"Client cert auth (mTLS)", "ClientAuth: tls.RequireAndVerifyClientCert"},
		{"Session tickets", "SessionTicketsDisabled: true for forward secrecy"},
		{"OCSP stapling", "Set in GetCertificate via tls.Certificate.OCSPStaple"},
		{"HSTS header", "w.Header().Set(\"Strict-Transport-Security\", \"max-age=63072000\")"},
	}

	for _, c := range checks {
		fmt.Printf("  %-30s  %s\n", c.item, c.config)
	}
}

func main() {
	demoHTTPS()
	demoRawTLS()
	demoTLSBestPractices()
}
