# Chapter 11 — Revision Checkpoint

## Questions

1. What does `&^` do?
2. Can `i++` be used as an expression?
3. Why doesn't `[]int == []int` compile?
4. What are the five precedence levels?
5. Which Go operator does the work of Java's `>>>`?

## Answers

1. Bit clear (AND NOT). `a &^ b` zeros the bits of `a` that are
   set in `b`. Equivalent to `a & ^b`.

2. No. `++` and `--` are statements; the result is not an
   expression. You can't write `j = i++`.

3. Slice equality would require an O(n) walk, and Go wants `==`
   to be semantically constant-time. Slices/maps/functions are
   not comparable except to `nil`. Use `slices.Equal` for deep
   comparison.

4. (5, highest) `*  /  %  <<  >>  &  &^`; (4) `+  -  |  ^`; (3)
   `==  !=  <  <=  >  >=`; (2) `&&`; (1) `||`.

5. Use a `uint` type and `>>`. Go picks logical vs arithmetic
   shift based on the operand's signedness, so `uint(x) >> n` is
   a logical shift (Java's `>>>`).
