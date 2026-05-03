# Chapter 74 Checkpoint — GraphQL

## Self-assessment questions

1. What is the N+1 problem in GraphQL? Give a concrete example and explain the dataloader fix.
2. How does cursor pagination differ from offset pagination? Why is it preferred for large datasets?
3. How does GraphQL handle errors differently from REST? What does a response look like when a query partially succeeds?
4. Why is query complexity limiting important in production? What kind of query would crash your database without it?
5. When would you choose GraphQL over REST? When would REST still be the better choice?

## Checklist

- [ ] Can define a GraphQL schema in SDL with Query, Mutation, and Subscription
- [ ] Can implement resolvers for root fields and nested fields
- [ ] Can explain the N+1 problem and how dataloader batch-loading solves it
- [ ] Can implement cursor-based pagination returning `Connection` / `Edge` / `PageInfo`
- [ ] Can implement auth guards in resolvers using context
- [ ] Can return partial data + field-level errors in a single response
- [ ] Can set up a subscription that pushes events over a channel

## Answers

1. When fetching a list of N users each with their orders, a naive resolver loads 1 query for users + 1 query per user for orders = N+1 queries. For 100 users that's 101 DB queries. Dataloader fixes this by collecting all user IDs within a single event-loop tick (or timer window) and issuing one `SELECT * FROM orders WHERE user_id IN (?)` query. The result is mapped back to each user. In Go, `graph-gophers/dataloader` implements this pattern.

2. Offset pagination (`OFFSET 50 LIMIT 10`) is unstable: if rows are inserted between requests, the same row can appear on two pages or a row can be skipped. Cursor pagination uses the last seen record's ID as a stable pointer; `WHERE id > cursor LIMIT 10` always returns the correct next page regardless of inserts. Cursors also avoid the performance cost of `OFFSET` scans on large tables.

3. REST returns a non-200 status code on error and typically has no data. GraphQL always returns 200 and the response always has a top-level `data` and optional `errors` array. A query for three users where one doesn't exist returns data for the two found users and an error entry (with `path`) for the missing one — partial success in a single request. Clients must check both `data` and `errors`.

4. A deeply nested query like `products { reviews { author { purchaseHistory { items { product { reviews { ... } } } } } } }` multiplies fan-out at each level: if each product has 10 reviews and each review has an author with 20 orders, the query fetches 10×20 = 200 rows for each product. Without cost analysis, a malicious client can craft a query that loads millions of rows and OOMs the server. Set a max query depth (e.g., 5 levels) and/or a cost budget per field.

5. **Choose GraphQL** when: the client needs flexible field selection (mobile app with bandwidth constraints), you're building a BFF aggregating multiple microservices, the schema evolves rapidly, or you need subscriptions. **Stick with REST** when: building public APIs (GraphQL introspection exposes your schema), need HTTP-level caching via CDN, handling file uploads/downloads, using webhooks, or the team is more familiar with REST tooling.
