// EXERCISE 27.1 — Storage abstraction with consumer-side interfaces.
//
// A TodoService uses only what it needs from storage.
// Implement two backends: MemoryStore and (stub) FileStore.
// Both satisfy the storage interface without knowing about TodoService.
//
// Run (from the chapter folder):
//   go run ./exercises/01_storage_abstraction

package main

import (
	"errors"
	"fmt"
)

type Todo struct {
	ID   int
	Text string
	Done bool
}

// ─── Consumer-side interfaces (defined in the service package) ────────────────

type TodoSaver interface {
	Save(t Todo) error
}

type TodoFinder interface {
	FindByID(id int) (Todo, error)
}

type TodoLister interface {
	List() ([]Todo, error)
}

type TodoDeleter interface {
	Delete(id int) error
}

// TodoStore combines all operations — used by the service.
type TodoStore interface {
	TodoSaver
	TodoFinder
	TodoLister
	TodoDeleter
}

// ─── Memory backend ───────────────────────────────────────────────────────────

type MemoryStore struct {
	items  map[int]Todo
	nextID int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{items: make(map[int]Todo), nextID: 1}
}

func (m *MemoryStore) Save(t Todo) error {
	if t.ID == 0 {
		t.ID = m.nextID
		m.nextID++
	}
	m.items[t.ID] = t
	return nil
}

func (m *MemoryStore) FindByID(id int) (Todo, error) {
	t, ok := m.items[id]
	if !ok {
		return Todo{}, fmt.Errorf("todo %d not found", id)
	}
	return t, nil
}

func (m *MemoryStore) List() ([]Todo, error) {
	result := make([]Todo, 0, len(m.items))
	for _, t := range m.items {
		result = append(result, t)
	}
	return result, nil
}

func (m *MemoryStore) Delete(id int) error {
	if _, ok := m.items[id]; !ok {
		return fmt.Errorf("todo %d not found", id)
	}
	delete(m.items, id)
	return nil
}

// ─── Todo service depends on TodoStore ───────────────────────────────────────

type TodoService struct {
	store TodoStore
}

func NewTodoService(store TodoStore) *TodoService {
	return &TodoService{store: store}
}

func (s *TodoService) Add(text string) (Todo, error) {
	t := Todo{Text: text}
	if err := s.store.Save(t); err != nil {
		return Todo{}, err
	}
	// Re-fetch to get the assigned ID (MemoryStore sets it on Save).
	all, _ := s.store.List()
	for _, item := range all {
		if item.Text == text && !item.Done {
			return item, nil
		}
	}
	return t, nil
}

func (s *TodoService) Complete(id int) error {
	t, err := s.store.FindByID(id)
	if err != nil {
		return err
	}
	t.Done = true
	return s.store.Save(t)
}

func (s *TodoService) Remove(id int) error {
	return s.store.Delete(id)
}

func (s *TodoService) ListPending() ([]Todo, error) {
	all, err := s.store.List()
	if err != nil {
		return nil, err
	}
	var pending []Todo
	for _, t := range all {
		if !t.Done {
			pending = append(pending, t)
		}
	}
	return pending, nil
}

func main() {
	store := NewMemoryStore()
	svc := NewTodoService(store)

	t1, _ := svc.Add("Buy groceries")
	t2, _ := svc.Add("Write Go code")
	t3, _ := svc.Add("Go for a run")

	fmt.Printf("Added: id=%d %q\n", t1.ID, t1.Text)
	fmt.Printf("Added: id=%d %q\n", t2.ID, t2.Text)
	fmt.Printf("Added: id=%d %q\n", t3.ID, t3.Text)

	_ = svc.Complete(t2.ID)

	pending, _ := svc.ListPending()
	fmt.Printf("\nPending (%d):\n", len(pending))
	for _, t := range pending {
		fmt.Printf("  [%d] %s\n", t.ID, t.Text)
	}

	_ = svc.Remove(t3.ID)
	fmt.Println("\nAfter removing id=3:")
	pending, _ = svc.ListPending()
	for _, t := range pending {
		fmt.Printf("  [%d] %s\n", t.ID, t.Text)
	}

	// Verify error handling
	_, err := store.FindByID(99)
	fmt.Println("\nFind missing:", err)
	fmt.Println("Is not-found:", errors.Is(err, fmt.Errorf("todo 99 not found")))
}
