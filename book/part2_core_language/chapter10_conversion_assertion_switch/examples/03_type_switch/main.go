// FILE: book/part2_core_language/chapter10_conversion_assertion_switch/examples/03_type_switch/main.go
// CHAPTER: 10 — Type Conversion, Assertion, Switch
// TOPIC: Type switch — multi-way dispatch on an interface's dynamic type.
//
// Run (from the chapter folder):
//   go run ./examples/03_type_switch

package main

import "fmt"

// Node types of a tiny arithmetic AST. Each is a distinct struct that
// implicitly satisfies the empty interface (any).
type (
	num    struct{ v float64 }
	addOp  struct{ l, r any }
	subOp  struct{ l, r any }
	mulOp  struct{ l, r any }
	divOp  struct{ l, r any }
)

// eval walks an AST and returns its value. The dispatch uses a type
// switch — readable, fast (one itab compare per case), no reflection.
func eval(node any) (float64, error) {
	switch n := node.(type) {
	case num:
		return n.v, nil
	case addOp:
		l, err := eval(n.l)
		if err != nil {
			return 0, err
		}
		r, err := eval(n.r)
		if err != nil {
			return 0, err
		}
		return l + r, nil
	case subOp:
		l, _ := eval(n.l)
		r, _ := eval(n.r)
		return l - r, nil
	case mulOp:
		l, _ := eval(n.l)
		r, _ := eval(n.r)
		return l * r, nil
	case divOp:
		l, _ := eval(n.l)
		r, _ := eval(n.r)
		if r == 0 {
			return 0, fmt.Errorf("divide by zero")
		}
		return l / r, nil
	case nil:
		return 0, fmt.Errorf("nil node")
	default:
		// %T prints the dynamic type; useful for "I don't recognize this".
		return 0, fmt.Errorf("unknown node type: %T", n)
	}
}

func main() {
	// ((1 + 2) * (5 - 3)) / 2 = 3
	expr := divOp{
		l: mulOp{
			l: addOp{l: num{1}, r: num{2}},
			r: subOp{l: num{5}, r: num{3}},
		},
		r: num{2},
	}

	v, err := eval(expr)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("result = %v\n", v)

	// Dispatch on multiple types in one case (note: `n` keeps the
	// interface type when multiple types share a case).
	values := []any{42, "hello", 3.14, true, nil}
	for _, v := range values {
		switch v.(type) {
		case int, float64:
			fmt.Printf("  numeric:  %v\n", v)
		case string:
			fmt.Printf("  string:   %v\n", v)
		case nil:
			fmt.Printf("  nil interface\n")
		default:
			fmt.Printf("  other:    %v (%T)\n", v, v)
		}
	}
}
