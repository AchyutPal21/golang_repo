// FILE: book/part2_core_language/chapter20_structs/examples/02_embedding/main.go
// CHAPTER: 20 — Structs and Composite Literals
// TOPIC: Struct embedding, field promotion, method promotion,
//        embedding vs inheritance, pointer embedding.
//
// Run (from the chapter folder):
//   go run ./examples/02_embedding

package main

import "fmt"

// --- Field promotion ---

type Animal struct {
	Name string
	Age  int
}

func (a Animal) Describe() string {
	return fmt.Sprintf("%s (age %d)", a.Name, a.Age)
}

type Dog struct {
	Animal         // embedded: fields and methods are promoted
	Breed  string
}

func (d Dog) Bark() string { return "Woof!" }

// --- Method shadowing ---

type Base struct {
	ID int
}

func (b Base) Describe() string {
	return fmt.Sprintf("Base{ID:%d}", b.ID)
}

type Derived struct {
	Base
	Extra string
}

// Derived.Describe shadows Base.Describe.
func (d Derived) Describe() string {
	return fmt.Sprintf("Derived{%s extra:%s}", d.Base.Describe(), d.Extra)
}

// --- Pointer embedding ---

type Logger struct {
	prefix string
}

func (l *Logger) Log(msg string) {
	fmt.Printf("[%s] %s\n", l.prefix, msg)
}

type Server struct {
	*Logger // embedded pointer — Logger methods promoted
	addr    string
}

func NewServer(addr, logPrefix string) *Server {
	return &Server{
		Logger: &Logger{prefix: logPrefix},
		addr:   addr,
	}
}

// --- Multiple embedding ---

type Closer struct{}

func (c Closer) Close() { fmt.Println("closed") }

type Reader struct{}

func (r Reader) Read() string { return "data" }

type ReadCloser struct {
	Reader
	Closer
}

// --- Embedding interface ---

// Embeding an interface in a struct is used to build mock/partial implementations.
type Saver interface {
	Save(data string) error
}

type LoggingService struct {
	Saver // promoted Save method
	log   *Logger
}

func (s *LoggingService) Save(data string) error {
	s.log.Log("saving: " + data)
	return s.Saver.Save(data) // delegate to inner Saver
}

// noopSaver is a minimal Saver for the demo.
type noopSaver struct{}

func (n noopSaver) Save(data string) error {
	fmt.Println("  [noop] saved:", data)
	return nil
}

func main() {
	// --- field promotion ---
	d := Dog{
		Animal: Animal{Name: "Rex", Age: 3},
		Breed:  "Labrador",
	}
	fmt.Println(d.Name)        // promoted from Animal
	fmt.Println(d.Describe())  // promoted from Animal
	fmt.Println(d.Bark())
	fmt.Println(d.Animal.Name) // explicit access also works

	fmt.Println()

	// --- method shadowing ---
	der := Derived{Base: Base{ID: 42}, Extra: "bonus"}
	fmt.Println(der.Describe())      // Derived.Describe
	fmt.Println(der.Base.Describe()) // explicit Base.Describe

	fmt.Println()

	// --- pointer embedding ---
	srv := NewServer(":8080", "HTTP")
	srv.Log("server starting")  // Logger.Log promoted through *Logger
	srv.Log("listening on " + srv.addr)

	fmt.Println()

	// --- multiple embedding ---
	rc := ReadCloser{}
	fmt.Println("read:", rc.Read())
	rc.Close()

	fmt.Println()

	// --- embedded interface delegation ---
	ls := &LoggingService{
		Saver: noopSaver{},
		log:   &Logger{prefix: "SVC"},
	}
	_ = ls.Save("important record")
}
