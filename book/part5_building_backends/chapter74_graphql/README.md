# Chapter 74 — GraphQL

## What you'll learn

How GraphQL works: schema definition, queries, mutations, subscriptions, resolvers, the N+1 problem and dataloader fix, cursor pagination, auth in resolvers, and field-level errors. Examples are in-process simulations; see the gqlgen section for real setup.

## Key concepts

| Concept | Description |
|---|---|
| Schema | SDL type definitions; the contract between client and server |
| Query | Read-only data fetch; client specifies exactly which fields it needs |
| Mutation | Write operation; returns the modified object |
| Subscription | Long-lived push channel (WebSocket); server sends events |
| Resolver | Function that returns data for a specific field |
| N+1 problem | Loading N child records with N+1 separate queries instead of 1 batched query |
| Dataloader | Batching pattern: collect IDs within a tick, then execute one query |
| Cursor pagination | `first` / `after` (cursor) instead of offset — stable under inserts |
| Field-level errors | GraphQL can return partial data and errors in the same response |
| Query complexity | Cost analysis to block deeply nested or expensive queries |

## Files

| File | Topic |
|---|---|
| `examples/01_graphql_basics/main.go` | Schema types, resolvers, N+1 vs dataloader, mutation, error handling |
| `examples/02_graphql_patterns/main.go` | Auth, cursor pagination, subscriptions, partial errors, REST/gRPC comparison |
| `exercises/01_product_api/main.go` | Product catalogue, reviews, auth guards, subscription, query complexity |

## gqlgen setup (real GraphQL)

```bash
go get github.com/99designs/gqlgen
go run github.com/99designs/gqlgen init
```

1. Write `graph/schema.graphqls`
2. Run `go generate ./...` (or `go run github.com/99designs/gqlgen generate`)
3. Implement resolver methods in `graph/resolver.go`

## Schema definition (SDL)

```graphql
type Query {
  user(id: ID!): User
  users: [User!]!
}

type Mutation {
  createOrder(input: CreateOrderInput!): Order!
}

type Subscription {
  orderUpdated(id: ID!): Order!
}

type User {
  id:     ID!
  name:   String!
  orders: [Order!]!    # resolved lazily by a resolver
}
```

## Resolver pattern (gqlgen)

```go
// QueryResolver
func (r *queryResolver) User(ctx context.Context, id string) (*model.User, error) {
    return r.store.FindUser(ctx, id)
}

// Nested resolver for User.Orders (called per User)
func (r *userResolver) Orders(ctx context.Context, u *model.User) ([]*model.Order, error) {
    return r.loaders.OrdersByUserID(ctx, u.ID) // dataloader batches these
}
```

## N+1 problem and dataloader fix

```go
// N+1: one query per user (bad)
for _, user := range users {
    user.Orders, _ = store.OrdersByUserID(ctx, user.ID) // N separate queries!
}

// Dataloader: collect IDs, one batched query (good)
// github.com/graph-gophers/dataloader
loader := dataloader.NewBatchedLoader(func(keys []string) []*dataloader.Result {
    orders, _ := store.BatchOrdersByUserIDs(ctx, keys)
    // Map results back to key positions.
    ...
})
```

## Auth in resolvers

```go
func (r *mutationResolver) CreateProduct(ctx context.Context, input model.CreateProductInput) (*model.Product, error) {
    user := auth.FromContext(ctx)
    if user.Role != auth.RoleAdmin {
        return nil, fmt.Errorf("permission denied")
    }
    return r.store.CreateProduct(ctx, input)
}
```

## Cursor pagination (Relay spec)

```graphql
type ProductConnection {
    edges:    [ProductEdge!]!
    pageInfo: PageInfo!
    total:    Int!
}
type ProductEdge {
    node:   Product!
    cursor: String!
}
type PageInfo {
    hasNextPage: Boolean!
    endCursor:   String
}
```

```go
// Query: products(first: 10, after: "cursor-xyz") { edges { node { id } } pageInfo { ... } }
```

## Production notes

- Always implement query depth limiting and cost analysis — unbounded nested queries can DDoS your DB
- Subscriptions require WebSocket; use a message broker (Redis pub/sub) to fan out across server instances
- N+1 is the #1 production issue with GraphQL — use dataloaders for every nested list resolver
- Introspection should be disabled in production (exposes your full schema)
- Use persisted queries in production to prevent arbitrary query execution from clients
