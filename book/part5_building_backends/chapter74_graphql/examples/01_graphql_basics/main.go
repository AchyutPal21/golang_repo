// FILE: book/part5_building_backends/chapter74_graphql/examples/01_graphql_basics/main.go
// CHAPTER: 74 — GraphQL
// TOPIC: GraphQL fundamentals — schema, queries, mutations, resolvers, N+1 problem.
//        Simulated in-process (no external library) to illustrate the execution model.
//        See README for real gqlgen setup.
//
// Run (from the chapter folder):
//   go run ./examples/01_graphql_basics

package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

// ─────────────────────────────────────────────────────────────────────────────
// SCHEMA TYPES (mirrors gqlgen generated types)
// ─────────────────────────────────────────────────────────────────────────────

type User struct {
	ID       string
	Name     string
	Email    string
	// Resolved lazily by a resolver.
	orders   []*Order
	ordersLoaded bool
}

type Order struct {
	ID         string
	UserID     string
	TotalCents int
	Status     string
	Items      []*OrderItem
}

type OrderItem struct {
	ProductID string
	Quantity  int
	Price     int
}

type Product struct {
	ID    string
	Name  string
	Price int
}

type CreateOrderInput struct {
	UserID string
	Items  []OrderItemInput
}

type OrderItemInput struct {
	ProductID string
	Quantity  int
}

// ─────────────────────────────────────────────────────────────────────────────
// DATA STORE
// ─────────────────────────────────────────────────────────────────────────────

type Store struct {
	mu       sync.RWMutex
	users    map[string]*User
	orders   map[string]*Order
	products map[string]*Product
	nextID   int

	// Metrics.
	dbQueries atomic.Int64
}

func NewStore() *Store {
	s := &Store{
		users:    make(map[string]*User),
		orders:   make(map[string]*Order),
		products: make(map[string]*Product),
		nextID:   1,
	}
	s.seed()
	return s
}

func (s *Store) seed() {
	s.users["u-1"] = &User{ID: "u-1", Name: "Alice", Email: "alice@example.com"}
	s.users["u-2"] = &User{ID: "u-2", Name: "Bob", Email: "bob@example.com"}
	s.users["u-3"] = &User{ID: "u-3", Name: "Carol", Email: "carol@example.com"}

	s.products["p-1"] = &Product{ID: "p-1", Name: "Go Book", Price: 3999}
	s.products["p-2"] = &Product{ID: "p-2", Name: "Keyboard", Price: 8999}

	s.orders["o-1"] = &Order{ID: "o-1", UserID: "u-1", TotalCents: 3999, Status: "delivered",
		Items: []*OrderItem{{ProductID: "p-1", Quantity: 1, Price: 3999}}}
	s.orders["o-2"] = &Order{ID: "o-2", UserID: "u-1", TotalCents: 8999, Status: "shipped",
		Items: []*OrderItem{{ProductID: "p-2", Quantity: 1, Price: 8999}}}
	s.orders["o-3"] = &Order{ID: "o-3", UserID: "u-2", TotalCents: 3999, Status: "placed",
		Items: []*OrderItem{{ProductID: "p-1", Quantity: 1, Price: 3999}}}
	s.nextID = 4
}

func (s *Store) GetUser(id string) (*User, bool) {
	s.dbQueries.Add(1)
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	return u, ok
}

func (s *Store) AllUsers() []*User {
	s.dbQueries.Add(1)
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, u)
	}
	return out
}

func (s *Store) OrdersByUserID(userID string) []*Order {
	s.dbQueries.Add(1) // one query per user = N+1 if not batched
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Order
	for _, o := range s.orders {
		if o.UserID == userID {
			out = append(out, o)
		}
	}
	return out
}

// BatchOrdersByUserIDs loads orders for multiple users in one query.
func (s *Store) BatchOrdersByUserIDs(userIDs []string) map[string][]*Order {
	s.dbQueries.Add(1) // one query total regardless of N users
	s.mu.RLock()
	defer s.mu.RUnlock()
	idSet := make(map[string]bool, len(userIDs))
	for _, id := range userIDs {
		idSet[id] = true
	}
	result := make(map[string][]*Order, len(userIDs))
	for _, o := range s.orders {
		if idSet[o.UserID] {
			result[o.UserID] = append(result[o.UserID], o)
		}
	}
	return result
}

func (s *Store) CreateOrder(input CreateOrderInput) *Order {
	s.dbQueries.Add(1)
	s.mu.Lock()
	defer s.mu.Unlock()
	id := fmt.Sprintf("o-%d", s.nextID)
	s.nextID++
	var total int
	var items []*OrderItem
	for _, i := range input.Items {
		p := s.products[i.ProductID]
		price := 0
		if p != nil {
			price = p.Price * i.Quantity
		}
		total += price
		items = append(items, &OrderItem{ProductID: i.ProductID, Quantity: i.Quantity, Price: price})
	}
	order := &Order{ID: id, UserID: input.UserID, TotalCents: total, Status: "placed", Items: items}
	s.orders[id] = order
	return order
}

// ─────────────────────────────────────────────────────────────────────────────
// RESOLVER (simulates gqlgen resolver struct)
// ─────────────────────────────────────────────────────────────────────────────

type Resolver struct {
	store *Store
}

// queryUser resolves: query { user(id: "u-1") { id name orders { id status } } }
func (r *Resolver) queryUser(ctx context.Context, id string, fields []string) string {
	u, ok := r.store.GetUser(id)
	if !ok {
		return `{"errors":[{"message":"user not found"}]}`
	}
	parts := r.resolveUserFields(u, fields)
	return fmt.Sprintf(`{"data":{"user":{%s}}}`, strings.Join(parts, ","))
}

func (r *Resolver) resolveUserFields(u *User, fields []string) []string {
	var parts []string
	for _, f := range fields {
		switch f {
		case "id":
			parts = append(parts, fmt.Sprintf(`"id":"%s"`, u.ID))
		case "name":
			parts = append(parts, fmt.Sprintf(`"name":"%s"`, u.Name))
		case "email":
			parts = append(parts, fmt.Sprintf(`"email":"%s"`, u.Email))
		case "orders":
			orders := r.store.OrdersByUserID(u.ID)
			var orderParts []string
			for _, o := range orders {
				orderParts = append(orderParts, fmt.Sprintf(`{"id":"%s","status":"%s","total":%d}`, o.ID, o.Status, o.TotalCents))
			}
			parts = append(parts, fmt.Sprintf(`"orders":[%s]`, strings.Join(orderParts, ",")))
		}
	}
	return parts
}

// queryAllUsers with N+1 problem.
func (r *Resolver) queryAllUsersNaive(ctx context.Context) string {
	users := r.store.AllUsers()
	var userParts []string
	for _, u := range users {
		// BUG: one DB query per user to load their orders — N+1!
		parts := r.resolveUserFields(u, []string{"id", "name", "orders"})
		userParts = append(userParts, fmt.Sprintf("{%s}", strings.Join(parts, ",")))
	}
	return fmt.Sprintf(`{"data":{"users":[%s]}}`, strings.Join(userParts, ","))
}

// queryAllUsersWithDataloader uses batching to avoid N+1.
func (r *Resolver) queryAllUsersWithDataloader(ctx context.Context) string {
	users := r.store.AllUsers()

	// Collect all user IDs first.
	userIDs := make([]string, 0, len(users))
	for _, u := range users {
		userIDs = append(userIDs, u.ID)
	}
	// One batched query for all orders.
	ordersByUser := r.store.BatchOrdersByUserIDs(userIDs)

	var userParts []string
	for _, u := range users {
		var parts []string
		parts = append(parts, fmt.Sprintf(`"id":"%s"`, u.ID))
		parts = append(parts, fmt.Sprintf(`"name":"%s"`, u.Name))

		var orderParts []string
		for _, o := range ordersByUser[u.ID] {
			orderParts = append(orderParts, fmt.Sprintf(`{"id":"%s","status":"%s"}`, o.ID, o.Status))
		}
		parts = append(parts, fmt.Sprintf(`"orders":[%s]`, strings.Join(orderParts, ",")))
		userParts = append(userParts, fmt.Sprintf("{%s}", strings.Join(parts, ",")))
	}
	return fmt.Sprintf(`{"data":{"users":[%s]}}`, strings.Join(userParts, ","))
}

// mutationCreateOrder resolves: mutation { createOrder(input: {...}) { id total } }
func (r *Resolver) mutationCreateOrder(ctx context.Context, input CreateOrderInput) string {
	if input.UserID == "" {
		return `{"errors":[{"message":"userID is required"}]}`
	}
	order := r.store.CreateOrder(input)
	return fmt.Sprintf(`{"data":{"createOrder":{"id":"%s","total":%d,"status":"%s"}}}`,
		order.ID, order.TotalCents, order.Status)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== GraphQL Basics (in-process simulation) ===")
	fmt.Println()

	store := NewStore()
	r := &Resolver{store: store}
	ctx := context.Background()

	// ── QUERY: single user ────────────────────────────────────────────────────
	fmt.Println("--- Query: user(id: \"u-1\") { id name email } ---")
	store.dbQueries.Store(0)
	result := r.queryUser(ctx, "u-1", []string{"id", "name", "email"})
	fmt.Printf("  result: %s\n", result)
	fmt.Printf("  db queries: %d\n", store.dbQueries.Load())

	// ── QUERY: user with nested orders ────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Query: user(id: \"u-1\") { id name orders { id status total } } ---")
	store.dbQueries.Store(0)
	result = r.queryUser(ctx, "u-1", []string{"id", "name", "orders"})
	fmt.Printf("  result: %s\n", result)
	fmt.Printf("  db queries: %d (1 for user + 1 for orders)\n", store.dbQueries.Load())

	// ── N+1 PROBLEM ──────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- N+1 problem: users { id orders { id } } ---")
	store.dbQueries.Store(0)
	result = r.queryAllUsersNaive(ctx)
	naive := store.dbQueries.Load()
	fmt.Printf("  naive: %d db queries for %d users\n", naive, 3)
	fmt.Printf("  (1 query to list users + 1 query per user for orders = N+1)\n")

	// ── DATALOADER FIX ────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Dataloader fix: batched query ---")
	store.dbQueries.Store(0)
	result = r.queryAllUsersWithDataloader(ctx)
	batched := store.dbQueries.Load()
	fmt.Printf("  batched: %d db queries for %d users\n", batched, 3)
	fmt.Printf("  (1 query to list users + 1 batched query for all orders)\n")
	_ = result

	// ── MUTATION ──────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Mutation: createOrder ---")
	store.dbQueries.Store(0)
	result = r.mutationCreateOrder(ctx, CreateOrderInput{
		UserID: "u-2",
		Items: []OrderItemInput{
			{ProductID: "p-1", Quantity: 2},
			{ProductID: "p-2", Quantity: 1},
		},
	})
	fmt.Printf("  result: %s\n", result)

	// ── ERROR CASE ────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Error: user not found ---")
	result = r.queryUser(ctx, "u-999", []string{"id", "name"})
	fmt.Printf("  result: %s\n", result)

	// ── GRAPHQL vs REST comparison ─────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- GraphQL vs REST ---")
	fmt.Println(`  REST: multiple round trips
    GET /users/u-1         → user fields
    GET /users/u-1/orders  → orders (second request)

  GraphQL: one request, client specifies fields
    POST /graphql
    { user(id: "u-1") { id name orders { id status total } } }
    → user + orders in a single response`)

	fmt.Println()
	fmt.Println("--- Schema definition (SDL) ---")
	schema := `
  type Query {
    user(id: ID!): User
    users: [User!]!
    order(id: ID!): Order
  }

  type Mutation {
    createOrder(input: CreateOrderInput!): Order!
  }

  type User {
    id:     ID!
    name:   String!
    email:  String!
    orders: [Order!]!
  }

  type Order {
    id:         ID!
    totalCents: Int!
    status:     String!
    items:      [OrderItem!]!
  }

  type OrderItem {
    productId: ID!
    quantity:  Int!
    price:     Int!
  }

  input CreateOrderInput {
    userId: ID!
    items:  [OrderItemInput!]!
  }

  input OrderItemInput {
    productId: ID!
    quantity:  Int!
  }`
	for _, line := range strings.Split(schema, "\n") {
		if strings.TrimSpace(line) != "" {
			fmt.Printf("  %s\n", line)
		}
	}
}
