// FILE: book/part2_core_language/chapter25_reflection/examples/01_reflect_basics/main.go
// CHAPTER: 25 — Reflection: Programming the Type System
// TOPIC: reflect.Type, reflect.Value, Kind, struct tag inspection,
//        setting values, reflect.DeepEqual, cost summary.
//
// Run (from the chapter folder):
//   go run ./examples/01_reflect_basics

package main

import (
	"fmt"
	"reflect"
	"strconv"
)

// --- Type and Kind ---

func typeInfo(v any) {
	t := reflect.TypeOf(v)
	val := reflect.ValueOf(v)
	fmt.Printf("Type=%-20s Kind=%-10s Value=%v\n", t, t.Kind(), val)
}

// --- Struct introspection ---

type Config struct {
	Host    string `env:"APP_HOST" default:"localhost"`
	Port    int    `env:"APP_PORT" default:"8080"`
	Debug   bool   `env:"APP_DEBUG" default:"false"`
	ignored string // unexported — not visible via reflect
}

func printStructFields(v any) {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	fmt.Printf("Struct: %s\n", t.Name())
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		fmt.Printf("  %-10s type=%-8s env=%q default=%q\n",
			f.Name, f.Type, f.Tag.Get("env"), f.Tag.Get("default"))
	}
}

// --- Setting values via reflection ---

// fillDefaults populates zero-value fields from their `default` struct tag.
func fillDefaults(v any) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		panic("fillDefaults requires a pointer to struct")
	}
	rv = rv.Elem()
	rt := rv.Type()

	for i := range rt.NumField() {
		f := rt.Field(i)
		fv := rv.Field(i)
		if !f.IsExported() || !fv.IsZero() {
			continue
		}
		def := f.Tag.Get("default")
		if def == "" {
			continue
		}
		switch f.Type.Kind() {
		case reflect.String:
			fv.SetString(def)
		case reflect.Int, reflect.Int64:
			n, _ := strconv.ParseInt(def, 10, 64)
			fv.SetInt(n)
		case reflect.Bool:
			b, _ := strconv.ParseBool(def)
			fv.SetBool(b)
		}
	}
}

// --- reflect.DeepEqual ---

func deepEqualDemo() {
	a := []int{1, 2, 3}
	b := []int{1, 2, 3}
	c := []int{1, 2, 4}

	fmt.Println("a==b (==):", false)               // slices not comparable with ==
	fmt.Println("DeepEqual(a,b):", reflect.DeepEqual(a, b)) // true
	fmt.Println("DeepEqual(a,c):", reflect.DeepEqual(a, c)) // false

	type Inner struct{ X int }
	m1 := map[string]Inner{"k": {1}}
	m2 := map[string]Inner{"k": {1}}
	fmt.Println("DeepEqual(maps):", reflect.DeepEqual(m1, m2)) // true
}

func main() {
	// --- type and kind ---
	fmt.Println("=== TypeOf / ValueOf ===")
	typeInfo(42)
	typeInfo(3.14)
	typeInfo("hello")
	typeInfo([]int{1, 2})
	typeInfo(Config{})

	fmt.Println()

	// --- struct introspection ---
	fmt.Println("=== struct fields ===")
	printStructFields(Config{})

	fmt.Println()

	// --- fillDefaults ---
	fmt.Println("=== fillDefaults ===")
	cfg := &Config{}
	fillDefaults(cfg)
	fmt.Printf("Host=%s Port=%d Debug=%v\n", cfg.Host, cfg.Port, cfg.Debug)

	// Partial override: only Debug is zero
	cfg2 := &Config{Host: "prod.example.com", Port: 9090}
	fillDefaults(cfg2)
	fmt.Printf("Host=%s Port=%d Debug=%v\n", cfg2.Host, cfg2.Port, cfg2.Debug)

	fmt.Println()

	// --- DeepEqual ---
	fmt.Println("=== DeepEqual ===")
	deepEqualDemo()

	fmt.Println()

	// --- cost summary ---
	fmt.Println("=== reflect cost ===")
	fmt.Println("reflect.TypeOf / ValueOf: single pointer chase (cheap)")
	fmt.Println("reflect.Value.Field:      index into struct (cheap)")
	fmt.Println("Method calls via reflect: ~10× slower than direct call")
	fmt.Println("reflect.DeepEqual:        traverses entire tree (use sparingly)")
	fmt.Println("")
	fmt.Println("Use reflection for: serialisation, ORMs, DI frameworks, testing.")
	fmt.Println("Avoid for: hot paths, simple type dispatch (use type switch).")
}
