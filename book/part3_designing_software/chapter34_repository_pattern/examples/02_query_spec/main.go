// FILE: book/part3_designing_software/chapter34_repository_pattern/examples/02_query_spec/main.go
// CHAPTER: 34 — Repository Pattern
// TOPIC: Specification pattern for composable queries; pagination; read models.
//
// Run (from the chapter folder):
//   go run ./examples/02_query_spec

package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN
// ─────────────────────────────────────────────────────────────────────────────

type ProductID int

type Product struct {
	ID         ProductID
	Name       string
	Category   string
	Price      float64
	Stock      int
	Active     bool
	LaunchedAt time.Time
}

// ─────────────────────────────────────────────────────────────────────────────
// SPECIFICATION PATTERN
//
// A Spec encapsulates a predicate. Specs compose with And/Or/Not.
// The repository's Query method accepts a Spec — no SQL leaks into the domain.
// ─────────────────────────────────────────────────────────────────────────────

type Spec interface {
	IsSatisfiedBy(p Product) bool
	Description() string
}

// ── Leaf specs ────────────────────────────────────────────────────────────────

type activeSpec struct{}

func Active() Spec { return activeSpec{} }
func (activeSpec) IsSatisfiedBy(p Product) bool { return p.Active }
func (activeSpec) Description() string           { return "active" }

type categorySpec struct{ cat string }

func InCategory(cat string) Spec { return categorySpec{cat} }
func (s categorySpec) IsSatisfiedBy(p Product) bool {
	return strings.EqualFold(p.Category, s.cat)
}
func (s categorySpec) Description() string { return "category=" + s.cat }

type priceRangeSpec struct{ min, max float64 }

func PriceBetween(min, max float64) Spec { return priceRangeSpec{min, max} }
func (s priceRangeSpec) IsSatisfiedBy(p Product) bool {
	return p.Price >= s.min && p.Price <= s.max
}
func (s priceRangeSpec) Description() string {
	return fmt.Sprintf("price[%.2f-%.2f]", s.min, s.max)
}

type inStockSpec struct{}

func InStock() Spec { return inStockSpec{} }
func (inStockSpec) IsSatisfiedBy(p Product) bool { return p.Stock > 0 }
func (inStockSpec) Description() string           { return "in-stock" }

// ── Composite specs ────────────────────────────────────────────────────────────

type andSpec struct{ a, b Spec }

func And(a, b Spec) Spec { return andSpec{a, b} }
func (s andSpec) IsSatisfiedBy(p Product) bool {
	return s.a.IsSatisfiedBy(p) && s.b.IsSatisfiedBy(p)
}
func (s andSpec) Description() string {
	return "(" + s.a.Description() + " AND " + s.b.Description() + ")"
}

type orSpec struct{ a, b Spec }

func Or(a, b Spec) Spec { return orSpec{a, b} }
func (s orSpec) IsSatisfiedBy(p Product) bool {
	return s.a.IsSatisfiedBy(p) || s.b.IsSatisfiedBy(p)
}
func (s orSpec) Description() string {
	return "(" + s.a.Description() + " OR " + s.b.Description() + ")"
}

type notSpec struct{ inner Spec }

func Not(inner Spec) Spec { return notSpec{inner} }
func (s notSpec) IsSatisfiedBy(p Product) bool { return !s.inner.IsSatisfiedBy(p) }
func (s notSpec) Description() string          { return "NOT " + s.inner.Description() }

// ─────────────────────────────────────────────────────────────────────────────
// PAGINATION
// ─────────────────────────────────────────────────────────────────────────────

type Page struct {
	Number int // 1-based
	Size   int
}

type PagedResult struct {
	Items      []Product
	TotalCount int
	Page       Page
}

func (r PagedResult) TotalPages() int {
	if r.Page.Size == 0 {
		return 0
	}
	return (r.TotalCount + r.Page.Size - 1) / r.Page.Size
}

// ─────────────────────────────────────────────────────────────────────────────
// REPOSITORY
// ─────────────────────────────────────────────────────────────────────────────

type SortField string

const (
	SortByName  SortField = "name"
	SortByPrice SortField = "price"
	SortByStock SortField = "stock"
)

type ProductRepository interface {
	Save(p Product) (Product, error)
	FindByID(id ProductID) (Product, error)
	Query(spec Spec, sortBy SortField, page Page) (PagedResult, error)
	Count(spec Spec) (int, error)
}

// ─────────────────────────────────────────────────────────────────────────────
// IN-MEMORY IMPLEMENTATION
// ─────────────────────────────────────────────────────────────────────────────

type memProductRepo struct {
	products map[ProductID]Product
	nextID   ProductID
}

func NewMemProductRepo() ProductRepository {
	return &memProductRepo{products: make(map[ProductID]Product), nextID: 1}
}

func (r *memProductRepo) Save(p Product) (Product, error) {
	if p.ID == 0 {
		p.ID = r.nextID
		r.nextID++
	}
	r.products[p.ID] = p
	return p, nil
}

func (r *memProductRepo) FindByID(id ProductID) (Product, error) {
	p, ok := r.products[id]
	if !ok {
		return Product{}, fmt.Errorf("product %d not found", id)
	}
	return p, nil
}

func (r *memProductRepo) Query(spec Spec, sortBy SortField, page Page) (PagedResult, error) {
	var matched []Product
	for _, p := range r.products {
		if spec.IsSatisfiedBy(p) {
			matched = append(matched, p)
		}
	}

	// Sort.
	sort.Slice(matched, func(i, j int) bool {
		switch sortBy {
		case SortByPrice:
			return matched[i].Price < matched[j].Price
		case SortByStock:
			return matched[i].Stock > matched[j].Stock
		default:
			return matched[i].Name < matched[j].Name
		}
	})

	total := len(matched)

	// Paginate.
	if page.Size > 0 {
		start := (page.Number - 1) * page.Size
		if start >= total {
			matched = nil
		} else {
			end := start + page.Size
			if end > total {
				end = total
			}
			matched = matched[start:end]
		}
	}

	return PagedResult{Items: matched, TotalCount: total, Page: page}, nil
}

func (r *memProductRepo) Count(spec Spec) (int, error) {
	count := 0
	for _, p := range r.products {
		if spec.IsSatisfiedBy(p) {
			count++
		}
	}
	return count, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SEED DATA
// ─────────────────────────────────────────────────────────────────────────────

func seedRepo(repo ProductRepository) {
	products := []Product{
		{Name: "Widget Alpha", Category: "widgets", Price: 9.99, Stock: 100, Active: true},
		{Name: "Widget Beta", Category: "widgets", Price: 14.99, Stock: 50, Active: true},
		{Name: "Widget Gamma", Category: "widgets", Price: 24.99, Stock: 0, Active: true},
		{Name: "Gadget One", Category: "gadgets", Price: 49.99, Stock: 25, Active: true},
		{Name: "Gadget Two", Category: "gadgets", Price: 99.99, Stock: 10, Active: false},
		{Name: "Gadget Three", Category: "gadgets", Price: 79.99, Stock: 30, Active: true},
		{Name: "Tool A", Category: "tools", Price: 19.99, Stock: 200, Active: true},
		{Name: "Tool B", Category: "tools", Price: 34.99, Stock: 5, Active: true},
		{Name: "Tool C", Category: "tools", Price: 59.99, Stock: 0, Active: false},
	}
	for _, p := range products {
		_, _ = repo.Save(p)
	}
}

func printResult(label string, result PagedResult) {
	fmt.Printf("  %s  [page %d/%d, total=%d]\n",
		label, result.Page.Number, result.TotalPages(), result.TotalCount)
	for _, p := range result.Items {
		status := "✓"
		if !p.Active {
			status = "✗"
		}
		fmt.Printf("    %s %-22s  $%6.2f  stock=%d\n", status, p.Name, p.Price, p.Stock)
	}
}

func main() {
	repo := NewMemProductRepo()
	seedRepo(repo)

	fmt.Println("=== Specification queries ===")

	// active widgets in stock
	spec1 := And(And(Active(), InCategory("widgets")), InStock())
	result1, _ := repo.Query(spec1, SortByPrice, Page{Number: 1, Size: 10})
	printResult(spec1.Description(), result1)

	fmt.Println()

	// active gadgets OR tools under $60
	spec2 := And(Active(), Or(InCategory("gadgets"), And(InCategory("tools"), PriceBetween(0, 60))))
	result2, _ := repo.Query(spec2, SortByName, Page{Number: 1, Size: 10})
	printResult(spec2.Description(), result2)

	fmt.Println()

	// all inactive products
	spec3 := Not(Active())
	result3, _ := repo.Query(spec3, SortByName, Page{Number: 1, Size: 10})
	printResult(spec3.Description(), result3)

	fmt.Println()
	fmt.Println("=== Pagination ===")
	all := And(Active(), InStock())
	for pageNum := 1; pageNum <= 3; pageNum++ {
		result, _ := repo.Query(all, SortByPrice, Page{Number: pageNum, Size: 3})
		printResult(fmt.Sprintf("page %d", pageNum), result)
		if result.Page.Number >= result.TotalPages() {
			break
		}
	}

	fmt.Println()
	fmt.Println("=== Count specs ===")
	for _, s := range []Spec{Active(), InStock(), And(Active(), InStock()), Not(Active())} {
		n, _ := repo.Count(s)
		fmt.Printf("  %-40s count=%d\n", s.Description(), n)
	}
}
