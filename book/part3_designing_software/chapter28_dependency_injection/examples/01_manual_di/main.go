// FILE: book/part3_designing_software/chapter28_dependency_injection/examples/01_manual_di/main.go
// CHAPTER: 28 — Dependency Injection
// TOPIC: Manual DI via constructor injection, the wiring layer (main),
//        test fakes, and why Go needs no DI framework.
//
// Run (from the chapter folder):
//   go run ./examples/01_manual_di

package main

import (
	"fmt"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// Dependency injection means: give a component its dependencies from
// the outside rather than letting it create them internally.
//
// In Go, this is trivially done via constructor parameters.
// No framework, no annotations, no code generation.
// ─────────────────────────────────────────────────────────────────────────────

// ─── Interfaces (defined by consumers) ───────────────────────────────────────

type UserStore interface {
	FindByEmail(email string) (User, error)
	Save(u User) error
}

type Mailer interface {
	Send(to, subject, body string) error
}

type Clock interface {
	Now() time.Time
}

// ─── Domain types ─────────────────────────────────────────────────────────────

type User struct {
	ID        int
	Email     string
	Name      string
	CreatedAt time.Time
}

// ─── Service ─────────────────────────────────────────────────────────────────

type UserService struct {
	store  UserStore
	mailer Mailer
	clock  Clock
}

// Constructor: all dependencies are explicit parameters.
// Callers cannot create UserService without providing every dependency.
func NewUserService(store UserStore, mailer Mailer, clock Clock) *UserService {
	return &UserService{store: store, mailer: mailer, clock: clock}
}

func (s *UserService) Register(email, name string) (User, error) {
	if _, err := s.store.FindByEmail(email); err == nil {
		return User{}, fmt.Errorf("email %s already registered", email)
	}
	u := User{Email: email, Name: name, CreatedAt: s.clock.Now()}
	if err := s.store.Save(u); err != nil {
		return User{}, err
	}
	_ = s.mailer.Send(email, "Welcome "+name, "Your account is ready.")
	return u, nil
}

// ─── Production implementations ───────────────────────────────────────────────

type memUserStore struct {
	users  map[string]User
	nextID int
}

func newMemUserStore() *memUserStore {
	return &memUserStore{users: make(map[string]User), nextID: 1}
}

func (m *memUserStore) FindByEmail(email string) (User, error) {
	u, ok := m.users[email]
	if !ok {
		return User{}, fmt.Errorf("not found: %s", email)
	}
	return u, nil
}

func (m *memUserStore) Save(u User) error {
	u.ID = m.nextID
	m.nextID++
	m.users[u.Email] = u
	return nil
}

type smtpMailer struct{ from string }

func (s *smtpMailer) Send(to, subject, body string) error {
	fmt.Printf("[SMTP] from=%s to=%s subj=%q\n", s.from, to, subject)
	return nil
}

type realClock struct{}

func (r realClock) Now() time.Time { return time.Now() }

// ─── Test fakes ───────────────────────────────────────────────────────────────

type fakeUserStore struct {
	users map[string]User
	saved []User
}

func newFakeUserStore(existing ...User) *fakeUserStore {
	f := &fakeUserStore{users: make(map[string]User)}
	for _, u := range existing {
		f.users[u.Email] = u
	}
	return f
}

func (f *fakeUserStore) FindByEmail(email string) (User, error) {
	u, ok := f.users[email]
	if !ok {
		return User{}, fmt.Errorf("not found")
	}
	return u, nil
}

func (f *fakeUserStore) Save(u User) error {
	f.users[u.Email] = u
	f.saved = append(f.saved, u)
	return nil
}

type fakeMailer struct{ sent []string }

func (f *fakeMailer) Send(to, subject, _ string) error {
	f.sent = append(f.sent, fmt.Sprintf("to=%s subj=%q", to, subject))
	return nil
}

type fixedClock struct{ t time.Time }

func (f fixedClock) Now() time.Time { return f.t }

func main() {
	fmt.Println("=== production wiring ===")
	store := newMemUserStore()
	mailer := &smtpMailer{from: "noreply@example.com"}
	clock := realClock{}
	svc := NewUserService(store, mailer, clock)

	u, err := svc.Register("alice@example.com", "Alice")
	if err != nil {
		fmt.Println("error:", err)
	} else {
		fmt.Printf("registered: id=%d name=%s\n", u.ID, u.Name)
	}

	_, err = svc.Register("alice@example.com", "Alice again")
	fmt.Println("duplicate:", err)

	fmt.Println()
	fmt.Println("=== test wiring (fakes, fixed clock) ===")
	fstore := newFakeUserStore()
	fmailer := &fakeMailer{}
	fclock := fixedClock{t: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	testSvc := NewUserService(fstore, fmailer, fclock)

	u2, err := testSvc.Register("bob@example.com", "Bob")
	if err != nil {
		fmt.Println("error:", err)
	} else {
		fmt.Printf("registered at: %s\n", u2.CreatedAt.Format(time.RFC3339))
	}

	fmt.Println("emails sent:", strings.Join(fmailer.sent, " | "))
	fmt.Println("users saved:", len(fstore.saved))
}
