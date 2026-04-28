// FILE: book/part2_core_language/chapter18_slices/examples/01_slice_header/main.go
// CHAPTER: 18 — Slices: The Most Important Type
// TOPIC: The slice header (ptr, len, cap), make, nil slice vs empty slice,
//        copy, 3-index slice, reflect.SliceHeader internals.
//
// Run (from the chapter folder):
//   go run ./examples/01_slice_header

package main

import (
	"fmt"
	"unsafe"
)

// sliceHeader prints the internal representation of a slice.
// We read the three words (ptr, len, cap) directly via unsafe.
func sliceHeader[T any](label string, s []T) {
	type header struct {
		ptr uintptr
		len int
		cap int
	}
	h := *(*header)(unsafe.Pointer(&s))
	fmt.Printf("%-20s ptr=0x%x  len=%d  cap=%d\n", label+":", h.ptr, h.len, h.cap)
}

func main() {
	// --- nil slice ---
	var ns []int
	sliceHeader("nil slice", ns)
	fmt.Println("nil slice == nil:", ns == nil)
	fmt.Println("len:", len(ns), "cap:", cap(ns))
	// nil slices are safe to range over and append to.
	for _, v := range ns {
		fmt.Println("never prints:", v)
	}

	fmt.Println()

	// --- empty (non-nil) slice ---
	es := []int{}
	sliceHeader("empty slice", es)
	fmt.Println("empty slice == nil:", es == nil) // false

	fmt.Println()

	// --- make ---
	// make([]T, len) — len elements, cap == len
	s1 := make([]int, 5)
	sliceHeader("make([]int,5)", s1)
	fmt.Println("s1:", s1)

	// make([]T, len, cap) — len elements, room for cap before reallocation
	s2 := make([]int, 3, 8)
	sliceHeader("make([]int,3,8)", s2)

	fmt.Println()

	// --- slice of existing array ---
	arr := [8]int{10, 20, 30, 40, 50, 60, 70, 80}
	s3 := arr[2:5] // len=3, cap=6 (from index 2 to end of arr)
	sliceHeader("arr[2:5]", s3)
	fmt.Println("s3:", s3)

	// --- len and cap ---
	fmt.Println("len(s3):", len(s3), "cap(s3):", cap(s3))

	fmt.Println()

	// --- copy ---
	src := []int{1, 2, 3, 4, 5}
	dst := make([]int, 3)
	n := copy(dst, src) // copies min(len(dst), len(src)) elements
	fmt.Println("copy:", n, "elements → dst:", dst)

	// Overlapping copy (safe with copy, unlike C memmove vs memcpy):
	overlap := []int{1, 2, 3, 4, 5}
	copy(overlap[1:], overlap) // shift right by 1
	fmt.Println("overlap shift:", overlap)

	fmt.Println()

	// --- 3-index slice: s[low:high:max] ---
	// Limits the capacity of the resulting slice, preventing accidental
	// writes beyond the intended window.
	base := []int{1, 2, 3, 4, 5, 6, 7, 8}
	limited := base[2:5:5] // len=3, cap=3 (max-low=5-2=3)
	sliceHeader("base[2:5:5]", limited)
	fmt.Println("limited:", limited)
	// Appending to limited will allocate a new backing array,
	// protecting base[5:] from accidental overwrite.
	limited = append(limited, 99)
	fmt.Println("after append:", limited)
	fmt.Println("base unchanged:", base)

	fmt.Println()

	// --- string to []byte and back ---
	str := "hello"
	b := []byte(str) // copy: strings are immutable
	b[0] = 'H'
	fmt.Println("modified []byte:", string(b))
	fmt.Println("original string:", str) // unchanged
}
