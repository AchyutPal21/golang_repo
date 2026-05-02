# Chapter 57 — REST API Design

## Questions

1. What is the difference between a **safe** and an **idempotent** HTTP method? Give an example of a method that is idempotent but not safe.
2. Why should `DELETE /articles/2` return `404` on the second call rather than `204`? Some argue it should always return `204` — what is the counter-argument?
3. When should you use `400 Bad Request` vs `422 Unprocessable Entity`?
4. What does the `Location` header carry on a `201 Created` response, and why is it important for clients?
5. What is HATEOAS, and what practical benefit does it provide over a Level 2 REST API?
6. Compare offset-based and cursor-based pagination: under what conditions does offset pagination produce incorrect results?

## Answers

1. **Safe** means the method has no observable side effects — reading does not change server state. **Idempotent** means calling the method N times produces the same result as calling it once. `DELETE` is idempotent but not safe: once a resource is deleted, deleting it again leaves the system in the same state (resource absent), but the first call did change state (removed the resource). `PUT` is also idempotent but not safe: replacing a resource with the same payload leaves state unchanged on subsequent calls.

2. The idempotency principle guarantees that repeated calls produce the same **server-side state**, not the same response code. After the first `DELETE /articles/2`, the article is absent. On the second call the article is still absent — the server state is identical, satisfying idempotency. Returning `404` is correct because the article genuinely does not exist; returning `204` would be misleading (it implies something was deleted). The counter-argument is that clients should not need to distinguish "just deleted" from "already absent." In practice, the `404` approach is more common because it accurately reflects the current resource state and helps clients detect bugs (e.g., deleting a wrong ID).

3. `400 Bad Request` signals a **syntactic** problem — the request cannot be parsed or is structurally invalid (malformed JSON, wrong Content-Type, missing required HTTP headers). `422 Unprocessable Entity` signals a **semantic** problem — the request was parsed successfully but the payload violates business rules (missing a required field, invalid enum value, referential integrity violation). In practice: if `json.Decode` fails → `400`; if validation of the decoded struct fails → `422`.

4. The `Location` header carries the URI of the newly created resource (e.g., `Location: /articles/6`). This is important because POST is the method for creation and the client does not know the server-assigned ID in advance. Without `Location`, the client must either parse the response body to extract the ID or make a follow-up `GET /articles` request to find the new resource. The `Location` header gives clients a direct, unambiguous pointer to the resource they just created, enabling them to immediately `GET`, `PUT`, or link to it.

5. **HATEOAS** (Hypermedia as the Engine of Application State) is REST Level 3: responses include `_links` describing which actions are available and how to invoke them (rel, href, method). A Level 2 API requires clients to hard-code URIs and know which methods are valid — any change to URI structure breaks clients. With HATEOAS, the server advertises available transitions dynamically; clients follow links rather than constructing URIs. Practical benefit: the API becomes self-describing and can omit links for unavailable actions (e.g., `publish` link disappears once an article is already published), guiding clients through valid state transitions without out-of-band documentation.

6. **Offset pagination** works by skipping `offset` rows before returning `limit` rows. If a new item is inserted between page 1 and page 2 while a client is paginating, all subsequent pages shift by one position — the client either sees a duplicate item or skips an item. This is called **page drift**. It is worst on high-write collections (social feeds, log streams). **Cursor pagination** avoids page drift by anchoring on a stable property (the last seen ID) and returning items with ID > cursor; inserts at the beginning do not affect what the client has already seen. The trade-off is that cursor pagination cannot jump to an arbitrary page — it only supports sequential forward traversal.
