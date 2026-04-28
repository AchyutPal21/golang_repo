# Chapter 19 — Exercises

## 19.1 — Word count with ranking

Run [`exercises/01_word_count`](exercises/01_word_count/main.go).

`TopN` builds a frequency map, sorts by count descending then alphabetically,
and returns the top n entries.

Try:
- Add a `Unique(text string) int` function that returns the number of distinct words.
- Extend TopN to accept a stop-word list (common words to exclude: "the", "a", "is").
- What is the time complexity of TopN? Can you do better for very large texts?

## 19.2 ★ — LRU cache

Implement an LRU (least-recently-used) cache with `Get(key int) (int, bool)` and
`Put(key, value int)`. Use a map for O(1) lookup and a doubly-linked list (or a
slice-based deque) to track access order.

## 19.3 ★ — Graph adjacency list

Represent a directed graph as `map[string][]string` (node → neighbours).
Implement:
- `addEdge(g map[string][]string, from, to string)`
- `bfs(g map[string][]string, start string) []string` — returns nodes in BFS order
- `hasCycle(g map[string][]string) bool`
