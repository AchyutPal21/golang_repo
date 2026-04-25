// FILE: book/part1_foundations/chapter06_coming_from_another_language/examples/01_translation_table/main.go
// CHAPTER: 06 — Coming From Another Language
// TOPIC: Print the translation table for a chosen source language.
//
// Run (from the chapter folder):
//   go run ./examples/01_translation_table             # all languages
//   go run ./examples/01_translation_table python      # only Python
//   go run ./examples/01_translation_table java javascript
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   The chapter's translation tables are easier to absorb in your terminal
//   than on a page. This program prints them filtered by source language so
//   you can focus on the one(s) you came from.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"fmt"
	"os"
	"strings"
)

// row is one entry in a translation table: the idiom in the source language
// and its idiomatic Go counterpart.
type row struct {
	source string
	go_    string
}

// table maps a language name to its translation rows. Order within a
// language is hand-curated for narrative — most-jarring transitions first.
var table = map[string][]row{
	"java": {
		{`class Foo { ... }`, `type Foo struct { ... }`},
		{`Foo extends Bar`, `embed: type Foo struct { Bar; ... }`},
		{`Foo implements Bar`, `implicit; just have the methods`},
		{`try { ... } catch (E e) { ... }`, `if err != nil { ... }`},
		{`throw new IllegalArgumentException(...)`, `return fmt.Errorf("invalid: %s", arg)`},
		{`Optional<T>`, `*T (nil = absent), or (T, bool)`},
		{`final` + ` field`, `unexported field, set only by constructor`},
		{`static member`, `package-level variable or function`},
		{`synchronized`, `sync.Mutex (or, better, a channel)`},
		{`CompletableFuture<T>`, `goroutine + channel`},
		{`@Nullable / @NotNull`, `nil-checks at boundaries; no annotations`},
		{`@Autowired`, `constructor injection by hand`},
	},
	"python": {
		{`def f(*args, **kwargs):`, `func F(opts ...Option) { ... }`},
		{`try: ... except: ...`, `if err := f(); err != nil { ... }`},
		{`[x*2 for x in xs]`, `for i, x := range xs { ys[i] = x*2 }`},
		{`with open(f) as fh:`, `f, err := os.Open(...); defer f.Close()`},
		{`class Animal: ...`, `type Animal struct { ... }`},
		{`@dataclass`, `a struct (no decorator needed)`},
		{`@property`, `unexported field + getter method`},
		{`async def f():`, `func F() { ... } (and `+"`go F()`"+`)`},
		{`asyncio.gather`, `errgroup.Group`},
		{`if x is None:`, `if x == nil: (for ptr/slice/map/iface)`},
		{`requests.get(url)`, `http.Get(url) // stdlib`},
	},
	"javascript": {
		{`class Foo { ... }`, `type Foo struct { ... }`},
		{`interface Foo { ... }`, `type Foo interface { ... }`},
		{`async function f() { ... }`, `func F() { ... } and go F()`},
		{`Promise<T>`, `chan T (or (T, error) + goroutine)`},
		{`await fetch(url)`, `http.Get(url) // synchronous; goroutine wraps it`},
		{`try { ... } catch (e) { ... }`, `if err != nil { ... }`},
		{`[1,2,3].map(x => x*2)`, `for i, x := range xs { ys[i] = x*2 }`},
		{`Object.assign({}, src)`, `struct copy (=) or for k, v := range src`},
		{`null | undefined`, `nil (one concept)`},
		{`import x from "y"`, `import "y"`},
		{`npm install`, `go mod tidy`},
		{`package.json`, `go.mod`},
		{`package-lock.json`, `go.sum`},
		{`.eslintrc`, `.golangci.yml`},
	},
	"cpp": {
		{`class Foo { public: ... };`, `type Foo struct { ... }`},
		{`virtual void f() = 0;`, `interface { F() }`},
		{`std::unique_ptr<T>`, `*T with explicit ownership convention`},
		{`std::shared_ptr<T>`, `*T (GC handles refcounts implicitly)`},
		{`std::string`, `string (immutable in Go)`},
		{`std::vector<T>`, `[]T (slice)`},
		{`std::map<K,V>`, `map[K]V (hash, not ordered)`},
		{`RAII destructor`, `defer cleanup()`},
		{`template<typename T>`, `func F[T any](x T) ...`},
		{`const T& parameter`, `T (value semantics)`},
		{`T& (out param)`, `*T parameter`},
		{`std::thread`, `go f()`},
		{`std::mutex`, `sync.Mutex`},
		{`std::condition_variable`, `channel + select`},
	},
	"rust": {
		{`struct Foo { ... }`, `type Foo struct { ... }`},
		{`impl Foo { fn bar(&self) ... }`, `func (f *Foo) Bar() { ... }`},
		{`trait Foo { ... }`, `type Foo interface { ... }`},
		{`Result<T, E>`, `(T, error)`},
		{`Option<T>`, `*T with nil, or (T, bool)`},
		{`?` + ` operator`, `if err != nil { return err }`},
		{`match x { ... }`, `switch x { ... }`},
		{`tokio::spawn(async { ... })`, `go func() { ... }()`},
		{`Arc<Mutex<T>>`, `*T + sync.Mutex (or use a channel)`},
		{`Box<dyn Trait>`, `interface { ... } value`},
		{`Cargo.toml`, `go.mod`},
		{`cargo build`, `go build`},
		{`clippy`, `golangci-lint`},
		{`derive macros`, `go:generate directives`},
	},
}

// order is the sequence languages are printed in, when no filter is given.
// We want Python and JS first (largest user base for transition guides),
// then statically-typed peers, then Rust.
var order = []string{"python", "javascript", "java", "cpp", "rust"}

func main() {
	requested := pickLanguages(os.Args[1:])
	for i, lang := range requested {
		if i > 0 {
			fmt.Println()
		}
		printLanguage(lang)
	}
}

func pickLanguages(args []string) []string {
	if len(args) == 0 {
		return order
	}
	picked := make([]string, 0, len(args))
	for _, a := range args {
		key := strings.ToLower(a)
		if _, ok := table[key]; ok {
			picked = append(picked, key)
		} else {
			fmt.Fprintf(os.Stderr, "unknown language %q (try: %s)\n",
				a, strings.Join(order, ", "))
			os.Exit(2)
		}
	}
	return picked
}

func printLanguage(lang string) {
	rows := table[lang]
	title := strings.ToUpper(lang) + " → Go"
	fmt.Println(title)
	fmt.Println(strings.Repeat("─", 70))

	srcWidth := 32
	for _, r := range rows {
		src := truncate(r.source, srcWidth)
		fmt.Printf("  %-*s  %s\n", srcWidth, src, r.go_)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
