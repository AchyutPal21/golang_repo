package main

import (
	"fmt"
)

// type Vertex struct {
// 	X int
// 	Y int
// }

type Vertex struct {
	Lat, Long float64
}

var m map[string]Vertex

func main() {

	/*

			1> Pointers
		Go has pointers. A pointer holds the memory address of a value.

		The type *T is a pointer to a T value. Its zero value is nil.

		var p *int
		The & operator generates a pointer to its operand.

		i := 42
		p = &i
		The * operator denotes the pointer's underlying value.

		fmt.Println(*p) // read i through the pointer p
		*p = 21         // set i through the pointer p
		This is known as "dereferencing" or "indirecting".

		Unlike C, Go has no pointer arithmetic.

	*/

	// var i int = 8909
	// var ptr1 *int

	// ptr1 = &i

	// fmt.Println("ptr1 value: ", ptr1)
	// fmt.Println("value at ptr1: ", *ptr1)

	// *ptr1 = 1000

	// fmt.Println("updated ptr1 value: ", ptr1)
	// fmt.Println("updated value at ptr1: ", *ptr1)

	/*

			2> Structs
		A struct is a collection of fields.

	*/

	// fmt.Println(Vertex{1, 2})
	// v := Vertex{10, 20}
	// v.X = 4
	// fmt.Println(v.X, v.Y)

	/*

		4> Pointers to structs
			Struct fields can be accessed through a struct pointer.

			To access the field X of a struct when we have the struct pointer p we could write (*p).X.
			However, that notation is cumbersome, so the language permits us instead to write just p.X,
			without the explicit dereference.

		=> NOTE
			In Go, structs are allocated on either stack or heap depending on escape analysis.
			If the variable does not escape the function, it is allocated on the stack.
			If it escapes (e.g., returned as pointer, captured by goroutine, stored globally), it is allocated on the heap.

	*/

	// v := Vertex{1, 2}
	// p := &v
	// p.X = 1e9
	// fmt.Println(v)

	/*

		5> Struct Literals
		A struct literal denotes a newly allocated struct value by listing the values of its fields.
		You can list just a subset of fields by using the Name: syntax. (And the order of named fields is irrelevant.)
		The special prefix & returns a pointer to the struct value.

	*/

	// var (
	// 	v1 = Vertex{1, 2}  // has type Vertex
	// 	v2 = Vertex{X: 1}  // Y:0 is implicit
	// 	v3 = Vertex{}      // X:0 and Y:0
	// 	p  = &Vertex{1, 2} // has type *Vertex
	// )

	// fmt.Println(p, v1, v2, v3)

	/*
			6> Arrays
		The type [n]T is an array of n values of type T.

		The expression
			var a [10]int
		declares a variable a as an array of ten integers.

		An array's length is part of its type, so arrays cannot be resized. This seems limiting, but don't worry;
		Go provides a convenient way of working with arrays.
	*/

	// var a [2]string
	// a[0] = "Hello"
	// a[1] = "World"
	// fmt.Println(a[0], a[1])
	// fmt.Println(a)

	// primes := [6]int{2, 3, 5, 7, 11, 13}
	// for i := range len(primes) {
	// 	fmt.Println(primes[i])
	// }

	// var s []int = primes[1:4]
	// fmt.Println(s)

	/*

			8> Slices are like references to arrays
		A slice does not store any data, it just describes a section of an underlying array.
		Changing the elements of a slice modifies the corresponding elements of its underlying array.
		Other slices that share the same underlying array will see those changes.
	*/
	// names := [4]string{
	// 	"John",
	// 	"Paul",
	// 	"George",
	// 	"Ringo",
	// }
	// fmt.Println(names)

	// a := names[0:2]
	// b := names[1:3]
	// fmt.Println(a, b)

	// b[0] = "XXX"
	// fmt.Println(a, b)
	// fmt.Println(names)

	/*
		9> Slice literals
		A slice literal is like an array literal without the length.

		This is an array literal:

		[3]bool{true, true, false}
		And this creates the same array as above, then builds a slice that references it:

		[]bool{true, true, false}

	*/

	// q := []int{2, 3, 5, 7, 11, 13}
	// fmt.Println(q)

	// r := []bool{true, false, true, true, false, true}
	// fmt.Println(r)

	// s := []struct {
	// 	i int
	// 	b bool
	// }{
	// 	{2, true},
	// 	{3, false},
	// 	{5, true},
	// 	{7, true},
	// 	{11, false},
	// 	{13, true},
	// }
	// // len(s) is length elements in the array
	// // cap(s) capacity of the array
	// fmt.Println(len(s), cap(s), s)

	/*
			13> Creating a slice with make
		Slices can be created with the built-in make function; this is how you create dynamically-sized arrays.

		The make function allocates a zeroed array and returns a slice that refers to that array:

		a := make([]int, 5)  // len(a)=5
		To specify a capacity, pass a third argument to make:

		b := make([]int, 0, 5) // len(b)=0, cap(b)=5

		b = b[:cap(b)] // len(b)=5, cap(b)=5
		b = b[1:]      // len(b)=4, cap(b)=4

	*/
	// a := make([]int, 5)
	// printSlice("a", a)

	// b := make([]int, 0, 5)
	// printSlice("b", b)

	// c := b[:2]
	// printSlice("c", c)

	// d := c[2:5]
	// printSlice("d", d)

	// // Create a tic-tac-toe board.
	// board := [][]string{
	// 	[]string{"_", "_", "_"},
	// 	[]string{"_", "_", "_"},
	// 	[]string{"_", "_", "_"},
	// }

	// // The players take turns.
	// board[0][0] = "X"
	// board[2][2] = "O"
	// board[1][2] = "X"
	// board[1][0] = "O"
	// board[0][2] = "X"

	// for i := 0; i < len(board); i++ {
	// 	fmt.Printf("%s\n", strings.Join(board[i], " "))
	// }

	// var s []int
	// printSlice(s)

	// // append works on nil slices.
	// s = append(s, 0)
	// printSlice(s)

	// // The slice grows as needed.
	// s = append(s, 1)
	// printSlice(s)

	// // We can add more than one element at a time.
	// s = append(s, 2, 3, 4)
	// printSlice(s)

	// var pow []int = []int{1, 2, 4, 8, 16, 32, 64, 128}
	// for i, v := range pow {
	// 	fmt.Printf("2**%d = %d\n", i, v)
	// }

	/*
		19> Maps

	*/

	// m = make(map[string]Vertex)
	// m["Bell Labs"] = Vertex{
	// 	40.68433, -74.39967,
	// }
	// fmt.Println(m["Bell Labs"])

	// var temp_map = map[string]Vertex{
	// 	"Bell Labs": Vertex{
	// 		40.68433, -74.39967,
	// 	},
	// 	"Google": Vertex{
	// 		37.42202, -122.08408,
	// 	},
	// }

	// fmt.Println(temp_map)

	// mutating_map := make(map[string]int)

	// mutating_map["Answer"] = 42
	// fmt.Println("The value:", mutating_map["Answer"])

	// mutating_map["Answer"] = 48
	// fmt.Println("The value:", mutating_map["Answer"])

	// delete(mutating_map, "Answer")
	// fmt.Println("The value:", mutating_map["Answer"])

	// v, ok := mutating_map["Answer"]
	// fmt.Println("The value:", v, "Present?", ok)

	/*
			24> Function values
		Functions are values too. They can be passed around just like other values.

		Function values may be used as function arguments and return values.
	*/

	// hypot := func(x, y float64) float64 {
	// 	return math.Sqrt(x*x + y*y)
	// }
	// fmt.Println(hypot(5, 12))

	// fmt.Println(compute(hypot))
	// fmt.Println(compute(math.Pow))

	/*
			25> Function closures
		Go functions may be closures. A closure is a function value that references
		variables from outside its body. The function may access and assign to the
		referenced variables; in this sense the function is "bound" to the variables.

		For example, the adder function returns a closure. Each closure is bound to its own sum variable.
	*/

	pos, neg := adder(), adder()
	for i := 0; i < 10; i++ {
		fmt.Println(
			pos(i),
			neg(-2*i),
		)
	}

	f := fibonacci()
	for i := 0; i < 10; i++ {
		fmt.Println(f())
	}

}

// func printSlice(s string, x []int) {
// 	fmt.Printf("%s len=%d cap=%d %v\n",
// 		s, len(x), cap(x), x)
// }

// func printSlice(s []int) {
// 	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
// }

// func compute(fn func(float64, float64) float64) float64 {
// 	return fn(3, 4)
// }

func adder() func(int) int {
	sum := 0
	return func(x int) int {
		sum += x
		return sum
	}
}

func fibonacci() func() int {
	a, b := 0, 1

	return func() int {
		result := a
		a, b = b, a+b
		return result
	}
}
