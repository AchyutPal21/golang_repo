# Chapter 55 — Networking III: TLS/H2/H3

## What you will learn

- TLS handshake mechanics: certificate exchange, ALPN negotiation, session keys
- Generating self-signed certificates in memory with `crypto/ecdsa` and `crypto/x509`
- Configuring `tls.Config`: `MinVersion`, `Certificates`, `RootCAs`, `ClientCAs`
- HTTPS server and TLS-aware `http.Client`
- Mutual TLS (mTLS): `ClientAuth: tls.RequireAndVerifyClientCert`
- HTTP/2: automatic upgrade via ALPN, multiplexing, header compression
- HTTP/3 overview: QUIC transport, 0-RTT, `quic-go`
- TLS best-practices: minimum version, HSTS, certificate rotation, OCSP

---

## TLS config skeleton

```go
// Server
serverTLS := &tls.Config{
    Certificates: []tls.Certificate{cert},
    MinVersion:   tls.VersionTLS12,
    ClientAuth:   tls.RequireAndVerifyClientCert, // for mTLS
    ClientCAs:    caPool,
}

// Client
clientTLS := &tls.Config{
    RootCAs:      serverCAPool,
    Certificates: []tls.Certificate{clientCert}, // for mTLS
    MinVersion:   tls.VersionTLS12,
}
client := &http.Client{
    Transport: &http.Transport{TLSClientConfig: clientTLS},
}
```

---

## Self-signed cert in memory

```go
key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
tmpl := &x509.Certificate{
    SerialNumber: big.NewInt(1),
    NotBefore:    time.Now(),
    NotAfter:     time.Now().Add(24 * time.Hour),
    KeyUsage:     x509.KeyUsageDigitalSignature,
    ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
    IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
}
certDER, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
cert, _ := tls.X509KeyPair(certPEM, keyPEM)
```

---

## Mutual TLS (mTLS)

Both sides present certificates; both sides verify the other's certificate against a trusted CA:

```
Client → Certificate (signed by shared CA)
Server → Certificate (signed by shared CA)
Server verifies client cert against ClientCAs
Client verifies server cert against RootCAs
```

Used in: Kubernetes service accounts, Istio mTLS mesh, internal service APIs.

---

## HTTP/2 in Go

Go's `net/http` enables H2 automatically when serving over TLS (via ALPN `"h2"` negotiation). No code changes needed. Benefits:

- **Multiplexing**: multiple requests/responses on one TCP connection
- **HPACK**: header compression (especially useful for repeated headers)
- **Stream priority**: hint the server on which responses matter most

---

## HTTP/3 (QUIC)

HTTP/3 runs over QUIC (UDP-based), eliminating TCP head-of-line blocking. Not in stdlib; use `github.com/quic-go/quic-go`. TLS 1.3 only. 0-RTT connection resumption for known servers.

---

## Examples

| File | Demonstrates |
|---|---|
| `examples/01_tls_server/main.go` | HTTPS, raw TLS, handshake state, best-practices table |
| `examples/02_http2_patterns/main.go` | H2 auto-upgrade, multiplexing benchmark, H1.1 vs H2 comparison |

## Exercise

`exercises/01_mtls/main.go` — CA-signed server cert + client cert, mTLS with `RequireAndVerifyClientCert`, three scenarios: valid cert (200), no cert (TLS error), rogue cert (TLS error).
