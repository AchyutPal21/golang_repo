package main

/*

1> Packages
Every Go program is made up of packages.

Programs start running in package main.

This program is using the packages with import paths "fmt" and "math/rand".

By convention, the package name is the same as the last element of the import path.
For instance, the "math/rand" package comprises files that begin with the statement package rand.
--------------------------------------------------------------------------------------------------
2> Imports
This code groups the imports into a parenthesized, "factored" import statement.

You can also write multiple import statements, like:

import "fmt"
import "math"
But it is good style to use the factored import statement.
--------------------------------------------------------------------------------------------------
3> Exported names
In Go, a name is exported if it begins with a capital letter.
For example, Pizza is an exported name, as is Pi, which is exported from the math package.
pizza and pi do not start with a capital letter, so they are not exported.

When importing a package, you can refer only to its exported names.
Any "unexported" names are not accessible from outside the package.

Run the code. Notice the error message.

To fix the error, rename math.pi to math.Pi and try it again.
--------------------------------------------------------------------------------------------------



*/

import (
	"fmt"
	"math"
	"math/cmplx"
	"math/rand"
)

// --------------------------------------------------------------------------------------------------

// 4>
func add(x int, y int) int {
	return x + y
}

// --------------------------------------------------------------------------------------------------
// 5>
// Short hand is you can remove the int multiple of times
func sum(x, y, z int) int {
	return x + y + z
}

// --------------------------------------------------------------------------------------------------
// 6>
// multiple return from function
// Multiple results
// A function can return any number of results.
// The swap function returns two strings.

func swap(x, y string) (string, string) {
	return y, x
}

// --------------------------------------------------------------------------------------------------
// 7> Named return values
// named returned value,
// we are declaring the value we will return from the function
// (x, y int) here x and y are the variables
func split(sum int) (x, y int) {
	x = sum * 4 / 9
	y = sum - x
	return
}

// --------------------------------------------------------------------------------------------------

// 8> Variables
// The var statement declares a list of variables; as in function argument lists, the type is last.
// A var statement can be at package or function level. We see both in this example.

var c, python, java bool

// --------------------------------------------------------------------------------------------------

// 9> Variables with initializers
// A var declaration can include initializers, one per variable.
// If an initializer is present, the type can be omitted; the variable will take the type of the initializer.

var ii, jj int = 1, 2

// --------------------------------------------------------------------------------------------------

// 10> Short variable declarations
// Inside a function, the := short assignment statement can be used in place of a var declaration with implicit type.
// Outside a function, every statement begins with a keyword (var, func, and so on) and so the := construct is not available.

// user_name := "Achyut Pal" // ERROR
// --------------------------------------------------------------------------------------------------

// 11> Basic types
/*
bool
string
int  int8  int16  int32  int64
uint uint8 uint16 uint32 uint64 uintptr
byte // alias for uint8
rune // alias for int32
	// represents a Unicode code point
float32 float64
complex64 complex128

The example shows variables of several types,
and also that variable declarations may be
"factored" into blocks, as with import statements.

The int, uint, and uintptr types are usually
32 bits wide on 32-bit systems and 64 bits
wide on 64-bit systems. When you need an integer
value you should use int unless you have a specific
reason to use a sized or unsigned integer type.

*/

var (
	ToBe   bool       = false
	MaxInt uint64     = 1<<64 - 1
	zz     complex128 = cmplx.Sqrt(-5 + 12i)
)

// --------------------------------------------------------------------------------------------------

// 12> Zero values
// Variables declared without an explicit initial value are given their zero value.
// The zero value is:
// 0 for numeric types,
// false for the boolean type, and
// "" (the empty string) for strings.
// --------------------------------------------------------------------------------------------------

// 13> Type conversions
// The expression T(v) converts the value v to the type T.

// Some numeric conversions:

// var i int = 42
// var f float64 = float64(i)
// var u uint = uint(f)
// Or, put more simply:

// i := 42
// f := float64(i)
// u := uint(f)
// Unlike in C, in Go assignment between items of different type requires an explicit conversion.
// Try removing the float64 or uint conversions in the example and see what happens.
// --------------------------------------------------------------------------------------------------

/*
14> Type inference
When declaring a variable without specifying an
explicit type (either by using the := syntax or var = expression syntax),
the variable's type is inferred from the value on the right hand side.

When the right hand side of the declaration is typed,
the new variable is of that same type:

var i int
j := i // j is an int
But when the right hand side contains an untyped numeric constant,
the new variable may be an int, float64, or complex128 depending on the precision of the constant:

i := 42           // int
f := 3.142        // float64
g := 0.867 + 0.5i // complex128
Try changing the initial value of v in the example code and observe how its type is affected.
*/
// --------------------------------------------------------------------------------------------------

// 15> Constants
// Constants are declared like variables, but with the const keyword.

// Constants can be character, string, boolean, or numeric values.

// Constants cannot be declared using the := syntax.
const Pi = 3.14

// --------------------------------------------------------------------------------------------------
// 16> Numeric Constants
// Numeric constants are high-precision values.

// An untyped constant takes the type needed by its context.

// Try printing needInt(Big) too.

// (An int can store at maximum a 64-bit integer, and sometimes less.)
const (
	// Create a huge number by shifting a 1 bit left 100 places.
	// In other words, the binary number that is 1 followed by 100 zeroes.
	Big = 1 << 100
	// Shift it right again 99 places, so we end up with 1<<1, or 2.
	Small = Big >> 99
)

func needInt(x int) int { return x*10 + 1 }
func needFloat(x float64) float64 {
	return x * 0.1
}

// --------------------------------------------------------------------------------------------------
// --------------------------------------------------------------------------------------------------

func main() {
	// 1>
	fmt.Println("My favorite number is", rand.Intn(10))
	// -------------------------------------------------
	// 2>
	fmt.Printf("Now you have %g problems.\n", math.Sqrt(7))
	// -------------------------------------------------
	// 3>
	fmt.Println("Value of PI:", math.Pi)
	// -------------------------------------------------
	// 4>
	fmt.Println("Sum of 34, 34:", add(34, 34))
	// -------------------------------------------------
	// 5>
	fmt.Println("Sum of 34, 34, 34:", sum(34, 34, 34))
	// -------------------------------------------------
	// 6>
	a, b := swap("hello", "world")
	fmt.Println(a, b)
	// -------------------------------------------------
	// 7>
	fmt.Println(split(17))
	// -------------------------------------------------
	// 8>
	var i int
	fmt.Println(i, c, python, java)
	// -------------------------------------------------
	// 9>
	var c_program, python_prg, java_prj = true, false, "no!"
	fmt.Println(ii, jj, c_program, python_prg, java_prj)
	// -------------------------------------------------
	// 10>
	var i_o, j_o int = 1, 2
	k_o := 3
	c_o, python_o, java_o := true, false, "no!"

	fmt.Println(i_o, j_o, k_o, c_o, python_o, java_o)
	// -------------------------------------------------
	// 11>
	fmt.Printf("Type: %T Value: %v\n", ToBe, ToBe)
	fmt.Printf("Type: %T Value: %v\n", MaxInt, MaxInt)
	fmt.Printf("Type: %T Value: %v\n", zz, zz)
	// -------------------------------------------------
	// 12>
	var ia int
	var fa float64
	var ba bool
	var sa string
	fmt.Printf("%v %v %v %q\n", ia, fa, ba, sa)
	// -------------------------------------------------

	// 13>
	var xb, yb int = 3, 4
	var fb float64 = float64(xb*xb + yb*yb)
	var zb uint = uint(fb)
	fmt.Println(xb, yb, fb, zb)
	// -------------------------------------------------
	// 14>
	vb := 73 // change me!
	fmt.Printf("vb is of type %T\n", vb)
	// -------------------------------------------------
	// 15>
	const World string = "WORLD!!!"
	fmt.Println("Hello", World)
	fmt.Println("Happy", Pi, "Day")

	const Truth = true
	fmt.Println("Go rules?", Truth)
	// -------------------------------------------------
	// 16>
	fmt.Println(needInt(Small))
	fmt.Println(needFloat(Small))
	fmt.Println(needFloat(Big))

	// -------------------------------------------------

}
