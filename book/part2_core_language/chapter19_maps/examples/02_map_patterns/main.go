// FILE: book/part2_core_language/chapter19_maps/examples/02_map_patterns/main.go
// CHAPTER: 19 — Maps: Hash Tables Built In
// TOPIC: Frequency counter, grouping, set, inverted index,
//        map of slices, struct as value, concurrent access with sync.Map.
//
// Run (from the chapter folder):
//   go run ./examples/02_map_patterns

package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// --- Frequency counter ---

func wordFreq(text string) map[string]int {
	freq := make(map[string]int)
	for _, w := range strings.Fields(text) {
		freq[strings.ToLower(w)]++
	}
	return freq
}

// --- Grouping (map of slices) ---

type Person struct {
	Name string
	City string
}

func groupByCity(people []Person) map[string][]Person {
	groups := make(map[string][]Person)
	for _, p := range people {
		groups[p.City] = append(groups[p.City], p)
	}
	return groups
}

// --- Set via map[T]struct{} ---

type Set[T comparable] map[T]struct{}

func (s Set[T]) Add(v T)          { s[v] = struct{}{} }
func (s Set[T]) Has(v T) bool     { _, ok := s[v]; return ok }
func (s Set[T]) Delete(v T)       { delete(s, v) }
func (s Set[T]) Len() int         { return len(s) }

func NewSet[T comparable](items ...T) Set[T] {
	s := make(Set[T])
	for _, v := range items {
		s.Add(v)
	}
	return s
}

func Intersect[T comparable](a, b Set[T]) Set[T] {
	result := make(Set[T])
	for k := range a {
		if b.Has(k) {
			result.Add(k)
		}
	}
	return result
}

// --- Inverted index ---

func buildInvertedIndex(docs map[int]string) map[string][]int {
	index := make(map[string][]int)
	for id, text := range docs {
		for _, w := range strings.Fields(strings.ToLower(text)) {
			index[w] = append(index[w], id)
		}
	}
	return index
}

// --- Concurrent map: sync.Map ---

func concurrentDemo() {
	var m sync.Map

	var wg sync.WaitGroup
	for i := range 5 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", n)
			m.Store(key, n*n)
		}(i)
	}
	wg.Wait()

	fmt.Println("sync.Map contents:")
	keys := []string{}
	m.Range(func(k, v any) bool {
		keys = append(keys, fmt.Sprintf("%v=%v", k, v))
		return true
	})
	sort.Strings(keys)
	for _, kv := range keys {
		fmt.Println(" ", kv)
	}
}

func main() {
	// --- frequency ---
	text := "the quick brown fox jumps over the lazy dog the fox"
	freq := wordFreq(text)
	// Print top words sorted
	type kv struct {
		k string
		v int
	}
	var pairs []kv
	for k, v := range freq {
		pairs = append(pairs, kv{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].v != pairs[j].v {
			return pairs[i].v > pairs[j].v
		}
		return pairs[i].k < pairs[j].k
	})
	fmt.Println("word frequencies (top 5):")
	for _, p := range pairs[:5] {
		fmt.Printf("  %-10s %d\n", p.k, p.v)
	}

	fmt.Println()

	// --- grouping ---
	people := []Person{
		{"Alice", "NYC"}, {"Bob", "LA"}, {"Carol", "NYC"},
		{"Dave", "LA"}, {"Eve", "NYC"},
	}
	groups := groupByCity(people)
	cities := []string{"NYC", "LA"}
	for _, city := range cities {
		names := make([]string, len(groups[city]))
		for i, p := range groups[city] {
			names[i] = p.Name
		}
		fmt.Printf("%s: %v\n", city, names)
	}

	fmt.Println()

	// --- set ---
	a := NewSet("go", "python", "rust", "java")
	b := NewSet("go", "rust", "c", "c++")
	inter := Intersect(a, b)
	fmt.Println("a has go:", a.Has("go"))
	fmt.Println("a has ruby:", a.Has("ruby"))
	fmt.Printf("intersection size: %d\n", inter.Len())
	a.Delete("java")
	fmt.Println("a after delete java, len:", a.Len())

	fmt.Println()

	// --- inverted index ---
	docs := map[int]string{
		1: "go is fast",
		2: "go is safe",
		3: "python is slow but easy",
	}
	index := buildInvertedIndex(docs)
	fmt.Println("docs containing 'go':", index["go"])
	fmt.Println("docs containing 'is':", index["is"])

	fmt.Println()

	// --- sync.Map ---
	concurrentDemo()
}
