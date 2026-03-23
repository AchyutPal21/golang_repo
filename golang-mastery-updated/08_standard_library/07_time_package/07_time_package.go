// FILE: 08_standard_library/07_time_package.go
// TOPIC: time Package — time.Time, format (THE reference time), duration, timers
//
// Run: go run 08_standard_library/07_time_package.go

package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: time Package")
	fmt.Println("════════════════════════════════════════")

	// ── time.Now and time.Time methods ─────────────────────────────────────
	now := time.Now()
	fmt.Printf("\n── time.Now() ──\n")
	fmt.Printf("  now: %v\n", now)
	fmt.Printf("  Year: %d  Month: %s  Day: %d\n", now.Year(), now.Month(), now.Day())
	fmt.Printf("  Hour: %d  Minute: %d  Second: %d\n", now.Hour(), now.Minute(), now.Second())
	fmt.Printf("  Weekday: %s\n", now.Weekday())
	fmt.Printf("  Unix (seconds): %d\n", now.Unix())
	fmt.Printf("  UnixMilli: %d\n", now.UnixMilli())
	fmt.Printf("  UnixNano: %d\n", now.UnixNano())

	// ── THE FORMAT REFERENCE TIME ─────────────────────────────────────────
	// Go's time formatting is UNLIKE any other language.
	// You don't use %Y %m %d — you use THE reference time:
	//
	//   Mon Jan 2 15:04:05 MST 2006
	//   Mon Jan 2 15:04:05 -0700 2006
	//
	// This is EXACTLY: 01/02 03:04:05PM '06 -0700 (in Go's layout encoding)
	//   Month: January = 01  (1=Jan, 2=Feb, ... 12=Dec — mnemonic: 1 2 3 4 5 6 7)
	//   Day:   02
	//   Hour:  15 (3PM in 24h)
	//   Min:   04
	//   Sec:   05
	//   Year:  2006 (06 in short form)
	//   TZ:    MST (-0700)
	//
	// WHY? The Go team chose a specific datetime so you can see what each part does.
	// It's the reference time. You substitute your desired format by rearranging
	// these EXACT values.

	fmt.Println("\n── Time formatting (the reference time trick) ──")
	fmt.Printf("  RFC3339:      %s\n", now.Format(time.RFC3339))
	fmt.Printf("  RFC3339Nano:  %s\n", now.Format(time.RFC3339Nano))
	fmt.Printf("  Kitchen:      %s\n", now.Format(time.Kitchen))  // 3:04PM
	fmt.Printf("  Custom YYYY-MM-DD:     %s\n", now.Format("2006-01-02"))
	fmt.Printf("  Custom HH:MM:SS:       %s\n", now.Format("15:04:05"))
	fmt.Printf("  Custom full:           %s\n", now.Format("Mon, 02 Jan 2006 15:04:05 MST"))
	fmt.Printf("  Custom 12h with AM/PM: %s\n", now.Format("03:04:05 PM"))
	fmt.Printf("  Custom with millis:    %s\n", now.Format("2006-01-02 15:04:05.000"))
	// Predefined constants:
	// time.RFC3339, time.RFC822, time.UnixDate, time.Kitchen, time.Stamp, etc.

	// ── Parsing time strings ────────────────────────────────────────────────
	fmt.Println("\n── time.Parse ──")
	// Same reference time layout used for both Format and Parse
	t, err := time.Parse("2006-01-02", "2024-03-15")
	if err != nil {
		fmt.Printf("  Parse error: %v\n", err)
	} else {
		fmt.Printf("  Parsed \"2024-03-15\": %v\n", t)
	}

	t2, _ := time.Parse(time.RFC3339, "2024-03-15T10:30:00Z")
	fmt.Printf("  Parsed RFC3339: %v\n", t2)

	// time.ParseInLocation for timezone-aware parsing:
	// loc, _ := time.LoadLocation("America/New_York")
	// t3, _ := time.ParseInLocation("2006-01-02 15:04:05", "2024-03-15 10:30:00", loc)

	// ── Duration ─────────────────────────────────────────────────────────────
	fmt.Println("\n── Duration ──")
	// Duration is just int64 nanoseconds under the hood.
	// Constants: time.Nanosecond, Microsecond, Millisecond, Second, Minute, Hour
	d := 2*time.Hour + 30*time.Minute + 15*time.Second
	fmt.Printf("  2h30m15s = %v\n", d)
	fmt.Printf("  In seconds: %.0f\n", d.Seconds())
	fmt.Printf("  In minutes: %.1f\n", d.Minutes())
	fmt.Printf("  In hours: %.4f\n", d.Hours())

	// Parsing duration string:
	d2, _ := time.ParseDuration("1h30m45s")
	fmt.Printf("  Parsed \"1h30m45s\": %v\n", d2)

	// ── Time arithmetic ──────────────────────────────────────────────────────
	fmt.Println("\n── Time arithmetic ──")
	future := now.Add(24 * time.Hour)
	fmt.Printf("  now + 24h: %v\n", future.Format("2006-01-02"))
	past := now.Add(-7 * 24 * time.Hour)
	fmt.Printf("  now - 7d:  %v\n", past.Format("2006-01-02"))

	diff := future.Sub(now)
	fmt.Printf("  future.Sub(now): %v\n", diff)

	fmt.Printf("  time.Since(now): ~%v\n", time.Since(now).Round(time.Microsecond))
	fmt.Printf("  time.Until(future): ~%v\n", time.Until(future).Round(time.Hour))

	// Comparison:
	fmt.Printf("  future.After(now): %v\n", future.After(now))
	fmt.Printf("  past.Before(now):  %v\n", past.Before(now))
	fmt.Printf("  now.Equal(now):    %v\n", now.Equal(now))

	// ── Timers and Tickers ────────────────────────────────────────────────
	fmt.Println("\n── Timer and Ticker ──")
	// time.After(d) returns a channel that receives once after d:
	timer := time.NewTimer(10 * time.Millisecond)
	<-timer.C  // wait for timer
	fmt.Println("  Timer fired after 10ms")
	// Always stop timers you don't use: timer.Stop()

	// time.Sleep — simplest way to pause:
	fmt.Print("  Sleeping 5ms...")
	time.Sleep(5 * time.Millisecond)
	fmt.Println(" done")

	// Ticker fires repeatedly at a fixed interval:
	ticker := time.NewTicker(5 * time.Millisecond)
	count := 0
	for range ticker.C {
		count++
		if count == 3 {
			ticker.Stop()  // ALWAYS stop tickers to prevent goroutine leak
			break
		}
	}
	fmt.Printf("  Ticker fired %d times\n", count)

	// ── Monotonic clock ─────────────────────────────────────────────────────
	fmt.Println("\n── Monotonic clock ──")
	// time.Now() includes both wall clock and monotonic clock reading.
	// Wall clock can go backward (NTP sync, leap seconds).
	// Monotonic clock only goes forward — use for measuring elapsed time.
	// Sub() automatically uses monotonic when both times have monotonic component.
	start := time.Now()
	time.Sleep(1 * time.Millisecond)
	elapsed := time.Since(start)  // uses monotonic — accurate even if wall clock changes
	fmt.Printf("  Elapsed (monotonic): %v\n", elapsed)

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  time.Now()                   → current time (wall + monotonic)")
	fmt.Println("  Format/Parse reference time: Mon Jan 2 15:04:05 MST 2006")
	fmt.Println("  time.RFC3339 = \"2006-01-02T15:04:05Z07:00\"")
	fmt.Println("  time.Since(t) / time.Until(t) → elapsed/remaining")
	fmt.Println("  time.Add(d) / t.Sub(t2)       → arithmetic")
	fmt.Println("  time.NewTimer  → one-shot, always Stop() if unused")
	fmt.Println("  time.NewTicker → repeating, always Stop() to avoid leak")
	fmt.Println("  time.Since uses monotonic clock — accurate for benchmarking")
}
