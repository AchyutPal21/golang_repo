// FILE: book/part3_designing_software/chapter26_oop_in_go/examples/01_classes_vs_go/main.go
// CHAPTER: 26 — OOP in Go
// TOPIC: How Go replaces classes, inheritance, and virtual dispatch.
//        Side-by-side: Java-style thinking vs Go idioms.
//
// Run (from the chapter folder):
//   go run ./examples/01_classes_vs_go

package main

import "fmt"

// ─────────────────────────────────────────────────────────────────────────────
// Java mental model:
//   abstract class Animal { abstract String speak(); }
//   class Dog extends Animal { String speak() { return "Woof"; } }
//   class Cat extends Animal { String speak() { return "Meow"; } }
//
// Go translation: replace the abstract class with an interface.
// There is no extends, no abstract, no class keyword.
// ─────────────────────────────────────────────────────────────────────────────

// Speaker is what Java calls an abstract method contract.
type Speaker interface {
	Speak() string
}

type Dog struct{ Name string }
type Cat struct{ Name string }
type Parrot struct {
	Name   string
	Phrase string
}

func (d Dog) Speak() string    { return "Woof!" }
func (c Cat) Speak() string    { return "Meow." }
func (p Parrot) Speak() string { return p.Phrase }

// makeNoise works on any Speaker — polymorphism without inheritance.
func makeNoise(s Speaker) {
	fmt.Println(s.Speak())
}

// ─────────────────────────────────────────────────────────────────────────────
// Java: private fields + getters/setters
// Go: unexported fields + methods on the same package
// ─────────────────────────────────────────────────────────────────────────────

type BankAccount struct {
	owner   string // unexported — enforces controlled access
	balance float64
}

func NewBankAccount(owner string, initial float64) *BankAccount {
	return &BankAccount{owner: owner, balance: initial}
}

func (a *BankAccount) Deposit(amount float64) { a.balance += amount }
func (a *BankAccount) Balance() float64        { return a.balance }
func (a *BankAccount) Owner() string           { return a.owner }

// ─────────────────────────────────────────────────────────────────────────────
// Java: static factory methods
// Go: NewXxx constructor functions — the canonical pattern
// ─────────────────────────────────────────────────────────────────────────────

type Config struct {
	host string
	port int
}

func NewConfig(host string, port int) (*Config, error) {
	if host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid port %d", port)
	}
	return &Config{host: host, port: port}, nil
}

func (c *Config) Addr() string { return fmt.Sprintf("%s:%d", c.host, c.port) }

// ─────────────────────────────────────────────────────────────────────────────
// Java: toString()
// Go: implement fmt.Stringer
// ─────────────────────────────────────────────────────────────────────────────

func (d Dog) String() string { return fmt.Sprintf("Dog(%s)", d.Name) }

// ─────────────────────────────────────────────────────────────────────────────
// Java: equals() / Comparable
// Go: implement == for comparable types, or a Less method
// ─────────────────────────────────────────────────────────────────────────────

type Point struct{ X, Y int }

// Points are comparable with == directly (no override needed).
// For sort ordering, implement a method:
func (p Point) Less(q Point) bool {
	if p.X != q.X {
		return p.X < q.X
	}
	return p.Y < q.Y
}

func main() {
	// Polymorphism via interface
	animals := []Speaker{
		Dog{Name: "Rex"},
		Cat{Name: "Whiskers"},
		Parrot{Name: "Polly", Phrase: "Pretty bird!"},
	}
	for _, a := range animals {
		makeNoise(a)
	}

	fmt.Println()

	// Encapsulation
	acc := NewBankAccount("Alice", 100)
	acc.Deposit(50)
	fmt.Printf("%s balance: %.2f\n", acc.Owner(), acc.Balance())

	fmt.Println()

	// Constructor with validation
	cfg, err := NewConfig("api.example.com", 8080)
	if err != nil {
		fmt.Println("error:", err)
	} else {
		fmt.Println("config:", cfg.Addr())
	}
	_, err = NewConfig("", 8080)
	fmt.Println("invalid config:", err)

	fmt.Println()

	// Stringer
	fmt.Println(Dog{Name: "Buddy"})

	// Comparable
	p1, p2 := Point{1, 2}, Point{1, 3}
	fmt.Println("p1==p2:", p1 == p2)
	fmt.Println("p1<p2:", p1.Less(p2))
}
