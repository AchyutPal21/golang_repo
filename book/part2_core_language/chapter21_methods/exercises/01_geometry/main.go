// EXERCISE 21.1 — Geometry shapes with a common interface.
//
// Implement Circle, Rectangle, and Triangle types each satisfying
// the Shape interface with Area(), Perimeter(), and String().
// Then write a TotalArea(shapes []Shape) float64 function.
//
// Run (from the chapter folder):
//   go run ./exercises/01_geometry

package main

import (
	"fmt"
	"math"
)

type Shape interface {
	Area() float64
	Perimeter() float64
	String() string
}

// Circle

type Circle struct{ Radius float64 }

func (c Circle) Area() float64      { return math.Pi * c.Radius * c.Radius }
func (c Circle) Perimeter() float64 { return 2 * math.Pi * c.Radius }
func (c Circle) String() string     { return fmt.Sprintf("Circle(r=%.2f)", c.Radius) }

// Rectangle

type Rectangle struct{ Width, Height float64 }

func (r Rectangle) Area() float64      { return r.Width * r.Height }
func (r Rectangle) Perimeter() float64 { return 2 * (r.Width + r.Height) }
func (r Rectangle) String() string {
	return fmt.Sprintf("Rectangle(%.2fx%.2f)", r.Width, r.Height)
}

// Triangle (sides a, b, c — Heron's formula)

type Triangle struct{ A, B, C float64 }

func (t Triangle) Perimeter() float64 { return t.A + t.B + t.C }
func (t Triangle) Area() float64 {
	s := t.Perimeter() / 2
	return math.Sqrt(s * (s - t.A) * (s - t.B) * (s - t.C))
}
func (t Triangle) String() string {
	return fmt.Sprintf("Triangle(%.2f,%.2f,%.2f)", t.A, t.B, t.C)
}

func TotalArea(shapes []Shape) float64 {
	total := 0.0
	for _, s := range shapes {
		total += s.Area()
	}
	return total
}

func printShape(s Shape) {
	fmt.Printf("%-35s area=%-8.2f perim=%.2f\n",
		s.String(), s.Area(), s.Perimeter())
}

func main() {
	shapes := []Shape{
		Circle{Radius: 5},
		Rectangle{Width: 4, Height: 6},
		Triangle{A: 3, B: 4, C: 5},
		Circle{Radius: 1},
	}

	for _, s := range shapes {
		printShape(s)
	}

	fmt.Printf("\nTotal area: %.2f\n", TotalArea(shapes))
}
