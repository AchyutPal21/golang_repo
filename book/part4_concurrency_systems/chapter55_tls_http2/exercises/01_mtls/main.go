// FILE: book/part4_concurrency_systems/chapter55_tls_http2/exercises/01_mtls/main.go
// CHAPTER: 55 — Networking III: TLS/H2/H3
// EXERCISE: Mutual TLS (mTLS) — both server and client authenticate with
//           certificates. Demonstrates CA → server cert + client cert signing,
//           RequireAndVerifyClientCert, and rejecting untrusted clients.
//
// Run (from the chapter folder):
//   go run ./exercises/01_mtls

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
// PKI HELPERS — generate CA + sign server/client certs
// ─────────────────────────────────────────────────────────────────────────────

type certBundle struct {
	cert    tls.Certificate
	certDER []byte
}

func generateCA() (*ecdsa.PrivateKey, *x509.Certificate, []byte, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"Golang Bible CA"}},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, nil, err
	}
	cert, err := x509.ParseCertificate(certDER)
	return key, cert, certDER, err
}

func signCert(caKey *ecdsa.PrivateKey, caCert *x509.Certificate, isServer bool, ipAddresses []net.IP) (certBundle, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return certBundle{}, err
	}
	extKeyUsage := x509.ExtKeyUsageClientAuth
	if isServer {
		extKeyUsage = x509.ExtKeyUsageServerAuth
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			Organization: []string{"Golang Bible"},
			CommonName:   func() string {
				if isServer {
					return "server"
				}
				return "client"
			}(),
		},
		NotBefore:   time.Now().Add(-time.Hour),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{extKeyUsage},
		IPAddresses: ipAddresses,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		return certBundle{}, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, _ := x509.MarshalECPrivateKey(key)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	return certBundle{cert: cert, certDER: certDER}, err
}

// ─────────────────────────────────────────────────────────────────────────────
// SCENARIO 1: valid client cert → 200
// SCENARIO 2: no client cert → 400 (TLS handshake fails)
// SCENARIO 3: wrong CA → 400 (TLS handshake fails — untrusted client)
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Mutual TLS (mTLS) ===")

	// Generate CA.
	caKey, caCert, caCertDER, err := generateCA()
	if err != nil {
		fmt.Println("CA error:", err)
		return
	}

	// Sign server cert.
	serverBundle, err := signCert(caKey, caCert, true, []net.IP{net.IPv4(127, 0, 0, 1)})
	if err != nil {
		fmt.Println("server cert error:", err)
		return
	}

	// Sign authorised client cert.
	clientBundle, err := signCert(caKey, caCert, false, nil)
	if err != nil {
		fmt.Println("client cert error:", err)
		return
	}

	// Generate a DIFFERENT CA and sign a rogue client cert.
	rogueCAKey, rogueCACert, _, err := generateCA()
	if err != nil {
		fmt.Println("rogue CA error:", err)
		return
	}
	rogueClientBundle, err := signCert(rogueCAKey, rogueCACert, false, nil)
	if err != nil {
		fmt.Println("rogue client cert error:", err)
		return
	}

	// Build CA pool for server (trusts our CA).
	caPool := x509.NewCertPool()
	parsed, _ := x509.ParseCertificate(caCertDER)
	caPool.AddCert(parsed)

	// Server TLS — require and verify client cert.
	serverTLS := &tls.Config{
		Certificates: []tls.Certificate{serverBundle.cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
		MinVersion:   tls.VersionTLS12,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/secure", func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, "no client cert", http.StatusUnauthorized)
			return
		}
		cn := r.TLS.PeerCertificates[0].Subject.CommonName
		fmt.Fprintf(w, "authenticated: CN=%s\n", cn)
	})

	srv := &http.Server{Handler: mux, TLSConfig: serverTLS}
	ln, _ := tls.Listen("tcp", "127.0.0.1:0", serverTLS)
	go srv.Serve(ln)
	defer srv.Close()
	addr := "https://" + ln.Addr().String()

	// CA pool for client (trusts server cert).
	serverCAPool := x509.NewCertPool()
	parsedServer, _ := x509.ParseCertificate(serverBundle.certDER)
	serverCAPool.AddCert(parsedServer)

	makeClient := func(clientCert *tls.Certificate) *http.Client {
		cfg := &tls.Config{
			RootCAs:    serverCAPool,
			MinVersion: tls.VersionTLS12,
		}
		if clientCert != nil {
			cfg.Certificates = []tls.Certificate{*clientCert}
		}
		return &http.Client{
			Transport: &http.Transport{TLSClientConfig: cfg},
			Timeout:   3 * time.Second,
		}
	}

	// Scenario 1: authorised client.
	fmt.Println()
	fmt.Print("Scenario 1 — valid client cert:    ")
	resp, err := makeClient(&clientBundle.cert).Get(addr + "/secure")
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		buf := make([]byte, 128)
		n, _ := resp.Body.Read(buf)
		resp.Body.Close()
		fmt.Printf("%d %s", resp.StatusCode, string(buf[:n]))
	}

	// Scenario 2: no client cert.
	fmt.Print("Scenario 2 — no client cert:       ")
	_, err2 := makeClient(nil).Get(addr + "/secure")
	if err2 != nil {
		fmt.Printf("TLS error (expected): handshake failed\n")
	} else {
		fmt.Println("unexpected success")
	}

	// Scenario 3: rogue client cert (different CA).
	fmt.Print("Scenario 3 — rogue client cert:    ")
	_, err3 := makeClient(&rogueClientBundle.cert).Get(addr + "/secure")
	if err3 != nil {
		fmt.Printf("TLS error (expected): certificate signed by unknown authority\n")
	} else {
		fmt.Println("unexpected success")
	}

	fmt.Println()
	fmt.Println("mTLS summary:")
	fmt.Println("  - Server presents cert signed by CA")
	fmt.Println("  - Client presents cert signed by the same CA")
	fmt.Println("  - Server verifies client cert against ClientCAs pool")
	fmt.Println("  - Client verifies server cert against RootCAs pool")
	fmt.Println("  - Used in service-to-service auth (Kubernetes, Istio, mutual API auth)")
}
