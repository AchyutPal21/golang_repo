# Chapter 55 — Revision Checkpoint

## Questions

1. What is ALPN and how does it enable HTTP/2 negotiation within a TLS connection?
2. What are the five `ClientAuth` modes in Go's `tls.Config`, and when would you use `RequireAndVerifyClientCert`?
3. Why is HTTP/2 multiplexing faster than HTTP/1.1 keep-alive for multiple concurrent requests?
4. What is the difference between `RootCAs` and `ClientCAs` in `tls.Config`, and which side sets each?
5. What makes HTTP/3 fundamentally different from HTTP/1.1 and HTTP/2 at the transport layer?

## Answers

1. ALPN (Application-Layer Protocol Negotiation) is a TLS extension that lets the client and server agree on an application protocol during the TLS handshake, before any application data is sent. The client sends a list of supported protocols in the `ClientHello` (e.g., `["h2", "http/1.1"]`); the server picks one and includes it in the `ServerHello`. If `"h2"` is selected, both sides know to use HTTP/2 framing on that connection. This eliminates the need for an additional round-trip Upgrade mechanism used by HTTP/1.1.

2. The five modes are: (1) `NoClientCert` — no client cert requested (default); (2) `RequestClientCert` — request a cert but do not require it; (3) `RequireAnyClientCert` — require a cert but don't verify it against a CA; (4) `VerifyClientCertIfGiven` — verify if provided, but don't require it; (5) `RequireAndVerifyClientCert` — require a cert and verify it against `ClientCAs`. Use `RequireAndVerifyClientCert` when you want mutual authentication (mTLS) — the client must present a certificate signed by a trusted CA before the TLS handshake completes. This is used in service-mesh architectures, internal APIs, and anywhere a machine-identity guarantee is needed without usernames/passwords.

3. HTTP/1.1 with keep-alive reuses the TCP connection but can only send one request at a time — the next request must wait for the previous response (head-of-line blocking at the application layer). HTTP/2 sends multiple requests as independent streams within one TCP connection simultaneously. Each stream has its own ID; responses can be interleaved in any order. A slow response (large file, slow query) does not block fast responses. For N concurrent requests each taking T time, HTTP/1.1 serial cost is N×T; HTTP/2 multiplexed cost approaches T (parallel) if the server can process them concurrently.

4. `RootCAs` is set on the **client** and contains the certificates the client trusts to sign the **server's** certificate. If nil, the system's default CA store is used. `ClientCAs` is set on the **server** and contains the certificates the server trusts to sign the **client's** certificate — used for mTLS. The naming reflects direction: "Root CAs" are the trust anchors for outbound connections; "Client CAs" are the trust anchors for verifying inbound client certificates.

5. HTTP/1.1 and HTTP/2 both run over TCP. TCP is a reliable, ordered byte stream — a single lost packet stalls the entire connection until retransmitted (TCP head-of-line blocking). HTTP/2's stream multiplexing mitigates this at the application layer but cannot eliminate it at the transport layer. HTTP/3 runs over **QUIC** (UDP-based), which implements reliability, ordering, and congestion control per-stream. A lost packet in stream 3 does not stall stream 1 — each stream has independent retransmission. QUIC also integrates TLS 1.3 natively (the initial QUIC handshake establishes the encrypted session), enabling 0-RTT connection resumption for known servers.
