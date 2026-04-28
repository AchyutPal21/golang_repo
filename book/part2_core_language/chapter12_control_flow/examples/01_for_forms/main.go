// FILE: book/part2_core_language/chapter12_control_flow/examples/01_for_forms/main.go
// CHAPTER: 12 — Control Flow
// TOPIC: All four `for` forms plus range over every iterable kind.
//
// Run (from the chapter folder):
//   go run ./examples/01_for_forms

package main

import "fmt"

func main() {
	// 1. Counter form — C-style three-clause.
	fmt.Print("counter:  ")
	for i := 0; i < 5; i++ {
		fmt.Print(i, " ")
	}
	fmt.Println()

	// 2. Condition-only — replaces `while` from other languages.
	fmt.Print("while:    ")
	x := 0
	for x < 5 {
		fmt.Print(x, " ")
		x++
	}
	fmt.Println()

	// 3. Infinite — used with break or with a select inside.
	fmt.Print("infinite: ")
	count := 0
	for {
		if count == 5 {
			break
		}
		fmt.Print(count, " ")
		count++
	}
	fmt.Println()

	// 4. Range over a slice
	fmt.Print("range slice: ")
	for i, v := range []string{"a", "b", "c"} {
		fmt.Printf("[%d]=%q ", i, v)
	}
	fmt.Println()

	// 5. Range over a string — i is byte index, r is rune
	fmt.Println("range string (byte index, rune):")
	for i, r := range "héllo" {
		fmt.Printf("  i=%d r=%q (U+%04X)\n", i, r, r)
	}

	// 6. Range over a map — order is RANDOMIZED, on purpose.
	fmt.Println("range map (order randomized; rerun and watch):")
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	for k, v := range m {
		fmt.Printf("  %s=%d\n", k, v)
	}

	// 7. Range over a channel — receives until close.
	fmt.Print("range channel: ")
	ch := make(chan int, 3)
	ch <- 10
	ch <- 20
	ch <- 30
	close(ch)
	for v := range ch {
		fmt.Print(v, " ")
	}
	fmt.Println()

	// 8. Range over an integer (Go 1.22+) — iterates 0..N-1.
	fmt.Print("range int (1.22+): ")
	for i := range 5 {
		fmt.Print(i, " ")
	}
	fmt.Println()

	// 9. Range with discarded value.
	fmt.Print("range with _ for value: ")
	for _, v := range []int{10, 20, 30} {
		fmt.Print(v, " ")
	}
	fmt.Println()

	// 10. Range to count — bind nothing.
	fmt.Print("range to count (no bindings): ")
	hits := 0
	for range []int{1, 2, 3, 4, 5} {
		hits++
	}
	fmt.Println(hits)
}
