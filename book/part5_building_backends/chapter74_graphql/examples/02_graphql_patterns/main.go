// FILE: book/part5_building_backends/chapter74_graphql/examples/02_graphql_patterns/main.go
// CHAPTER: 74 — GraphQL
// TOPIC: GraphQL patterns — auth in resolvers, subscriptions (WebSocket push),
//        pagination (cursor + offset), field-level errors, and when to use
//        GraphQL vs REST vs gRPC.
//
// Run (from the chapter folder):
//   go run ./examples/02_graphql_patterns

package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// AUTH CONTEXT
// ─────────────────────────────────────────────────────────────────────────────

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleUser    Role = "user"
	RoleGuest   Role = "guest"
)

type AuthUser struct {
	ID   string
	Role Role
}

type authKey struct{}

func WithAuth(ctx context.Context, u AuthUser) context.Context {
	return context.WithValue(ctx, authKey{}, u)
}

func AuthFromCtx(ctx context.Context) (AuthUser, bool) {
	u, ok := ctx.Value(authKey{}).(AuthUser)
	return u, ok
}

// ─────────────────────────────────────────────────────────────────────────────
// PARTIAL ERROR RESPONSE
// GraphQL can return partial data + errors in the same response.
// ─────────────────────────────────────────────────────────────────────────────

type GQLError struct {
	Message string
	Path    []string
}

type GQLResponse struct {
	Data   map[string]any
	Errors []GQLError
}

func (r *GQLResponse) String() string {
	var parts []string
	if len(r.Data) > 0 {
		var dp []string
		for k, v := range r.Data {
			dp = append(dp, fmt.Sprintf("%q:%v", k, v))
		}
		parts = append(parts, fmt.Sprintf(`"data":{%s}`, strings.Join(dp, ",")))
	}
	if len(r.Errors) > 0 {
		var ep []string
		for _, e := range r.Errors {
			ep = append(ep, fmt.Sprintf(`{"message":%q,"path":[%s]}`,
				e.Message, strings.Join(quoted(e.Path), ",")))
		}
		parts = append(parts, fmt.Sprintf(`"errors":[%s]`, strings.Join(ep, ",")))
	}
	return fmt.Sprintf("{%s}", strings.Join(parts, ","))
}

func quoted(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = fmt.Sprintf("%q", s)
	}
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// CURSOR PAGINATION
// ─────────────────────────────────────────────────────────────────────────────

type PageInfo struct {
	HasNextPage bool
	EndCursor   string
}

type Connection[T any] struct {
	Edges    []Edge[T]
	PageInfo PageInfo
	Total    int
}

type Edge[T any] struct {
	Node   T
	Cursor string
}

type Item struct {
	ID    string
	Name  string
	Price int
}

func paginateItems(items []Item, after string, first int) Connection[Item] {
	startIdx := 0
	if after != "" {
		for i, it := range items {
			if it.ID == after {
				startIdx = i + 1
				break
			}
		}
	}
	end := startIdx + first
	if end > len(items) {
		end = len(items)
	}
	slice := items[startIdx:end]

	edges := make([]Edge[Item], len(slice))
	for i, it := range slice {
		edges[i] = Edge[Item]{Node: it, Cursor: it.ID}
	}

	conn := Connection[Item]{
		Edges: edges,
		Total: len(items),
		PageInfo: PageInfo{
			HasNextPage: end < len(items),
		},
	}
	if len(edges) > 0 {
		conn.PageInfo.EndCursor = edges[len(edges)-1].Cursor
	}
	return conn
}

// ─────────────────────────────────────────────────────────────────────────────
// SUBSCRIPTION (simulated push)
// ─────────────────────────────────────────────────────────────────────────────

type OrderEvent struct {
	OrderID string
	Status  string
	At      time.Time
}

type Subscription struct {
	mu        sync.Mutex
	listeners map[string][]chan OrderEvent
}

func NewSubscription() *Subscription {
	return &Subscription{listeners: make(map[string][]chan OrderEvent)}
}

func (s *Subscription) Subscribe(orderID string) <-chan OrderEvent {
	ch := make(chan OrderEvent, 8)
	s.mu.Lock()
	s.listeners[orderID] = append(s.listeners[orderID], ch)
	s.mu.Unlock()
	return ch
}

func (s *Subscription) Publish(event OrderEvent) {
	s.mu.Lock()
	chs := s.listeners[event.OrderID]
	s.mu.Unlock()
	for _, ch := range chs {
		select {
		case ch <- event:
		default:
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// AUTH RESOLVER
// ─────────────────────────────────────────────────────────────────────────────

type AdminResolver struct{}

func (a *AdminResolver) AllOrders(ctx context.Context) *GQLResponse {
	auth, ok := AuthFromCtx(ctx)
	if !ok || auth.Role != RoleAdmin {
		return &GQLResponse{
			Errors: []GQLError{{Message: "permission denied: admin role required", Path: []string{"allOrders"}}},
		}
	}
	return &GQLResponse{
		Data: map[string]any{`"allOrders"`: `[{"id":"o-1"},{"id":"o-2"},{"id":"o-3"}]`},
	}
}

// UserOrders returns orders for the authenticated user only.
func (a *AdminResolver) UserOrders(ctx context.Context, requestedUserID string) *GQLResponse {
	auth, ok := AuthFromCtx(ctx)
	if !ok {
		return &GQLResponse{Errors: []GQLError{{Message: "unauthenticated"}}}
	}
	// Users can only see their own orders; admins can see any.
	if auth.Role != RoleAdmin && auth.ID != requestedUserID {
		return &GQLResponse{
			Errors: []GQLError{{
				Message: fmt.Sprintf("forbidden: cannot view orders for user %s", requestedUserID),
				Path:    []string{"userOrders"},
			}},
		}
	}
	return &GQLResponse{
		Data: map[string]any{`"orders"`: fmt.Sprintf(`[{"userID":"%s","count":2}]`, requestedUserID)},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// PARTIAL SUCCESS (field-level errors)
// ─────────────────────────────────────────────────────────────────────────────

func resolveWithPartialError(userIDs []string) *GQLResponse {
	resp := &GQLResponse{Data: make(map[string]any)}
	var users []string
	for _, id := range userIDs {
		if id == "u-bad" {
			// Partial failure: return error for this field but continue.
			resp.Errors = append(resp.Errors, GQLError{
				Message: fmt.Sprintf("user %q not found", id),
				Path:    []string{"users", id},
			})
			continue
		}
		users = append(users, fmt.Sprintf(`{"id":%q}`, id))
	}
	resp.Data[`"users"`] = fmt.Sprintf("[%s]", strings.Join(users, ","))
	return resp
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== GraphQL Patterns ===")
	fmt.Println()

	// ── AUTH IN RESOLVERS ─────────────────────────────────────────────────────
	fmt.Println("--- Auth in resolvers ---")
	r := &AdminResolver{}

	// Unauthenticated.
	resp := r.AllOrders(context.Background())
	fmt.Printf("  no auth: %s\n", resp.String())

	// User role (not admin).
	userCtx := WithAuth(context.Background(), AuthUser{ID: "u-1", Role: RoleUser})
	resp = r.AllOrders(userCtx)
	fmt.Printf("  user role: %s\n", resp.String())

	// Admin role.
	adminCtx := WithAuth(context.Background(), AuthUser{ID: "u-admin", Role: RoleAdmin})
	resp = r.AllOrders(adminCtx)
	fmt.Printf("  admin role: %s\n", resp.String())

	// User viewing own orders (allowed).
	resp = r.UserOrders(userCtx, "u-1")
	fmt.Printf("  user viewing own orders: %s\n", resp.String())

	// User viewing other's orders (forbidden).
	resp = r.UserOrders(userCtx, "u-2")
	fmt.Printf("  user viewing other orders: %s\n", resp.String())

	// ── CURSOR PAGINATION ─────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Cursor pagination ---")
	items := []Item{
		{ID: "i-1", Name: "Alpha", Price: 100},
		{ID: "i-2", Name: "Beta", Price: 200},
		{ID: "i-3", Name: "Gamma", Price: 300},
		{ID: "i-4", Name: "Delta", Price: 400},
		{ID: "i-5", Name: "Epsilon", Price: 500},
	}

	// Page 1: first 2.
	page1 := paginateItems(items, "", 2)
	fmt.Printf("  page 1: hasNext=%v endCursor=%s total=%d\n",
		page1.PageInfo.HasNextPage, page1.PageInfo.EndCursor, page1.Total)
	for _, e := range page1.Edges {
		fmt.Printf("    %s %s\n", e.Cursor, e.Node.Name)
	}

	// Page 2: next 2 after cursor.
	page2 := paginateItems(items, page1.PageInfo.EndCursor, 2)
	fmt.Printf("  page 2: hasNext=%v endCursor=%s\n",
		page2.PageInfo.HasNextPage, page2.PageInfo.EndCursor)
	for _, e := range page2.Edges {
		fmt.Printf("    %s %s\n", e.Cursor, e.Node.Name)
	}

	// Page 3: last item.
	page3 := paginateItems(items, page2.PageInfo.EndCursor, 2)
	fmt.Printf("  page 3: hasNext=%v items=%d\n",
		page3.PageInfo.HasNextPage, len(page3.Edges))
	for _, e := range page3.Edges {
		fmt.Printf("    %s %s\n", e.Cursor, e.Node.Name)
	}

	// ── PARTIAL ERRORS ────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Partial success (field-level errors) ---")
	partial := resolveWithPartialError([]string{"u-1", "u-bad", "u-2"})
	fmt.Printf("  response: %s\n", partial.String())
	fmt.Println("  (GraphQL returns partial data + errors in the same response)")

	// ── SUBSCRIPTIONS ─────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Subscription: order status updates ---")
	sub := NewSubscription()
	ch := sub.Subscribe("ord-1")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		count := 0
		for event := range ch {
			fmt.Printf("  [sub] order=%s status=%s\n", event.OrderID, event.Status)
			count++
			if count == 3 {
				return
			}
		}
	}()

	// Simulate server pushing events.
	for _, status := range []string{"confirmed", "shipped", "delivered"} {
		sub.Publish(OrderEvent{OrderID: "ord-1", Status: status, At: time.Now()})
		time.Sleep(10 * time.Millisecond)
	}
	wg.Wait()

	// ── WHEN TO USE WHAT ──────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- When to use GraphQL vs REST vs gRPC ---")
	fmt.Println(`
  GraphQL:  client-driven queries, mobile apps with varied field needs,
            BFF (backend for frontend) aggregating multiple services,
            rapid prototyping where schema evolves frequently.

  REST:     public APIs, simple CRUD, HTTP caching via CDN,
            teams unfamiliar with GraphQL tooling,
            file upload/download, webhooks.

  gRPC:     service-to-service communication, high-throughput internal APIs,
            streaming (live prices, telemetry), strong typing via .proto.
            Not browser-friendly without grpc-gateway.`)
}
