package main

import (
	"fmt"
	"math"
)

/*
1> For
Go has only one looping construct, the for loop.

The basic for loop has three components separated by semicolons:

the init statement: executed before the first iteration
the condition expression: evaluated before every iteration
the post statement: executed at the end of every iteration
The init statement will often be a short variable declaration, and the variables declared there are visible only in the scope of the for statement.

The loop will stop iterating once the boolean condition evaluates to false.

Note: Unlike other languages like C, Java, or JavaScript there are no parentheses surrounding the three components of the for statement and the braces { } are always required.
*/

/*
2> For continued
The init and post statements are optional.
*/

/*
3> For is Go's "while"
At that point you can drop the semicolons: C's while is spelled for in Go.
*/

/*
4>
Forever
If you omit the loop condition it loops forever, so an infinite loop is compactly expressed.
*/

/*
5> If
Go's if statements are like its for loops;
the expression need not be surrounded by parentheses ( ) but the braces { } are required.
*/

func sqrt(x float64) string {
	if x < 0 {
		return sqrt(-x) + "i"
	}
	return fmt.Sprint(math.Sqrt(x))
}

/*
6> If with a short statement
Like for, the if statement can start with a short statement to execute before the condition.

Variables declared by the statement are only in scope until the end of the if.

(Try using v in the last return statement.)
*/

func pow(x, n, lim float64) float64 {
	if v := math.Pow(x, n); v < lim {
		return v
	}

	// return v // v is now out of scope
	return lim
}

/*
7> If and else
Variables declared inside an if short statement are also available inside any of the else blocks.

(Both calls to pow return their results before the call to fmt.Println in main begins.)
*/

func pow2(x, n, lim float64) float64 {
	if v := math.Pow(x, n); v < lim {
		return v
	} else {
		fmt.Printf("%g >= %g\n", v, lim)
	}
	// can't use v here, though
	return lim
}

// 8> EXERCISE
func Sqrt(x float64) float64 {
	z := 1.0

	for i := 0; i < 10; i++ {
		z -= (z*z - x) / (2 * z)
		fmt.Println("Iteration", i+1, ":", z)
	}

	return z
}

func Sqrt_V2(x float64) float64 {
	z := 1.0
	prev := 0.0

	for math.Abs(z-prev) > 1e-10 {
		prev = z
		z -= (z*z - x) / (2 * z)
	}
	return z
}

func main() {
	// 1>
	sum := 0
	for i := 0; i < 10; i++ {
		sum += i
	}
	fmt.Println("Sum value:", sum)

	// 2>
	var k int = 0
	for k <= 50 {
		k += 10
	}

	fmt.Println("Sum k:", k)

	// 3>
	m := 1
	for m < 1000 {
		m += m
	}
	fmt.Println(m)

	// 4>
	// for {
	// }

	// 5>
	fmt.Println(sqrt(2), sqrt(-4))

	// 6>
	fmt.Println(
		pow(3, 2, 10),
		pow(3, 3, 20),
	)

	// 7>
	fmt.Println(
		pow2(3, 2, 10),
		pow2(3, 3, 20),
	)

	// 8>
	fmt.Println("Result:", Sqrt(2))
	fmt.Println("math.Sqrt:", math.Sqrt(2))
	fmt.Println("Result Sqrt_V2:", Sqrt_V2(2))
}
