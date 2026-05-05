// FILE: book/part6_production_engineering/chapter101_microservices_vs_monolith/examples/02_service_decomposition/main.go
// CHAPTER: 101 — Microservices vs Monolith
// TOPIC: Service decomposition — strangler fig pattern, seam detection,
//        dependency graph, and data ownership rules.
//
// Run:
//   go run ./book/part6_production_engineering/chapter101_microservices_vs_monolith/examples/02_service_decomposition

package main

import (
	"fmt"
	"sort"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// DEPENDENCY GRAPH
// ─────────────────────────────────────────────────────────────────────────────

type Package struct {
	Name         string
	Imports      []string // packages this one depends on
	OwnedTables  []string // DB tables this package writes to
	SharedTables []string // DB tables this package reads but doesn't own
	ChangeFreq   int      // commits per month (higher = more churn)
	TeamOwner    string
}

type DependencyGraph struct {
	packages map[string]*Package
}

func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{packages: map[string]*Package{}}
}

func (g *DependencyGraph) Add(p Package) {
	g.packages[p.Name] = &p
}

// FanIn returns how many packages import the given package.
func (g *DependencyGraph) FanIn(name string) int {
	count := 0
	for _, p := range g.packages {
		for _, imp := range p.Imports {
			if imp == name {
				count++
			}
		}
	}
	return count
}

// FanOut returns how many packages the given package imports.
func (g *DependencyGraph) FanOut(name string) int {
	p, ok := g.packages[name]
	if !ok {
		return 0
	}
	return len(p.Imports)
}

func (g *DependencyGraph) Print() {
	names := make([]string, 0, len(g.packages))
	for n := range g.packages {
		names = append(names, n)
	}
	sort.Strings(names)

	fmt.Printf("  %-25s  %-8s  %-8s  %-20s  %s\n",
		"Package", "Fan-in", "Fan-out", "Owned tables", "Shared tables")
	fmt.Printf("  %s\n", strings.Repeat("-", 85))
	for _, n := range names {
		p := g.packages[n]
		owned := strings.Join(p.OwnedTables, ",")
		shared := strings.Join(p.SharedTables, ",")
		if owned == "" {
			owned = "—"
		}
		if shared == "" {
			shared = "—"
		}
		fmt.Printf("  %-25s  %-8d  %-8d  %-20s  %s\n",
			n, g.FanIn(n), g.FanOut(n), owned, shared)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SEAM DETECTOR
// ─────────────────────────────────────────────────────────────────────────────

type Seam struct {
	Package     string
	Score       int    // 0–100: higher = better extraction candidate
	Explanation string
}

// DetectSeams identifies packages that could be extracted as independent services.
// A good seam has: low fan-in, clear data ownership, stable API (low churn).
func DetectSeams(g *DependencyGraph) []Seam {
	var seams []Seam
	for name, p := range g.packages {
		score := 100
		var reasons []string

		fanIn := g.FanIn(name)
		if fanIn > 3 {
			penalty := (fanIn - 3) * 10
			score -= penalty
			reasons = append(reasons, fmt.Sprintf("high fan-in=%d (-%d)", fanIn, penalty))
		}

		if len(p.SharedTables) > 0 {
			penalty := len(p.SharedTables) * 15
			score -= penalty
			reasons = append(reasons, fmt.Sprintf("%d shared tables (-%d)", len(p.SharedTables), penalty))
		}

		if p.ChangeFreq > 10 {
			penalty := 20
			score -= penalty
			reasons = append(reasons, fmt.Sprintf("high churn=%d/mo (-%d)", p.ChangeFreq, penalty))
		}

		if len(p.OwnedTables) == 0 {
			score -= 10
			reasons = append(reasons, "no owned tables (-10)")
		}

		if score < 0 {
			score = 0
		}

		explanation := "good candidate"
		if score < 40 {
			explanation = "poor candidate — " + strings.Join(reasons, "; ")
		} else if score < 70 {
			explanation = "marginal — " + strings.Join(reasons, "; ")
		}

		seams = append(seams, Seam{Package: name, Score: score, Explanation: explanation})
	}
	sort.Slice(seams, func(i, j int) bool {
		return seams[i].Score > seams[j].Score
	})
	return seams
}

// ─────────────────────────────────────────────────────────────────────────────
// STRANGLER FIG ROUTER
// ─────────────────────────────────────────────────────────────────────────────

type Route struct {
	Path    string
	Handler string // "legacy" | service name
}

type StranglerRouter struct {
	routes []Route
}

func NewStranglerRouter() *StranglerRouter { return &StranglerRouter{} }

func (r *StranglerRouter) Register(path, handler string) {
	r.routes = append(r.routes, Route{Path: path, Handler: handler})
}

// Route finds the most specific matching handler for a path.
func (r *StranglerRouter) Route(path string) string {
	best := ""
	bestLen := -1
	for _, rt := range r.routes {
		if path == rt.Path || strings.HasPrefix(path, rt.Path+"/") {
			if len(rt.Path) > bestLen {
				bestLen = len(rt.Path)
				best = rt.Handler
			}
		}
	}
	if best == "" {
		return "legacy"
	}
	return best
}

func (r *StranglerRouter) MigratedPct() float64 {
	if len(r.routes) == 0 {
		return 0
	}
	migrated := 0
	for _, rt := range r.routes {
		if rt.Handler != "legacy" {
			migrated++
		}
	}
	return 100 * float64(migrated) / float64(len(r.routes))
}

func (r *StranglerRouter) PrintRoutes() {
	fmt.Printf("  %-35s  %s\n", "Path", "Handler")
	fmt.Printf("  %s\n", strings.Repeat("-", 55))
	for _, rt := range r.routes {
		flag := ""
		if rt.Handler != "legacy" {
			flag = "  ← migrated"
		}
		fmt.Printf("  %-35s  %s%s\n", rt.Path, rt.Handler, flag)
	}
	fmt.Printf("  Migration progress: %.0f%%\n", r.MigratedPct())
}

// ─────────────────────────────────────────────────────────────────────────────
// DATA OWNERSHIP CHECKER
// ─────────────────────────────────────────────────────────────────────────────

type TableOwnership struct {
	Table  string
	Owners []string // services/packages that write to this table
}

func checkDataOwnership(packages []Package) []TableOwnership {
	tableWriters := map[string][]string{}
	for _, p := range packages {
		for _, t := range p.OwnedTables {
			tableWriters[t] = append(tableWriters[t], p.Name)
		}
	}
	var result []TableOwnership
	tables := make([]string, 0, len(tableWriters))
	for t := range tableWriters {
		tables = append(tables, t)
	}
	sort.Strings(tables)
	for _, t := range tables {
		result = append(result, TableOwnership{Table: t, Owners: tableWriters[t]})
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 101: Service Decomposition ===")
	fmt.Println()

	// ── BUILD SAMPLE MONOLITH GRAPH ───────────────────────────────────────────
	g := NewDependencyGraph()
	packages := []Package{
		{
			Name: "order", Imports: []string{"catalog", "billing", "notification"},
			OwnedTables: []string{"orders", "order_items"},
			SharedTables: []string{}, ChangeFreq: 8, TeamOwner: "platform",
		},
		{
			Name: "catalog", Imports: []string{"search"},
			OwnedTables: []string{"products", "categories"},
			SharedTables: []string{}, ChangeFreq: 4, TeamOwner: "catalog-team",
		},
		{
			Name: "billing", Imports: []string{"payment_gateway"},
			OwnedTables: []string{"invoices", "payments"},
			SharedTables: []string{}, ChangeFreq: 3, TeamOwner: "billing-team",
		},
		{
			Name: "notification", Imports: []string{"template"},
			OwnedTables: []string{},
			SharedTables: []string{"orders"}, // violation: reads orders table directly
			ChangeFreq: 6, TeamOwner: "platform",
		},
		{
			Name: "search", Imports: []string{},
			OwnedTables: []string{"search_index"},
			SharedTables: []string{"products", "categories"}, // reads catalog tables
			ChangeFreq: 12, TeamOwner: "search-team",
		},
		{
			Name: "payment_gateway", Imports: []string{},
			OwnedTables: []string{"payment_methods"},
			SharedTables: []string{}, ChangeFreq: 1, TeamOwner: "billing-team",
		},
		{
			Name: "template", Imports: []string{},
			OwnedTables: []string{"email_templates"},
			SharedTables: []string{}, ChangeFreq: 2, TeamOwner: "platform",
		},
		{
			Name: "reporting", Imports: []string{"order", "catalog", "billing"},
			OwnedTables: []string{},
			SharedTables: []string{"orders", "products", "invoices"}, // reads 3 other-owned tables
			ChangeFreq: 5, TeamOwner: "analytics",
		},
	}
	for _, p := range packages {
		g.Add(p)
	}

	fmt.Println("--- Monolith dependency graph ---")
	g.Print()
	fmt.Println()

	// ── SEAM DETECTION ────────────────────────────────────────────────────────
	fmt.Println("--- Seam detection (extraction candidates, scored 0–100) ---")
	seams := DetectSeams(g)
	fmt.Printf("  %-25s  %-6s  %s\n", "Package", "Score", "Assessment")
	fmt.Printf("  %s\n", strings.Repeat("-", 75))
	for _, s := range seams {
		bar := strings.Repeat("█", s.Score/10) + strings.Repeat("░", 10-s.Score/10)
		fmt.Printf("  %-25s  %3d    %s  %s\n", s.Package, s.Score, bar, s.Explanation)
	}
	fmt.Println()

	// ── DATA OWNERSHIP CHECK ──────────────────────────────────────────────────
	fmt.Println("--- Data ownership violations ---")
	ownership := checkDataOwnership(packages)
	violations := 0
	for _, o := range ownership {
		if len(o.Owners) > 1 {
			fmt.Printf("  [VIOLATION] Table %-20s owned by: %s\n",
				o.Table, strings.Join(o.Owners, ", "))
			violations++
		}
	}
	fmt.Println("  Checking shared-table cross-reads...")
	for _, p := range packages {
		if len(p.SharedTables) > 0 {
			fmt.Printf("  [WARNING] %-20s reads tables it doesn't own: %s\n",
				p.Name, strings.Join(p.SharedTables, ", "))
			violations++
		}
	}
	if violations == 0 {
		fmt.Println("  No violations found.")
	} else {
		fmt.Printf("  Total violations: %d\n", violations)
	}
	fmt.Println()

	// ── STRANGLER FIG DEMO ────────────────────────────────────────────────────
	fmt.Println("--- Strangler fig: migrating /api/billing to billing-service ---")
	router := NewStranglerRouter()

	// Phase 1: everything in legacy
	router.Register("/api/orders", "legacy")
	router.Register("/api/catalog", "legacy")
	router.Register("/api/billing", "legacy")
	router.Register("/api/notifications", "legacy")
	router.Register("/api/search", "legacy")

	fmt.Println("  Phase 1 — all traffic in legacy:")
	router.PrintRoutes()
	fmt.Println()

	// Phase 2: billing extracted
	for i, rt := range router.routes {
		if rt.Path == "/api/billing" {
			router.routes[i].Handler = "billing-service"
		}
	}
	// search extracted too
	for i, rt := range router.routes {
		if rt.Path == "/api/search" {
			router.routes[i].Handler = "search-service"
		}
	}
	fmt.Println("  Phase 2 — billing and search extracted:")
	router.PrintRoutes()
	fmt.Println()

	// Test routing
	fmt.Println("  Route resolution examples:")
	paths := []string{"/api/billing/invoices", "/api/orders/123", "/api/search?q=laptop"}
	for _, p := range paths {
		fmt.Printf("    %-35s → %s\n", p, router.Route(p))
	}
	fmt.Println()

	// ── DECOMPOSITION PHASES ──────────────────────────────────────────────────
	fmt.Println("--- Strangler fig phases ---")
	fmt.Println(`  Phase 1 — Add abstraction
    • Define an interface in front of the module being extracted
    • Route all internal calls through the interface
    • No behavior change yet; just a seam

  Phase 2 — Build new service (dark launch)
    • Implement the interface as an RPC call to the new service
    • Deploy the new service; route 0% of traffic to it
    • Run shadow mode: send requests to both, compare responses

  Phase 3 — Migrate traffic
    • Gradually shift traffic: 5% → 25% → 100% (canary style)
    • Monitor error rate, latency at each stage
    • On success: remove legacy code path

  Rule: each phase is independently deployable and reversible.`)
}
