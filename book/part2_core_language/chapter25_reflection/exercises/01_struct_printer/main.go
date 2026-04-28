// EXERCISE 25.1 — Generic struct pretty-printer using reflection.
//
// Implement PrintStruct(v any) that prints each exported field of a struct
// with its name, type, and value — similar to fmt.Printf("%+v") but one
// field per line and handling nested structs recursively.
//
// Run (from the chapter folder):
//   go run ./exercises/01_struct_printer

package main

import (
	"fmt"
	"reflect"
	"strings"
)

func PrintStruct(v any) {
	printStructIndent(reflect.ValueOf(v), 0)
}

func printStructIndent(rv reflect.Value, depth int) {
	indent := strings.Repeat("  ", depth)

	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			fmt.Println(indent + "<nil>")
			return
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		fmt.Printf("%s%v\n", indent, rv.Interface())
		return
	}

	rt := rv.Type()
	fmt.Printf("%s%s{\n", indent, rt.Name())
	for i := range rt.NumField() {
		f := rt.Field(i)
		fv := rv.Field(i)
		if !f.IsExported() {
			continue
		}
		if fv.Kind() == reflect.Struct || (fv.Kind() == reflect.Ptr && !fv.IsNil()) {
			fmt.Printf("%s  %s (%s):\n", indent, f.Name, f.Type)
			printStructIndent(fv, depth+2)
		} else {
			fmt.Printf("%s  %-12s %-12s = %v\n", indent, f.Name, f.Type, fv.Interface())
		}
	}
	fmt.Printf("%s}\n", indent)
}

type Address struct {
	Street string
	City   string
	Zip    string
}

type Person struct {
	Name    string
	Age     int
	Email   string
	Active  bool
	Address Address
	Tags    []string
}

func main() {
	p := Person{
		Name:   "Alice",
		Age:    30,
		Email:  "alice@example.com",
		Active: true,
		Address: Address{
			Street: "123 Main St",
			City:   "Springfield",
			Zip:    "12345",
		},
		Tags: []string{"admin", "user"},
	}

	PrintStruct(p)
	fmt.Println()
	PrintStruct(&p)
}
