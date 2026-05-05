// FILE: book/part7_capstone_projects/capstone_g_file_upload/main.go
// CAPSTONE G — File Upload Service
// Self-contained simulation: single upload, resumable multipart upload,
// content validation, virus-scan hook, and S3-compatible storage interface.
//
// Run:
//   go run ./book/part7_capstone_projects/capstone_g_file_upload

package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// STORAGE BACKEND INTERFACE
// ─────────────────────────────────────────────────────────────────────────────

type Object struct {
	Key         string
	ContentType string
	Size        int64
	Data        []byte
	UploadedAt  time.Time
}

type StorageBackend interface {
	Put(key, contentType string, r io.Reader) (Object, error)
	Get(key string) (Object, bool)
	Delete(key string) error
	List() []Object
}

type memoryStorage struct {
	mu      sync.RWMutex
	objects map[string]Object
}

func newMemoryStorage() *memoryStorage { return &memoryStorage{objects: map[string]Object{}} }

func (s *memoryStorage) Put(key, contentType string, r io.Reader) (Object, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return Object{}, err
	}
	obj := Object{Key: key, ContentType: contentType, Size: int64(len(data)), Data: data, UploadedAt: time.Now()}
	s.mu.Lock()
	s.objects[key] = obj
	s.mu.Unlock()
	return obj, nil
}

func (s *memoryStorage) Get(key string) (Object, bool) {
	s.mu.RLock()
	obj, ok := s.objects[key]
	s.mu.RUnlock()
	return obj, ok
}

func (s *memoryStorage) Delete(key string) error {
	s.mu.Lock()
	delete(s.objects, key)
	s.mu.Unlock()
	return nil
}

func (s *memoryStorage) List() []Object {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Object, 0, len(s.objects))
	for _, o := range s.objects {
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// CONTENT VALIDATOR
// ─────────────────────────────────────────────────────────────────────────────

var allowedTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	"application/pdf": true,
	"text/plain":      true,
	"video/mp4":       true,
}

const maxSizeBytes = 50 * 1024 * 1024 // 50 MB

type contentValidator struct{}

func (v *contentValidator) Validate(contentType string, size int64) error {
	if !allowedTypes[contentType] {
		return fmt.Errorf("content type %q not allowed", contentType)
	}
	if size > maxSizeBytes {
		return fmt.Errorf("file too large: %d bytes (max %d)", size, maxSizeBytes)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// VIRUS SCAN HOOK
// ─────────────────────────────────────────────────────────────────────────────

type VirusScanHook func(data []byte) error

func noopScan(_ []byte) error { return nil }

func simulatedVirusScan(data []byte) error {
	// Simulate detecting a "virus signature" in content
	if bytes.Contains(data, []byte("EICAR-VIRUS")) {
		return errors.New("virus detected: EICAR test signature")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// RESUMABLE UPLOAD TRACKER
// ─────────────────────────────────────────────────────────────────────────────

type UploadPart struct {
	Number int
	Data   []byte
}

type UploadSession struct {
	UploadID    string
	Key         string
	ContentType string
	Parts       map[int]UploadPart
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

type uploadTracker struct {
	mu       sync.Mutex
	sessions map[string]*UploadSession
}

func newUploadTracker() *uploadTracker {
	return &uploadTracker{sessions: map[string]*UploadSession{}}
}

func (t *uploadTracker) Init(key, contentType string) string {
	id := generateID()
	t.mu.Lock()
	t.sessions[id] = &UploadSession{
		UploadID:    id,
		Key:         key,
		ContentType: contentType,
		Parts:       map[int]UploadPart{},
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}
	t.mu.Unlock()
	return id
}

func (t *uploadTracker) AddPart(uploadID string, partNum int, data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	s, ok := t.sessions[uploadID]
	if !ok {
		return fmt.Errorf("upload %s not found", uploadID)
	}
	if time.Now().After(s.ExpiresAt) {
		return errors.New("upload session expired")
	}
	s.Parts[partNum] = UploadPart{Number: partNum, Data: data}
	return nil
}

func (t *uploadTracker) Complete(uploadID string) (*UploadSession, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	s, ok := t.sessions[uploadID]
	if !ok {
		return nil, fmt.Errorf("upload %s not found", uploadID)
	}
	if len(s.Parts) == 0 {
		return nil, errors.New("no parts uploaded")
	}
	delete(t.sessions, uploadID)
	return s, nil
}

func (t *uploadTracker) Abort(uploadID string) {
	t.mu.Lock()
	delete(t.sessions, uploadID)
	t.mu.Unlock()
}

// ─────────────────────────────────────────────────────────────────────────────
// MULTIPART ASSEMBLER
// ─────────────────────────────────────────────────────────────────────────────

func assembleParts(parts map[int]UploadPart) io.Reader {
	nums := make([]int, 0, len(parts))
	for n := range parts {
		nums = append(nums, n)
	}
	sort.Ints(nums)
	var buf bytes.Buffer
	for _, n := range nums {
		buf.Write(parts[n].Data)
	}
	return &buf
}

// ─────────────────────────────────────────────────────────────────────────────
// UPLOAD SERVICE
// ─────────────────────────────────────────────────────────────────────────────

type UploadService struct {
	storage   StorageBackend
	validator *contentValidator
	virusScan VirusScanHook
	tracker   *uploadTracker
}

func NewUploadService(storage StorageBackend, virusScan VirusScanHook) *UploadService {
	return &UploadService{
		storage:   storage,
		validator: &contentValidator{},
		virusScan: virusScan,
		tracker:   newUploadTracker(),
	}
}

// Upload handles a complete single-shot upload.
func (s *UploadService) Upload(key, contentType string, data []byte) (Object, error) {
	if err := s.validator.Validate(contentType, int64(len(data))); err != nil {
		return Object{}, fmt.Errorf("validation: %w", err)
	}
	if err := s.virusScan(data); err != nil {
		return Object{}, fmt.Errorf("scan: %w", err)
	}
	return s.storage.Put(key, contentType, bytes.NewReader(data))
}

// InitResumable starts a multipart upload session.
func (s *UploadService) InitResumable(key, contentType string) string {
	return s.tracker.Init(key, contentType)
}

// UploadPart adds a part to an in-progress resumable upload.
func (s *UploadService) UploadPart(uploadID string, partNum int, data []byte) error {
	return s.tracker.AddPart(uploadID, partNum, data)
}

// CompleteResumable assembles all parts and stores the final object.
func (s *UploadService) CompleteResumable(uploadID string) (Object, error) {
	session, err := s.tracker.Complete(uploadID)
	if err != nil {
		return Object{}, err
	}
	assembled := assembleParts(session.Parts)
	data, _ := io.ReadAll(assembled)
	if err := s.virusScan(data); err != nil {
		return Object{}, fmt.Errorf("scan: %w", err)
	}
	return s.storage.Put(session.Key, session.ContentType, bytes.NewReader(data))
}

// AbortResumable cleans up an incomplete upload.
func (s *UploadService) AbortResumable(uploadID string) {
	s.tracker.Abort(uploadID)
}

// Download retrieves a stored file.
func (s *UploadService) Download(key string) (Object, error) {
	obj, ok := s.storage.Get(key)
	if !ok {
		return Object{}, fmt.Errorf("object %q not found", key)
	}
	return obj, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// HELPERS
// ─────────────────────────────────────────────────────────────────────────────

func generateID() string {
	b := make([]byte, 12)
	rand.Read(b) //nolint:errcheck
	return hex.EncodeToString(b)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Capstone G: File Upload Service ===")
	fmt.Println()

	svc := NewUploadService(newMemoryStorage(), simulatedVirusScan)

	// ── SINGLE UPLOAD ─────────────────────────────────────────────────────────
	fmt.Println("--- Single uploads ---")
	obj, err := svc.Upload("images/profile.jpg", "image/jpeg", bytes.Repeat([]byte("JFIF"), 512))
	fmt.Printf("  Uploaded: key=%s size=%d bytes type=%s\n", obj.Key, obj.Size, obj.ContentType)

	_, err = svc.Upload("script.exe", "application/octet-stream", []byte("binary"))
	fmt.Printf("  Blocked type:  %v\n", err)

	_, err = svc.Upload("docs/report.pdf", "application/pdf", bytes.Repeat([]byte("PDF"), maxSizeBytes/3+1))
	fmt.Printf("  Too large:     %v\n", err)

	_, err = svc.Upload("malware.txt", "text/plain", []byte("some EICAR-VIRUS content here"))
	fmt.Printf("  Virus blocked: %v\n", err)
	fmt.Println()

	// ── RESUMABLE UPLOAD ──────────────────────────────────────────────────────
	fmt.Println("--- Resumable multipart upload ---")
	uploadID := svc.InitResumable("videos/demo.mp4", "video/mp4")
	fmt.Printf("  Upload ID: %s\n", uploadID)

	// Upload 3 parts (simulating a 15MB video split into 5MB chunks)
	for i := 1; i <= 3; i++ {
		part := bytes.Repeat([]byte(fmt.Sprintf("PART%d", i)), 1024)
		err := svc.UploadPart(uploadID, i, part)
		fmt.Printf("  Part %d uploaded: %v\n", i, err)
	}

	assembled, err := svc.CompleteResumable(uploadID)
	fmt.Printf("  Assembled: key=%s size=%d bytes\n", assembled.Key, assembled.Size)
	fmt.Println()

	// ── OUT-OF-ORDER PARTS ────────────────────────────────────────────────────
	fmt.Println("--- Out-of-order part assembly ---")
	uid2 := svc.InitResumable("docs/contract.pdf", "application/pdf")
	// Upload parts out of order: 3, 1, 2
	svc.UploadPart(uid2, 3, []byte("PART_THREE"))  //nolint:errcheck
	svc.UploadPart(uid2, 1, []byte("PART_ONE"))    //nolint:errcheck
	svc.UploadPart(uid2, 2, []byte("PART_TWO"))    //nolint:errcheck
	result, _ := svc.CompleteResumable(uid2)
	content, _ := io.ReadAll(bytes.NewReader(result.Data))
	fmt.Printf("  Assembled content: %s\n", string(content))
	fmt.Println()

	// ── ABORT UPLOAD ──────────────────────────────────────────────────────────
	fmt.Println("--- Abort resumable upload ---")
	uid3 := svc.InitResumable("tmp/draft.pdf", "application/pdf")
	svc.UploadPart(uid3, 1, []byte("PART_ONE")) //nolint:errcheck
	svc.AbortResumable(uid3)
	_, err = svc.CompleteResumable(uid3)
	fmt.Printf("  Complete after abort: %v\n", err)
	fmt.Println()

	// ── DOWNLOAD ──────────────────────────────────────────────────────────────
	fmt.Println("--- Download ---")
	obj2, err := svc.Download("images/profile.jpg")
	fmt.Printf("  Download OK: key=%s size=%d\n", obj2.Key, obj2.Size)
	_, err = svc.Download("nonexistent.jpg")
	fmt.Printf("  Not found:   %v\n", err)
	fmt.Println()

	// ── STORAGE LISTING ───────────────────────────────────────────────────────
	fmt.Println("--- Stored objects ---")
	storage := svc.storage.(*memoryStorage)
	for _, o := range storage.List() {
		fmt.Printf("  %-35s  %s  %d bytes\n", o.Key, o.ContentType, o.Size)
	}
	fmt.Println()

	// ── ALLOWED TYPES ─────────────────────────────────────────────────────────
	fmt.Println("--- Allowed content types ---")
	types := make([]string, 0, len(allowedTypes))
	for t := range allowedTypes {
		types = append(types, t)
	}
	sort.Strings(types)
	fmt.Printf("  %s\n", strings.Join(types, ", "))
}
