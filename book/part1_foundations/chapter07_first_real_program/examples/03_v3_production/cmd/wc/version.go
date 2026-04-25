// FILE: examples/03_v3_production/cmd/wc/version.go
// TOPIC: --version implementation. Reads VCS info from runtime/debug.

package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// version is set at build time via:
//   go build -ldflags '-X main.version=1.2.3' ./cmd/wc
// It defaults to "dev" so `go run` works without flags.
var version = "dev"

func printVersion() {
	fmt.Printf("wc %s", version)

	// Since Go 1.18, debug.ReadBuildInfo exposes VCS info if -buildvcs=true
	// (the default). Prefer this over a -ldflags-stamped commit; it works
	// without any build orchestration.
	if info, ok := debug.ReadBuildInfo(); ok {
		var revision, modified string
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				revision = s.Value
			case "vcs.modified":
				modified = s.Value
			}
		}
		if revision != "" {
			short := revision
			if len(short) > 8 {
				short = short[:8]
			}
			fmt.Printf(" (%s", short)
			if modified == "true" {
				fmt.Print(", dirty")
			}
			fmt.Print(")")
		}
	}
	fmt.Printf(" %s\n", runtime.Version())
}
