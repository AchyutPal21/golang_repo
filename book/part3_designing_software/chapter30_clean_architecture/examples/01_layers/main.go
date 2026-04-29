// FILE: book/part3_designing_software/chapter30_clean_architecture/examples/01_layers/main.go
// CHAPTER: 30 — Clean / Hexagonal Architecture
// TOPIC: Four-layer architecture — domain, application, infrastructure, transport.
//        Dependencies always point inward; the domain has no imports.
//
// Run (from the chapter folder):
//   go run ./examples/01_layers

package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// LAYER 1 — DOMAIN
//
// Pure business entities and rules. No imports from other layers.
// No framework, no database, no HTTP. Just Go structs and errors.
// ─────────────────────────────────────────────────────────────────────────────

// Article is the core domain entity.
type Article struct {
	ID          string
	Title       string
	Body        string
	AuthorEmail string
	PublishedAt *time.Time // nil = draft
}

func (a Article) IsDraft() bool { return a.PublishedAt == nil }

// Domain errors — defined in the domain, not infrastructure.
var (
	ErrEmptyTitle   = errors.New("title cannot be empty")
	ErrEmptyBody    = errors.New("body cannot be empty")
	ErrNotFound     = errors.New("article not found")
	ErrAlreadyPublished = errors.New("article already published")
)

// NewArticle enforces domain invariants at construction.
func NewArticle(id, title, body, authorEmail string) (Article, error) {
	if strings.TrimSpace(title) == "" {
		return Article{}, ErrEmptyTitle
	}
	if strings.TrimSpace(body) == "" {
		return Article{}, ErrEmptyBody
	}
	return Article{ID: id, Title: title, Body: body, AuthorEmail: authorEmail}, nil
}

// Publish is a domain operation — the rule "already published" is domain logic.
func (a *Article) Publish(now time.Time) error {
	if !a.IsDraft() {
		return ErrAlreadyPublished
	}
	a.PublishedAt = &now
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// LAYER 2 — APPLICATION (USE CASES)
//
// Orchestrates domain objects. Defines the ports (interfaces) it needs.
// Depends on domain; does NOT depend on infrastructure or transport.
// ─────────────────────────────────────────────────────────────────────────────

// ArticleRepository is a port — defined here, implemented in infrastructure.
type ArticleRepository interface {
	Save(a Article) error
	FindByID(id string) (Article, error)
	All() ([]Article, error)
}

// IDGenerator is a port for ID creation — injectable for tests.
type IDGenerator interface {
	Next() string
}

// Clock is a port for time — injectable for deterministic tests.
type Clock interface {
	Now() time.Time
}

// PublishNotifier is an outbound port — infrastructure decides how to notify.
type PublishNotifier interface {
	NotifyPublished(a Article) error
}

// ArticleService is the application use-case layer.
type ArticleService struct {
	repo     ArticleRepository
	ids      IDGenerator
	clock    Clock
	notifier PublishNotifier
}

func NewArticleService(
	repo ArticleRepository,
	ids IDGenerator,
	clock Clock,
	notifier PublishNotifier,
) *ArticleService {
	return &ArticleService{repo: repo, ids: ids, clock: clock, notifier: notifier}
}

func (s *ArticleService) Draft(title, body, authorEmail string) (Article, error) {
	a, err := NewArticle(s.ids.Next(), title, body, authorEmail)
	if err != nil {
		return Article{}, fmt.Errorf("Draft: %w", err)
	}
	if err := s.repo.Save(a); err != nil {
		return Article{}, fmt.Errorf("Draft: %w", err)
	}
	return a, nil
}

func (s *ArticleService) Publish(id string) (Article, error) {
	a, err := s.repo.FindByID(id)
	if err != nil {
		return Article{}, fmt.Errorf("Publish: %w", err)
	}
	if err := a.Publish(s.clock.Now()); err != nil {
		return Article{}, fmt.Errorf("Publish: %w", err)
	}
	if err := s.repo.Save(a); err != nil {
		return Article{}, fmt.Errorf("Publish: %w", err)
	}
	_ = s.notifier.NotifyPublished(a) // best-effort; don't fail the operation
	return a, nil
}

func (s *ArticleService) List() ([]Article, error) {
	return s.repo.All()
}

// ─────────────────────────────────────────────────────────────────────────────
// LAYER 3 — INFRASTRUCTURE (ADAPTERS)
//
// Concrete implementations of the application ports.
// Depends on application (implements its interfaces); domain types pass through.
// ─────────────────────────────────────────────────────────────────────────────

type memArticleRepo struct{ articles map[string]Article }

func newMemArticleRepo() *memArticleRepo {
	return &memArticleRepo{articles: make(map[string]Article)}
}

func (r *memArticleRepo) Save(a Article) error {
	r.articles[a.ID] = a
	return nil
}

func (r *memArticleRepo) FindByID(id string) (Article, error) {
	a, ok := r.articles[id]
	if !ok {
		return Article{}, ErrNotFound
	}
	return a, nil
}

func (r *memArticleRepo) All() ([]Article, error) {
	out := make([]Article, 0, len(r.articles))
	for _, a := range r.articles {
		out = append(out, a)
	}
	return out, nil
}

type seqIDGenerator struct{ n int }

func (g *seqIDGenerator) Next() string {
	g.n++
	return fmt.Sprintf("ART-%04d", g.n)
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

type stdoutNotifier struct{}

func (stdoutNotifier) NotifyPublished(a Article) error {
	fmt.Printf("  [NOTIFY] article %q published at %s\n",
		a.Title, a.PublishedAt.Format(time.RFC3339))
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// LAYER 4 — TRANSPORT (ENTRY POINTS)
//
// Translates external inputs into application use-case calls.
// In a real app this would be HTTP handlers, gRPC, CLI flags, etc.
// Here: a simple CLI-style runner.
// ─────────────────────────────────────────────────────────────────────────────

type CLI struct{ svc *ArticleService }

func (c *CLI) Run() {
	fmt.Println("=== draft two articles ===")
	a1, err := c.svc.Draft("Go Interfaces", "Interfaces are implicit in Go...", "alice@example.com")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("  created %s: %q (draft=%v)\n", a1.ID, a1.Title, a1.IsDraft())

	a2, err := c.svc.Draft("", "body without title", "bob@example.com")
	if err != nil {
		fmt.Println("  domain error (expected):", err)
	} else {
		fmt.Println("  unexpected success:", a2.ID)
	}

	a3, _ := c.svc.Draft("Clean Arch", "Layers keep concerns separated.", "carol@example.com")
	fmt.Printf("  created %s: %q (draft=%v)\n", a3.ID, a3.Title, a3.IsDraft())

	fmt.Println()
	fmt.Println("=== publish first article ===")
	published, err := c.svc.Publish(a1.ID)
	if err != nil {
		fmt.Println("error:", err)
	} else {
		fmt.Printf("  %s is now published\n", published.ID)
	}

	fmt.Println()
	fmt.Println("=== publish again (idempotency error) ===")
	_, err = c.svc.Publish(a1.ID)
	fmt.Println("  expected error:", err)

	fmt.Println()
	fmt.Println("=== list all articles ===")
	articles, _ := c.svc.List()
	for _, a := range articles {
		status := "draft"
		if !a.IsDraft() {
			status = "published"
		}
		fmt.Printf("  %s  %-20s  %s\n", a.ID, a.Title, status)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// COMPOSITION ROOT — main()
//
// The only place where all layers are imported and wired together.
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	repo := newMemArticleRepo()
	ids := &seqIDGenerator{}
	clock := realClock{}
	notifier := stdoutNotifier{}

	svc := NewArticleService(repo, ids, clock, notifier)
	cli := &CLI{svc: svc}
	cli.Run()
}
