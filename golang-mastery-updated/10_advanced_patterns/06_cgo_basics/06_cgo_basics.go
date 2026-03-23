// FILE: 10_advanced_patterns/06_cgo_basics.go
// TOPIC: cgo — calling C from Go (conceptual + safe demo)
//
// Run: go run 10_advanced_patterns/06_cgo_basics.go
//
// NOTE: This file explains cgo WITHOUT actually using it (no C compiler required).
// Real cgo examples require: a C compiler, import "C", and build with cgo enabled.
// This file shows the patterns and explains when/why to use cgo.

package main

import (
	"fmt"
	"math"
	"runtime"
)

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: cgo Basics")
	fmt.Println("════════════════════════════════════════")

	// ── WHAT IS CGO? ──────────────────────────────────────────────────────
	fmt.Println(`
── What is cgo? ──

  cgo lets you call C code from Go and Go code from C.
  When you write:
    import "C"
  Go's build system invokes a C compiler for the C code
  embedded in special // comments above the import.

── Basic cgo pattern ──

  /*
  #include <stdio.h>
  #include <stdlib.h>

  int add(int a, int b) {
      return a + b;
  }

  char* greet(const char* name) {
      char* result = malloc(100);
      sprintf(result, "Hello, %s!", name);
      return result;
  }
  */
  import "C"
  import "unsafe"

  func CallCAdd(a, b int) int {
      return int(C.add(C.int(a), C.int(b)))
  }

  func CallCGreet(name string) string {
      cName := C.CString(name)          // Go string → C string (malloc!)
      defer C.free(unsafe.Pointer(cName)) // MUST free C memory
      result := C.greet(cName)
      defer C.free(unsafe.Pointer(result))
      return C.GoString(result)         // C string → Go string (copies)
  }
`)

	// ── CGO TYPE CONVERSIONS ───────────────────────────────────────────────
	fmt.Println("── cgo Type Conversions ──")
	fmt.Println(`
  Go type    →  C type
  int        →  C.int
  int64      →  C.longlong
  float64    →  C.double
  bool       →  C.int (0/1)
  string     →  C.CString(s)  (allocates, must C.free!)
  []byte     →  C.CBytes(b)   (allocates, must C.free!)
  unsafe.Pointer → void*

  C.GoString(ptr)   → Go string (copies from C memory)
  C.GoBytes(ptr, n) → []byte   (copies from C memory)
`)

	// ── MEMORY MANAGEMENT RULES ────────────────────────────────────────────
	fmt.Println("── Memory Management Rules ──")
	fmt.Println(`
  CRITICAL:
    - C.CString() allocates C memory → MUST call C.free() on it
    - C.malloc() → MUST call C.free()
    - C code that returns malloc'd memory → MUST free in Go

    - Do NOT pass Go pointers to C and store them (GC can move Go objects)
    - Do NOT keep C pointers after the C function returns (if C might free them)

  Safe pattern:
    cs := C.CString(goStr)
    defer C.free(unsafe.Pointer(cs))
    C.someFunction(cs)
`)

	// ── CGO OVERHEAD ──────────────────────────────────────────────────────
	fmt.Println("── cgo Performance Overhead ──")
	fmt.Println(`
  Each cgo call costs ~50-100ns overhead (crossing the Go/C boundary).
  This includes:
    - Saving goroutine state
    - Switching from Go's managed stack to the C stack
    - Any GC lock/unlock needed

  WHEN CGO IS WORTH IT:
    - Wrapping large, battle-tested C libraries (OpenSSL, SQLite, OpenCV)
    - Hardware acceleration (CUDA, BLAS)
    - OS-specific APIs without a pure Go wrapper
    - Calling C code that takes milliseconds (overhead is negligible)

  WHEN TO AVOID CGO:
    - Simple math/string operations (Go is fast enough)
    - Cross-compilation (cgo makes it much harder)
    - When a pure Go library exists
    - When call frequency is very high (overhead adds up)
`)

	// ── ALTERNATIVES TO CGO ────────────────────────────────────────────────
	fmt.Println("── Alternatives to cgo ──")
	fmt.Println(`
  1. Pure Go libraries — often exist: SQLite has modernc.org/sqlite
  2. net.Conn to subprocess — run C program, communicate over socket/stdio
  3. gRPC — run C service separately, call via RPC
  4. WebAssembly — Go + WASM for some use cases
  5. plugin package — Go plugins (limited, Linux only mostly)
`)

	// ── Demo: Pure Go doing what cgo might do ─────────────────────────────
	fmt.Println("── Demo: Math (what you'd use cgo for with a C math library) ──")
	// Pure Go math is already well-optimized — no cgo needed:
	fmt.Printf("  math.Sqrt(2):   %.15f\n", math.Sqrt(2))
	fmt.Printf("  math.Sin(π/6):  %.15f  (= 0.5)\n", math.Sin(math.Pi/6))
	fmt.Printf("  math.Pow(2,32): %.0f\n", math.Pow(2, 32))

	// ── Current platform info ──────────────────────────────────────────────
	fmt.Printf("\n  Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  CGO_ENABLED: check with: go env CGO_ENABLED\n")

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  cgo: call C from Go via import \"C\"")
	fmt.Println("  C.CString → malloc, MUST C.free() (memory leak if missed)")
	fmt.Println("  ~50-100ns per cgo call overhead")
	fmt.Println("  Use cgo: wrapping C libraries, OS APIs, infrequent calls")
	fmt.Println("  Avoid cgo: hot paths, cross-compilation, simple operations")
	fmt.Println("  Prefer pure Go libraries when available")
}
