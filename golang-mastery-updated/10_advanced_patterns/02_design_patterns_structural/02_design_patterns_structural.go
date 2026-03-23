// 02_design_patterns_structural.go
//
// Structural Design Patterns in Go
// ==================================
// Structural patterns describe how to compose objects and interfaces
// to form larger, more flexible structures.
//
// In Go, structural patterns are natural because:
//   - Interface satisfaction is implicit (duck typing)
//   - Embedding provides composition without inheritance
//   - Any type can wrap any other type
//
// Patterns covered:
//   1. Decorator  — wrap an interface to add behavior transparently
//   2. Adapter    — make incompatible interfaces compatible
//   3. Proxy      — same interface, different behavior (cache, auth, log)
//   4. Composite  — tree structures where leaves and branches are uniform
//   5. Facade     — simple API over a complex subsystem

package main

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// =============================================================================
// PATTERN 1: DECORATOR
// =============================================================================
//
// Intent: Attach additional behavior to an object dynamically.
//         Decorators are an alternative to subclassing for extending behavior.
//
// In Go: A decorator is a struct that:
//   - Holds a reference to the interface it decorates
//   - Implements the same interface
//   - Calls through to the wrapped object, adding behavior before/after
//
// Real-world uses:
//   - HTTP middleware (the most common Go decorator)
//   - Logging wrappers around service interfaces
//   - Retry wrappers around network clients
//   - Metrics wrappers (count calls, measure latency)
//   - Caching wrappers
//
// Key insight: decorators are composable. You can stack them:
//   svc = NewLogging(NewRetry(NewMetrics(realService)))
// Each layer adds its concern without modifying the original.

// DataSource is the interface we'll decorate.
// Could be a file, database, S3 bucket — callers don't care.
type DataSource interface {
	WriteData(data string) error
	ReadData() (string, error)
}

// FileDataSource is the real implementation.
type FileDataSource struct {
	filename string
	content  string // in-memory for demo purposes
}

func NewFileDataSource(filename string) *FileDataSource {
	return &FileDataSource{filename: filename}
}

func (f *FileDataSource) WriteData(data string) error {
	f.content = data
	return nil
}

func (f *FileDataSource) ReadData() (string, error) {
	return f.content, nil
}

// EncryptionDecorator transparently encrypts/decrypts data.
// Notice it embeds DataSource (the interface), not the concrete type.
// This is the power of the decorator: it works with ANY DataSource.
type EncryptionDecorator struct {
	wrapped DataSource // the component being decorated
}

func NewEncryptionDecorator(ds DataSource) *EncryptionDecorator {
	return &EncryptionDecorator{wrapped: ds}
}

// A trivially simple "cipher" for demo. Real code uses crypto/aes.
func encrypt(data string) string {
	runes := []rune(data)
	for i, r := range runes {
		runes[i] = r + 1
	}
	return string(runes)
}

func decrypt(data string) string {
	runes := []rune(data)
	for i, r := range runes {
		runes[i] = r - 1
	}
	return string(runes)
}

func (e *EncryptionDecorator) WriteData(data string) error {
	encrypted := encrypt(data)
	fmt.Printf("  [encryption] encrypting before write: %q → %q\n", data, encrypted)
	return e.wrapped.WriteData(encrypted) // delegate to wrapped
}

func (e *EncryptionDecorator) ReadData() (string, error) {
	data, err := e.wrapped.ReadData() // delegate to wrapped
	if err != nil {
		return "", err
	}
	decrypted := decrypt(data)
	fmt.Printf("  [encryption] decrypting after read: %q → %q\n", data, decrypted)
	return decrypted, nil
}

// CompressionDecorator adds compression on top of whatever it wraps.
// It doesn't know or care whether the wrapped object also encrypts.
type CompressionDecorator struct {
	wrapped DataSource
}

func NewCompressionDecorator(ds DataSource) *CompressionDecorator {
	return &CompressionDecorator{wrapped: ds}
}

// Trivial "compression" for demo purposes.
func compress(data string) string   { return "[COMPRESSED]" + data }
func decompress(data string) string { return strings.TrimPrefix(data, "[COMPRESSED]") }

func (c *CompressionDecorator) WriteData(data string) error {
	compressed := compress(data)
	fmt.Printf("  [compression] compressing before write (%d→%d bytes)\n",
		len(data), len(compressed))
	return c.wrapped.WriteData(compressed)
}

func (c *CompressionDecorator) ReadData() (string, error) {
	data, err := c.wrapped.ReadData()
	if err != nil {
		return "", err
	}
	decompressed := decompress(data)
	fmt.Printf("  [compression] decompressing after read (%d→%d bytes)\n",
		len(data), len(decompressed))
	return decompressed, nil
}

// =============================================================================
// PATTERN 2: ADAPTER
// =============================================================================
//
// Intent: Make one interface work where a different interface is expected.
//         The adapter converts one interface into another.
//
// Analogy: A power plug adapter — the device hasn't changed, but now it
//          fits into a different socket.
//
// When to use:
//   - Integrating a third-party library whose interface doesn't match yours
//   - Wrapping legacy code with a modern interface
//   - Making two independently designed systems interoperate
//
// Object Adapter vs Class Adapter:
//   Go only has "object adapter" (holds reference to adaptee).
//   "Class adapter" (inheriting from adaptee) doesn't exist in Go.

// Our application's logging interface (Target interface).
// This is what our code expects.
type Logger interface {
	Log(level, message string)
}

// ThirdPartyLogger is a library we must use but can't modify.
// Its interface is completely different from ours.
type ThirdPartyLogger struct {
	prefix string
}

// These methods don't match our Logger interface.
func (t *ThirdPartyLogger) EmitInfo(msg string) {
	fmt.Printf("[%s][INFO] %s\n", t.prefix, msg)
}
func (t *ThirdPartyLogger) EmitError(msg string) {
	fmt.Printf("[%s][ERROR] %s\n", t.prefix, msg)
}
func (t *ThirdPartyLogger) EmitWarn(msg string) {
	fmt.Printf("[%s][WARN] %s\n", t.prefix, msg)
}

// LoggerAdapter adapts ThirdPartyLogger to our Logger interface.
// It wraps the adaptee and translates calls.
type LoggerAdapter struct {
	adaptee *ThirdPartyLogger // the object being adapted
}

func NewLoggerAdapter(prefix string) *LoggerAdapter {
	return &LoggerAdapter{adaptee: &ThirdPartyLogger{prefix: prefix}}
}

// Log implements our Logger interface by delegating to ThirdPartyLogger's methods.
func (a *LoggerAdapter) Log(level, message string) {
	switch strings.ToLower(level) {
	case "info":
		a.adaptee.EmitInfo(message)
	case "error":
		a.adaptee.EmitError(message)
	case "warn":
		a.adaptee.EmitWarn(message)
	default:
		a.adaptee.EmitInfo(fmt.Sprintf("[%s] %s", level, message))
	}
}

// Application uses only Logger interface — it's decoupled from ThirdPartyLogger.
type Application struct {
	logger Logger
}

func NewApplication(logger Logger) *Application {
	return &Application{logger: logger}
}

func (app *Application) Start() {
	app.logger.Log("info", "application starting")
	app.logger.Log("warn", "config file not found, using defaults")
	app.logger.Log("info", "application started")
}

// =============================================================================
// PATTERN 3: PROXY
// =============================================================================
//
// Intent: Provide a surrogate or placeholder for another object.
//         The proxy controls access to the real object.
//
// Proxy vs Decorator:
//   - Decorator adds behavior and always delegates
//   - Proxy controls access (may not delegate, may block, may cache)
//   - Both implement the same interface — the distinction is intent
//
// Proxy variants:
//   - Virtual Proxy: lazy initialization (don't create until needed)
//   - Protection Proxy: access control (auth checks)
//   - Caching Proxy: memoize results (avoid redundant calls)
//   - Remote Proxy: represent object in another address space (gRPC client)
//   - Logging Proxy: record all method calls for audit

// ImageLoader is the service interface.
type ImageLoader interface {
	LoadImage(url string) ([]byte, error)
}

// RealImageLoader does the actual HTTP call.
type RealImageLoader struct{}

func (r *RealImageLoader) LoadImage(url string) ([]byte, error) {
	// Simulate a network call.
	fmt.Printf("  [real loader] fetching from network: %s\n", url)
	time.Sleep(5 * time.Millisecond) // simulate latency
	return []byte(fmt.Sprintf("<image data from %s>", url)), nil
}

// CachingProxy sits in front of RealImageLoader and caches results.
// Callers use ImageLoader interface — they never know a proxy is involved.
type CachingProxy struct {
	real  ImageLoader
	cache map[string][]byte
	hits  int
	misses int
}

func NewCachingProxy(real ImageLoader) *CachingProxy {
	return &CachingProxy{
		real:  real,
		cache: make(map[string][]byte),
	}
}

func (p *CachingProxy) LoadImage(url string) ([]byte, error) {
	if data, ok := p.cache[url]; ok {
		p.hits++
		fmt.Printf("  [cache proxy] cache HIT for %s (hits=%d)\n", url, p.hits)
		return data, nil
	}

	p.misses++
	fmt.Printf("  [cache proxy] cache MISS for %s (misses=%d)\n", url, p.misses)
	data, err := p.real.LoadImage(url)
	if err != nil {
		return nil, err
	}
	p.cache[url] = data
	return data, nil
}

// AuthProxy adds authorization before delegating.
type AuthProxy struct {
	real  ImageLoader
	token string
}

func NewAuthProxy(real ImageLoader, token string) *AuthProxy {
	return &AuthProxy{real: real, token: token}
}

func (p *AuthProxy) LoadImage(url string) ([]byte, error) {
	if p.token == "" {
		return nil, fmt.Errorf("auth proxy: no token provided")
	}
	if strings.HasPrefix(url, "private://") && p.token != "secret" {
		return nil, fmt.Errorf("auth proxy: unauthorized access to %s", url)
	}
	fmt.Printf("  [auth proxy] authorized (token=%s), delegating\n", p.token)
	return p.real.LoadImage(url)
}

// Stacking proxies: auth → cache → real
// The auth proxy wraps the caching proxy which wraps the real loader.

// =============================================================================
// PATTERN 4: COMPOSITE
// =============================================================================
//
// Intent: Compose objects into tree structures. Let clients treat individual
//         objects (leaves) and compositions (branches) uniformly.
//
// Classic example: filesystem (files = leaves, directories = branches).
// Both implement the same interface: you can call Size() on a file or directory.
//
// In Go: define a Component interface that both Leaf and Composite implement.
//
// When to use:
//   - File systems
//   - UI widget hierarchies
//   - Organization charts
//   - Expression trees (arithmetic expressions)
//   - HTML/XML document trees

// FileSystemComponent is the component interface.
// Both files and directories implement this.
type FileSystemComponent interface {
	Name() string
	Size() int64
	Print(indent int)
}

// File is a leaf node — no children.
type File struct {
	name string
	size int64
}

func NewFile(name string, size int64) *File {
	return &File{name: name, size: size}
}

func (f *File) Name() string { return f.name }
func (f *File) Size() int64  { return f.size }
func (f *File) Print(indent int) {
	fmt.Printf("%s📄 %s (%d bytes)\n", strings.Repeat("  ", indent), f.name, f.size)
}

// Directory is a composite node — contains other components.
type Directory struct {
	name     string
	children []FileSystemComponent // can hold files OR other directories
}

func NewDirectory(name string) *Directory {
	return &Directory{name: name}
}

func (d *Directory) Add(component FileSystemComponent) {
	d.children = append(d.children, component)
}

func (d *Directory) Name() string { return d.name }

// Size() recursively sums children — works uniformly for files and dirs.
func (d *Directory) Size() int64 {
	var total int64
	for _, child := range d.children {
		total += child.Size() // polymorphic call
	}
	return total
}

func (d *Directory) Print(indent int) {
	fmt.Printf("%s📁 %s/ (%d bytes total)\n",
		strings.Repeat("  ", indent), d.name, d.Size())
	for _, child := range d.children {
		child.Print(indent + 1) // recursive — works for nested dirs
	}
}

// =============================================================================
// PATTERN 5: FACADE
// =============================================================================
//
// Intent: Provide a simplified interface to a complex subsystem.
//         The facade hides the complexity and presents a clean API.
//
// Facade vs Adapter:
//   - Adapter makes incompatible interfaces compatible (1:1 mapping)
//   - Facade simplifies a complex subsystem (many classes → one interface)
//
// Real-world examples:
//   - net/http is a facade over sockets, TLS, HTTP parsing, connection pooling
//   - A service layer is a facade over repositories, caches, external APIs
//   - AWS SDK clients are facades over HTTP, JSON, auth, retries
//
// The facade doesn't prevent advanced users from accessing subsystems directly.
// It just provides a simpler path for the common case.

// --- Subsystem components (complex, internal) ---

// VideoEncoder encodes video (complex subsystem component 1).
type VideoEncoder struct{}

func (v *VideoEncoder) Initialize(codec string) error {
	fmt.Printf("  [VideoEncoder] initializing with codec: %s\n", codec)
	return nil
}
func (v *VideoEncoder) Encode(input, output string) error {
	fmt.Printf("  [VideoEncoder] encoding %s → %s\n", input, output)
	return nil
}
func (v *VideoEncoder) Shutdown() {
	fmt.Println("  [VideoEncoder] shutting down")
}

// AudioMixer handles audio processing (complex subsystem component 2).
type AudioMixer struct{}

func (a *AudioMixer) LoadTrack(path string) error {
	fmt.Printf("  [AudioMixer] loading audio track: %s\n", path)
	return nil
}
func (a *AudioMixer) SetVolume(level float64) {
	fmt.Printf("  [AudioMixer] setting volume to %.0f%%\n", level*100)
}
func (a *AudioMixer) Mix(output string) error {
	fmt.Printf("  [AudioMixer] mixing to: %s\n", output)
	return nil
}

// ThumbnailGenerator creates preview images (complex subsystem component 3).
type ThumbnailGenerator struct{}

func (t *ThumbnailGenerator) Extract(videoPath string, timestamp float64) error {
	fmt.Printf("  [ThumbnailGenerator] extracting frame at %.1fs from %s\n",
		timestamp, videoPath)
	return nil
}
func (t *ThumbnailGenerator) Resize(width, height int) error {
	fmt.Printf("  [ThumbnailGenerator] resizing to %dx%d\n", width, height)
	return nil
}
func (t *ThumbnailGenerator) Save(path string) error {
	fmt.Printf("  [ThumbnailGenerator] saving thumbnail: %s\n", path)
	return nil
}

// --- The Facade ---

// VideoProcessingFacade is the simplified API.
// Users don't need to know about VideoEncoder, AudioMixer, or ThumbnailGenerator.
type VideoProcessingFacade struct {
	encoder   *VideoEncoder
	mixer     *AudioMixer
	thumbnailer *ThumbnailGenerator
}

func NewVideoProcessingFacade() *VideoProcessingFacade {
	return &VideoProcessingFacade{
		encoder:     &VideoEncoder{},
		mixer:       &AudioMixer{},
		thumbnailer: &ThumbnailGenerator{},
	}
}

// ProcessVideo is the single simplified method.
// It orchestrates all subsystems internally.
func (f *VideoProcessingFacade) ProcessVideo(inputVideo, audioTrack, outputDir string) error {
	fmt.Println("  [Facade] starting video processing pipeline...")

	// Coordinate all subsystems. Users don't see this complexity.
	if err := f.encoder.Initialize("h264"); err != nil {
		return fmt.Errorf("encoder init: %w", err)
	}

	if err := f.mixer.LoadTrack(audioTrack); err != nil {
		return fmt.Errorf("audio load: %w", err)
	}
	f.mixer.SetVolume(0.85)
	if err := f.mixer.Mix(outputDir + "/mixed_audio.aac"); err != nil {
		return fmt.Errorf("audio mix: %w", err)
	}

	if err := f.encoder.Encode(inputVideo, outputDir+"/video.mp4"); err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	if err := f.thumbnailer.Extract(inputVideo, 5.0); err != nil {
		return fmt.Errorf("thumbnail extract: %w", err)
	}
	if err := f.thumbnailer.Resize(320, 180); err != nil {
		return fmt.Errorf("thumbnail resize: %w", err)
	}
	if err := f.thumbnailer.Save(outputDir + "/thumb.jpg"); err != nil {
		return fmt.Errorf("thumbnail save: %w", err)
	}

	f.encoder.Shutdown()
	fmt.Println("  [Facade] processing complete!")
	return nil
}

// =============================================================================
// MAIN
// =============================================================================

func main() {
	fmt.Println("=== STRUCTURAL DESIGN PATTERNS IN GO ===")
	fmt.Println()

	// ------------------------------------------------------------------
	// 1. DECORATOR
	// ------------------------------------------------------------------
	fmt.Println("--- 1. DECORATOR ---")
	fmt.Println("  Stack: Compression(Encryption(File))")

	// Build the decorated stack: file ← encrypt ← compress
	// Writes go through: compress → encrypt → file
	// Reads come back:  file → decrypt → decompress
	file := NewFileDataSource("data.bin")
	encrypted := NewEncryptionDecorator(file)
	compressed := NewCompressionDecorator(encrypted)

	// Write through the full stack.
	fmt.Println("  Writing 'Hello, World!':")
	if err := compressed.WriteData("Hello, World!"); err != nil {
		fmt.Println("  Error:", err)
	}

	// Read through the full stack (decorators apply in reverse order).
	fmt.Println("  Reading back:")
	data, _ := compressed.ReadData()
	fmt.Printf("  Final data: %q\n", data)
	fmt.Println()

	// ------------------------------------------------------------------
	// 2. ADAPTER
	// ------------------------------------------------------------------
	fmt.Println("--- 2. ADAPTER ---")
	fmt.Println("  Adapting ThirdPartyLogger to Logger interface:")

	// Create adapter — application only knows Logger interface.
	logger := NewLoggerAdapter("MyApp")
	app := NewApplication(logger)
	app.Start()
	fmt.Println()

	// ------------------------------------------------------------------
	// 3. PROXY
	// ------------------------------------------------------------------
	fmt.Println("--- 3. PROXY ---")

	real := &RealImageLoader{}
	cache := NewCachingProxy(real)

	// Stack: auth → cache → real
	auth := NewAuthProxy(cache, "secret")

	fmt.Println("  Loading images (first time — cache miss):")
	auth.LoadImage("https://cdn.example.com/logo.png")
	auth.LoadImage("https://cdn.example.com/banner.jpg")

	fmt.Println("  Loading same images again (cache hit):")
	auth.LoadImage("https://cdn.example.com/logo.png")
	auth.LoadImage("https://cdn.example.com/logo.png")

	fmt.Println("  Unauthorized access attempt:")
	authWeak := NewAuthProxy(cache, "wrong-token")
	_, err := authWeak.LoadImage("private://secret-doc.pdf")
	fmt.Println("  Error (expected):", err)

	fmt.Printf("  Cache stats: hits=%d, misses=%d\n", cache.hits, cache.misses)
	fmt.Println()

	// ------------------------------------------------------------------
	// 4. COMPOSITE
	// ------------------------------------------------------------------
	fmt.Println("--- 4. COMPOSITE (File System Tree) ---")

	// Build a directory tree.
	root := NewDirectory("project")

	src := NewDirectory("src")
	src.Add(NewFile("main.go", 1024))
	src.Add(NewFile("handler.go", 2048))
	src.Add(NewFile("service.go", 3072))

	tests := NewDirectory("tests")
	tests.Add(NewFile("main_test.go", 512))
	tests.Add(NewFile("handler_test.go", 1024))

	assets := NewDirectory("assets")
	images := NewDirectory("images")
	images.Add(NewFile("logo.png", 51200))
	images.Add(NewFile("banner.jpg", 102400))
	assets.Add(images)
	assets.Add(NewFile("style.css", 8192))

	root.Add(src)
	root.Add(tests)
	root.Add(assets)
	root.Add(NewFile("go.mod", 256))
	root.Add(NewFile("README.md", 4096))

	// Print the entire tree — same Print() call for files and directories.
	root.Print(0)
	fmt.Printf("  Total project size: %d bytes (%.1f KB)\n",
		root.Size(), float64(root.Size())/1024)

	// Demonstrate uniformity: both File and Directory are FileSystemComponent.
	components := []FileSystemComponent{
		NewFile("single_file.go", 500),
		root, // the entire tree is just another component
	}
	fmt.Println("  Uniform treatment of file and directory:")
	for _, c := range components {
		fmt.Printf("    %s → size=%d\n", c.Name(), c.Size())
	}

	// Demonstrate math.Sqrt to avoid unused import
	_ = math.Sqrt(2.0)
	fmt.Println()

	// ------------------------------------------------------------------
	// 5. FACADE
	// ------------------------------------------------------------------
	fmt.Println("--- 5. FACADE ---")
	fmt.Println("  Simple API call hides complex subsystem orchestration:")

	facade := NewVideoProcessingFacade()
	err = facade.ProcessVideo(
		"input/raw_video.mp4",
		"input/background_music.mp3",
		"output/",
	)
	if err != nil {
		fmt.Println("  Processing failed:", err)
	}

	fmt.Println()
	fmt.Println("=== END STRUCTURAL PATTERNS ===")
}
