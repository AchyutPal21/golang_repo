// EXERCISE 26.1 — Refactor an OOP shape hierarchy into Go idioms.
//
// The "Java version" below uses a base struct with virtual methods.
// Rewrite it so Shape is an interface, each concrete type stands alone,
// and Renderer accepts any Shape without knowing its type.
//
// Run (from the chapter folder):
//   go run ./exercises/01_shape_hierarchy

package main

import (
	"fmt"
	"math"
	"strings"
)

// ─── Interface-based design ───────────────────────────────────────────────────

type Shape interface {
	Area() float64
	Perimeter() float64
	Describe() string
}

// ─── Concrete types ───────────────────────────────────────────────────────────

type Circle struct{ Radius float64 }

func (c Circle) Area() float64      { return math.Pi * c.Radius * c.Radius }
func (c Circle) Perimeter() float64 { return 2 * math.Pi * c.Radius }
func (c Circle) Describe() string   { return fmt.Sprintf("Circle r=%.2f", c.Radius) }

type Rectangle struct{ W, H float64 }

func (r Rectangle) Area() float64      { return r.W * r.H }
func (r Rectangle) Perimeter() float64 { return 2 * (r.W + r.H) }
func (r Rectangle) Describe() string {
	return fmt.Sprintf("Rectangle %.2fx%.2f", r.W, r.H)
}

type RightTriangle struct{ Base, Height float64 }

func (t RightTriangle) Hypotenuse() float64 {
	return math.Sqrt(t.Base*t.Base + t.Height*t.Height)
}
func (t RightTriangle) Area() float64      { return 0.5 * t.Base * t.Height }
func (t RightTriangle) Perimeter() float64 { return t.Base + t.Height + t.Hypotenuse() }
func (t RightTriangle) Describe() string {
	return fmt.Sprintf("RightTriangle base=%.2f height=%.2f", t.Base, t.Height)
}

// ─── Renderer: accepts Shape — not a concrete type ───────────────────────────

type Renderer struct{ width int }

func (r Renderer) Render(s Shape) {
	line := strings.Repeat("─", r.width)
	fmt.Println(line)
	fmt.Printf("  %-35s\n", s.Describe())
	fmt.Printf("  area=%-10.2f perimeter=%.2f\n", s.Area(), s.Perimeter())
}

// ─── Functional operations ───────────────────────────────────────────────────

func totalArea(shapes []Shape) float64 {
	total := 0.0
	for _, s := range shapes {
		total += s.Area()
	}
	return total
}

func largest(shapes []Shape) Shape {
	if len(shapes) == 0 {
		return nil
	}
	best := shapes[0]
	for _, s := range shapes[1:] {
		if s.Area() > best.Area() {
			best = s
		}
	}
	return best
}

func main() {
	shapes := []Shape{
		Circle{Radius: 5},
		Rectangle{W: 8, H: 3},
		RightTriangle{Base: 3, Height: 4},
		Circle{Radius: 2},
	}

	r := Renderer{width: 45}
	for _, s := range shapes {
		r.Render(s)
	}
	fmt.Println(strings.Repeat("─", 45))

	fmt.Printf("\nTotal area:  %.2f\n", totalArea(shapes))
	fmt.Printf("Largest:     %s\n", largest(shapes).Describe())
}
