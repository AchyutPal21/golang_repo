// FILE: 08_standard_library/06_os_package.go
// TOPIC: os Package — files, env, args, signals, filepath
//
// Run: go run 08_standard_library/06_os_package.go

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: os Package")
	fmt.Println("════════════════════════════════════════")

	// ── os.Args — command-line arguments ──────────────────────────────────
	fmt.Println("\n── os.Args ──")
	fmt.Printf("  os.Args[0] = %q  (program name)\n", os.Args[0])
	fmt.Printf("  os.Args length: %d\n", len(os.Args))
	// Real usage: os.Args[1], os.Args[2], ... are user-provided args.
	// For complex CLIs use the flag package or cobra library.

	// ── Environment variables ─────────────────────────────────────────────
	fmt.Println("\n── Environment variables ──")
	// os.Getenv: returns empty string if not set (no way to distinguish "not set" from "empty")
	home := os.Getenv("HOME")
	fmt.Printf("  HOME=%q\n", home)

	// os.LookupEnv: distinguishes "not set" from "set to empty"
	val, exists := os.LookupEnv("HOME")
	fmt.Printf("  LookupEnv(HOME): val=%q, exists=%v\n", val, exists)

	val2, exists2 := os.LookupEnv("DEFINITELY_NOT_SET")
	fmt.Printf("  LookupEnv(DEFINITELY_NOT_SET): val=%q, exists=%v\n", val2, exists2)

	// Set env var for this process:
	os.Setenv("MY_APP_ENV", "production")
	fmt.Printf("  After Setenv: MY_APP_ENV=%q\n", os.Getenv("MY_APP_ENV"))

	// ── File operations ───────────────────────────────────────────────────
	fmt.Println("\n── File operations ──")

	tmpFile := "/tmp/go_test_demo.txt"

	// Write a file (creates or truncates):
	content := "Hello from Go!\nSecond line.\n"
	err := os.WriteFile(tmpFile, []byte(content), 0644)  // 0644 = rw-r--r--
	if err != nil {
		fmt.Printf("  WriteFile error: %v\n", err)
		return
	}
	fmt.Printf("  WriteFile: wrote %d bytes to %s\n", len(content), tmpFile)

	// Read a file (entire content at once):
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		fmt.Printf("  ReadFile error: %v\n", err)
		return
	}
	fmt.Printf("  ReadFile: %q\n", string(data))

	// os.Open — read-only, returns *os.File (implements io.Reader)
	f, err := os.Open(tmpFile)
	if err != nil {
		fmt.Printf("  Open error: %v\n", err)
		return
	}
	defer f.Close()  // ALWAYS defer Close on opened files
	fmt.Printf("  Opened file: %s\n", f.Name())

	// os.OpenFile — full control over flags and permissions
	// Flags: os.O_RDONLY, os.O_WRONLY, os.O_RDWR, os.O_CREATE, os.O_TRUNC, os.O_APPEND
	appendFile := "/tmp/go_append_demo.txt"
	af, err := os.OpenFile(appendFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		fmt.Fprintf(af, "appended line\n")
		af.Close()
		fmt.Printf("  Appended to %s\n", appendFile)
	}

	// ── os.Stat — file info ────────────────────────────────────────────────
	fmt.Println("\n── os.Stat ──")
	info, err := os.Stat(tmpFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("  File does not exist")
		}
	} else {
		fmt.Printf("  Name: %s\n", info.Name())
		fmt.Printf("  Size: %d bytes\n", info.Size())
		fmt.Printf("  Mode: %s\n", info.Mode())
		fmt.Printf("  IsDir: %v\n", info.IsDir())
		fmt.Printf("  ModTime: %v\n", info.ModTime())
	}

	// ── Directory operations ──────────────────────────────────────────────
	fmt.Println("\n── Directory operations ──")
	tmpDir := "/tmp/go_demo_dir"
	err = os.MkdirAll(tmpDir+"/sub/deep", 0755)  // creates all intermediate dirs
	if err == nil {
		fmt.Printf("  Created: %s\n", tmpDir)
	}

	// List directory:
	entries, _ := os.ReadDir("/tmp")
	count := 0
	for _, e := range entries {
		if count < 3 {
			fmt.Printf("  /tmp entry: %s (dir=%v)\n", e.Name(), e.IsDir())
			count++
		}
	}

	// Cleanup:
	os.RemoveAll(tmpDir)
	os.Remove(tmpFile)
	os.Remove(appendFile)

	// ── filepath package ───────────────────────────────────────────────────
	fmt.Println("\n── filepath package ──")
	p := "/home/user/projects/myapp/main.go"
	fmt.Printf("  filepath.Dir:  %q\n", filepath.Dir(p))
	fmt.Printf("  filepath.Base: %q\n", filepath.Base(p))
	fmt.Printf("  filepath.Ext:  %q\n", filepath.Ext(p))
	joined := filepath.Join("/home", "user", "projects", "app")
	fmt.Printf("  filepath.Join: %q\n", joined)

	// filepath.Abs — get absolute path:
	abs, _ := filepath.Abs("relative/path")
	fmt.Printf("  filepath.Abs(\"relative/path\"): %q\n", abs)

	// ── os.Stdin/Stdout/Stderr ─────────────────────────────────────────────
	fmt.Println("\n── os.Stdin/Stdout/Stderr ──")
	fmt.Println("  os.Stdout is an *os.File that implements io.Writer")
	fmt.Fprintln(os.Stdout, "  Writing directly to os.Stdout")
	fmt.Fprintln(os.Stderr, "  Writing to os.Stderr (may appear separately)")

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  os.ReadFile/WriteFile   → simple whole-file read/write")
	fmt.Println("  os.Open                 → read-only file handle")
	fmt.Println("  os.OpenFile             → full control: flags + permissions")
	fmt.Println("  defer f.Close()         → ALWAYS close opened files")
	fmt.Println("  os.Stat / os.IsNotExist → file info and existence check")
	fmt.Println("  os.MkdirAll             → create nested directories")
	fmt.Println("  os.Getenv / LookupEnv   → read environment variables")
	fmt.Println("  filepath.Join/Dir/Base  → OS-portable path operations")
}
