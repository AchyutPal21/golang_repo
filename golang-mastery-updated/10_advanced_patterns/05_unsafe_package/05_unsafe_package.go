// FILE: 10_advanced_patterns/05_unsafe_package.go
// TOPIC: unsafe Package — Sizeof, Alignof, Offsetof, pointer rules
//
// Run: go run 10_advanced_patterns/05_unsafe_package.go

package main

import (
	"fmt"
	"unsafe"
)

type SmallStruct struct {
	A bool    // 1 byte
	B int32   // 4 bytes
	C bool    // 1 byte
	D int64   // 8 bytes
}

// Optimized layout (same fields, reordered to minimize padding):
type OptimizedStruct struct {
	D int64  // 8 bytes
	B int32  // 4 bytes
	A bool   // 1 byte
	C bool   // 1 byte
	// 2 bytes padding at end (to make size multiple of 8)
}

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: unsafe Package")
	fmt.Println("════════════════════════════════════════")

	// ── unsafe.Sizeof — size of a type in bytes ───────────────────────────
	fmt.Println("\n── unsafe.Sizeof ──")
	fmt.Printf("  bool:    %d byte\n", unsafe.Sizeof(bool(false)))
	fmt.Printf("  int8:    %d byte\n", unsafe.Sizeof(int8(0)))
	fmt.Printf("  int16:   %d bytes\n", unsafe.Sizeof(int16(0)))
	fmt.Printf("  int32:   %d bytes\n", unsafe.Sizeof(int32(0)))
	fmt.Printf("  int64:   %d bytes\n", unsafe.Sizeof(int64(0)))
	fmt.Printf("  float32: %d bytes\n", unsafe.Sizeof(float32(0)))
	fmt.Printf("  float64: %d bytes\n", unsafe.Sizeof(float64(0)))
	fmt.Printf("  string:  %d bytes  (ptr+len header)\n", unsafe.Sizeof(string("")))
	fmt.Printf("  slice:   %d bytes  (ptr+len+cap header)\n", unsafe.Sizeof([]int{}))
	fmt.Printf("  map:     %d bytes  (pointer to runtime.hmap)\n", unsafe.Sizeof(map[int]int{}))

	// ── Struct layout and padding ─────────────────────────────────────────
	fmt.Println("\n── Struct layout and padding ──")
	fmt.Printf("  SmallStruct    size: %d bytes\n", unsafe.Sizeof(SmallStruct{}))
	fmt.Printf("  OptimizedStruct size: %d bytes  (reordered fields!)\n", unsafe.Sizeof(OptimizedStruct{}))
	fmt.Println("  Tip: order fields largest→smallest to minimize padding")

	// ── unsafe.Alignof ────────────────────────────────────────────────────
	fmt.Println("\n── unsafe.Alignof (alignment requirements) ──")
	fmt.Printf("  bool:    alignment %d\n", unsafe.Alignof(bool(false)))
	fmt.Printf("  int32:   alignment %d\n", unsafe.Alignof(int32(0)))
	fmt.Printf("  int64:   alignment %d\n", unsafe.Alignof(int64(0)))
	fmt.Printf("  float64: alignment %d\n", unsafe.Alignof(float64(0)))

	// ── unsafe.Offsetof ───────────────────────────────────────────────────
	fmt.Println("\n── unsafe.Offsetof (field byte offsets) ──")
	var s SmallStruct
	fmt.Printf("  SmallStruct.A offset: %d\n", unsafe.Offsetof(s.A))
	fmt.Printf("  SmallStruct.B offset: %d  (3 bytes padding after A)\n", unsafe.Offsetof(s.B))
	fmt.Printf("  SmallStruct.C offset: %d\n", unsafe.Offsetof(s.C))
	fmt.Printf("  SmallStruct.D offset: %d  (3 bytes padding after C)\n", unsafe.Offsetof(s.D))

	// ── unsafe.Pointer rules ──────────────────────────────────────────────
	fmt.Println("\n── unsafe.Pointer conversion rules ──")
	fmt.Println(`
  The ONLY safe conversions involving unsafe.Pointer:

  Rule 1: *T → unsafe.Pointer
    p := unsafe.Pointer(&x)

  Rule 2: unsafe.Pointer → *T  (reinterpret as different type)
    q := (*float64)(unsafe.Pointer(&n))

  Rule 3: unsafe.Pointer ↔ uintptr  (for arithmetic — then back IMMEDIATELY)
    // DANGEROUS: GC can move objects between the uintptr and the Pointer conversion
    // Only valid in a SINGLE expression, never store uintptr separately.

  Rule 4: unsafe.Pointer from reflect.Value.Pointer() or SliceHeader

  NEVER:
    - Store a uintptr and later convert to unsafe.Pointer (GC may have moved it)
    - Pass unsafe.Pointer to C and keep a Go reference (use C.malloc, not Go alloc)
`)

	// ── Practical: int32 → float32 reinterpret (bit cast) ─────────────────
	fmt.Println("── Bit reinterpret (int32 bits as float32) ──")
	n := int32(0x3F800000)  // IEEE 754 representation of 1.0
	f := *(*float32)(unsafe.Pointer(&n))
	fmt.Printf("  int32(0x3F800000) reinterpreted as float32: %f\n", f)

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  unsafe.Sizeof  → size in bytes (compile-time constant)")
	fmt.Println("  unsafe.Alignof → alignment requirement")
	fmt.Println("  unsafe.Offsetof → field offset in struct (optimize layout)")
	fmt.Println("  Order struct fields: largest→smallest to reduce padding")
	fmt.Println("  unsafe.Pointer: 4 valid conversion patterns only")
	fmt.Println("  For normal Go code: NEVER use unsafe")
	fmt.Println("  Legitimate uses: cgo, runtime internals, zero-copy serialization")
}
