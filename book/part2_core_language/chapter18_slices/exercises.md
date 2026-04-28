# Chapter 18 — Exercises

## 18.1 — Generic stack

Run [`exercises/01_stack`](exercises/01_stack/main.go).

Study how `Pop` shrinks the slice and how the zero value of `Stack[T]` is
immediately usable without a constructor.

Try:
- Add `Clear()` that resets the stack to empty.
- Implement `Drain() []T` that pops all elements and returns them in LIFO order.
- Why does `Pop` return `(T, bool)` instead of `(T, error)`?

## 18.2 ★ — Sliding window maximum

Implement `slidingMax(nums []int, k int) []int` that returns the maximum element
in each window of size k. For `[1,3,-1,-3,5,3,6,7]` with k=3, the answer is
`[3,3,5,5,6,7]`. The O(n) solution uses a deque (a slice used as a ring buffer
of indices).

## 18.3 ★ — In-place string reverse words

Implement `reverseWords(s []byte)` that reverses the order of words in a byte
slice in place — no additional allocation. For `"the sky is blue"` the result
is `"blue is sky the"`. Hint: reverse the whole slice, then reverse each word.
