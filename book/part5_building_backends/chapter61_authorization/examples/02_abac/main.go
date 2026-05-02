// FILE: book/part5_building_backends/chapter61_authorization/examples/02_abac/main.go
// CHAPTER: 61 — Authorization
// TOPIC: Attribute-Based Access Control (ABAC) —
//        policy functions, a policy set with deny-unless-permit evaluation,
//        and an HTTP handler that enforces article-access rules based on
//        subject/resource/action/environment attributes.
//
// Run (from the chapter folder):
//   go run ./examples/02_abac

package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// POLICY ENGINE
// ─────────────────────────────────────────────────────────────────────────────

// Attrs is a free-form attribute map passed to every policy.
type Attrs map[string]any

// Policy is a single authorization rule.
// Return true  → permit (if PolicySet is deny-unless-permit, evaluation stops).
// Return false → this policy does not permit (try next policy).
type Policy func(subject, resource, action, environment Attrs) bool

// PolicySet evaluates a slice of policies in order.
// Decision: deny-unless-permit — access is denied unless at least one policy
// returns true.  The first policy that returns true wins immediately.
type PolicySet []Policy

// Evaluate runs the policy set.  Returns true (permit) if any policy permits.
func (ps PolicySet) Evaluate(subject, resource, action, environment Attrs) bool {
	for _, p := range ps {
		if p(subject, resource, action, environment) {
			return true
		}
	}
	return false
}

// ─────────────────────────────────────────────────────────────────────────────
// CONCRETE POLICIES
// ─────────────────────────────────────────────────────────────────────────────

// ownerOnly — the subject is the owner of the resource.
var ownerOnly Policy = func(subject, resource, _, _ Attrs) bool {
	sid, _ := subject["id"].(string)
	rid, _ := resource["owner_id"].(string)
	return sid != "" && sid == rid
}

// adminBypass — admin role bypasses all restrictions.
var adminBypass Policy = func(subject, _, _, _ Attrs) bool {
	role, _ := subject["role"].(string)
	return role == "admin"
}

// publishedDuringBusinessHours — non-owner can read a published article only
// during business hours (9-17 inclusive).  Both conditions must hold.
var publishedDuringBusinessHours Policy = func(subject, resource, _, environment Attrs) bool {
	sid, _ := subject["id"].(string)
	rid, _ := resource["owner_id"].(string)
	if sid == rid {
		return false // let ownerOnly handle it
	}
	pub, _ := resource["published"].(bool)
	if !pub {
		return false
	}
	hour, _ := environment["hour"].(int)
	return hour >= 9 && hour <= 17
}

// ─────────────────────────────────────────────────────────────────────────────
// ARTICLE DATA STORE
// ─────────────────────────────────────────────────────────────────────────────

type Article struct {
	ID        int
	Title     string
	OwnerID   string
	Published bool
}

var articles = map[int]Article{
	1: {ID: 1, Title: "Draft by Alice", OwnerID: "alice", Published: false},
	2: {ID: 2, Title: "Published by Bob", OwnerID: "bob", Published: true},
	3: {ID: 3, Title: "Published by Alice", OwnerID: "alice", Published: true},
}

// ─────────────────────────────────────────────────────────────────────────────
// HTTP HANDLER
// ─────────────────────────────────────────────────────────────────────────────

// buildPolicySet returns the application's article-read policy set.
func buildPolicySet() PolicySet {
	return PolicySet{adminBypass, ownerOnly, publishedDuringBusinessHours}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// subjectFromHeader parses the demo header:
//
//	Authorization: Bearer user-<id>-role-<role>
//
// e.g. "Bearer user-alice-role-viewer"
func subjectFromHeader(r *http.Request) (Attrs, bool) {
	auth := r.Header.Get("Authorization")
	auth = strings.TrimPrefix(auth, "Bearer ")
	// format: user-<id>-role-<role>
	if !strings.HasPrefix(auth, "user-") {
		return nil, false
	}
	auth = strings.TrimPrefix(auth, "user-")
	idx := strings.Index(auth, "-role-")
	if idx < 0 {
		return nil, false
	}
	uid := auth[:idx]
	role := auth[idx+6:]
	if uid == "" || role == "" {
		return nil, false
	}
	return Attrs{"id": uid, "role": role}, true
}

func handleGetArticle(ps PolicySet, currentHour func() int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		subject, ok := subjectFromHeader(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or malformed Authorization"})
			return
		}

		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
			return
		}

		art, exists := articles[id]
		if !exists {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}

		resource := Attrs{
			"id":        art.ID,
			"owner_id":  art.OwnerID,
			"published": art.Published,
		}
		action := Attrs{"type": "read"}
		environment := Attrs{"hour": currentHour()}

		if !ps.Evaluate(subject, resource, action, environment) {
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error": fmt.Sprintf("access denied for user %q to article %d", subject["id"], art.ID),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"id":        art.ID,
			"title":     art.Title,
			"published": art.Published,
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	ps := buildPolicySet()

	// Controllable clock for the test harness.
	testHour := 12 // default: business hours
	currentHour := func() int { return testHour }

	mux := http.NewServeMux()
	mux.HandleFunc("GET /articles/{id}", handleGetArticle(ps, currentHour))

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	base := "http://" + ln.Addr().String()
	go (&http.Server{Handler: mux}).Serve(ln) //nolint:errcheck

	client := &http.Client{Timeout: 3 * time.Second}

	do := func(path, auth string) int {
		req, _ := http.NewRequest("GET", base+path, nil)
		if auth != "" {
			req.Header.Set("Authorization", "Bearer "+auth)
		}
		resp, err := client.Do(req)
		if err != nil {
			return 0
		}
		resp.Body.Close()
		return resp.StatusCode
	}

	check := func(label string, got, want int) {
		mark := "✓"
		if got != want {
			mark = "✗"
		}
		fmt.Printf("  %s %-60s got=%d want=%d\n", mark, label, got, want)
	}

	fmt.Printf("=== ABAC Policy Tests — %s ===\n\n", base)

	fmt.Println("--- Policy descriptions ---")
	fmt.Println("  adminBypass                   : admin bypasses all restrictions")
	fmt.Println("  ownerOnly                     : owner reads their own article (published or not, any hour)")
	fmt.Println("  publishedDuringBusinessHours  : non-owner reads a published article only during 9-17")
	fmt.Println()

	fmt.Println("--- No auth → 401 ---")
	check("GET /articles/1 (no auth)", do("/articles/1", ""), 401)

	fmt.Println()
	fmt.Println("--- Article 1: unpublished, owner=alice (hour=12, business hours) ---")
	check("alice reads her own unpublished article  (ownerOnly → permit)", do("/articles/1", "user-alice-role-viewer"), 200)
	check("bob cannot read alice's unpublished article (deny)", do("/articles/1", "user-bob-role-viewer"), 403)
	check("admin can read alice's unpublished article (adminBypass)", do("/articles/1", "user-admin-role-admin"), 200)

	fmt.Println()
	fmt.Println("--- Article 2: published, owner=bob (hour=12, business hours) ---")
	check("alice reads bob's published article (publishedDuringBusinessHours)", do("/articles/2", "user-alice-role-viewer"), 200)
	check("bob reads his own published article (ownerOnly → permit)", do("/articles/2", "user-bob-role-viewer"), 200)

	fmt.Println()
	fmt.Println("--- Outside business hours (hour=22) ---")
	testHour = 22
	check("alice reads bob's published article outside hours (deny)", do("/articles/2", "user-alice-role-viewer"), 403)
	check("bob reads his own article outside hours (ownerOnly still permits)", do("/articles/2", "user-bob-role-viewer"), 200)
	check("admin reads any article outside hours (adminBypass)", do("/articles/1", "user-admin-role-admin"), 200)

	fmt.Println()
	fmt.Println("--- Back to business hours (hour=10) ---")
	testHour = 10
	check("alice reads bob's published article during business hours", do("/articles/2", "user-alice-role-viewer"), 200)
	check("alice reads alice's unpublished article (ownerOnly)", do("/articles/1", "user-alice-role-viewer"), 200)
	check("carol reads alice's unpublished article (deny: not owner, not published)", do("/articles/1", "user-carol-role-viewer"), 403)
}
