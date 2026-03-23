// 06_higher_order_functions.go
//
// TOPIC: Higher-Order Functions — Functions as Values, Callbacks, Filter/Map/Reduce,
//        Function Composition, Adapter Pattern, Strategy Pattern
//
// A "higher-order function" is any function that either:
//   a) Takes one or more functions as arguments, OR
//   b) Returns a function as its result
//
// Higher-order functions are the foundation of functional programming.
// In Go, they're enabled by the fact that functions are first-class values
// (covered in 01_function_basics.go).
//
// WHY HIGHER-ORDER FUNCTIONS?
//   They let you abstract over BEHAVIOR, not just data. Instead of writing
//   separate "filterEven", "filterOdd", "filterGreaterThan10" functions, you
//   write one "filter" that takes a predicate function. The predicate encodes
//   the specific behavior.
//
//   In Java 8+ this is done via lambdas + Stream API.
//   In Python: map(), filter(), functools.reduce(), list comprehensions.
//   In Go: explicit higher-order functions (no built-in stream API, but clean syntax).
//
// NOTE: Go 1.18 added generics which make filter/map/reduce type-safe for any type.
//       In this file we use []int for clarity. Module 09_generics covers the generic
//       versions (Filter[T any], Map[T, R any], Reduce[T, R any]).

package main

import (
	"fmt"
	"sort"
	"strings"
)

// ─── 1. FUNCTION TYPES ────────────────────────────────────────────────────────
//
// Every function in Go has a type determined by its signature.
// Functions with the same signature are the same TYPE — they're interchangeable.
//
// You can write the type inline:  func(int) bool
// Or define a named type:         type Predicate func(int) bool
//
// Named function types improve readability and let you attach documentation.

// Predicate is a function that tests a single int and returns true or false.
// Naming the type makes parameter lists and return types more readable.
type Predicate func(int) bool

// Transformer maps an int to another int.
type Transformer func(int) int

// Reducer combines two ints into one (used in Reduce/fold operations).
type Reducer func(acc, val int) int

// ─── 2. FILTER — KEEP ELEMENTS MATCHING A PREDICATE ──────────────────────────
//
// filter returns a new slice containing only elements for which predicate returns true.
// The original slice is NOT modified (pure function).
//
// In Python: [x for x in nums if predicate(x)]  or  filter(predicate, nums)
// In Java:   stream.filter(predicate).collect(...)

func filter(nums []int, predicate Predicate) []int {
	// Use nil as initial result — append will allocate when needed.
	// This avoids allocating an empty slice when there are no matches.
	var result []int
	for _, n := range nums {
		if predicate(n) {
			result = append(result, n)
		}
	}
	return result
}

// ─── 3. MAP — TRANSFORM EVERY ELEMENT ────────────────────────────────────────
//
// mapInts applies a transformer to every element and returns a new slice.
// The name 'mapInts' avoids shadowing the builtin 'map' keyword.
//
// In Python: [transform(x) for x in nums]  or  map(transform, nums)
// In Java:   stream.map(transform).collect(...)

func mapInts(nums []int, transform Transformer) []int {
	result := make([]int, len(nums)) // pre-allocate: we know exact size
	for i, n := range nums {
		result[i] = transform(n)
	}
	return result
}

// ─── 4. REDUCE — FOLD A SLICE INTO A SINGLE VALUE ────────────────────────────
//
// reduce folds a slice into a single value by repeatedly applying reducer.
// 'initial' is the starting accumulator value (identity element).
//
// For sum: reduce(nums, 0, func(acc, v int) int { return acc + v })
// For product: reduce(nums, 1, func(acc, v int) int { return acc * v })
// For max: reduce(nums, math.MinInt, func(acc, v int) int { if v > acc { return v }; return acc })
//
// In Python: functools.reduce(reducer, nums, initial)
// In Haskell: foldl reducer initial nums

func reduce(nums []int, initial int, reducer Reducer) int {
	acc := initial
	for _, n := range nums {
		acc = reducer(acc, n)
	}
	return acc
}

// ─── 5. COMBINING FILTER + MAP + REDUCE ──────────────────────────────────────
//
// These three primitives can be chained to express complex data transformations
// declaratively. This is the "pipeline" style.

func sumOfSquaresOfEvens(nums []int) int {
	evens := filter(nums, func(n int) bool { return n%2 == 0 })
	squares := mapInts(evens, func(n int) int { return n * n })
	return reduce(squares, 0, func(acc, v int) int { return acc + v })
}

// ─── 6. CALLBACKS ─────────────────────────────────────────────────────────────
//
// A callback is a function passed as an argument to be called by the receiving
// function at some point. The classic use: event handlers, completion handlers.
//
// Go uses callbacks extensively:
//   - sort.Slice(s, func(i, j int) bool { ... })
//   - http.HandleFunc("/path", func(w http.ResponseWriter, r *http.Request) { ... })
//   - filepath.Walk(root, func(path string, info fs.FileInfo, err error) error { ... })

type Event struct {
	Name string
	Data any
}

type EventHandler func(Event)

// EventBus is a simplified event dispatcher using callbacks.
type EventBus struct {
	handlers map[string][]EventHandler
}

func NewEventBus() *EventBus {
	return &EventBus{handlers: make(map[string][]EventHandler)}
}

func (eb *EventBus) Subscribe(event string, handler EventHandler) {
	eb.handlers[event] = append(eb.handlers[event], handler)
}

func (eb *EventBus) Publish(event Event) {
	for _, h := range eb.handlers[event.Name] {
		h(event) // invoke each registered callback
	}
}

// ─── 7. FUNCTION COMPOSITION ──────────────────────────────────────────────────
//
// Function composition: given f and g, create h = f ∘ g where h(x) = f(g(x)).
// The output of g becomes the input to f.
//
// In math: (f ∘ g)(x) = f(g(x))
// In Haskell: h = f . g
// In Go: we build compose() explicitly.

// compose returns a new function that applies g first, then f.
func compose(f, g Transformer) Transformer {
	return func(n int) int {
		return f(g(n)) // g runs first, f runs on g's result
	}
}

// composeMany composes any number of transformers from RIGHT to LEFT.
// composeMany(f, g, h)(x) == f(g(h(x)))
func composeMany(fns ...Transformer) Transformer {
	return func(n int) int {
		result := n
		// Apply right-to-left (last function in list runs first)
		for i := len(fns) - 1; i >= 0; i-- {
			result = fns[i](result)
		}
		return result
	}
}

// pipe is like compose but applies left-to-right (more natural reading order).
// pipe(f, g, h)(x) == h(g(f(x)))
func pipe(fns ...Transformer) Transformer {
	return func(n int) int {
		result := n
		for _, fn := range fns {
			result = fn(result)
		}
		return result
	}
}

// ─── 8. STRATEGY PATTERN USING FUNCTIONS ──────────────────────────────────────
//
// The Strategy pattern defines a family of algorithms and makes them
// interchangeable. In Java this requires an interface + multiple implementing
// classes. In Go, a function type IS the strategy.
//
// Example: sorting with different comparison strategies.

type SortStrategy func([]int) []int

func sortWith(nums []int, strategy SortStrategy) []int {
	// Work on a copy so we don't mutate the input
	cp := make([]int, len(nums))
	copy(cp, nums)
	return strategy(cp)
}

var (
	ascending SortStrategy = func(nums []int) []int {
		sort.Ints(nums)
		return nums
	}

	descending SortStrategy = func(nums []int) []int {
		sort.Sort(sort.Reverse(sort.IntSlice(nums)))
		return nums
	}

	absSort SortStrategy = func(nums []int) []int {
		abs := func(n int) int {
			if n < 0 {
				return -n
			}
			return n
		}
		sort.Slice(nums, func(i, j int) bool {
			return abs(nums[i]) < abs(nums[j])
		})
		return nums
	}
)

// ─── 9. ADAPTER PATTERN USING FUNCTIONS ───────────────────────────────────────
//
// An adapter converts a function with one signature into a function with
// another signature. This is useful when APIs expect specific signatures but
// you have functions with compatible-but-different signatures.
//
// Real example: adapting a func(string) to an http.HandlerFunc
// (covered in depth in 09_function_types_interfaces.go)

// Logger has signature func(string).
type Logger func(string)

// prefixLogger adapts a Logger by prepending a prefix to every message.
// It takes a Logger and returns a Logger — adapting the behavior.
func prefixLogger(prefix string, log Logger) Logger {
	return func(msg string) {
		log("[" + prefix + "] " + msg)
	}
}

// timestampLogger adds a timestamp (simulated) to every log message.
func timestampLogger(log Logger) Logger {
	return func(msg string) {
		log("2024-01-01T00:00:00Z " + msg)
	}
}

// ─── 10. PARTIAL APPLICATION ──────────────────────────────────────────────────
//
// Partial application: fix some arguments of a function, producing a new
// function that takes the remaining arguments.
//
// It's related to currying (common in Haskell/ML), but in Go we do it manually
// via closures.

// add3 takes 3 ints.
func add3(a, b, c int) int { return a + b + c }

// partial2 partially applies the first argument.
func partial2(f func(int, int, int) int, a int) func(int, int) int {
	return func(b, c int) int {
		return f(a, b, c)
	}
}

// ─── MAIN ─────────────────────────────────────────────────────────────────────

func main() {
	sep := strings.Repeat("═", 55)
	fmt.Println(sep)
	fmt.Println("  HIGHER-ORDER FUNCTIONS IN GO")
	fmt.Println(sep)

	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	fmt.Println("  source:", nums)

	// 2. filter
	fmt.Println("\n── 2. filter ──")
	evens := filter(nums, func(n int) bool { return n%2 == 0 })
	odds := filter(nums, func(n int) bool { return n%2 != 0 })
	gt5 := filter(nums, func(n int) bool { return n > 5 })
	fmt.Println("  evens:", evens)
	fmt.Println("  odds: ", odds)
	fmt.Println("  >5:   ", gt5)

	// 3. mapInts
	fmt.Println("\n── 3. mapInts ──")
	doubled := mapInts(nums, func(n int) int { return n * 2 })
	squared := mapInts(nums, func(n int) int { return n * n })
	negated := mapInts(nums, func(n int) int { return -n })
	fmt.Println("  doubled:", doubled)
	fmt.Println("  squared:", squared)
	fmt.Println("  negated:", negated)

	// 4. reduce
	fmt.Println("\n── 4. reduce ──")
	sum := reduce(nums, 0, func(acc, v int) int { return acc + v })
	product := reduce(nums[:5], 1, func(acc, v int) int { return acc * v })
	max := reduce(nums, nums[0], func(acc, v int) int {
		if v > acc {
			return v
		}
		return acc
	})
	fmt.Println("  sum:", sum)
	fmt.Println("  product of first 5:", product)
	fmt.Println("  max:", max)

	// 5. Chained pipeline
	fmt.Println("\n── 5. Chained Pipeline ──")
	result := sumOfSquaresOfEvens(nums)
	fmt.Printf("  sumOfSquaresOfEvens(%v)\n", nums)
	fmt.Printf("  = filter(even) → map(square) → reduce(sum) = %d\n", result)

	// 6. Callbacks (event bus)
	fmt.Println("\n── 6. Callbacks (Event Bus) ──")
	bus := NewEventBus()

	bus.Subscribe("user.login", func(e Event) {
		fmt.Printf("  [audit] user logged in: %v\n", e.Data)
	})
	bus.Subscribe("user.login", func(e Event) {
		fmt.Printf("  [email] sending welcome email to: %v\n", e.Data)
	})
	bus.Subscribe("user.logout", func(e Event) {
		fmt.Printf("  [audit] user logged out: %v\n", e.Data)
	})

	bus.Publish(Event{Name: "user.login", Data: "alice@example.com"})
	bus.Publish(Event{Name: "user.logout", Data: "alice@example.com"})

	// 7. Function composition
	fmt.Println("\n── 7. Function Composition ──")
	double := func(n int) int { return n * 2 }
	addOne := func(n int) int { return n + 1 }
	square := func(n int) int { return n * n }

	doubleThenadd := compose(addOne, double) // addOne(double(x))
	fmt.Printf("  compose(addOne, double)(5) = addOne(double(5)) = %d\n", doubleThenadd(5))

	pipeline := pipe(double, addOne, square) // square(addOne(double(x)))
	fmt.Printf("  pipe(double, addOne, square)(3):\n")
	fmt.Printf("    double(3)=%d → addOne(6)=%d → square(7)=%d\n",
		double(3), addOne(double(3)), pipeline(3))

	composed := composeMany(square, addOne, double) // same as pipe but reversed
	fmt.Printf("  composeMany(square,addOne,double)(3) = %d\n", composed(3))

	// 8. Strategy pattern
	fmt.Println("\n── 8. Strategy Pattern ──")
	unsorted := []int{5, -3, 8, 1, -7, 4, 2}
	fmt.Println("  source:     ", unsorted)
	fmt.Println("  ascending:  ", sortWith(unsorted, ascending))
	fmt.Println("  descending: ", sortWith(unsorted, descending))
	fmt.Println("  abs-sort:   ", sortWith(unsorted, absSort))

	// 9. Adapter pattern
	fmt.Println("\n── 9. Adapter Pattern (Logger) ──")
	baseLog := Logger(func(msg string) { fmt.Println(" ", msg) })
	appLog := prefixLogger("APP", baseLog)
	dbLog := prefixLogger("DB", timestampLogger(baseLog))

	appLog("server started on :8080")
	dbLog("connected to postgres")

	// 10. Partial application
	fmt.Println("\n── 10. Partial Application ──")
	add10 := partial2(add3, 10) // fix first arg as 10
	fmt.Println("  add3(10, 20, 30) =", add3(10, 20, 30))
	fmt.Println("  add10(20, 30)    =", add10(20, 30)) // same result

	fmt.Println("\n" + sep)
	fmt.Println("Key Takeaways:")
	fmt.Println("  • Higher-order functions abstract over BEHAVIOR, not just data")
	fmt.Println("  • filter/map/reduce are the core primitives of functional programming")
	fmt.Println("  • Named function types (type Predicate func(int) bool) improve readability")
	fmt.Println("  • Function composition creates new functions from existing ones")
	fmt.Println("  • Strategy pattern: pass a function instead of an interface implementation")
	fmt.Println("  • Adapter pattern: wrap a function to change its behavior/signature")
	fmt.Println("  • Go 1.18 generics enable type-safe filter/map/reduce — see module 09")
	fmt.Println(sep)
}
