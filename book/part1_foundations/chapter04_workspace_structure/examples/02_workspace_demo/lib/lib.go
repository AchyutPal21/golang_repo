// Package lib is a tiny library used to demonstrate go.work multi-module
// development. In a real project this would be a separate Git repo; for
// the demo it's a sibling directory bound by the workspace's go.work file.
package lib

import "fmt"

// Greeting returns a polite greeting addressed to name.
//
// Greeting is exported (capital G) so the svc module can call it.
func Greeting(name string) string {
	return fmt.Sprintf("hello from lib, %s!", name)
}
