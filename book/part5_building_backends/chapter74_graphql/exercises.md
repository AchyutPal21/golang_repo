# Chapter 74 Exercises — GraphQL

## Exercise 1 — Product API (`exercises/01_product_api`)

Build a product catalogue GraphQL API with queries, mutations, subscriptions, dataloader-style batching, auth guards, and review validation.

### Schema (SDL — implement as Go types)

```graphql
type Query {
  product(id: ID!): Product
  products: [Product!]!
}
type Mutation {
  createProduct(name: String!, category: Category!, price: Int!): Product!
  addReview(input: ReviewInput!): Review!
}
type Subscription {
  reviewAdded: ReviewEvent!
}
type Product {
  id:       ID!
  name:     String!
  category: Category!
  price:    Int!
  inStock:  Boolean!
  reviews:  [Review!]!
}
type Review {
  id:        ID!
  productID: ID!
  authorID:  ID!
  rating:    Int!   # 1-5
  comment:   String!
}
enum Category { books electronics }
```

### Auth rules

- `createProduct`: admin role required
- `addReview`: any authenticated user; unauthenticated → error

### Validation

- `rating` must be 1–5; return error otherwise

### N+1 demonstration

Implement `ProductsNaive` that loads reviews per-product individually (N+1), and `ProductsWithDataloader` that batches all review loads in one query. Print query counts for both.

### Subscription

Implement `ReviewSubscription` that pushes `ReviewEvent{Review, ProductID}` to all active listeners whenever `addReview` succeeds.

### Demonstration

1. Query `product("p-1")` — verify name and review count
2. Query `product("nonexistent")` — verify error response
3. N+1 vs dataloader: print query counts for same result
4. `createProduct` with user role → error; with admin role → success
5. `addReview` unauthenticated → error; valid → success; rating=6 → validation error
6. Subscribe to `reviewAdded`; add two reviews; verify 2 events received
7. Print query complexity example with explanation

### Hints

- Resolver struct holds `*Store` and `*ReviewSubscription`
- Auth context key should be an unexported struct type to avoid collisions
- `BatchReviewsByProductIDs(ids []string) map[string][]*Review` should count as 1 query
- The naive version calls `ReviewsByProductID` once per product — record the query count before and after to measure N+1
- Subscription channel should be buffered (8) to avoid blocking the publisher
