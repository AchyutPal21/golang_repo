// 05_recursion.go
//
// TOPIC: Recursion — Fundamentals, Stack Overflow Risk, Tail Recursion (Go Does NOT
//        Optimize It), When to Use Recursion vs Iteration, and Mutual Recursion
//
// WHAT IS RECURSION?
//   A function that calls itself (directly or indirectly). Every recursive solution
//   has two parts:
//     1. Base case:     the condition under which the function does NOT recurse
//     2. Recursive case: the smaller sub-problem the function delegates to itself
//
// Without a base case (or with an incorrect one), recursion runs forever
// and eventually causes a stack overflow (goroutine stack exhaustion in Go).
//
// GO'S STACK MODEL:
//   Unlike C/C++, Go goroutines start with a small stack (typically 2–8 KB) that
//   GROWS DYNAMICALLY as needed (up to a configurable limit, default 1 GB on 64-bit).
//   Each function call pushes a frame; each return pops one. Deep recursion means
//   many frames — you can exhaust memory before hitting a hard limit.
//   The default goroutine stack limit is set by GOMAXSTACK / runtime/debug.SetMaxStack.
//
// TAIL RECURSION — GO DOES NOT OPTIMIZE IT:
//   In some languages (Scheme, Haskell, Kotlin with tailrec, Scala), a compiler
//   can detect a "tail call" (a recursive call that is the LAST operation before
//   returning) and optimize it into a loop — no new stack frame is needed.
//   Go does NOT perform Tail Call Optimization (TCO).
//   A tail-recursive Go function still creates a new stack frame for each call.
//   This means for very large inputs, even a "tail recursive" Go function can
//   overflow the goroutine stack.
//   The idiomatic Go answer: convert recursion to iteration for large inputs.

package main

import (
	"fmt"
	"strings"
)

// ─── 1. FACTORIAL — CLASSIC EXAMPLE ──────────────────────────────────────────
//
// factorial(n) = n * factorial(n-1), base case: factorial(0) = 1
//
// IMPORTANT: factorial grows incredibly fast. factorial(20) already overflows int64.
// We use int here for clarity; in production, use math/big for large factorials.

func factorial(n int) int {
	// Base case: 0! = 1 and 1! = 1
	if n <= 1 {
		return 1
	}
	// Recursive case: n! = n * (n-1)!
	// Each call creates a new stack frame holding its own 'n'.
	return n * factorial(n-1)
}

// factorialIterative — the iterative equivalent.
// For factorial, iteration is always preferred in production Go code.
// Same result, no stack growth, more efficient.
func factorialIterative(n int) int {
	result := 1
	for i := 2; i <= n; i++ {
		result *= i
	}
	return result
}

// ─── 2. FIBONACCI — NAIVE VS MEMOIZED ────────────────────────────────────────
//
// Naive recursive Fibonacci is the canonical example of BAD recursion:
//   fib(5) = fib(4) + fib(3)
//   fib(4) = fib(3) + fib(2)    ← fib(3) computed TWICE
//   ...
// Time complexity: O(2^n) — exponential!
// For fib(50), this makes ~2^50 ≈ 10^15 calls. Practically unusable.

func fibNaive(n int) int {
	if n <= 1 {
		return n
	}
	return fibNaive(n-1) + fibNaive(n-2)
}

// fibMemo uses a cache to avoid redundant computation.
// Time complexity drops to O(n). This is dynamic programming via memoization.
// We pass the cache explicitly here (vs closure approach in 02_closures.go).
func fibMemo(n int, cache map[int]int) int {
	if n <= 1 {
		return n
	}
	if v, ok := cache[n]; ok {
		return v
	}
	result := fibMemo(n-1, cache) + fibMemo(n-2, cache)
	cache[n] = result
	return result
}

// fibIterative — O(n) time, O(1) space. The best solution for production.
// Recursion is elegant here but iteration is strictly better.
func fibIterative(n int) int {
	if n <= 1 {
		return n
	}
	a, b := 0, 1
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}
	return b
}

// ─── 3. SUM — DEMONSTRATING BASE CASE IMPORTANCE ─────────────────────────────

func sumSlice(nums []int) int {
	// Base case: empty slice sums to 0
	if len(nums) == 0 {
		return 0
	}
	// Recursive case: first element + sum of rest
	// 'nums[1:]' creates a new slice header (no copy) pointing to the remaining elements
	return nums[0] + sumSlice(nums[1:])
}

// ─── 4. TAIL RECURSION — GO DOES NOT OPTIMIZE ─────────────────────────────────
//
// A tail call is when the recursive call is the VERY LAST operation —
// no work is done with its return value in the current frame.
//
// factorialTailHelper is written in tail-recursive style:
//   - The accumulator carries the running result
//   - The recursive call is the last expression
//   - No multiplication AFTER the call (unlike naive factorial which does n * recurse())
//
// In TCO languages, this compiles to a loop. In Go, it STILL creates stack frames.
// We show this to understand WHY Go developers default to iteration.

func factorialTailHelper(n, acc int) int {
	if n <= 1 {
		return acc
	}
	// Tail call: the result of this call is returned directly, no post-processing
	return factorialTailHelper(n-1, n*acc)
}

func factorialTail(n int) int {
	return factorialTailHelper(n, 1)
}

// ─── 5. WHEN TO USE RECURSION — TREE TRAVERSAL ────────────────────────────────
//
// Recursion SHINES when the data structure itself is recursive (trees, graphs,
// nested structures). The code mirrors the structure, making it much clearer
// than the iterative equivalent (which would need an explicit stack).
//
// Rule of thumb:
//   Recursion: when the problem naturally decomposes into smaller same-shaped problems
//              OR when the data structure is inherently recursive (tree, nested JSON)
//   Iteration: when depth could be large (stack overflow risk), or for simple loops

// TreeNode represents a binary search tree node.
type TreeNode struct {
	Value       int
	Left, Right *TreeNode
}

// insert adds a value to the BST — recursive is natural here.
func insert(node *TreeNode, val int) *TreeNode {
	if node == nil {
		return &TreeNode{Value: val}
	}
	if val < node.Value {
		node.Left = insert(node.Left, val)
	} else if val > node.Value {
		node.Right = insert(node.Right, val)
	}
	return node
}

// inorder traversal: Left → Root → Right — naturally recursive.
// Iterative equivalent needs an explicit stack, making it more complex.
func inorder(node *TreeNode, result *[]int) {
	if node == nil {
		return // base case
	}
	inorder(node.Left, result)
	*result = append(*result, node.Value)
	inorder(node.Right, result)
}

// treeHeight — another naturally recursive tree function.
func treeHeight(node *TreeNode) int {
	if node == nil {
		return 0
	}
	leftH := treeHeight(node.Left)
	rightH := treeHeight(node.Right)
	if leftH > rightH {
		return leftH + 1
	}
	return rightH + 1
}

// ─── 6. RECURSIVE DIRECTORY-LIKE TRAVERSAL ────────────────────────────────────
//
// Flattening nested structures is another place where recursion is natural.

type Dir struct {
	Name     string
	Children []Dir
}

// collectPaths returns all leaf paths in a directory tree.
func collectPaths(dir Dir, prefix string) []string {
	path := prefix + dir.Name
	if len(dir.Children) == 0 {
		return []string{path} // base case: leaf directory
	}
	var paths []string
	for _, child := range dir.Children {
		// Recursive case: delegate to children
		paths = append(paths, collectPaths(child, path+"/")...)
	}
	return paths
}

// ─── 7. MUTUAL RECURSION ──────────────────────────────────────────────────────
//
// Mutual recursion: function A calls function B, which calls function A.
// Both functions must be declared in the same scope (or package).
// In Go this works naturally since all package-level functions are visible.
//
// Classic example: even/odd determination (silly but illustrative).
// Practical example: parsing a grammar where expressions can contain terms
// which can contain expressions (recursive descent parsers).

func isEven(n int) bool {
	if n < 0 {
		return isEven(-n) // handle negatives
	}
	if n == 0 {
		return true
	}
	return isOdd(n - 1) // delegates to isOdd
}

func isOdd(n int) bool {
	if n < 0 {
		return isOdd(-n)
	}
	if n == 0 {
		return false
	}
	return isEven(n - 1) // delegates back to isEven
}

// A more practical mutual recursion: evaluating simple nested expressions.
// expr → term ('+' term)*
// term → number | '(' expr ')'

// ─── 8. FLATTENING NESTED SLICES ──────────────────────────────────────────────
//
// Another classic recursive problem: flattening arbitrarily nested structure.
// In Go without generics (pre-1.18), we use []interface{} (any).
// With generics (1.18+), you'd define a Nested[T any] type.

func flatten(nested []any) []int {
	var result []int
	for _, item := range nested {
		switch v := item.(type) {
		case int:
			result = append(result, v) // base case: it's an int, collect it
		case []any:
			result = append(result, flatten(v)...) // recursive: flatten the sub-slice
		}
	}
	return result
}

// ─── MAIN ─────────────────────────────────────────────────────────────────────

func main() {
	sep := strings.Repeat("═", 55)
	fmt.Println(sep)
	fmt.Println("  RECURSION IN GO")
	fmt.Println(sep)

	// 1. Factorial
	fmt.Println("\n── 1. Factorial — Recursive vs Iterative ──")
	for _, n := range []int{0, 1, 5, 10, 12} {
		r := factorial(n)
		it := factorialIterative(n)
		match := "✓"
		if r != it {
			match = "✗ MISMATCH"
		}
		fmt.Printf("  factorial(%2d) = %10d  iterative = %10d  %s\n", n, r, it, match)
	}

	// 2. Fibonacci
	fmt.Println("\n── 2. Fibonacci — Naive vs Memoized vs Iterative ──")
	cache := make(map[int]int)
	for _, n := range []int{0, 1, 5, 10, 20, 35} {
		naive := "skip (too slow for n>30)"
		if n <= 30 {
			naive = fmt.Sprintf("%d", fibNaive(n))
		}
		memo := fibMemo(n, cache)
		iter := fibIterative(n)
		fmt.Printf("  fib(%2d): naive=%-6s  memo=%d  iter=%d\n", n, naive, memo, iter)
	}

	// 3. sumSlice
	fmt.Println("\n── 3. Recursive Sum of Slice ──")
	data := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	fmt.Printf("  sumSlice(%v) = %d\n", data, sumSlice(data))
	fmt.Println("  sumSlice([]) =", sumSlice([]int{}))

	// 4. Tail recursion (Go does NOT optimize)
	fmt.Println("\n── 4. Tail Recursion (No TCO in Go) ──")
	fmt.Println("  factorialTail(10) =", factorialTail(10))
	fmt.Println("  NOTE: tail-recursive Go still creates stack frames — use iteration for large n")

	// 5. Tree traversal
	fmt.Println("\n── 5. Binary Search Tree Traversal ──")
	var root *TreeNode
	for _, v := range []int{5, 3, 7, 1, 4, 6, 8, 2} {
		root = insert(root, v)
	}
	var sorted []int
	inorder(root, &sorted)
	fmt.Println("  BST inorder (sorted):", sorted)
	fmt.Println("  tree height:", treeHeight(root))

	// 6. Directory traversal
	fmt.Println("\n── 6. Recursive Directory Traversal ──")
	fs := Dir{Name: "root", Children: []Dir{
		{Name: "src", Children: []Dir{
			{Name: "main.go"},
			{Name: "utils.go"},
		}},
		{Name: "docs", Children: []Dir{
			{Name: "api.md"},
			{Name: "guide", Children: []Dir{
				{Name: "quick-start.md"},
			}},
		}},
		{Name: "go.mod"},
	}}
	for _, p := range collectPaths(fs, "") {
		fmt.Println(" ", p)
	}

	// 7. Mutual recursion
	fmt.Println("\n── 7. Mutual Recursion (isEven / isOdd) ──")
	for _, n := range []int{0, 1, 6, 7, -4, -5} {
		fmt.Printf("  isEven(%3d) = %-5v  isOdd(%3d) = %v\n", n, isEven(n), n, isOdd(n))
	}

	// 8. Flatten nested slice
	fmt.Println("\n── 8. Flatten Nested Slice ──")
	nested := []any{1, []any{2, 3, []any{4, 5}}, 6, []any{7}}
	fmt.Printf("  nested: %v\n", nested)
	fmt.Printf("  flat:   %v\n", flatten(nested))

	fmt.Println("\n" + sep)
	fmt.Println("Key Takeaways:")
	fmt.Println("  • Every recursion needs a base case — missing one = infinite loop/crash")
	fmt.Println("  • Go goroutines have dynamic stacks, but deep recursion can still OOM")
	fmt.Println("  • Go does NOT optimize tail calls — tail-recursive still uses stack frames")
	fmt.Println("  • Prefer iteration for large inputs (factorial, fibonacci, sum)")
	fmt.Println("  • Prefer recursion for recursive data structures (trees, nested data)")
	fmt.Println("  • Memoization transforms O(2^n) naive recursion to O(n)")
	fmt.Println("  • Mutual recursion works naturally in Go (package-level visibility)")
	fmt.Println(sep)
}
