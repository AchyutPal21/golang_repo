// 02_struct_embedding.go
//
// STRUCT EMBEDDING — Go's mechanism for composition.
//
// Go deliberately omits inheritance. The designers (Griesemer, Pike, Thompson)
// observed that deep inheritance hierarchies create tight coupling, fragile
// base class problems, and make refactoring painful. Instead, Go uses:
//
//   "Favor composition over inheritance" — a principle from the Gang of Four.
//
// Embedding is NOT inheritance. It is DELEGATION with syntactic sugar.
// When you embed type T into S, Go:
//   1. Creates an anonymous field whose name IS the type name T.
//   2. Promotes all exported fields and methods of T to S's method set.
//   3. Allows you to access them directly on S as if S defined them.
//
// But S IS NOT a T. You cannot pass S where T is expected without explicit
// conversion or accessing the embedded field.

package main

import "fmt"

// ─── 1. The Baseline: Named Field (no embedding) ──────────────────────────────
//
// Before embedding, let's see what life looks like WITHOUT it.
// With a named field you must always qualify: dog.Animal.Speak()

type AnimalNamed struct {
	Name string
}

func (a AnimalNamed) Speak() string {
	return a.Name + " makes a sound"
}

type DogNamed struct {
	Animal AnimalNamed // named field — must qualify access
	Breed  string
}

// ─── 2. Embedding (Composition with Promotion) ────────────────────────────────
//
// Embedding: list the type WITHOUT a field name.
// The field name implicitly becomes the type name: Animal.
//
// After embedding:
//   d.Name        is the same as d.Animal.Name      (field promotion)
//   d.Speak()     is the same as d.Animal.Speak()   (method promotion)

type Animal struct {
	Name string
	Age  int
}

// Speak is a method on Animal that will be promoted to Dog.
func (a Animal) Speak() string {
	return fmt.Sprintf("%s says: ...", a.Name)
}

// Breathe is another promoted method.
func (a Animal) Breathe() string {
	return fmt.Sprintf("%s breathes oxygen", a.Name)
}

// Dog EMBEDS Animal.
// Dog gets Name, Age, Speak(), and Breathe() for free.
type Dog struct {
	Animal        // embedded — anonymous field, name is "Animal"
	Breed  string
}

// Dog OVERRIDES the promoted Speak() method.
// When you call d.Speak(), Go calls Dog.Speak(), NOT Animal.Speak().
// This is "method overriding" via shadowing — the embedded method
// is still accessible via d.Animal.Speak() explicitly.
func (d Dog) Speak() string {
	return fmt.Sprintf("%s says: Woof! (breed: %s)", d.Name, d.Breed)
}

// ─── 3. Multi-Level Embedding ─────────────────────────────────────────────────
//
// Embedding can be chained. ServiceDog embeds Dog which embeds Animal.
// Fields and methods are promoted transitively.

type ServiceDog struct {
	Dog               // embeds Dog (which embeds Animal)
	ServiceType string
}

// ServiceDog also overrides Speak.
func (s ServiceDog) Speak() string {
	return fmt.Sprintf("%s says: I am a %s dog. Woof!", s.Name, s.ServiceType)
}

// ─── 4. Multiple Embedding ────────────────────────────────────────────────────
//
// You can embed multiple types. Fields and methods from all are promoted.
// AMBIGUITY: if two embedded types have a method with the same name,
// accessing that method on the outer type is a compile-time error unless
// the outer type defines its own method (which shadows the ambiguity).

type Swimmer struct {
	MaxSpeed float64
}

func (s Swimmer) Swim() string {
	return fmt.Sprintf("swimming at %.1f km/h", s.MaxSpeed)
}

type Runner struct {
	Stamina int
}

func (r Runner) Run() string {
	return fmt.Sprintf("running with stamina %d", r.Stamina)
}

// Triathlete embeds both. Gets Swim() and Run() promoted.
type Triathlete struct {
	Swimmer
	Runner
	Name string
}

// ─── 5. Embedding Interfaces ──────────────────────────────────────────────────
//
// You can embed an INTERFACE in a struct. This means:
//   - The struct has an anonymous field of interface type.
//   - The struct satisfies the interface (because the embedded interface does).
//   - At runtime, calling those methods delegates to whatever is stored in
//     the interface field.
//
// WHY: Primarily used for:
//   a) Partial implementation / wrapper types (override only some methods)
//   b) Building mock types in tests
//
// WARNING: If the interface field is nil at runtime and you call an unoverridden
// method, you get a nil pointer panic.

type Speaker interface {
	Speak() string
}

// SpeakerWrapper wraps any Speaker and adds logging.
// It only overrides Speak() — any other methods of the embedded interface
// are delegated automatically.
type SpeakerLogger struct {
	Speaker               // embedded interface
	prefix  string
}

func (sl SpeakerLogger) Speak() string {
	// Override: add prefix logging, then delegate to the embedded Speaker
	result := sl.Speaker.Speak() // explicit delegation to embedded field
	return fmt.Sprintf("[%s] %s", sl.prefix, result)
}

// ─── 6. Real-World Example: HTTP Handler Middleware Pattern ───────────────────
//
// A common real-world use: embed a base struct to inherit shared behavior,
// then specialize in the outer struct.

type BaseHandler struct {
	Route string
}

func (b BaseHandler) Log(msg string) {
	fmt.Printf("[BaseHandler|%s] %s\n", b.Route, msg)
}

func (b BaseHandler) Authenticate() bool {
	// In real code: check JWT, session, etc.
	return true
}

type UserHandler struct {
	BaseHandler       // gets Log() and Authenticate() for free
	DB          string // pretend this is a *sql.DB
}

func (u UserHandler) HandleGet() {
	if !u.Authenticate() { // promoted method call
		u.Log("Unauthorized") // promoted method call
		return
	}
	u.Log("Handling GET /users") // promoted method call
	fmt.Printf("  Fetching users from DB: %s\n", u.DB)
}

// ─── main ─────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("========================================")
	fmt.Println("  Struct Embedding & Composition")
	fmt.Println("========================================")

	// ── Named Field vs Embedding ─────────────────────────────────────────────
	fmt.Println("\n── Named Field (no embedding) ───────────────────────")

	dn := DogNamed{
		Animal: AnimalNamed{Name: "Rex"},
		Breed:  "Labrador",
	}
	// Must qualify: dn.Animal.Speak(), dn.Animal.Name
	fmt.Printf("Name (via field): %s\n", dn.Animal.Name)
	fmt.Printf("Speak (via field): %s\n", dn.Animal.Speak())

	// ── Embedding with Promoted Fields and Methods ───────────────────────────
	fmt.Println("\n── Embedding — Promoted Fields & Methods ────────────")

	d := Dog{
		Animal: Animal{Name: "Buddy", Age: 3}, // initialize embedded field
		Breed:  "Golden Retriever",
	}

	// Promoted field access — no need to write d.Animal.Name
	fmt.Printf("d.Name (promoted):  %s\n", d.Name)    // same as d.Animal.Name
	fmt.Printf("d.Age (promoted):   %d\n", d.Age)     // same as d.Animal.Age
	fmt.Printf("d.Breed (own field): %s\n", d.Breed)

	// Promoted method — calls Animal.Breathe() because Dog doesn't override it
	fmt.Printf("d.Breathe() (promoted): %s\n", d.Breathe())

	// Overridden method — calls Dog.Speak(), NOT Animal.Speak()
	fmt.Printf("d.Speak() (overridden): %s\n", d.Speak())

	// You can still reach the original via explicit field access
	fmt.Printf("d.Animal.Speak() (original): %s\n", d.Animal.Speak())

	// ── Embedded field name IS the type name ─────────────────────────────────
	fmt.Println("\n── Accessing the Embedded Field Explicitly ──────────")

	// The embedded field name is "Animal" (the type name)
	fmt.Printf("Type of d.Animal: %T\n", d.Animal)

	// This lets you pass the embedded part where Animal is expected
	printAnimalInfo(d.Animal) // works: d.Animal IS an Animal

	// NOTE: you CANNOT pass d itself where Animal is expected (not inheritance)
	// printAnimalInfo(d)  // compile error: cannot use Dog as Animal

	// ── Multi-Level Embedding ────────────────────────────────────────────────
	fmt.Println("\n── Multi-Level Embedding ────────────────────────────")

	sd := ServiceDog{
		Dog:         Dog{Animal: Animal{Name: "Rex", Age: 5}, Breed: "German Shepherd"},
		ServiceType: "Guide",
	}

	// sd.Name is promoted from Animal through Dog through ServiceDog
	fmt.Printf("sd.Name (2 levels deep): %s\n", sd.Name)
	fmt.Printf("sd.Breed (1 level deep): %s\n", sd.Breed)
	fmt.Printf("sd.Speak() (overridden at ServiceDog): %s\n", sd.Speak())
	fmt.Printf("sd.Dog.Speak() (Dog's override): %s\n", sd.Dog.Speak())
	fmt.Printf("sd.Animal.Speak() (original): %s\n", sd.Animal.Speak())
	fmt.Printf("sd.Breathe() (promoted from Animal): %s\n", sd.Breathe())

	// ── Multiple Embedding ───────────────────────────────────────────────────
	fmt.Println("\n── Multiple Embedding ───────────────────────────────")

	t := Triathlete{
		Swimmer: Swimmer{MaxSpeed: 3.5},
		Runner:  Runner{Stamina: 90},
		Name:    "Jan",
	}

	fmt.Printf("%s: %s\n", t.Name, t.Swim()) // promoted from Swimmer
	fmt.Printf("%s: %s\n", t.Name, t.Run())  // promoted from Runner

	// Accessing embedded fields directly
	fmt.Printf("t.MaxSpeed: %.1f  t.Stamina: %d\n", t.MaxSpeed, t.Stamina)

	// ── Embedding an Interface ───────────────────────────────────────────────
	fmt.Println("\n── Embedding an Interface ───────────────────────────")

	cat := Animal{Name: "Whiskers"}
	// Animal satisfies Speaker because it has a Speak() method

	logger := SpeakerLogger{
		Speaker: cat,   // store a concrete Animal in the interface field
		prefix:  "LOG",
	}
	fmt.Printf("Logged speak: %s\n", logger.Speak())

	// Now use it as a Speaker — SpeakerLogger satisfies Speaker
	var s Speaker = logger
	fmt.Printf("As Speaker: %s\n", s.Speak())

	// ── Real-World: HTTP Handler Base ────────────────────────────────────────
	fmt.Println("\n── Real-World: Embedded Base Handler ────────────────")

	handler := UserHandler{
		BaseHandler: BaseHandler{Route: "/users"},
		DB:          "postgres://localhost/app",
	}
	handler.HandleGet()

	// ── Embedding vs Named Field Summary ─────────────────────────────────────
	fmt.Println("\n── Summary: Embedding vs Named Field ────────────────")
	fmt.Println(`
  Named field  (type T as field):
    - Access: outer.FieldName.Method()
    - Explicit — clear ownership, no surprise promotions
    - USE when the relationship is "has-a" with clear distinction

  Embedding (anonymous field):
    - Access: outer.Method() (promoted) or outer.TypeName.Method()
    - Provides promotion — outer type gains inner's methods/fields
    - USE when you want to "extend" or "specialize" a type
    - USE for mixins, adding shared behavior (logging, locking, etc.)

  KEY INSIGHT:
    Embedding is syntactic sugar for delegation.
    Go promotes methods so you type less, but underneath it's
    just: outer.Inner.Method() — no vtable, no inheritance chain.
  `)
}

func printAnimalInfo(a Animal) {
	fmt.Printf("  Animal: name=%s, age=%d\n", a.Name, a.Age)
}
