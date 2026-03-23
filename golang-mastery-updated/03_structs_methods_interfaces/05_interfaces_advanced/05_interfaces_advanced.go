// 05_interfaces_advanced.go
//
// ADVANCED INTERFACES — composition, type assertions, type switches,
// interface vs concrete types in function signatures.
//
// The proverb: "Accept interfaces, return concrete types."
//   - Accepting interfaces makes your functions flexible (callers can pass
//     any satisfying type, including mocks in tests).
//   - Returning concrete types makes your API clear and lets callers access
//     all methods without type assertions.
//
// Interface SEGREGATION (from SOLID):
//   "No client should be forced to depend on methods it does not use."
//   In Go: define small, focused interfaces. Compose them when needed.

package main

import (
	"fmt"
	"math"
)

// ─── 1. Interface Composition ─────────────────────────────────────────────────
//
// Interfaces can embed other interfaces. The resulting interface requires ALL
// methods from all embedded interfaces. This is composition applied to interfaces.
//
// The standard library uses this extensively:
//   io.ReadWriter = io.Reader + io.Writer
//   io.ReadWriteCloser = io.Reader + io.Writer + io.Closer
//   io.ReadWriteSeeker = io.Reader + io.Writer + io.Seeker

type Reader interface {
	Read(p []byte) (n int, err error)
}

type Writer interface {
	Write(p []byte) (n int, err error)
}

type Closer interface {
	Close() error
}

// ReadWriter composes Reader and Writer.
// Any type with both Read() and Write() satisfies ReadWriter.
type ReadWriter interface {
	Reader
	Writer
}

// ReadWriteCloser composes all three.
type ReadWriteCloser interface {
	Reader
	Writer
	Closer
}

// ─── 2. Our Domain: Animal Hierarchy via Interfaces ───────────────────────────
//
// Instead of inheritance, we express capabilities as interfaces.
// Types opt in to the capabilities they have.

type Mover interface {
	Move() string
}

type Talker interface {
	Talk() string
}

type Swimmer interface {
	Swim() string
}

type Flyer interface {
	Fly() string
}

// MoverTalker is a composed interface.
type MoverTalker interface {
	Mover
	Talker
}

// Concrete types — each implements whichever interfaces make sense.

type Dog struct{ Name string }

func (d Dog) Move() string { return d.Name + " runs on four legs" }
func (d Dog) Talk() string { return d.Name + " barks: Woof!" }
func (d Dog) Swim() string { return d.Name + " paddles in the water" }

type Bird struct{ Name string }

func (b Bird) Move() string { return b.Name + " hops around" }
func (b Bird) Talk() string { return b.Name + " chirps: Tweet!" }
func (b Bird) Fly() string  { return b.Name + " soars through the sky" }

type Fish struct{ Name string }

func (f Fish) Move() string { return f.Name + " glides through the water" }
func (f Fish) Swim() string { return f.Name + " swims with fins" }

// Duck can do everything
type Duck struct{ Name string }

func (d Duck) Move() string { return d.Name + " waddles" }
func (d Duck) Talk() string { return d.Name + " quacks: Quack!" }
func (d Duck) Swim() string { return d.Name + " drifts on the pond" }
func (d Duck) Fly() string  { return d.Name + " flaps and takes flight" }

// ─── 3. Accept Interfaces, Return Concrete Types ──────────────────────────────
//
// This function accepts an interface → works with any Mover.
// Returns void here, but if we returned, we'd return the concrete type.

func makeItMove(m Mover) {
	fmt.Printf("  [Mover] %s\n", m.Move())
}

func makeItTalk(t Talker) {
	fmt.Printf("  [Talker] %s\n", t.Talk())
}

// ─── 4. Type Assertion — Single-Value Form ────────────────────────────────────
//
// Syntax: concrete := iface.(ConcreteType)
//
// Extracts the underlying concrete value from an interface.
// PANICS if the interface does not hold a value of ConcreteType.
// Use this only when you are CERTAIN of the type (e.g., after a type check).
//
// When to use: rarely. Most code should work through the interface.
// Exception: when you need to access a method NOT in the interface.

// ─── 5. Type Assertion — Two-Value Form (safe) ────────────────────────────────
//
// Syntax: concrete, ok := iface.(ConcreteType)
//
// ok is true if the assertion succeeded. concrete is zero value if ok is false.
// NEVER panics. Always use this form when unsure.

// ─── 6. Asserting to Multiple Interfaces ──────────────────────────────────────
//
// You can assert whether a value (stored as one interface) ALSO implements
// a DIFFERENT interface. This is called "interface-to-interface assertion."

func describeCapabilities(m Mover) {
	fmt.Printf("  %T can move: %s\n", m, m.Move())

	// Can it also talk? Check with a type assertion to Talker interface.
	if t, ok := m.(Talker); ok {
		fmt.Printf("  %T can talk: %s\n", m, t.Talk())
	}

	// Can it swim?
	if s, ok := m.(Swimmer); ok {
		fmt.Printf("  %T can swim: %s\n", m, s.Swim())
	}

	// Can it fly?
	if f, ok := m.(Flyer); ok {
		fmt.Printf("  %T can fly: %s\n", m, f.Fly())
	}
}

// ─── 7. Type Switch ───────────────────────────────────────────────────────────
//
// A type switch is a clean way to branch on the dynamic type of an interface.
// It's like a switch statement but for types.
//
// Syntax:
//   switch v := iface.(type) {
//   case ConcreteType1:
//       // v is ConcreteType1 here
//   case ConcreteType2:
//       // v is ConcreteType2 here
//   default:
//       // v is the original interface type
//   }
//
// WHY prefer type switch over a chain of if/else type assertions:
//   - Cleaner, more readable
//   - Exhaustive (you can add a default)
//   - The compiler can help with dead code analysis

func whatCanItDo(m Mover) string {
	switch v := m.(type) {
	case Dog:
		return fmt.Sprintf("Dog '%s': can move, talk, swim", v.Name)
	case Bird:
		return fmt.Sprintf("Bird '%s': can move, talk, fly", v.Name)
	case Fish:
		return fmt.Sprintf("Fish '%s': can move, swim", v.Name)
	case Duck:
		return fmt.Sprintf("Duck '%s': can move, talk, swim, fly — does it all!", v.Name)
	default:
		// v has type Mover — we only know it can Move
		return fmt.Sprintf("Unknown mover %T: can only move", v)
	}
}

// ─── 8. Type Switch for Polymorphic Behavior ──────────────────────────────────
//
// Real-world example: a function that processes heterogeneous event types.

type Event interface {
	EventName() string
}

type ClickEvent struct {
	X, Y int
}

func (e ClickEvent) EventName() string { return "click" }

type KeyEvent struct {
	Key      string
	Modifier string
}

func (e KeyEvent) EventName() string { return "keypress" }

type ScrollEvent struct {
	Delta float64
}

func (e ScrollEvent) EventName() string { return "scroll" }

// processEvent uses a type switch to handle different event types.
// This is Go's pattern for "polymorphic dispatch" without inheritance.
func processEvent(e Event) {
	switch ev := e.(type) {
	case ClickEvent:
		fmt.Printf("  Clicked at (%d, %d)\n", ev.X, ev.Y)
	case KeyEvent:
		fmt.Printf("  Key pressed: %s+%s\n", ev.Modifier, ev.Key)
	case ScrollEvent:
		direction := "down"
		if ev.Delta < 0 {
			direction = "up"
		}
		fmt.Printf("  Scrolled %s by %.1f units\n", direction, math.Abs(ev.Delta))
	default:
		fmt.Printf("  Unknown event: %s (%T)\n", ev.EventName(), ev)
	}
}

// ─── 9. Interface Segregation in Practice ────────────────────────────────────
//
// BAD: one fat interface that forces implementors to provide ALL methods
type AllInOneStorage interface {
	Read(key string) ([]byte, error)
	Write(key string, data []byte) error
	Delete(key string) error
	List() ([]string, error)
	Stats() map[string]int
	Backup() error
	Restore() error
}

// GOOD: segregated into small, focused interfaces
// Callers only depend on what they actually need.

type DataReader interface {
	Read(key string) ([]byte, error)
}

type DataWriter interface {
	Write(key string, data []byte) error
}

type DataDeleter interface {
	Delete(key string) error
}

type DataLister interface {
	List() ([]string, error)
}

// ReadWriter storage — only for components that need to read AND write
type DataReadWriter interface {
	DataReader
	DataWriter
}

// A mock/stub that satisfies DataReader without implementing everything
type MemoryStore struct {
	data map[string][]byte
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string][]byte)}
}

func (m *MemoryStore) Read(key string) ([]byte, error) {
	v, ok := m.data[key]
	if !ok {
		return nil, fmt.Errorf("key %q not found", key)
	}
	return v, nil
}

func (m *MemoryStore) Write(key string, data []byte) error {
	m.data[key] = data
	return nil
}

func (m *MemoryStore) Delete(key string) error {
	delete(m.data, key)
	return nil
}

func (m *MemoryStore) List() ([]string, error) {
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys, nil
}

// This function only needs a DataReader — it's more flexible and testable.
func fetchUserData(r DataReader, userID string) ([]byte, error) {
	return r.Read("user:" + userID)
}

// ─── main ─────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("========================================")
	fmt.Println("  Advanced Interfaces")
	fmt.Println("========================================")

	// ── Accept Interfaces ────────────────────────────────────────────────────
	fmt.Println("\n── Accept Interfaces — Polymorphism ─────────────────")

	movers := []Mover{
		Dog{Name: "Rex"},
		Bird{Name: "Tweety"},
		Fish{Name: "Nemo"},
		Duck{Name: "Donald"},
	}

	for _, m := range movers {
		makeItMove(m)
	}

	// ── Asserting to Multiple Interfaces ─────────────────────────────────────
	fmt.Println("\n── Asserting to Multiple Interfaces ─────────────────")

	for _, m := range movers {
		describeCapabilities(m)
		fmt.Println()
	}

	// ── Type Assertion: Single-Value (panics on failure) ──────────────────────
	fmt.Println("\n── Type Assertion (single-value, careful!) ──────────")

	var m Mover = Dog{Name: "Buddy"}

	// We know it's a Dog — safe to use single-value form here
	dog := m.(Dog)
	fmt.Printf("Asserted to Dog: %s can swim: %s\n", dog.Name, dog.Swim())

	// Demonstrate panic protection using recover (normally you'd use two-value form)
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Panic caught: %v\n", r)
			}
		}()
		var m2 Mover = Fish{Name: "Nemo"}
		_ = m2.(Dog) // PANIC: Fish is not a Dog
	}()

	// ── Type Assertion: Two-Value (safe) ─────────────────────────────────────
	fmt.Println("\n── Type Assertion (two-value, safe) ─────────────────")

	animals := []Mover{
		Dog{Name: "Rex"},
		Bird{Name: "Tweety"},
		Fish{Name: "Nemo"},
	}

	for _, a := range animals {
		if dog, ok := a.(Dog); ok {
			fmt.Printf("  %T is a Dog, name: %s\n", a, dog.Name)
		} else if bird, ok := a.(Bird); ok {
			fmt.Printf("  %T is a Bird, name: %s\n", a, bird.Name)
		} else {
			fmt.Printf("  %T is something else\n", a)
		}
	}

	// ── Type Switch ──────────────────────────────────────────────────────────
	fmt.Println("\n── Type Switch ──────────────────────────────────────")

	for _, m := range movers {
		fmt.Printf("  %s\n", whatCanItDo(m))
	}

	// ── Type Switch for Events ───────────────────────────────────────────────
	fmt.Println("\n── Type Switch for Event Processing ─────────────────")

	events := []Event{
		ClickEvent{X: 100, Y: 200},
		KeyEvent{Key: "S", Modifier: "Ctrl"},
		ScrollEvent{Delta: 3.5},
		ScrollEvent{Delta: -1.0},
		ClickEvent{X: 50, Y: 75},
	}

	for _, e := range events {
		processEvent(e)
	}

	// ── Interface-to-Interface Assertion ─────────────────────────────────────
	fmt.Println("\n── Interface-to-Interface Assertion ─────────────────")

	var mover Mover = Duck{Name: "Daffy"}

	// Assert that mover also satisfies Swimmer
	if swimmer, ok := mover.(Swimmer); ok {
		fmt.Printf("  Mover is also a Swimmer: %s\n", swimmer.Swim())
	}

	// Assert that mover also satisfies Flyer
	if flyer, ok := mover.(Flyer); ok {
		fmt.Printf("  Mover is also a Flyer: %s\n", flyer.Fly())
	}

	// Assert to composed interface
	if mt, ok := mover.(MoverTalker); ok {
		fmt.Printf("  MoverTalker: %s | %s\n", mt.Move(), mt.Talk())
	}

	// ── Interface Segregation ────────────────────────────────────────────────
	fmt.Println("\n── Interface Segregation ────────────────────────────")

	store := NewMemoryStore()
	_ = store.Write("user:42", []byte(`{"name":"Alice","age":30}`))
	_ = store.Write("user:99", []byte(`{"name":"Bob","age":25}`))

	// fetchUserData only needs a DataReader — MemoryStore satisfies it
	data, err := fetchUserData(store, "42")
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Printf("  User 42 data: %s\n", string(data))
	}

	// MemoryStore satisfies multiple small interfaces
	var reader DataReader = store
	var writer DataWriter = store
	var lister DataLister = store

	_ = writer.Write("config:theme", []byte("dark"))
	keys, _ := lister.List()
	fmt.Printf("  All keys: %v\n", keys)

	val, _ := reader.Read("config:theme")
	fmt.Printf("  theme config: %s\n", string(val))

	// ── Key Takeaways ────────────────────────────────────────────────────────
	fmt.Println("\n── Key Takeaways ────────────────────────────────────")
	fmt.Println(`
  1. Compose interfaces from smaller ones (io.ReadWriter pattern).
  2. Type assertion one-value form panics — use only when type is certain.
  3. Type assertion two-value form is always safe (ok pattern).
  4. Type switch is cleaner than chained if/else type assertions.
  5. Accept interfaces → flexible; return concrete types → clear API.
  6. Interface segregation: prefer many small interfaces over one large one.
  7. You can assert from one interface type to another interface type.
  `)
}
