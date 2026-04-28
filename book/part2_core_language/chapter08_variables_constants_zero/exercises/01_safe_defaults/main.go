// FILE: book/part2_core_language/chapter08_variables_constants_zero/exercises/01_safe_defaults/main.go
// EXERCISE 8.3 — Find a shadowing bug.
//
// Run (from the chapter folder):
//   go run ./exercises/01_safe_defaults
//
// The program is a tiny config loader. It looks correct at a glance — it
// even runs and prints something — but it has a SHADOWING BUG that hides
// the real error path. Read the code, find the bug, fix it, then run:
//
//   go vet ./exercises/01_safe_defaults
//
// to confirm vet sees the issue too. Hint: look at how `err` is used.

package main

import (
	"errors"
	"fmt"
)

func loadConfig() (string, error) {
	return "", errors.New("config file missing")
}

func loadDefaults() string {
	return "default-config"
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		// BUG: := here SHADOWS the outer err. The fix: use = (single equals).
		// Until you fix it, the outer err is still set, but the if-branch
		// log message is the inner err — same value here, but if the
		// fallback later modifies "the error", we'd be modifying the inner
		// one and leaving the outer untouched.
		err := fmt.Errorf("primary load failed: %w", err)
		fmt.Println("warn:", err)
		cfg = loadDefaults()
	}
	if err == nil {
		fmt.Println("loaded config:", cfg)
	} else {
		fmt.Println("note: outer err is still", err)
	}
}
