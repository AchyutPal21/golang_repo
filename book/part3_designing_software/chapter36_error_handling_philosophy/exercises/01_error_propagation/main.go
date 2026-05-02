// EXERCISE 36.1 — Trace error propagation through a three-layer stack.
//
// A request flows through transport → service → repository.
// Each layer wraps errors with context. The boundary (main) handles the final error
// using errors.Is and errors.As to classify and report it.
//
// Run (from the chapter folder):
//   go run ./exercises/01_error_propagation

package main

import (
	"errors"
	"fmt"
	"strings"
)

// ─── Domain sentinels ─────────────────────────────────────────────────────────

var (
	ErrNotFound   = errors.New("not found")
	ErrForbidden  = errors.New("forbidden")
	ErrConflict   = errors.New("conflict")
)

// ─── Domain types ─────────────────────────────────────────────────────────────

type Post struct {
	ID      int
	Title   string
	Body    string
	Author  string
	Private bool
}

// ─── Repository layer ─────────────────────────────────────────────────────────

type PostRepo struct{ data map[int]Post }

func newPostRepo() *PostRepo {
	return &PostRepo{data: map[int]Post{
		1: {1, "Go Interfaces", "Interfaces are implicit.", "alice", false},
		2: {2, "Secret Notes", "Internal only.", "alice", true},
	}}
}

func (r *PostRepo) FindByID(id int) (Post, error) {
	p, ok := r.data[id]
	if !ok {
		return Post{}, fmt.Errorf("PostRepo.FindByID id=%d: %w", id, ErrNotFound)
	}
	return p, nil
}

func (r *PostRepo) Save(p Post) error {
	if _, exists := r.data[p.ID]; exists {
		return fmt.Errorf("PostRepo.Save id=%d: %w", p.ID, ErrConflict)
	}
	r.data[p.ID] = p
	return nil
}

// ─── Service layer ────────────────────────────────────────────────────────────

type PostService struct{ repo *PostRepo }

func (s *PostService) GetPost(callerUser string, id int) (Post, error) {
	post, err := s.repo.FindByID(id)
	if err != nil {
		return Post{}, fmt.Errorf("PostService.GetPost: %w", err)
	}
	if post.Private && post.Author != callerUser {
		return Post{}, fmt.Errorf("PostService.GetPost id=%d caller=%s: %w",
			id, callerUser, ErrForbidden)
	}
	return post, nil
}

func (s *PostService) CreatePost(author, title, body string) (Post, error) {
	if strings.TrimSpace(title) == "" {
		return Post{}, fmt.Errorf("PostService.CreatePost: title is required")
	}
	// Use len(data)+1 as ID — simplified for example.
	id := len(s.repo.data) + 1
	p := Post{ID: id, Title: title, Body: body, Author: author}
	if err := s.repo.Save(p); err != nil {
		return Post{}, fmt.Errorf("PostService.CreatePost: %w", err)
	}
	return p, nil
}

// ─── Transport layer (simulated HTTP handler) ─────────────────────────────────

type HTTPResponse struct {
	Status int
	Body   string
}

func handleGetPost(svc *PostService, userID string, postID int) HTTPResponse {
	post, err := svc.GetPost(userID, postID)
	if err == nil {
		return HTTPResponse{200, fmt.Sprintf(`{"id":%d,"title":%q}`, post.ID, post.Title)}
	}

	// Classify the error at the boundary.
	switch {
	case errors.Is(err, ErrNotFound):
		return HTTPResponse{404, `{"error":"not found"}`}
	case errors.Is(err, ErrForbidden):
		return HTTPResponse{403, `{"error":"forbidden"}`}
	default:
		fmt.Println("  [LOG] unexpected error:", err)
		return HTTPResponse{500, `{"error":"internal server error"}`}
	}
}

func handleCreatePost(svc *PostService, userID, title, body string) HTTPResponse {
	post, err := svc.CreatePost(userID, title, body)
	if err == nil {
		return HTTPResponse{201, fmt.Sprintf(`{"id":%d,"title":%q}`, post.ID, post.Title)}
	}

	switch {
	case errors.Is(err, ErrConflict):
		return HTTPResponse{409, `{"error":"conflict"}`}
	default:
		// Non-sentinel → likely a validation error; return 400.
		return HTTPResponse{400, fmt.Sprintf(`{"error":%q}`, err.Error())}
	}
}

func printResp(label string, resp HTTPResponse) {
	fmt.Printf("  %-40s  HTTP %d  %s\n", label, resp.Status, resp.Body)
}

func main() {
	repo := newPostRepo()
	svc := &PostService{repo: repo}

	fmt.Println("=== GET requests ===")
	printResp("alice reads public post 1", handleGetPost(svc, "alice", 1))
	printResp("bob reads public post 1", handleGetPost(svc, "bob", 1))
	printResp("alice reads her private post 2", handleGetPost(svc, "alice", 2))
	printResp("bob reads alice's private post 2", handleGetPost(svc, "bob", 2))
	printResp("anyone reads missing post 99", handleGetPost(svc, "alice", 99))

	fmt.Println()
	fmt.Println("=== POST requests ===")
	printResp("create valid post", handleCreatePost(svc, "bob", "New Post", "body"))
	printResp("create with empty title", handleCreatePost(svc, "bob", "", "body"))

	fmt.Println()
	fmt.Println("=== errors.Is chain unwrapping ===")
	_, err := svc.GetPost("bob", 2)
	fmt.Println("  raw error:", err)
	fmt.Println("  Is ErrForbidden:", errors.Is(err, ErrForbidden))
	fmt.Println("  Is ErrNotFound: ", errors.Is(err, ErrNotFound))
}
