// FILE: book/part6_production_engineering/chapter82_testing_fundamentals/examples/02_subtests/main.go
// CHAPTER: 82 — Testing Fundamentals
// TOPIC: Subtests, test fixtures, setup/teardown, golden files, and fuzz
//        testing patterns.
//
// Run:
//   go run ./examples/02_subtests

package main

import (
	"fmt"
	"sort"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN: order processor
// ─────────────────────────────────────────────────────────────────────────────

type Item struct {
	Name     string
	Price    int // cents
	Quantity int
}

type Order struct {
	ID       string
	Items    []Item
	Discount int // percentage 0-100
}

func (o *Order) Total() int {
	var subtotal int
	for _, item := range o.Items {
		subtotal += item.Price * item.Quantity
	}
	if o.Discount > 0 {
		subtotal = subtotal * (100 - o.Discount) / 100
	}
	return subtotal
}

func (o *Order) Summary() string {
	lines := make([]string, 0, len(o.Items)+2)
	lines = append(lines, fmt.Sprintf("Order %s:", o.ID))
	for _, item := range o.Items {
		lines = append(lines, fmt.Sprintf("  %s x%d @ %d¢", item.Name, item.Quantity, item.Price))
	}
	lines = append(lines, fmt.Sprintf("Total: %d¢", o.Total()))
	return strings.Join(lines, "\n")
}

// ─────────────────────────────────────────────────────────────────────────────
// MINI TEST FRAMEWORK
// ─────────────────────────────────────────────────────────────────────────────

type T struct {
	name   string
	failed bool
	logs   []string
}

func (t *T) Errorf(format string, args ...any) {
	t.failed = true
	t.logs = append(t.logs, "    FAIL: "+fmt.Sprintf(format, args...))
}

type Suite struct{ passed, failed int }

func (s *Suite) Run(name string, fn func(*T)) {
	t := &T{name: name}
	fn(t)
	if t.failed {
		s.failed++
		fmt.Printf("  --- FAIL: %s\n", name)
		for _, l := range t.logs {
			fmt.Println(l)
		}
	} else {
		s.passed++
		fmt.Printf("  --- PASS: %s\n", name)
	}
}

func (s *Suite) Report() {
	fmt.Printf("  %d/%d passed\n", s.passed, s.passed+s.failed)
}

// ─────────────────────────────────────────────────────────────────────────────
// FIXTURE: shared test setup
// ─────────────────────────────────────────────────────────────────────────────

type OrderFixture struct {
	simple   *Order
	discount *Order
	empty    *Order
}

func newOrderFixture() *OrderFixture {
	return &OrderFixture{
		simple: &Order{
			ID:    "ord-1",
			Items: []Item{{"Widget", 1000, 2}, {"Gadget", 500, 1}},
		},
		discount: &Order{
			ID:       "ord-2",
			Items:    []Item{{"Widget", 1000, 3}},
			Discount: 10,
		},
		empty: &Order{ID: "ord-3"},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SUBTEST-STYLE TESTS
// ─────────────────────────────────────────────────────────────────────────────

func runOrderTests(s *Suite) {
	fx := newOrderFixture()

	// Group: Total()
	s.Run("Order/Total/simple", func(t *T) {
		// 2×1000 + 1×500 = 2500
		got := fx.simple.Total()
		if got != 2500 {
			t.Errorf("Total() = %d, want 2500", got)
		}
	})
	s.Run("Order/Total/with_discount", func(t *T) {
		// 3×1000 = 3000, 10% off = 2700
		got := fx.discount.Total()
		if got != 2700 {
			t.Errorf("Total() = %d, want 2700", got)
		}
	})
	s.Run("Order/Total/empty", func(t *T) {
		got := fx.empty.Total()
		if got != 0 {
			t.Errorf("Total() = %d, want 0", got)
		}
	})

	// Group: Summary()
	s.Run("Order/Summary/contains_id", func(t *T) {
		summary := fx.simple.Summary()
		if !strings.Contains(summary, "ord-1") {
			t.Errorf("Summary missing order ID: %q", summary)
		}
	})
	s.Run("Order/Summary/contains_total", func(t *T) {
		summary := fx.simple.Summary()
		if !strings.Contains(summary, "2500") {
			t.Errorf("Summary missing total: %q", summary)
		}
	})
	s.Run("Order/Summary/contains_items", func(t *T) {
		summary := fx.simple.Summary()
		if !strings.Contains(summary, "Widget") || !strings.Contains(summary, "Gadget") {
			t.Errorf("Summary missing item names: %q", summary)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// GOLDEN FILE PATTERN (simulated in-memory)
// ─────────────────────────────────────────────────────────────────────────────

// In production: golden files live at testdata/golden/<name>.txt
// Use -update flag to regenerate them.
var goldenFiles = map[string]string{
	"order_summary": `Order ord-1:
  Widget x2 @ 1000¢
  Gadget x1 @ 500¢
Total: 2500¢`,
}

func runGoldenTests(s *Suite) {
	fx := newOrderFixture()
	s.Run("Golden/order_summary", func(t *T) {
		got := fx.simple.Summary()
		want := goldenFiles["order_summary"]
		if got != want {
			t.Errorf("summary mismatch:\ngot:\n%s\nwant:\n%s", got, want)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// SETUP / TEARDOWN PATTERN
// ─────────────────────────────────────────────────────────────────────────────

type DB struct {
	records map[string]*Order
	calls   int
}

func newTestDB() *DB {
	return &DB{records: make(map[string]*Order)}
}

func (db *DB) Save(o *Order) {
	db.calls++
	db.records[o.ID] = o
}

func (db *DB) Load(id string) (*Order, bool) {
	db.calls++
	o, ok := db.records[id]
	return o, ok
}

func (db *DB) Reset() {
	db.records = make(map[string]*Order)
	db.calls = 0
}

func runDBTests(s *Suite) {
	db := newTestDB() // shared fixture — each test calls Reset() for isolation

	s.Run("DB/save_and_load", func(t *T) {
		db.Reset()
		order := &Order{ID: "test-1", Items: []Item{{"X", 100, 1}}}
		db.Save(order)
		got, ok := db.Load("test-1")
		if !ok {
			t.Errorf("Load: not found after Save")
			return
		}
		if got.ID != "test-1" {
			t.Errorf("Load: got ID %q, want test-1", got.ID)
		}
	})

	s.Run("DB/load_missing", func(t *T) {
		db.Reset()
		_, ok := db.Load("nonexistent")
		if ok {
			t.Errorf("Load: expected false for missing key, got true")
		}
	})

	s.Run("DB/overwrite", func(t *T) {
		db.Reset()
		db.Save(&Order{ID: "upd-1", Items: []Item{{"A", 100, 1}}})
		db.Save(&Order{ID: "upd-1", Items: []Item{{"B", 200, 1}}})
		got, _ := db.Load("upd-1")
		if got.Items[0].Name != "B" {
			t.Errorf("overwrite: got item %q, want B", got.Items[0].Name)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// PROPERTY-BASED / FUZZ PATTERN (hand-rolled)
// ─────────────────────────────────────────────────────────────────────────────

func runPropertyTests(s *Suite) {
	// Property: total is always non-negative
	s.Run("Property/total_non_negative", func(t *T) {
		orders := []*Order{
			{Items: []Item{{"A", 100, 1}}},
			{Items: []Item{{"A", 100, 1}}, Discount: 100},
			{Items: []Item{{"A", 0, 5}}},
			{},
		}
		for _, o := range orders {
			if total := o.Total(); total < 0 {
				t.Errorf("Total() = %d, want >= 0 (order: %+v)", total, o)
			}
		}
	})

	// Property: discount=0 same as no discount
	s.Run("Property/zero_discount_identity", func(t *T) {
		items := []Item{{"A", 500, 3}, {"B", 200, 1}}
		o1 := &Order{Items: items}
		o2 := &Order{Items: items, Discount: 0}
		if o1.Total() != o2.Total() {
			t.Errorf("0%% discount should equal no discount: %d != %d", o1.Total(), o2.Total())
		}
	})

	// Property: items order shouldn't affect total
	s.Run("Property/item_order_independent", func(t *T) {
		items := []Item{{"A", 100, 1}, {"B", 200, 2}, {"C", 50, 3}}
		reversed := make([]Item, len(items))
		copy(reversed, items)
		sort.Slice(reversed, func(i, j int) bool { return reversed[i].Name > reversed[j].Name })
		o1 := &Order{Items: items}
		o2 := &Order{Items: reversed}
		if o1.Total() != o2.Total() {
			t.Errorf("item order affects total: %d != %d", o1.Total(), o2.Total())
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Subtests, Fixtures, Golden Files ===")
	fmt.Println()

	fmt.Println("--- Order subtests ---")
	s1 := &Suite{}
	runOrderTests(s1)
	s1.Report()

	fmt.Println()
	fmt.Println("--- Golden file tests ---")
	s2 := &Suite{}
	runGoldenTests(s2)
	s2.Report()

	fmt.Println()
	fmt.Println("--- Setup/teardown (shared DB fixture) ---")
	s3 := &Suite{}
	runDBTests(s3)
	s3.Report()

	fmt.Println()
	fmt.Println("--- Property-based tests ---")
	s4 := &Suite{}
	runPropertyTests(s4)
	s4.Report()

	fmt.Println()
	fmt.Println("--- Patterns reference ---")
	fmt.Println(`  Subtest grouping (real Go):
    t.Run("Group/case", func(t *testing.T) { ... })
    // Run only matching: go test -run TestOrders/Total
    // Run parallel:      t.Parallel() inside the subtest

  Golden file (real Go):
    var update = flag.Bool("update", false, "update golden files")
    func TestSummary(t *testing.T) {
        got := order.Summary()
        golden := "testdata/summary.golden"
        if *update { os.WriteFile(golden, []byte(got), 0644) }
        want, _ := os.ReadFile(golden)
        if got != string(want) { t.Errorf(...) }
    }

  Fuzz test (real Go, Go 1.18+):
    func FuzzTotal(f *testing.F) {
        f.Add(100, 2, 10) // seed corpus
        f.Fuzz(func(t *testing.T, price, qty, discount int) {
            o := &Order{Items: []Item{{"x", price, qty}}, Discount: discount}
            if o.Total() < 0 { t.Errorf("negative total") }
        })
    }
    // go test -fuzz=FuzzTotal`)
}
