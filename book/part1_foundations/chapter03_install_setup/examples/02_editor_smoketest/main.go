// FILE: book/part1_foundations/chapter03_install_setup/examples/02_editor_smoketest/main.go
// CHAPTER: 03 — Installing Go and Setting Up Your Environment
// TOPIC: A file your editor should react to, to confirm gopls is alive.
//
// Run (from the chapter folder):
//   go run ./examples/02_editor_smoketest
//
// Then OPEN this file in your editor. You should see:
//
//   - hover over fmt.Println: doc comment in tooltip
//   - hover over Greet: the doc comment we wrote
//   - the import block is exactly the imports we use (gopls organises them)
//   - on save, gofmt rewrites the file (try adding a stray space)
//
// If any of those fail, your editor is not talking to gopls. See Chapter 3
// section 8.6 to wire it up.

package main

import (
	"fmt"
	"strings"
)

// Greet returns a polite greeting addressed to name.
//
// Greet is exported precisely so we can hover over its name in an editor
// and confirm gopls renders this doc comment in the tooltip.
func Greet(name string) string {
	return fmt.Sprintf("hello, %s", strings.TrimSpace(name))
}

func main() {
	// Hover over fmt.Println in your editor. You should see its stdlib
	// doc comment (something starting with "Println formats…").
	fmt.Println(Greet("editor"))

	// Hover over Greet. You should see the doc comment we wrote above.
	fmt.Println(Greet("  gopls  "))
}
