// EXERCISE 11.1 — Implement a Flag-set type with Set/Clear/Has/Toggle.
//
// Run (from the chapter folder):
//   go run ./exercises/01_flag_set

package main

import (
	"fmt"
	"strings"
)

type Flag uint16

const (
	FlagDirty Flag = 1 << iota
	FlagDeleted
	FlagHidden
	FlagPinned
)

type FlagSet Flag

func (s FlagSet) Set(f Flag) FlagSet     { return s | FlagSet(f) }
func (s FlagSet) Clear(f Flag) FlagSet   { return s &^ FlagSet(f) }
func (s FlagSet) Has(f Flag) bool        { return uint16(s)&uint16(f) == uint16(f) }
func (s FlagSet) Toggle(f Flag) FlagSet  { return s ^ FlagSet(f) }

func (s FlagSet) String() string {
	names := []struct {
		f    Flag
		name string
	}{
		{FlagDirty, "dirty"},
		{FlagDeleted, "deleted"},
		{FlagHidden, "hidden"},
		{FlagPinned, "pinned"},
	}
	var parts []string
	for _, e := range names {
		if s.Has(e.f) {
			parts = append(parts, e.name)
		}
	}
	if len(parts) == 0 {
		return "[]"
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func main() {
	var s FlagSet
	fmt.Println("initial:    ", s)
	s = s.Set(FlagDirty)
	s = s.Set(FlagPinned)
	fmt.Println("set D+P:    ", s)
	s = s.Toggle(FlagDirty)
	fmt.Println("toggle D:   ", s)
	s = s.Set(FlagHidden)
	fmt.Println("set H:      ", s)
	s = s.Clear(FlagPinned)
	fmt.Println("clear P:    ", s)
	fmt.Println("Has Hidden: ", s.Has(FlagHidden))
	fmt.Println("Has Pinned: ", s.Has(FlagPinned))
}
