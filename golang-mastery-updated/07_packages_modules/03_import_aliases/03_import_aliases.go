// 03_import_aliases.go
//
// IMPORT SYNTAX, ALIASES, BLANK IMPORTS, DOT IMPORTS — Deep Dive
//
// The import statement is how Go brings external packages into scope.
// It looks simple but has significant nuance: aliases, blank imports,
// dot imports, grouping conventions, and the relationship between
// import paths, go.mod, and go.sum.
//
// This file covers:
//   - Basic import syntax (single and grouped)
//   - Import aliases: renaming a package at the call site
//   - Blank import: _ "path" — import for side effects only
//   - Dot import: . "path" — merge namespace (avoid this!)
//   - Import grouping conventions (stdlib / external / internal)
//   - When and why to use aliases
//   - The go.mod / go.sum relationship explained
//   - Unused import compile error — the rule and why it exists

package main

import (
	// --- stdlib group ---
	"fmt"
	"math/rand" // import path: "math/rand", package name: "rand"
	"os"
	"sort"
	"strings"
	// Note: there is no blank line separating stdlib from the next group
	// because this is a single-file demo using only stdlib.
	// In a real project with external deps, you'd have three groups:
	//   1. stdlib
	//   2. external (third-party modules)
	//   3. internal (your own module's packages)
	// Each group separated by a blank line. goimports enforces this automatically.
)

// ============================================================
// PART 1: BASIC IMPORT SYNTAX
// ============================================================
//
// Single import (rarely used for more than one package):
//   import "fmt"
//
// Grouped import (the idiomatic form when importing 2+ packages):
//   import (
//       "fmt"
//       "os"
//   )
//
// The string in quotes is the IMPORT PATH — it identifies the package
// in the module system. It is NOT necessarily the name you use in code.
//
// Example:
//   import "math/rand"
//   rand.Intn(10)   ← "rand" is the PACKAGE NAME (declared inside the package)
//
// The package name is determined by the "package <name>" declaration at the
// top of the source files in that directory — NOT by the import path.
// Conventionally they match, but they don't have to.
//
// COMPILE ERROR — unused import:
//   Go refuses to compile a program that imports a package but never uses it.
//   WHY: Unused imports slow compilation (the package must be loaded anyway),
//        indicate dead code, and make the import list misleading.
//   This is one of the most complained-about rules by Go newcomers,
//   but it pays dividends in large codebases: every import is intentional.

// ============================================================
// PART 2: IMPORT ALIASES
// ============================================================
//
// Syntax:
//   import alias "import/path"
//
// The alias becomes the identifier you use in code instead of the package name.
//
// When to use aliases:
//
//   Case 1 — NAME COLLISION: Two packages with the same name.
//     import (
//         crand "crypto/rand"    // alias because "rand" is already taken
//         "math/rand"
//     )
//     crand.Read(buf)   // crypto/rand
//     rand.Intn(10)     // math/rand
//
//   Case 2 — LONG OR OBSCURE PATH: The last path segment is a poor name.
//     import (
//         pb "github.com/acme/protos/gen/go/user/v1"
//     )
//     pb.CreateUserRequest{...}   // cleaner than user_v1.CreateUserRequest
//
//   Case 3 — DISAMBIGUATING VERSIONS: Importing v1 and v2 of the same module.
//     import (
//         configv1 "github.com/acme/config/v1"
//         configv2 "github.com/acme/config/v2"
//     )
//
//   Case 4 — TESTING RENAME: In test files, rename the package under test
//     to avoid a collision with the test package name.
//     import (
//         . "testing"          // sometimes used in test helpers (avoid for clarity)
//         subject "myapp/foo"  // alias to avoid conflict with test package name
//     )
//
// Best practice: use aliases sparingly. If you need many aliases,
// reconsider your package naming.

// demoAliases shows the alias syntax in action using stdlib packages
// that have identical package names but different import paths.
func demoAliases() {
	fmt.Println("--- Import alias demo ---")

	// math/rand and crypto/rand both declare "package rand".
	// In a real file importing both, you'd alias one:
	//   import (
	//       mrand "math/rand"
	//       crand "crypto/rand"
	//   )
	// Here we only have math/rand imported (as "rand"), so no alias needed.

	// Using rand with its natural name:
	r := rand.New(rand.NewSource(42))
	nums := make([]int, 5)
	for i := range nums {
		nums[i] = r.Intn(100)
	}
	fmt.Printf("  math/rand (no alias needed here): %v\n", nums)

	// Demonstrate that the alias IS the package name at call sites.
	// If we had:  import str "strings"
	// We'd write: str.ToUpper(...)
	// Here we just use strings normally:
	fmt.Printf("  strings.ToUpper: %s\n", strings.ToUpper("hello, aliases"))
	fmt.Println()
}

// ============================================================
// PART 3: BLANK IMPORT — import _ "path"
// ============================================================
//
// Syntax:
//   import _ "some/package"
//
// Effect:
//   - The package IS compiled and its init() function(s) run.
//   - No identifier from the package is introduced into the current file's scope.
//   - You cannot call any functions from the package directly.
//
// WHY would you import a package just for its side effects?
//
//   Pattern — SELF-REGISTERING PLUGINS:
//     Many Go packages register themselves in a global registry inside init().
//     The most famous example: database drivers.
//
//     import _ "github.com/lib/pq"  // registers PostgreSQL driver with database/sql
//
//     database/sql maintains an internal map of driver names to driver implementations.
//     The pq package's init() calls sql.Register("postgres", &Driver{}).
//     After the blank import, your code can use sql.Open("postgres", dsn).
//
//   Other common uses:
//     _ "image/png"   // register PNG decoder in the image package
//     _ "image/jpeg"  // register JPEG decoder
//     _ "net/http/pprof"  // register /debug/pprof HTTP handlers
//
// Common mistake: blank import in the wrong file.
//   The blank import must be in a file that is actually compiled into your binary.
//   If you put it only in a test file, the driver won't be registered in production.
//
// The underscore identifier _ is special in Go:
//   - As an import alias: discard all exported names, keep the side effects.
//   - As a variable name: discard the value (used with multi-return functions).
//   - As a struct field: force alignment padding (rare, advanced).

func demoBlankImport() {
	fmt.Println("--- Blank import demo (conceptual) ---")
	fmt.Println("  Blank import: import _ \"some/pkg\"")
	fmt.Println("  The package's init() runs → side effects take effect.")
	fmt.Println("  You cannot use any names from the package directly.")
	fmt.Println()
	fmt.Println("  Classic example:")
	fmt.Println("    import _ \"github.com/lib/pq\"")
	fmt.Println("    // pq.init() registers the 'postgres' driver with database/sql")
	fmt.Println("    db, _ := sql.Open(\"postgres\", dsn)  // works because pq registered it")
	fmt.Println()

	// In THIS file, we import "os" normally (not as blank).
	// Demonstrating: if we never used os, the compiler would reject the file.
	// We use it here to avoid the "imported and not used" error.
	_ = os.Args // use os to satisfy the compiler (os.Args is always defined)
	fmt.Println("  (os used via os.Args to satisfy compiler's unused-import rule)")
	fmt.Println()
}

// ============================================================
// PART 4: DOT IMPORT — import . "path"
// ============================================================
//
// Syntax:
//   import . "some/package"
//
// Effect:
//   ALL exported identifiers from the package are merged into the current
//   file's scope. You call them WITHOUT a package qualifier.
//
// Example:
//   import . "fmt"
//   Println("hello")  // no "fmt." prefix needed!
//
// WHY YOU SHOULD (ALMOST) NEVER USE DOT IMPORTS:
//
//   1. AMBIGUITY: If two dot-imported packages export the same name, the
//      compiler reports a conflict. Even if they don't conflict today, a
//      future version of the imported package might add a name that conflicts.
//
//   2. READABILITY: "Println(...)" tells you nothing about which package it
//      comes from. "fmt.Println(...)" is unambiguous. In large files, tracing
//      a bare function call to its package is painful.
//
//   3. MAINTENANCE: When you remove a dot import, all the bare names stop
//      working. IDE tooling has to work harder to resolve references.
//
//   4. CODE REVIEW: Reviewers cannot tell at a glance where bare names come from.
//
// LEGITIMATE use cases (very narrow):
//
//   1. Test files (package foo_test) testing package foo.
//      By convention, test packages sometimes use dot import of the package
//      under test to write tests that read like the package's own code:
//        import . "myapp/calculator"
//        result := Add(1, 2)  // reads naturally in tests
//      Even here, most style guides discourage it.
//
//   2. Gomega/Ginkgo testing DSL:
//        import . "github.com/onsi/gomega"
//        Expect(result).To(Equal(42))
//      The DSL is designed for dot import. This is an accepted exception.
//
// Rule of thumb: if you're not using a testing DSL explicitly designed for it,
// use a regular import or an alias.

func demoDotImport() {
	fmt.Println("--- Dot import demo (conceptual — AVOID in production) ---")
	fmt.Println("  import . \"fmt\"")
	fmt.Println("  Println(\"hello\")   ← no fmt. prefix")
	fmt.Println()
	fmt.Println("  Problems:")
	fmt.Println("    - Ambiguous: which package does Println come from?")
	fmt.Println("    - Name conflicts if two dot-imported packages clash")
	fmt.Println("    - Hurts readability and code review")
	fmt.Println()
	fmt.Println("  Acceptable only with testing DSLs (Gomega, Ginkgo) designed for it.")
	fmt.Println()
}

// ============================================================
// PART 5: IMPORT GROUPING CONVENTIONS
// ============================================================
//
// The Go community (and goimports tool) organizes imports into groups
// separated by blank lines:
//
//   import (
//       // Group 1: stdlib
//       "fmt"
//       "os"
//       "strings"
//
//       // Group 2: external (third-party modules)
//       "github.com/pkg/errors"
//       "go.uber.org/zap"
//
//       // Group 3: internal (your own module)
//       "myapp/internal/config"
//       "myapp/pkg/auth"
//   )
//
// WHY group them?
//   - At a glance, you can see the dependency profile of a file.
//   - Standard library vs third-party vs internal is a meaningful distinction.
//   - goimports (and gopls) enforce this automatically — just save the file.
//
// Some teams add a fourth group for "generated code" or "test helpers".
// The important thing is consistency within a codebase, enforced by tooling.
//
// Within each group, imports are alphabetically sorted by goimports.
// Never sort manually — let the tool do it.

func demoImportGrouping() {
	fmt.Println("--- Import grouping conventions ---")
	groups := []string{
		"Group 1: stdlib     (fmt, os, strings, math/rand, ...)",
		"Group 2: external   (github.com/pkg/errors, go.uber.org/zap, ...)",
		"Group 3: internal   (myapp/internal/config, myapp/pkg/auth, ...)",
	}
	for _, g := range groups {
		fmt.Printf("  %s\n", g)
	}
	fmt.Println("  Blank line between groups. Alphabetical within groups.")
	fmt.Println("  Use goimports to enforce automatically.")
	fmt.Println()
}

// ============================================================
// PART 6: go.mod AND go.sum RELATIONSHIP
// ============================================================
//
// go.mod — the manifest of your module:
//   - Declares the module path (the root of all your import paths).
//   - Declares the minimum Go version.
//   - Lists direct dependencies with minimum required versions.
//   - Can have replace and exclude directives.
//   - HUMAN-EDITABLE (but use "go get" or "go mod tidy" rather than editing by hand).
//
// go.sum — the cryptographic ledger:
//   - Contains expected SHA-256 hashes for every module version used.
//   - Two lines per module version:
//       github.com/pkg/errors v0.9.1 h1:<hash-of-zip>
//       github.com/pkg/errors v0.9.1/go.mod h1:<hash-of-go.mod>
//   - The go command verifies downloaded modules against go.sum before using them.
//   - If the hash doesn't match → build fails immediately (tampered module).
//   - Should be committed to version control (it's your trust anchor).
//   - NEVER manually edited. Always generated by go commands.
//   - The checksum database (sum.golang.org) is a transparency log; go can
//     verify hashes against it (configurable via GONOSUMDB, GONOSUMCHECK).
//
// Relationship between go.mod and go.sum:
//   go.mod says WHAT you need and which MINIMUM version.
//   go.sum says WHAT the bits of each version MUST look like.
//   Together they give you reproducible, tamper-evident builds.
//
// When you run "go get github.com/pkg/errors@v0.9.1":
//   1. go.mod updated: adds/updates the require line.
//   2. go.sum updated: adds the hash lines for that version.
//   3. Module downloaded to $GOMODCACHE.
//
// When you run "go mod tidy":
//   1. Adds missing dependencies (packages imported but not yet in go.mod).
//   2. Removes unused dependencies (packages in go.mod but no longer imported).
//   3. Updates go.sum accordingly.

func demoGoModSum() {
	fmt.Println("--- go.mod and go.sum relationship ---")

	gomodExample := `
  module github.com/acme/myapp   ← module path (root of all import paths)

  go 1.22                        ← minimum Go version

  require (
      github.com/pkg/errors v0.9.1   ← direct dependency, minimum version
      go.uber.org/zap v1.27.0        ← direct dependency
  )

  require (
      go.uber.org/atomic v1.11.0     ← indirect dependency (transitive)
  )

  // replace: swap a dep with a local copy (useful during development)
  replace github.com/acme/shared => ../shared
`
	fmt.Println(gomodExample)

	gosumExample := `
  go.sum lines (one pair per module version):
    github.com/pkg/errors v0.9.1 h1:FEBLx1zS214owpjy7qsBeixbURkuhQAwrK5UwLGTwt38=
    github.com/pkg/errors v0.9.1/go.mod h1:bwawxfHBFNV+L2hUp1rHADufV3IMtnDRdf1r5NINEl0=
`
	fmt.Println(gosumExample)
	fmt.Println("  go.mod = WHAT you need (manifest)")
	fmt.Println("  go.sum = WHAT the bits must look like (integrity)")
	fmt.Println()
}

// ============================================================
// PART 7: COMMON IMPORT MISTAKES AND HOW TO AVOID THEM
// ============================================================

func demoCommonMistakes() {
	fmt.Println("--- Common import mistakes ---")

	mistakes := []struct {
		mistake    string
		correction string
	}{
		{
			"import \"math/rand\" then use math.Rand()",
			"Package name is 'rand', use rand.Intn() (the path segment after / is not the name)",
		},
		{
			"Unused import left in file",
			"The compiler will refuse to compile. Remove it or use blank import _ if side effects needed.",
		},
		{
			"import . \"fmt\" for brevity",
			"Use fmt.Println(). The dot import hides where names come from.",
		},
		{
			"Forgetting to run go mod tidy after adding an import",
			"Run 'go mod tidy' to add the dependency to go.mod and go.sum.",
		},
		{
			"Manually editing go.sum",
			"Never edit go.sum by hand. It's generated by the go toolchain.",
		},
		{
			"Committing go.sum to .gitignore",
			"go.sum SHOULD be committed — it's your integrity anchor for reproducible builds.",
		},
	}

	for i, m := range mistakes {
		fmt.Printf("  Mistake %d: %s\n", i+1, m.mistake)
		fmt.Printf("  Fix:       %s\n", m.correction)
		fmt.Println()
	}
}

// ============================================================
// MAIN
// ============================================================

func main() {
	fmt.Println("=== 03: Import Syntax, Aliases, Blank/Dot Imports ===")
	fmt.Println()

	demoAliases()
	demoBlankImport()
	demoDotImport()
	demoImportGrouping()
	demoGoModSum()
	demoCommonMistakes()

	// --- Using sort and strings to show normal imports in action ---
	fmt.Println("--- Normal imports in action (sort, strings, rand) ---")
	words := []string{"module", "package", "import", "alias", "blank", "dot"}
	sort.Strings(words)
	fmt.Printf("  Sorted: %s\n", strings.Join(words, ", "))

	r := rand.New(rand.NewSource(99))
	fmt.Printf("  Random pick: %s\n", words[r.Intn(len(words))])
	fmt.Println()

	fmt.Println("=== End of 03_import_aliases.go ===")
}
