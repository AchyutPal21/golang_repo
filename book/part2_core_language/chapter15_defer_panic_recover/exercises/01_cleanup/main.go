// EXERCISE 15.1 — Resource cleanup chain.
//
// Implement openResource(name string) (*Resource, error) and a
// runPipeline function that opens three resources and ensures all
// are closed even if one step fails. Use defer for each resource.
//
// Implement safeParseInt(s string) (n int, err error) that converts
// a panic from a hypothetical parseInt(s) into a proper error.
//
// Run (from the chapter folder):
//   go run ./exercises/01_cleanup

package main

import (
	"errors"
	"fmt"
	"strconv"
)

type Resource struct {
	name   string
	closed bool
}

func (r *Resource) Use() error {
	if r.closed {
		return fmt.Errorf("%s: already closed", r.name)
	}
	fmt.Printf("[%s] in use\n", r.name)
	return nil
}

func (r *Resource) Close() {
	if !r.closed {
		r.closed = true
		fmt.Printf("[%s] closed\n", r.name)
	}
}

func openResource(name string) (*Resource, error) {
	if name == "bad" {
		return nil, fmt.Errorf("openResource: failed to open %q", name)
	}
	fmt.Printf("[%s] opened\n", name)
	return &Resource{name: name}, nil
}

// runPipeline opens A, B, C and uses them; closes all on any failure.
func runPipeline(names []string) error {
	resources := make([]*Resource, 0, len(names))

	// Register cleanup before opening so partial success is handled.
	defer func() {
		for i := len(resources) - 1; i >= 0; i-- {
			resources[i].Close()
		}
	}()

	for _, name := range names {
		r, err := openResource(name)
		if err != nil {
			return err
		}
		resources = append(resources, r)
	}

	for _, r := range resources {
		if err := r.Use(); err != nil {
			return err
		}
	}
	return nil
}

// panicParseInt panics instead of returning an error (bad design).
func panicParseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		panic(fmt.Sprintf("panicParseInt: invalid %q", s))
	}
	return n
}

// safeParseInt wraps panicParseInt and converts panics to errors.
func safeParseInt(s string) (n int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	return panicParseInt(s), nil
}

func main() {
	fmt.Println("=== successful pipeline ===")
	err := runPipeline([]string{"A", "B", "C"})
	fmt.Println("result:", err)

	fmt.Println()

	fmt.Println("=== failed pipeline (B fails to open) ===")
	err = runPipeline([]string{"A", "bad", "C"})
	fmt.Println("result:", err)

	fmt.Println()

	fmt.Println("=== safeParseInt ===")
	n, err := safeParseInt("42")
	fmt.Printf("safeParseInt(%q) = %d, %v\n", "42", n, err)

	n, err = safeParseInt("oops")
	fmt.Printf("safeParseInt(%q) = %d, %v\n", "oops", n, err)

	_ = errors.New // used by the exercise framework
}
