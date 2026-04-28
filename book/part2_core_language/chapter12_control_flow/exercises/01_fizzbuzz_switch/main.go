// EXERCISE 12.1 — FizzBuzz with a single tagless switch.
//
// Run (from the chapter folder):
//   go run ./exercises/01_fizzbuzz_switch

package main

import "fmt"

func main() {
	for n := 1; n <= 20; n++ {
		switch {
		case n%15 == 0:
			fmt.Println("FizzBuzz")
		case n%3 == 0:
			fmt.Println("Fizz")
		case n%5 == 0:
			fmt.Println("Buzz")
		default:
			fmt.Println(n)
		}
	}
}
