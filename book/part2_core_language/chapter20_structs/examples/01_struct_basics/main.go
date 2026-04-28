// FILE: book/part2_core_language/chapter20_structs/examples/01_struct_basics/main.go
// CHAPTER: 20 — Structs and Composite Literals
// TOPIC: Declaration, composite literals, field tags, anonymous structs,
//        struct comparison, JSON marshalling, memory layout.
//
// Run (from the chapter folder):
//   go run ./examples/01_struct_basics

package main

import (
	"encoding/json"
	"fmt"
	"unsafe"
)

// --- Basic struct ---

type Point struct {
	X, Y float64
}

// --- Struct with tags ---

// User demonstrates JSON field tags: custom names, omitempty, and "-" to skip.
type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email,omitempty"` // omitted when empty
	password  string // unexported — not marshalled regardless
	CreatedAt string `json:"created_at"`
	Internal  string `json:"-"` // always omitted from JSON
}

// --- Nested struct ---

type Address struct {
	Street string
	City   string
	Zip    string
}

type Employee struct {
	Name    string
	Age     int
	Address Address // embedded by value — a copy, not a pointer
}

// --- Anonymous struct ---

func anonymousStructDemo() {
	// Anonymous structs are useful for one-off JSON payloads and test fixtures.
	req := struct {
		Method string
		Path   string
		Body   []byte
	}{
		Method: "POST",
		Path:   "/api/users",
		Body:   []byte(`{"name":"Alice"}`),
	}
	fmt.Printf("request: %s %s\n", req.Method, req.Path)
}

// --- Memory layout ---

type Padded struct {
	A bool    // 1 byte + 7 pad
	B int64   // 8 bytes
	C bool    // 1 byte + 3 pad
	D int32   // 4 bytes
}

type Packed struct {
	B int64   // 8 bytes — largest first
	D int32   // 4 bytes
	A bool    // 1 byte
	C bool    // 1 byte + 2 pad
}

func main() {
	// --- construction ---
	p1 := Point{1.0, 2.0}               // positional — fragile, avoid for > 2 fields
	p2 := Point{X: 3.0, Y: 4.0}         // keyed — preferred
	var p3 Point                          // zero value: {0 0}
	fmt.Println(p1, p2, p3)

	// --- field access ---
	p2.X = 10
	fmt.Println("after p2.X=10:", p2)

	fmt.Println()

	// --- struct comparison ---
	a := Point{1, 2}
	b := Point{1, 2}
	c := Point{1, 3}
	fmt.Println("a==b:", a == b) // true
	fmt.Println("a==c:", a == c) // false

	fmt.Println()

	// --- nested ---
	emp := Employee{
		Name: "Alice",
		Age:  30,
		Address: Address{
			Street: "123 Main St",
			City:   "Springfield",
			Zip:    "12345",
		},
	}
	fmt.Println("city:", emp.Address.City)

	fmt.Println()

	// --- JSON marshalling ---
	u := User{
		ID:       1,
		Name:     "Alice",
		password: "secret", // will not appear in JSON
		Internal: "skip",   // will not appear in JSON
	}
	data, _ := json.Marshal(u)
	fmt.Println("json (no email — omitempty):", string(data))

	u.Email = "alice@example.com"
	data, _ = json.Marshal(u)
	fmt.Println("json (with email):          ", string(data))

	// Unmarshal
	var u2 User
	_ = json.Unmarshal([]byte(`{"id":2,"name":"Bob","email":"bob@x.com"}`), &u2)
	fmt.Printf("unmarshalled: %+v\n", u2)

	fmt.Println()

	// --- anonymous struct ---
	anonymousStructDemo()

	fmt.Println()

	// --- memory layout ---
	fmt.Printf("Padded size: %d  align: %d\n", unsafe.Sizeof(Padded{}), unsafe.Alignof(Padded{}))
	fmt.Printf("Packed size: %d  align: %d\n", unsafe.Sizeof(Packed{}), unsafe.Alignof(Packed{}))
	fmt.Printf("Saving %d bytes by reordering fields\n",
		unsafe.Sizeof(Padded{})-unsafe.Sizeof(Packed{}))
}
