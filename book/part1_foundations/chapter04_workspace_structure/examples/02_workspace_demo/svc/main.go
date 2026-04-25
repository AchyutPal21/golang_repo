// FILE: examples/02_workspace_demo/svc/main.go
// Run from examples/02_workspace_demo (NOT from the chapter root):
//   cd examples/02_workspace_demo
//   go run ./svc
//
// The go.work in this directory binds the local lib/ module so this
// import resolves to the source on disk, not to a published version.

package main

import (
	"fmt"

	"example.com/wsdemo/lib"
)

func main() {
	fmt.Println(lib.Greeting("workspace user"))
	fmt.Println("(now edit ../lib/lib.go and re-run; changes are picked up immediately)")
}
