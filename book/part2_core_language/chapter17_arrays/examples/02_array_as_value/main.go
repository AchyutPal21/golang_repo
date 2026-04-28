// FILE: book/part2_core_language/chapter17_arrays/examples/02_array_as_value/main.go
// CHAPTER: 17 — Arrays: The Real Underlying Type
// TOPIC: Value semantics (copy on assignment/pass), pointer to array,
//        array as slice backing store, fixed-size type uses.
//
// Run (from the chapter folder):
//   go run ./examples/02_array_as_value

package main

import "fmt"

// double attempts to double every element — but works on a COPY.
// The caller's array is unchanged.
func doubleCopy(arr [5]int) {
	for i := range arr {
		arr[i] *= 2
	}
	fmt.Println("inside doubleCopy:", arr)
}

// doublePointer works on the original via a pointer to the array.
func doublePointer(arr *[5]int) {
	for i := range arr {
		arr[i] *= 2
	}
}

// --- Fixed-size type uses ---

// SHA256 digest is exactly 32 bytes — an array enforces this at compile time.
type SHA256Digest [32]byte

func hexPrefix(d SHA256Digest) string {
	return fmt.Sprintf("%02x%02x%02x%02x...", d[0], d[1], d[2], d[3])
}

// IPv4Address is 4 bytes, comparable (can be used as map key).
type IPv4Address [4]byte

func (ip IPv4Address) String() string {
	return fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3])
}

// --- Array as slice backing store ---

func arraySliceRelationship() {
	arr := [5]int{1, 2, 3, 4, 5}

	// A slice is a window into an array.
	s := arr[1:4] // shares memory with arr
	fmt.Println("slice:", s)

	s[0] = 200      // modifies arr
	fmt.Println("arr after s[0]=200:", arr)

	// The full-slice expression: arr[:] gives a slice over the whole array.
	full := arr[:]
	fmt.Println("full slice:", full)

	// Pointer to array and slice header
	p := &arr
	ps := p[:]  // same as arr[:]
	ps[0] = 999
	fmt.Println("arr after ps[0]=999:", arr)
}

func main() {
	// --- Value semantics: copy on pass ---
	original := [5]int{1, 2, 3, 4, 5}
	doubleCopy(original)
	fmt.Println("original after doubleCopy:", original) // unchanged

	fmt.Println()

	// --- Pointer: mutate the original ---
	doublePointer(&original)
	fmt.Println("original after doublePointer:", original) // doubled

	fmt.Println()

	// --- Assignment copies ---
	a := [3]int{10, 20, 30}
	b := a            // full copy
	b[0] = 999
	fmt.Println("a:", a, "b:", b) // a unchanged

	fmt.Println()

	// --- Fixed-size types ---
	var digest SHA256Digest
	digest[0], digest[1] = 0xde, 0xad
	fmt.Println("digest prefix:", hexPrefix(digest))

	ip := IPv4Address{192, 168, 1, 1}
	fmt.Println("IP:", ip)

	// IPv4 as map key (comparable)
	hosts := map[IPv4Address]string{
		{127, 0, 0, 1}: "localhost",
		{10, 0, 0, 1}:  "gateway",
	}
	fmt.Println("lookup:", hosts[IPv4Address{127, 0, 0, 1}])

	fmt.Println()

	// --- Array/slice relationship ---
	arraySliceRelationship()
}
