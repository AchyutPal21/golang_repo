// FILE: 01_fundamentals/04_types_floats_complex/04_types_floats_complex.go
// TOPIC: Floating Point & Complex Numbers — IEEE 754, precision pitfalls, complex
//
// Run: go run 01_fundamentals/04_types_floats_complex/04_types_floats_complex.go
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS MATTERS:
//   Floating point is the source of notorious bugs in ALL languages.
//   "0.1 + 0.2 != 0.3" is not a Go bug — it's how IEEE 754 works.
//   Not understanding this causes incorrect financial calculations,
//   physics simulations, and comparison bugs. Know how floats actually work.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"fmt"
	"math"
	"math/cmplx"
)

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Floats & Complex Numbers")
	fmt.Println("════════════════════════════════════════")

	// ─────────────────────────────────────────────────────────────────────
	// float32 vs float64 — The Two Floating Point Types
	// ─────────────────────────────────────────────────────────────────────
	//
	// float32 = 32 bits = 4 bytes
	//   - Sign: 1 bit
	//   - Exponent: 8 bits  → range ~1.18×10^-38 to 3.4×10^38
	//   - Mantissa: 23 bits → ~7 significant decimal digits of precision
	//
	// float64 = 64 bits = 8 bytes (the default)
	//   - Sign: 1 bit
	//   - Exponent: 11 bits → range ~2.2×10^-308 to 1.8×10^308
	//   - Mantissa: 52 bits → ~15-17 significant decimal digits of precision
	//
	// RULE: ALWAYS use float64 unless you have a specific reason to use float32.
	//   float32 reasons: GPU work, large arrays where memory is critical,
	//                    talking to C APIs that require float.
	//   Otherwise: float64. The extra precision prevents accumulation errors.
	//
	// UNTYPED float constants default to float64 (just like untyped int → int).

	var f32 float32 = 3.14159265358979323846 // only ~7 digits are stored
	var f64 float64 = 3.14159265358979323846 // ~15-17 digits are stored

	fmt.Printf("\n── float32 vs float64 precision ──\n")
	fmt.Printf("  float32: %.20f\n", f32) // See precision cutoff around digit 7
	fmt.Printf("  float64: %.20f\n", f64) // Much more precise

	// ─────────────────────────────────────────────────────────────────────
	// THE GOLDEN RULE: Never compare floats with ==
	// ─────────────────────────────────────────────────────────────────────
	//
	// Floating point numbers are represented in binary.
	// Many decimal fractions (0.1, 0.2, 0.3) CANNOT be exactly represented
	// in binary — they become infinite repeating binary fractions,
	// rounded to the nearest representable value.
	//
	// This means arithmetic results can have tiny rounding errors.
	// These errors accumulate through multiple operations.

	a := 0.1 + 0.2
	b := 0.3
	fmt.Printf("\n── The famous float comparison bug ──\n")
	fmt.Printf("  0.1 + 0.2       = %.17f\n", a)
	fmt.Printf("  0.3             = %.17f\n", b)
	fmt.Printf("  0.1+0.2 == 0.3 → %v  ← NOT equal!\n", a == b)

	// CORRECT WAY: Use an epsilon (tolerance) comparison
	epsilon := 1e-9 // tolerance: numbers within 1 billionth are "equal"
	diff := math.Abs(a - b)
	fmt.Printf("  |diff|=%.2e < epsilon=%.2e → %v  ← correct comparison\n",
		diff, epsilon, diff < epsilon)

	// For financial calculations: NEVER use float. Use integer cents,
	// or a decimal library. Float accumulation errors in money = fraud.

	// ─────────────────────────────────────────────────────────────────────
	// SPECIAL FLOAT VALUES: Inf, NaN
	// ─────────────────────────────────────────────────────────────────────
	//
	// IEEE 754 defines special values:
	//   +Inf  = positive infinity (e.g., 1.0 / 0.0)
	//   -Inf  = negative infinity (e.g., -1.0 / 0.0)
	//   NaN   = Not a Number (e.g., 0.0 / 0.0, sqrt(-1))
	//
	// IMPORTANT: NaN != NaN is ALWAYS TRUE.
	//   NaN is not equal to anything, including itself.
	//   This is the IEEE standard. Use math.IsNaN() to check for NaN.
	//
	// Go does NOT panic on division by zero for floats (unlike integers!).
	// Integer division by zero PANICS. Float division by zero gives +Inf/-Inf.

	posInf := math.Inf(1)
	negInf := math.Inf(-1)
	nan := math.NaN()

	fmt.Printf("\n── Special float values ──\n")
	fmt.Printf("  +Inf: %v  IsInf: %v\n", posInf, math.IsInf(posInf, 1))
	fmt.Printf("  -Inf: %v  IsInf: %v\n", negInf, math.IsInf(negInf, -1))
	fmt.Printf("  NaN:  %v  IsNaN: %v\n", nan, math.IsNaN(nan))
	fmt.Printf("  NaN == NaN: %v  ← NaN is never equal to itself!\n", nan == nan)

	// Float division by zero → Inf (no panic)
	x := 1.0 / 0.0
	fmt.Printf("  1.0/0.0 = %v (float: no panic, returns Inf)\n", x)

	// ─────────────────────────────────────────────────────────────────────
	// MATH PACKAGE — Essential float functions
	// ─────────────────────────────────────────────────────────────────────

	fmt.Printf("\n── math package functions ──\n")
	fmt.Printf("  math.Pi         = %.10f\n", math.Pi)
	fmt.Printf("  math.E          = %.10f\n", math.E)
	fmt.Printf("  math.Sqrt(2)    = %.10f\n", math.Sqrt(2))
	fmt.Printf("  math.Pow(2, 10) = %.0f\n", math.Pow(2, 10))
	fmt.Printf("  math.Log(math.E)= %.10f\n", math.Log(math.E))
	fmt.Printf("  math.Log2(1024) = %.10f\n", math.Log2(1024))
	fmt.Printf("  math.Log10(100) = %.10f\n", math.Log10(100))
	fmt.Printf("  math.Abs(-3.14) = %.2f\n", math.Abs(-3.14))
	fmt.Printf("  math.Ceil(4.1)  = %.0f\n", math.Ceil(4.1))
	fmt.Printf("  math.Floor(4.9) = %.0f\n", math.Floor(4.9))
	fmt.Printf("  math.Round(4.5) = %.0f\n", math.Round(4.5))
	fmt.Printf("  math.Min(3,5)   = %.0f\n", math.Min(3, 5))
	fmt.Printf("  math.Max(3,5)   = %.0f\n", math.Max(3, 5))
	fmt.Printf("  math.MaxFloat64 = %e\n", math.MaxFloat64)
	fmt.Printf("  math.SmallestNonzeroFloat64 = %e\n", math.SmallestNonzeroFloat64)

	// ─────────────────────────────────────────────────────────────────────
	// COMPLEX NUMBERS — Go has native complex type support
	// ─────────────────────────────────────────────────────────────────────
	//
	// Most languages need a library for complex numbers. Go has them built in.
	//
	// complex64  = real float32 + imaginary float32
	// complex128 = real float64 + imaginary float64 (the default)
	//
	// A complex number: a + bi
	//   a = real part
	//   b = imaginary part
	//   i = imaginary unit (sqrt(-1))
	//
	// USE CASES:
	//   - Signal processing (FFT)
	//   - Electrical engineering (impedance, phasors)
	//   - Fractal computation (Mandelbrot set)
	//   - Quantum computing simulations
	//
	// In most backend/systems Go code, you won't use complex numbers.
	// But Go has them natively, which is a design choice for scientific work.

	// Creating complex numbers: complex(real, imag)
	c1 := complex(3.0, 4.0)  // 3 + 4i
	c2 := complex(1.0, -2.0) // 1 - 2i
	// Or with literal syntax:
	c3 := 2 + 3i // complex128 literal (i suffix)
	_ = c3

	fmt.Printf("\n── Complex numbers ──\n")
	fmt.Printf("  c1 = %v  (type: %T)\n", c1, c1)
	fmt.Printf("  c2 = %v  (type: %T)\n", c2, c2)

	// Arithmetic on complex numbers:
	sum := c1 + c2
	product := c1 * c2
	fmt.Printf("  c1 + c2 = %v\n", sum)
	fmt.Printf("  c1 * c2 = %v\n", product)

	// Extract real and imaginary parts:
	fmt.Printf("  real(c1)  = %.1f\n", real(c1))
	fmt.Printf("  imag(c1)  = %.1f\n", imag(c1))

	// math/cmplx package for complex math:
	fmt.Printf("  cmplx.Abs(3+4i)  = %.1f  (magnitude: sqrt(3²+4²))\n", cmplx.Abs(c1))
	fmt.Printf("  cmplx.Phase(3+4i)= %.4f radians\n", cmplx.Phase(c1))
	fmt.Printf("  cmplx.Sqrt(-1)   = %v  (should be i)\n", cmplx.Sqrt(-1))
	fmt.Printf("  cmplx.Exp(πi)    = %v  (Euler: e^(πi) ≈ -1)\n", cmplx.Exp(complex(0, math.Pi)))

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  float32 → 7 digits precision, use only when memory critical")
	fmt.Println("  float64 → 15-17 digits precision, the default for everything")
	fmt.Println("  NEVER compare floats with ==, use epsilon tolerance")
	fmt.Println("  NEVER use float for money — use integer cents or decimal lib")
	fmt.Println("  NaN != NaN always — use math.IsNaN()")
	fmt.Println("  complex64/128 → built-in complex number support")
}
