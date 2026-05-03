# Chapter 78 Checkpoint — Rate Limiting, Circuit Breakers, Retries

## Self-assessment questions

1. What is the "burst edge" problem with fixed-window rate limiting? How does a sliding window fix it?
2. Describe the three states of a circuit breaker. What triggers each transition?
3. Why must exponential backoff include jitter? What happens without it in a system with many clients?
4. When should you NOT retry a failed request? Give three concrete examples.
5. What is a retry budget and why does it prevent cascade failures?

## Checklist

- [ ] Can implement a token bucket with burst capacity and a configurable refill rate
- [ ] Can implement a per-key limiter (one bucket per user/IP)
- [ ] Can implement a circuit breaker with Closed, Open, and Half-Open states
- [ ] Can implement exponential backoff with configurable multiplier and jitter
- [ ] Can implement a `RetryOn` predicate to skip retries on non-retriable errors
- [ ] Can implement hedged requests that issue a duplicate after a delay
- [ ] Can combine rate limiter + circuit breaker + retry into a single policy

## Answers

1. Fixed window: if the limit is 100 req/min, a client can send 100 at 00:59 and 100 at 01:00 — 200 requests in 2 seconds, doubling the effective rate at the window boundary. Sliding window tracks the count over the last N seconds at any point in time (using sub-buckets), so the true rolling count never exceeds the limit. There is no boundary spike.

2. **Closed**: normal operation; all requests pass through; failure counter increments on each error. **Open**: entered when consecutive failures reach `FailureThreshold`; all requests are rejected immediately with an error, protecting the downstream service from more load. **Half-Open**: entered after `OpenTimeout`; one probe request is allowed through; if it succeeds (and success count reaches `SuccessThreshold`), transitions back to Closed; if it fails, returns to Open.

3. Without jitter, N clients that all backed off for the same duration all retry simultaneously — creating a thundering herd that sends the same spike load that triggered the failure in the first place. Jitter spreads retries over a window (`delay ± delay × jitter`), so the load ramps up gradually and the system can absorb it. The fix is cheap (one `rand.Float64()` call) and critical for any system with more than a handful of clients.

4. Do not retry: (1) **400 Bad Request** — the request is malformed; retrying sends the same bad input again. (2) **401 Unauthorized** — the token is invalid; retrying without a new token always fails. (3) **409 Conflict** — a resource conflict exists (e.g., duplicate key); retrying creates the same conflict. Also do not retry non-idempotent writes that may have partially succeeded.

5. A retry budget is a global token bucket that caps the total number of retries per second across all goroutines. Without it, if 50% of requests fail, each client retries 3× — tripling the total load on an already overloaded service, making recovery impossible. The retry budget limits retries to, say, 10% of total request volume. Requests that can't get a retry token fail fast instead of amplifying the load, giving the downstream service headroom to recover.
