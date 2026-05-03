// FILE: book/part5_building_backends/chapter74_graphql/exercises/01_product_api/main.go
// CHAPTER: 74 — GraphQL
// TOPIC: Product catalogue API — query, mutation, subscription, dataloader,
//        auth, and cursor pagination.
//
// Run (from the chapter folder):
//   go run ./exercises/01_product_api

package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN TYPES
// ─────────────────────────────────────────────────────────────────────────────

type Category string

const (
	CategoryBooks       Category = "books"
	CategoryElectronics Category = "electronics"
)

type Product struct {
	ID       string
	Name     string
	Category Category
	Price    int
	InStock  bool
	Reviews  []*Review
}

type Review struct {
	ID        string
	ProductID string
	AuthorID  string
	Rating    int // 1-5
	Comment   string
}

type ReviewInput struct {
	ProductID string
	AuthorID  string
	Rating    int
	Comment   string
}

// ─────────────────────────────────────────────────────────────────────────────
// STORE
// ─────────────────────────────────────────────────────────────────────────────

type Store struct {
	mu       sync.RWMutex
	products map[string]*Product
	reviews  map[string][]*Review // productID → reviews
	nextID   int
	Queries  atomic.Int64
}

func NewStore() *Store {
	s := &Store{products: make(map[string]*Product), reviews: make(map[string][]*Review), nextID: 1}
	s.seed()
	return s
}

func (s *Store) seed() {
	prods := []*Product{
		{Name: "The Go Programming Language", Category: CategoryBooks, Price: 3999, InStock: true},
		{Name: "Clean Architecture", Category: CategoryBooks, Price: 2999, InStock: true},
		{Name: "Mechanical Keyboard", Category: CategoryElectronics, Price: 8999, InStock: false},
		{Name: "USB-C Dock", Category: CategoryElectronics, Price: 4999, InStock: true},
	}
	for _, p := range prods {
		p.ID = fmt.Sprintf("p-%d", s.nextID)
		s.products[p.ID] = p
		s.nextID++
	}
	// Seed some reviews.
	s.reviews["p-1"] = []*Review{
		{ID: "r-1", ProductID: "p-1", AuthorID: "u-1", Rating: 5, Comment: "excellent"},
		{ID: "r-2", ProductID: "p-1", AuthorID: "u-2", Rating: 4, Comment: "good"},
	}
	s.reviews["p-2"] = []*Review{
		{ID: "r-3", ProductID: "p-2", AuthorID: "u-1", Rating: 5, Comment: "must read"},
	}
}

func (s *Store) GetProduct(id string) (*Product, bool) {
	s.Queries.Add(1)
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.products[id]
	return p, ok
}

func (s *Store) AllProducts() []*Product {
	s.Queries.Add(1)
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Product, 0, len(s.products))
	for _, p := range s.products {
		out = append(out, p)
	}
	return out
}

func (s *Store) ProductsByCategory(cat Category) []*Product {
	s.Queries.Add(1)
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Product
	for _, p := range s.products {
		if p.Category == cat {
			out = append(out, p)
		}
	}
	return out
}

func (s *Store) ReviewsByProductID(productID string) []*Review {
	s.Queries.Add(1)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reviews[productID]
}

// BatchReviewsByProductIDs loads reviews for multiple products in one query.
func (s *Store) BatchReviewsByProductIDs(ids []string) map[string][]*Review {
	s.Queries.Add(1)
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string][]*Review, len(ids))
	for _, id := range ids {
		out[id] = s.reviews[id]
	}
	return out
}

func (s *Store) AddReview(input ReviewInput) *Review {
	s.Queries.Add(1)
	s.mu.Lock()
	defer s.mu.Unlock()
	r := &Review{
		ID:        fmt.Sprintf("r-%d", s.nextID),
		ProductID: input.ProductID,
		AuthorID:  input.AuthorID,
		Rating:    input.Rating,
		Comment:   input.Comment,
	}
	s.nextID++
	s.reviews[input.ProductID] = append(s.reviews[input.ProductID], r)
	return r
}

func (s *Store) CreateProduct(name string, cat Category, price int) *Product {
	s.Queries.Add(1)
	s.mu.Lock()
	defer s.mu.Unlock()
	p := &Product{ID: fmt.Sprintf("p-%d", s.nextID), Name: name, Category: cat, Price: price, InStock: true}
	s.nextID++
	s.products[p.ID] = p
	return p
}

// ─────────────────────────────────────────────────────────────────────────────
// AUTH
// ─────────────────────────────────────────────────────────────────────────────

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

type authCtxKey struct{}

type AuthUser struct {
	ID   string
	Role Role
}

func WithAuth(ctx context.Context, u AuthUser) context.Context {
	return context.WithValue(ctx, authCtxKey{}, u)
}

func AuthFromCtx(ctx context.Context) (AuthUser, bool) {
	u, ok := ctx.Value(authCtxKey{}).(AuthUser)
	return u, ok
}

// ─────────────────────────────────────────────────────────────────────────────
// SUBSCRIPTION (review added events)
// ─────────────────────────────────────────────────────────────────────────────

type ReviewEvent struct {
	Review    *Review
	ProductID string
}

type ReviewSubscription struct {
	mu        sync.Mutex
	listeners []chan ReviewEvent
}

func (rs *ReviewSubscription) Subscribe() <-chan ReviewEvent {
	ch := make(chan ReviewEvent, 8)
	rs.mu.Lock()
	rs.listeners = append(rs.listeners, ch)
	rs.mu.Unlock()
	return ch
}

func (rs *ReviewSubscription) Publish(e ReviewEvent) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	for _, ch := range rs.listeners {
		select {
		case ch <- e:
		default:
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// RESOLVER
// ─────────────────────────────────────────────────────────────────────────────

type Resolver struct {
	store   *Store
	reviewSub *ReviewSubscription
}

func NewResolver(store *Store) *Resolver {
	return &Resolver{store: store, reviewSub: &ReviewSubscription{}}
}

func (res *Resolver) Product(ctx context.Context, id string) string {
	p, ok := res.store.GetProduct(id)
	if !ok {
		return fmt.Sprintf(`{"errors":[{"message":"product %q not found"}]}`, id)
	}
	reviews := res.store.ReviewsByProductID(p.ID)
	return fmt.Sprintf(`{"data":{"product":{"id":%q,"name":%q,"price":%d,"reviews":%d}}}`,
		p.ID, p.Name, p.Price, len(reviews))
}

func (res *Resolver) ProductsNaive() string {
	products := res.store.AllProducts()
	var parts []string
	for _, p := range products {
		revs := res.store.ReviewsByProductID(p.ID) // N+1!
		parts = append(parts, fmt.Sprintf(`{"id":%q,"reviews":%d}`, p.ID, len(revs)))
	}
	return fmt.Sprintf(`{"data":{"products":[%s]}}`, strings.Join(parts, ","))
}

func (res *Resolver) ProductsWithDataloader() string {
	products := res.store.AllProducts()
	ids := make([]string, len(products))
	for i, p := range products {
		ids[i] = p.ID
	}
	reviewsByProduct := res.store.BatchReviewsByProductIDs(ids)
	var parts []string
	for _, p := range products {
		revs := reviewsByProduct[p.ID]
		parts = append(parts, fmt.Sprintf(`{"id":%q,"reviews":%d}`, p.ID, len(revs)))
	}
	return fmt.Sprintf(`{"data":{"products":[%s]}}`, strings.Join(parts, ","))
}

func (res *Resolver) CreateProduct(ctx context.Context, name string, cat Category, price int) string {
	auth, ok := AuthFromCtx(ctx)
	if !ok || auth.Role != RoleAdmin {
		return `{"errors":[{"message":"admin role required to create products"}]}`
	}
	p := res.store.CreateProduct(name, cat, price)
	return fmt.Sprintf(`{"data":{"createProduct":{"id":%q,"name":%q,"price":%d}}}`, p.ID, p.Name, p.Price)
}

func (res *Resolver) AddReview(ctx context.Context, input ReviewInput) string {
	_, ok := AuthFromCtx(ctx)
	if !ok {
		return `{"errors":[{"message":"authentication required to leave a review"}]}`
	}
	if input.Rating < 1 || input.Rating > 5 {
		return `{"errors":[{"message":"rating must be between 1 and 5"}]}`
	}
	r := res.store.AddReview(input)
	res.reviewSub.Publish(ReviewEvent{Review: r, ProductID: input.ProductID})
	return fmt.Sprintf(`{"data":{"addReview":{"id":%q,"rating":%d,"comment":%q}}}`,
		r.ID, r.Rating, r.Comment)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Product API (GraphQL exercise) ===")
	fmt.Println()

	store := NewStore()
	res := NewResolver(store)

	ctx := context.Background()
	adminCtx := WithAuth(ctx, AuthUser{ID: "u-admin", Role: RoleAdmin})
	userCtx := WithAuth(ctx, AuthUser{ID: "u-1", Role: RoleUser})

	// ── QUERY ─────────────────────────────────────────────────────────────────
	fmt.Println("--- Query: product ---")
	fmt.Println("  " + res.Product(ctx, "p-1"))
	fmt.Println("  " + res.Product(ctx, "nonexistent"))

	// ── N+1 vs DATALOADER ─────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- N+1 comparison ---")
	store.Queries.Store(0)
	res.ProductsNaive()
	naiveQueries := store.Queries.Load()

	store.Queries.Store(0)
	res.ProductsWithDataloader()
	batchedQueries := store.Queries.Load()

	fmt.Printf("  naive:   %d queries for %d products\n", naiveQueries, len(store.AllProducts())-1) // subtract last allProducts call
	fmt.Printf("  batched: %d queries for %d products\n", batchedQueries, len(store.AllProducts())-1)

	// ── MUTATION: auth required ────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Mutation: createProduct (admin only) ---")
	fmt.Println("  " + res.CreateProduct(userCtx, "Test", CategoryBooks, 999))
	fmt.Println("  " + res.CreateProduct(adminCtx, "New Book", CategoryBooks, 1999))

	// ── MUTATION: addReview ────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Mutation: addReview ---")
	fmt.Println("  " + res.AddReview(ctx, ReviewInput{ProductID: "p-1", AuthorID: "u-3", Rating: 3, Comment: "decent"}))
	fmt.Println("  " + res.AddReview(userCtx, ReviewInput{ProductID: "p-1", AuthorID: "u-1", Rating: 5, Comment: "great"}))
	fmt.Println("  " + res.AddReview(userCtx, ReviewInput{ProductID: "p-1", AuthorID: "u-1", Rating: 6, Comment: "invalid"}))

	// ── SUBSCRIPTION ──────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Subscription: reviewAdded ---")
	ch := res.reviewSub.Subscribe()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		received := 0
		for event := range ch {
			fmt.Printf("  [sub] new review on %s: rating=%d comment=%q\n",
				event.ProductID, event.Review.Rating, event.Review.Comment)
			received++
			if received == 2 {
				return
			}
		}
	}()

	res.AddReview(userCtx, ReviewInput{ProductID: "p-2", AuthorID: "u-2", Rating: 4, Comment: "very good"})
	res.AddReview(userCtx, ReviewInput{ProductID: "p-3", AuthorID: "u-1", Rating: 3, Comment: "okay"})
	time.Sleep(20 * time.Millisecond)
	wg.Wait()

	// ── QUERY COMPLEXITY ──────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Query complexity (conceptual) ---")
	fmt.Println(`  Nested queries can be expensive:
    products {                 # complexity: 1
      reviews {                # complexity: N
        author {               # complexity: N*M
          purchaseHistory {    # complexity: N*M*K — must be blocked!
            ...
          }
        }
      }
    }`)
	fmt.Println("  Use query depth limiting and cost analysis in production.")
	fmt.Println("  gqlgen: github.com/99designs/gqlgen/graphql/handler/extension")
}
