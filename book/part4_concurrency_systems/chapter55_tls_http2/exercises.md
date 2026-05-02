# Chapter 55 — Exercises

## 55.1 — Mutual TLS

Run [`exercises/01_mtls`](exercises/01_mtls/main.go).

Full CA-signed PKI: generate a CA, sign a server cert, sign an authorised client cert, and sign a rogue client cert from a different CA. Three scenarios: valid client cert (200), no client cert (TLS handshake error), rogue cert (handshake error — untrusted authority).

Try:
- Add a CN check in the handler: return 403 if `PeerCertificates[0].Subject.CommonName != "allowed-service"`.
- Implement certificate rotation: replace the server's `tls.Config.Certificates` via `GetCertificate` without restarting the server.

## 55.2 ★ — HTTPS redirect

Write middleware that redirects all HTTP traffic to HTTPS:

```go
func httpsRedirect(tlsPort string) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        target := "https://" + r.Host + r.URL.RequestURI()
        http.Redirect(w, r, target, http.StatusMovedPermanently)
    })
}
```

Run two servers: one HTTP on port 8080, one HTTPS on port 8443. Verify that an HTTP request is redirected to HTTPS.

## 55.3 ★★ — TLS certificate pinning

Implement certificate pinning on the client side: compute the SHA-256 fingerprint of the server's leaf certificate and reject any connection whose cert fingerprint does not match a known-good value.

```go
transport := &http.Transport{
    TLSClientConfig: &tls.Config{
        VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
            // compute sha256(rawCerts[0]) and compare against pinned hash
        },
    },
}
```

Test by rotating the server certificate and verifying the client rejects it.
