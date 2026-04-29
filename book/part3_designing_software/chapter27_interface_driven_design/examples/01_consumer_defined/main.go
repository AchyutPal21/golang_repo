// FILE: book/part3_designing_software/chapter27_interface_driven_design/examples/01_consumer_defined/main.go
// CHAPTER: 27 — Interface-Driven Design
// TOPIC: Consumer-side interfaces — define the interface where it is used,
//        not where it is implemented. Decoupling packages.
//
// Run (from the chapter folder):
//   go run ./examples/01_consumer_defined

package main

import (
	"fmt"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// BAD: producer-side interface
//
// The database package defines a big interface. Every consumer that wants
// one operation still imports and depends on the full interface.
// ─────────────────────────────────────────────────────────────────────────────

type BigDBInterface interface {
	GetUser(id int) (string, error)
	SaveUser(id int, name string) error
	DeleteUser(id int) error
	ListUsers() ([]string, error)
	GetProduct(id int) (string, error)
	SaveProduct(id int, name string) error
	// ... 20 more methods
}

// ─────────────────────────────────────────────────────────────────────────────
// GOOD: consumer-side interface
//
// Each consumer defines exactly the interface it needs.
// The concrete type (realDB) satisfies all of them without knowing they exist.
// ─────────────────────────────────────────────────────────────────────────────

// UserService only needs two operations. Define that interface here.
type UserReader interface {
	GetUser(id int) (string, error)
}

type UserWriter interface {
	SaveUser(id int, name string) error
}

// AuditService only needs to read users (for logging who did what).
type AuditUserReader interface {
	GetUser(id int) (string, error)
}

// ReportService only needs to list users.
type UserLister interface {
	ListUsers() ([]string, error)
}

// ─── Concrete implementation (lives in a "db" package in real code) ───────────

type realDB struct {
	users map[int]string
}

func newRealDB() *realDB {
	return &realDB{users: map[int]string{1: "Alice", 2: "Bob", 3: "Carol"}}
}

func (db *realDB) GetUser(id int) (string, error) {
	name, ok := db.users[id]
	if !ok {
		return "", fmt.Errorf("user %d not found", id)
	}
	return name, nil
}

func (db *realDB) SaveUser(id int, name string) error {
	db.users[id] = name
	return nil
}

func (db *realDB) DeleteUser(id int) error {
	delete(db.users, id)
	return nil
}

func (db *realDB) ListUsers() ([]string, error) {
	names := make([]string, 0, len(db.users))
	for _, n := range db.users {
		names = append(names, n)
	}
	return names, nil
}

// ─── Services depend only on their narrow interfaces ─────────────────────────

type UserService struct {
	reader UserReader
	writer UserWriter
}

func (s *UserService) Greet(id int) string {
	name, err := s.reader.GetUser(id)
	if err != nil {
		return "Hello, stranger"
	}
	return "Hello, " + name
}

func (s *UserService) Register(id int, name string) error {
	return s.writer.SaveUser(id, name)
}

type ReportService struct {
	lister UserLister
}

func (r *ReportService) Summary() string {
	users, _ := r.lister.ListUsers()
	return fmt.Sprintf("%d users: %s", len(users), strings.Join(users, ", "))
}

// ─── Audit log ────────────────────────────────────────────────────────────────

type AuditLog struct {
	reader AuditUserReader
	log    []string
}

func (a *AuditLog) Record(userID int, action string) {
	name, _ := a.reader.GetUser(userID)
	entry := fmt.Sprintf("[%s] user=%s action=%s",
		time.Now().Format("15:04:05"), name, action)
	a.log = append(a.log, entry)
}

func (a *AuditLog) Print() {
	for _, e := range a.log {
		fmt.Println(" ", e)
	}
}

func main() {
	db := newRealDB()

	// Each service receives only the slice of the db it needs.
	userSvc := &UserService{reader: db, writer: db}
	reportSvc := &ReportService{lister: db}
	audit := &AuditLog{reader: db}

	// Operations
	fmt.Println(userSvc.Greet(1))
	fmt.Println(userSvc.Greet(99))

	_ = userSvc.Register(4, "Dave")
	audit.Record(1, "login")
	audit.Record(4, "register")

	fmt.Println()
	fmt.Println("Report:", reportSvc.Summary())

	fmt.Println()
	fmt.Println("Audit log:")
	audit.Print()

	fmt.Println()
	fmt.Println("Key: realDB satisfies UserReader, UserWriter, UserLister, AuditUserReader")
	fmt.Println("     without knowing any of those interfaces exist.")
}
