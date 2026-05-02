# Chapter 60 — Authentication

## Questions

1. Why should passwords be stored as bcrypt hashes rather than SHA-256 hashes?
2. What are the three critical `Set-Cookie` flags for a session cookie, and what attack does each prevent?
3. Explain the three segments of a JWT and how the signature is computed.
4. What is refresh token rotation, and why is it more secure than a long-lived, reusable refresh token?
5. What is the main disadvantage of JWTs compared to sessions when it comes to token revocation?

## Answers

1. **Speed.** SHA-256 is designed to be fast — a modern GPU can compute billions of SHA-256 hashes per second, making brute-force and dictionary attacks feasible even on a salted hash. bcrypt is designed to be slow: at cost 10, it takes ~100ms per hash, which is negligible for legitimate logins but makes brute-forcing ~10 billion times more expensive. bcrypt also incorporates a random salt internally, preventing precomputed rainbow table attacks. If the password database is stolen, bcrypt-hashed passwords require years to crack rather than minutes.

2. **(1) `HttpOnly: true`** — the cookie cannot be read by JavaScript (`document.cookie`). Prevents an XSS vulnerability from stealing the session token — even if the attacker injects script, they cannot exfiltrate the cookie. **(2) `SameSite: Lax` or `Strict`** — the browser does not send the cookie on cross-site requests initiated by a third-party site. Prevents CSRF attacks where a malicious page tricks the user's browser into making an authenticated request to your site. **(3) `Secure: true`** — the cookie is only sent over HTTPS. Prevents a man-in-the-middle attack on an unencrypted connection from stealing the session token.

3. A JWT is `base64url(header) + "." + base64url(payload) + "." + base64url(signature)`. The **header** specifies the algorithm (`{"alg":"HS256","typ":"JWT"}`). The **payload** contains claims — structured data about the user and the token itself (`sub`, `exp`, `iat`, `role`). The **signature** is `HMAC-SHA256(secretKey, header + "." + payload)`. Since the signature is computed over the exact header and payload bytes, any tampering with either segment (changing the role, extending the expiry) produces a different signature that does not match, and validation fails.

4. **Refresh token rotation** means each refresh token is single-use: when a client exchanges it for a new access token, the server deletes the old refresh token and issues a brand-new one. This limits the damage from a stolen refresh token — if an attacker uses it once to get a new access token, the legitimate client's next refresh attempt will fail (the token was already consumed), alerting the system to a possible compromise. Without rotation, a stolen refresh token is usable forever until its TTL expires. Rotation also enables detection: if the same refresh token is presented twice, the second use is a signal that the token has been stolen and the account should be locked.

5. **Revocation.** A session can be invalidated instantly by deleting it from the server-side session store. Because JWTs are stateless — the server stores nothing — there is no built-in way to revoke a token before its expiry (`exp`). If a user logs out or an account is compromised, the issued access tokens remain valid until they expire. Solutions are either (a) maintain a blocklist of revoked JWI IDs (adds server state, partially negates the stateless benefit) or (b) use very short-lived access tokens (15 minutes) and rely on refresh token rotation to limit the exposure window.
