// FILE: book/part6_production_engineering/chapter101_microservices_vs_monolith/exercises/01_migration_planner/main.go
// CHAPTER: 101 — Microservices vs Monolith
// EXERCISE: Migration readiness planner — score packages on coupling, churn,
//           and data isolation; produce a ranked decomposition roadmap.
//
// Run:
//   go run ./book/part6_production_engineering/chapter101_microservices_vs_monolith/exercises/01_migration_planner

package main

import (
	"fmt"
	"sort"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// PACKAGE PROFILE
// ─────────────────────────────────────────────────────────────────────────────

type PackageProfile struct {
	Name           string
	FanIn          int     // number of packages that import this one
	FanOut         int     // number of packages this one imports
	OwnedTables    int     // tables exclusively owned by this package
	SharedTables   int     // tables this package reads but doesn't own
	ChangeFreqMo   int     // commits per month
	HasDedicatedTeam bool  // true if a team owns this package exclusively
	APIStability   int     // 0–10: how often the public API changes (lower = more stable)
}

// ─────────────────────────────────────────────────────────────────────────────
// SCORING
// ─────────────────────────────────────────────────────────────────────────────

type PackageScore struct {
	Profile         PackageProfile
	CouplingScore   int // 0–100: higher = more coupled, harder to extract
	BenefitScore    int // 0–100: higher = more benefit from extraction
	MigrationCost   int // 0–100: higher = more expensive to migrate
	NetScore        int // BenefitScore - MigrationCost (positive = worth extracting)
	Recommendation  string
	Reasons         []string
}

func CouplingScore(p PackageProfile) int {
	score := 0

	// Fan-in penalty: highly imported = hard to extract without breaking callers
	if p.FanIn > 5 {
		score += 30
	} else if p.FanIn > 2 {
		score += 15
	}

	// Fan-out penalty: imports many others = pulls in transitive deps
	if p.FanOut > 5 {
		score += 20
	} else if p.FanOut > 2 {
		score += 10
	}

	// Shared tables = cross-service data dependency
	score += p.SharedTables * 15

	// Unstable API = extracting creates a moving contract
	if p.APIStability < 4 {
		score += 20
	} else if p.APIStability < 7 {
		score += 10
	}

	if score > 100 {
		score = 100
	}
	return score
}

func BenefitScore(p PackageProfile) int {
	score := 0

	// Dedicated team = clear ownership after extraction
	if p.HasDedicatedTeam {
		score += 30
	}

	// Owned tables = self-contained data store
	if p.OwnedTables > 0 {
		score += 20
	}

	// No shared tables = no data dependency leakage
	if p.SharedTables == 0 {
		score += 20
	}

	// High change frequency = bottlenecks the monolith release train
	if p.ChangeFreqMo > 10 {
		score += 20
	} else if p.ChangeFreqMo > 5 {
		score += 10
	}

	// Stable API = easy to define a service contract
	if p.APIStability >= 7 {
		score += 10
	}

	if score > 100 {
		score = 100
	}
	return score
}

func MigrationCost(p PackageProfile) int {
	cost := 0

	// High fan-in requires updating all callers
	cost += p.FanIn * 8

	// Each shared table needs a data migration strategy
	cost += p.SharedTables * 12

	// High churn = moving target during migration
	if p.ChangeFreqMo > 15 {
		cost += 20
	} else if p.ChangeFreqMo > 8 {
		cost += 10
	}

	// No dedicated team = migration needs cross-team coordination
	if !p.HasDedicatedTeam {
		cost += 15
	}

	if cost > 100 {
		cost = 100
	}
	return cost
}

func ScorePackage(p PackageProfile) PackageScore {
	coupling := CouplingScore(p)
	benefit := BenefitScore(p)
	cost := MigrationCost(p)
	net := benefit - cost

	var recommendation string
	var reasons []string

	switch {
	case net >= 20 && coupling < 40:
		recommendation = "EXTRACT NOW"
		reasons = append(reasons, "high benefit, low coupling, manageable cost")
	case net >= 5:
		recommendation = "PLAN EXTRACTION"
		reasons = append(reasons, "positive ROI, prepare strangler fig")
	case net >= -10 && coupling < 50:
		recommendation = "MONITOR"
		reasons = append(reasons, "marginal ROI, revisit in 6 months")
	default:
		recommendation = "LEAVE IN MONOLITH"
		reasons = append(reasons, "negative ROI or too tightly coupled")
	}

	if p.SharedTables > 0 {
		reasons = append(reasons, fmt.Sprintf("must resolve %d shared table(s) first", p.SharedTables))
	}
	if !p.HasDedicatedTeam {
		reasons = append(reasons, "assign team ownership before extracting")
	}

	return PackageScore{
		Profile:        p,
		CouplingScore:  coupling,
		BenefitScore:   benefit,
		MigrationCost:  cost,
		NetScore:       net,
		Recommendation: recommendation,
		Reasons:        reasons,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MIGRATION PLAN
// ─────────────────────────────────────────────────────────────────────────────

type MigrationPlan struct {
	Packages []PackageScore
}

func BuildPlan(profiles []PackageProfile) MigrationPlan {
	scores := make([]PackageScore, 0, len(profiles))
	for _, p := range profiles {
		scores = append(scores, ScorePackage(p))
	}
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].NetScore > scores[j].NetScore
	})
	return MigrationPlan{Packages: scores}
}

func (plan MigrationPlan) Print() {
	fmt.Printf("  %-22s  %-8s  %-8s  %-8s  %-8s  %-18s  %s\n",
		"Package", "Couple", "Benefit", "Cost", "Net", "Recommendation", "Notes")
	fmt.Printf("  %s\n", strings.Repeat("-", 110))
	for _, s := range plan.Packages {
		notes := ""
		if len(s.Reasons) > 0 {
			notes = s.Reasons[0]
		}
		fmt.Printf("  %-22s  %6d    %6d    %6d  %+6d    %-18s  %s\n",
			s.Profile.Name,
			s.CouplingScore,
			s.BenefitScore,
			s.MigrationCost,
			s.NetScore,
			s.Recommendation,
			notes)
	}
}

func (plan MigrationPlan) PrintRoadmap() {
	groups := map[string][]PackageScore{}
	order := []string{"EXTRACT NOW", "PLAN EXTRACTION", "MONITOR", "LEAVE IN MONOLITH"}
	for _, s := range plan.Packages {
		groups[s.Recommendation] = append(groups[s.Recommendation], s)
	}

	wave := 1
	for _, label := range order {
		pkgs := groups[label]
		if len(pkgs) == 0 {
			continue
		}
		switch label {
		case "EXTRACT NOW":
			fmt.Printf("  Wave %d (immediate): %s\n", wave, label)
		case "PLAN EXTRACTION":
			fmt.Printf("  Wave %d (next quarter): %s\n", wave, label)
		case "MONITOR":
			fmt.Printf("  Wave %d (deferred): %s\n", wave, label)
		default:
			fmt.Printf("  Wave %d (stay): %s\n", wave, label)
		}
		for _, s := range pkgs {
			notes := strings.Join(s.Reasons, "; ")
			fmt.Printf("    • %-22s  net=%+d  [%s]\n", s.Profile.Name, s.NetScore, notes)
		}
		wave++
		fmt.Println()
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DISTRIBUTED MONOLITH DETECTOR
// ─────────────────────────────────────────────────────────────────────────────

type ServiceDep struct {
	Service      string
	SharedTables []string // tables read across service boundary
}

func detectDistributedMonolith(services []ServiceDep) int {
	tableOwners := map[string][]string{}
	for _, svc := range services {
		for _, t := range svc.SharedTables {
			tableOwners[t] = append(tableOwners[t], svc.Service)
		}
	}

	violations := 0
	for table, svcs := range tableOwners {
		if len(svcs) > 1 {
			fmt.Printf("  [DM VIOLATION] Table %-20s accessed by: %s\n",
				table, strings.Join(svcs, ", "))
			violations++
		}
	}

	score := violations * 20
	if score > 100 {
		score = 100
	}
	return score
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 101 Exercise: Migration Planner ===")
	fmt.Println()

	// ── PACKAGE PROFILES ──────────────────────────────────────────────────────
	profiles := []PackageProfile{
		{
			Name: "billing", FanIn: 1, FanOut: 1,
			OwnedTables: 2, SharedTables: 0,
			ChangeFreqMo: 3, HasDedicatedTeam: true, APIStability: 8,
		},
		{
			Name: "catalog", FanIn: 3, FanOut: 1,
			OwnedTables: 2, SharedTables: 0,
			ChangeFreqMo: 4, HasDedicatedTeam: true, APIStability: 7,
		},
		{
			Name: "notification", FanIn: 2, FanOut: 1,
			OwnedTables: 0, SharedTables: 1, // reads orders
			ChangeFreqMo: 6, HasDedicatedTeam: false, APIStability: 5,
		},
		{
			Name: "search", FanIn: 1, FanOut: 0,
			OwnedTables: 1, SharedTables: 2, // reads products + categories
			ChangeFreqMo: 14, HasDedicatedTeam: true, APIStability: 6,
		},
		{
			Name: "order", FanIn: 4, FanOut: 3,
			OwnedTables: 2, SharedTables: 0,
			ChangeFreqMo: 9, HasDedicatedTeam: false, APIStability: 4,
		},
		{
			Name: "reporting", FanIn: 0, FanOut: 3,
			OwnedTables: 0, SharedTables: 3, // reads from 3 other services
			ChangeFreqMo: 5, HasDedicatedTeam: false, APIStability: 3,
		},
		{
			Name: "payment_gateway", FanIn: 1, FanOut: 0,
			OwnedTables: 1, SharedTables: 0,
			ChangeFreqMo: 1, HasDedicatedTeam: true, APIStability: 9,
		},
		{
			Name: "user_auth", FanIn: 6, FanOut: 0,
			OwnedTables: 2, SharedTables: 0,
			ChangeFreqMo: 2, HasDedicatedTeam: true, APIStability: 9,
		},
	}

	// ── FULL SCORING TABLE ────────────────────────────────────────────────────
	fmt.Println("--- Package scores (sorted by net ROI) ---")
	plan := BuildPlan(profiles)
	plan.Print()
	fmt.Println()

	// ── DECOMPOSITION ROADMAP ─────────────────────────────────────────────────
	fmt.Println("--- Decomposition roadmap ---")
	plan.PrintRoadmap()

	// ── DISTRIBUTED MONOLITH CHECK ────────────────────────────────────────────
	fmt.Println("--- Distributed monolith detection ---")
	fmt.Println("  (Simulating a partially-migrated system with shared table access)")
	services := []ServiceDep{
		{Service: "order-service", SharedTables: []string{"payments", "products"}},
		{Service: "reporting-service", SharedTables: []string{"orders", "payments", "products"}},
		{Service: "notification-service", SharedTables: []string{"orders"}},
		{Service: "billing-service", SharedTables: []string{}},
	}
	dmScore := detectDistributedMonolith(services)
	fmt.Printf("  Distributed monolith score: %d/100", dmScore)
	if dmScore >= 60 {
		fmt.Println("  ← CRITICAL: fix data ownership before continuing extraction")
	} else if dmScore >= 20 {
		fmt.Println("  ← WARNING: some cross-service table reads present")
	} else {
		fmt.Println("  ← OK")
	}
	fmt.Println()

	// ── MIGRATION COST FORMULA ────────────────────────────────────────────────
	fmt.Println("--- Migration cost model ---")
	fmt.Println(`  CouplingScore  = fan_in×weight + fan_out×weight + shared_tables×15 + api_instability
  BenefitScore   = dedicated_team + owned_tables + no_shared_tables + high_churn + stable_api
  MigrationCost  = fan_in×8 + shared_tables×12 + high_churn_penalty + no_team_penalty
  NetScore       = BenefitScore - MigrationCost

  Thresholds:
    Net ≥ 20 and coupling < 40  → EXTRACT NOW
    Net ≥  5                    → PLAN EXTRACTION (strangler fig next quarter)
    Net ≥ -10 and coupling < 50 → MONITOR (revisit in 6 months)
    else                        → LEAVE IN MONOLITH`)
}
