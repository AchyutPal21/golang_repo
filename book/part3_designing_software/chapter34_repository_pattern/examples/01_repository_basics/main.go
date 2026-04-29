// FILE: book/part3_designing_software/chapter34_repository_pattern/examples/01_repository_basics/main.go
// CHAPTER: 34 — Repository Pattern
// TOPIC: Repository interface, in-memory implementation, multiple implementations,
//        unit of work, and the contract a repository must honour.
//
// Run (from the chapter folder):
//   go run ./examples/01_repository_basics

package main

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN
// ─────────────────────────────────────────────────────────────────────────────

type UserID int

type User struct {
	ID        UserID
	Email     string
	Name      string
	Active    bool
	CreatedAt time.Time
}

// Sentinel errors defined in the domain layer — implementations must use these.
var (
	ErrUserNotFound  = errors.New("user not found")
	ErrEmailTaken    = errors.New("email already registered")
	ErrInvalidUserID = errors.New("invalid user ID")
)

// ─────────────────────────────────────────────────────────────────────────────
// REPOSITORY INTERFACE (defined in the application/domain layer)
//
// Rules:
//   1. Methods return domain types, not DB-specific types
//   2. Save() handles both insert and update (upsert semantics)
//   3. Errors use domain sentinels, not driver errors
//   4. No SQL, no JSON, no HTTP in the interface
// ─────────────────────────────────────────────────────────────────────────────

type UserRepository interface {
	Save(u User) (User, error)
	FindByID(id UserID) (User, error)
	FindByEmail(email string) (User, error)
	FindAll() ([]User, error)
	FindActive() ([]User, error)
	Delete(id UserID) error
	Count() (int, error)
}

// ─────────────────────────────────────────────────────────────────────────────
// IN-MEMORY IMPLEMENTATION
// ─────────────────────────────────────────────────────────────────────────────

type memUserRepo struct {
	users  map[UserID]User
	nextID UserID
}

func NewMemUserRepo() UserRepository {
	return &memUserRepo{
		users:  make(map[UserID]User),
		nextID: 1,
	}
}

func (r *memUserRepo) Save(u User) (User, error) {
	// insert: ID == 0
	if u.ID == 0 {
		// check email uniqueness
		for _, existing := range r.users {
			if existing.Email == u.Email {
				return User{}, ErrEmailTaken
			}
		}
		u.ID = r.nextID
		r.nextID++
		if u.CreatedAt.IsZero() {
			u.CreatedAt = time.Now()
		}
	} else {
		// update: must exist
		if _, ok := r.users[u.ID]; !ok {
			return User{}, ErrUserNotFound
		}
	}
	r.users[u.ID] = u
	return u, nil
}

func (r *memUserRepo) FindByID(id UserID) (User, error) {
	if id <= 0 {
		return User{}, ErrInvalidUserID
	}
	u, ok := r.users[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return u, nil
}

func (r *memUserRepo) FindByEmail(email string) (User, error) {
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return User{}, ErrUserNotFound
}

func (r *memUserRepo) FindAll() ([]User, error) {
	users := make([]User, 0, len(r.users))
	for _, u := range r.users {
		users = append(users, u)
	}
	sort.Slice(users, func(i, j int) bool { return users[i].ID < users[j].ID })
	return users, nil
}

func (r *memUserRepo) FindActive() ([]User, error) {
	var users []User
	for _, u := range r.users {
		if u.Active {
			users = append(users, u)
		}
	}
	sort.Slice(users, func(i, j int) bool { return users[i].ID < users[j].ID })
	return users, nil
}

func (r *memUserRepo) Delete(id UserID) error {
	if _, ok := r.users[id]; !ok {
		return ErrUserNotFound
	}
	delete(r.users, id)
	return nil
}

func (r *memUserRepo) Count() (int, error) {
	return len(r.users), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SECOND IMPLEMENTATION: append-only (audit) repository
// Demonstrates that the same interface can have radically different backends.
// ─────────────────────────────────────────────────────────────────────────────

type auditUserRepo struct {
	log     []User        // append-only; versions are preserved
	current map[UserID]int // id → latest log index
	nextID  UserID
}

func NewAuditUserRepo() UserRepository {
	return &auditUserRepo{current: make(map[UserID]int)}
}

func (r *auditUserRepo) Save(u User) (User, error) {
	if u.ID == 0 {
		u.ID = r.nextID + 1
		r.nextID++
		u.CreatedAt = time.Now()
	}
	r.log = append(r.log, u)
	r.current[u.ID] = len(r.log) - 1
	fmt.Printf("    [AUDIT] saved version %d of user %d\n", len(r.log)-1, u.ID)
	return u, nil
}

func (r *auditUserRepo) FindByID(id UserID) (User, error) {
	idx, ok := r.current[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return r.log[idx], nil
}

func (r *auditUserRepo) FindByEmail(email string) (User, error) {
	for id, idx := range r.current {
		if r.log[idx].Email == email {
			_ = id
			return r.log[idx], nil
		}
	}
	return User{}, ErrUserNotFound
}

func (r *auditUserRepo) FindAll() ([]User, error) {
	var users []User
	for _, idx := range r.current {
		users = append(users, r.log[idx])
	}
	sort.Slice(users, func(i, j int) bool { return users[i].ID < users[j].ID })
	return users, nil
}

func (r *auditUserRepo) FindActive() ([]User, error) {
	var users []User
	for _, idx := range r.current {
		if r.log[idx].Active {
			users = append(users, r.log[idx])
		}
	}
	return users, nil
}

func (r *auditUserRepo) Delete(id UserID) error {
	if _, ok := r.current[id]; !ok {
		return ErrUserNotFound
	}
	delete(r.current, id)
	return nil
}

func (r *auditUserRepo) Count() (int, error) {
	return len(r.current), nil
}

func (r *auditUserRepo) VersionCount() int { return len(r.log) }

// ─────────────────────────────────────────────────────────────────────────────
// APPLICATION SERVICE — depends only on the UserRepository interface
// ─────────────────────────────────────────────────────────────────────────────

type UserService struct{ repo UserRepository }

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Register(email, name string) (User, error) {
	u := User{Email: email, Name: name, Active: true}
	created, err := s.repo.Save(u)
	if err != nil {
		return User{}, fmt.Errorf("Register: %w", err)
	}
	return created, nil
}

func (s *UserService) Deactivate(id UserID) error {
	u, err := s.repo.FindByID(id)
	if err != nil {
		return fmt.Errorf("Deactivate: %w", err)
	}
	u.Active = false
	_, err = s.repo.Save(u)
	return err
}

func (s *UserService) Summary() {
	all, _ := s.repo.FindAll()
	active, _ := s.repo.FindActive()
	total, _ := s.repo.Count()
	fmt.Printf("  total=%d  active=%d\n", total, len(active))
	for _, u := range all {
		status := "active"
		if !u.Active {
			status = "inactive"
		}
		fmt.Printf("    [%d] %-20s  %s\n", u.ID, u.Email, status)
	}
	_ = all
}

func runDemo(label string, repo UserRepository) {
	fmt.Printf("=== %s ===\n", label)
	svc := NewUserService(repo)

	alice, _ := svc.Register("alice@example.com", "Alice")
	bob, _ := svc.Register("bob@example.com", "Bob")
	_, _ = svc.Register("carol@example.com", "Carol")

	// Duplicate email
	_, err := svc.Register("alice@example.com", "Alice2")
	if errors.Is(err, ErrEmailTaken) {
		fmt.Println("  duplicate email blocked (expected)")
	}

	_ = svc.Deactivate(bob.ID)
	_ = alice

	svc.Summary()
}

func main() {
	runDemo("in-memory repository", NewMemUserRepo())
	fmt.Println()
	runDemo("audit repository", NewAuditUserRepo())
}
