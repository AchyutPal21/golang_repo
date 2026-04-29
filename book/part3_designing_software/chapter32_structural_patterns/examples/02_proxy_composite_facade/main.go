// FILE: book/part3_designing_software/chapter32_structural_patterns/examples/02_proxy_composite_facade/main.go
// CHAPTER: 32 — Structural Patterns
// TOPIC: Proxy (controlled access), Composite (tree of same-interface nodes),
//        and Facade (simplified front for a complex subsystem).
//
// Run (from the chapter folder):
//   go run ./examples/02_proxy_composite_facade

package main

import (
	"fmt"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// PROXY
//
// Controls access to an object. Implements the same interface as the real
// subject. Common uses: caching, access control, lazy initialisation.
// ─────────────────────────────────────────────────────────────────────────────

type DataLoader interface {
	Load(key string) (string, error)
}

// realLoader — expensive: simulates a DB or remote API call.
type realLoader struct{ callCount int }

func (r *realLoader) Load(key string) (string, error) {
	r.callCount++
	fmt.Printf("  [DB] loading %q (call #%d)\n", key, r.callCount)
	return fmt.Sprintf("data-for-%s", key), nil
}

// cachingProxy — caches results; delegates to real loader on miss.
type cachingProxy struct {
	inner DataLoader
	cache map[string]string
}

func NewCachingProxy(inner DataLoader) DataLoader {
	return &cachingProxy{inner: inner, cache: make(map[string]string)}
}

func (p *cachingProxy) Load(key string) (string, error) {
	if v, ok := p.cache[key]; ok {
		fmt.Printf("  [CACHE HIT] %q\n", key)
		return v, nil
	}
	v, err := p.inner.Load(key)
	if err != nil {
		return "", err
	}
	p.cache[key] = v
	return v, nil
}

// accessControlProxy — enforces role-based access.
type accessControlProxy struct {
	inner    DataLoader
	allowed  map[string]bool
	userRole string
}

func NewACProxy(inner DataLoader, role string, allowedKeys ...string) DataLoader {
	m := make(map[string]bool)
	for _, k := range allowedKeys {
		m[k] = true
	}
	return &accessControlProxy{inner: inner, allowed: m, userRole: role}
}

func (p *accessControlProxy) Load(key string) (string, error) {
	if !p.allowed[key] {
		return "", fmt.Errorf("access denied: role=%q key=%q", p.userRole, key)
	}
	return p.inner.Load(key)
}

// ─────────────────────────────────────────────────────────────────────────────
// COMPOSITE
//
// Treats individual objects and groups of objects uniformly through a shared
// interface. Builds a tree: leaf nodes do real work; composite nodes delegate
// to their children.
// ─────────────────────────────────────────────────────────────────────────────

type FileSystemNode interface {
	Name() string
	Size() int
	Display(indent int)
}

// File — leaf node.
type File struct {
	name string
	size int
}

func (f *File) Name() string     { return f.name }
func (f *File) Size() int        { return f.size }
func (f *File) Display(indent int) {
	fmt.Printf("%s📄 %s (%d B)\n", strings.Repeat("  ", indent), f.name, f.size)
}

// Directory — composite node; contains other FileSystemNodes.
type Directory struct {
	name     string
	children []FileSystemNode
}

func NewDir(name string) *Directory { return &Directory{name: name} }

func (d *Directory) Add(node FileSystemNode) { d.children = append(d.children, node) }
func (d *Directory) Name() string             { return d.name }
func (d *Directory) Size() int {
	total := 0
	for _, c := range d.children {
		total += c.Size()
	}
	return total
}

func (d *Directory) Display(indent int) {
	fmt.Printf("%s📁 %s/ (%d B total)\n", strings.Repeat("  ", indent), d.name, d.Size())
	for _, c := range d.children {
		c.Display(indent + 1)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// FACADE
//
// Provides a simplified interface to a complex subsystem. The subsystem
// classes remain available for advanced use; the facade handles the 80% case.
// ─────────────────────────────────────────────────────────────────────────────

// Subsystem: four independent components, each with a complex API.

type videoDecoder struct{}

func (v *videoDecoder) Decode(src string) string {
	fmt.Printf("  [VIDEO] decoding %s\n", src)
	return "raw-frames"
}

func (v *videoDecoder) SetCodec(codec string) {
	fmt.Printf("  [VIDEO] codec set to %s\n", codec)
}

type audioDecoder struct{}

func (a *audioDecoder) Decode(src string) string {
	fmt.Printf("  [AUDIO] decoding %s\n", src)
	return "raw-audio"
}

func (a *audioDecoder) SetSampleRate(hz int) {
	fmt.Printf("  [AUDIO] sample rate %d Hz\n", hz)
}

type videoEncoder struct{}

func (e *videoEncoder) Encode(frames, output string) {
	fmt.Printf("  [ENCODE] frames=%s → %s\n", frames, output)
}

type audioMixer struct{}

func (m *audioMixer) Mix(audio, output string) {
	fmt.Printf("  [MIX] audio=%s → %s\n", audio, output)
}

// MediaConverter — the facade. Hides the four-step pipeline behind one call.
type MediaConverter struct {
	vdec  *videoDecoder
	adec  *audioDecoder
	venc  *videoEncoder
	mixer *audioMixer
}

func NewMediaConverter() *MediaConverter {
	return &MediaConverter{
		vdec:  &videoDecoder{},
		adec:  &audioDecoder{},
		venc:  &videoEncoder{},
		mixer: &audioMixer{},
	}
}

func (m *MediaConverter) ConvertToMP4(input, output string) {
	fmt.Printf("  [FACADE] converting %s → %s\n", input, output)
	m.vdec.SetCodec("h264")
	m.adec.SetSampleRate(44100)
	frames := m.vdec.Decode(input)
	audio := m.adec.Decode(input)
	m.venc.Encode(frames, output+".tmp")
	m.mixer.Mix(audio, output)
	fmt.Printf("  [FACADE] done: %s\n", output)
}

func main() {
	fmt.Println("=== Proxy: caching ===")
	real := &realLoader{}
	cached := NewCachingProxy(real)
	for _, key := range []string{"user:1", "user:2", "user:1", "user:2", "user:3"} {
		v, err := cached.Load(key)
		if err != nil {
			fmt.Println("error:", err)
		} else {
			fmt.Printf("  → %s\n", v)
		}
	}
	fmt.Printf("  real loader called %d times (2 unique keys + 1 new = 3)\n", real.callCount)

	fmt.Println()
	fmt.Println("=== Proxy: access control ===")
	acProxy := NewACProxy(real, "viewer", "public:1", "public:2")
	v, err := acProxy.Load("public:1")
	fmt.Printf("  public:1 → %q  err=%v\n", v, err)
	_, err = acProxy.Load("secret:admin")
	fmt.Printf("  secret:admin → err=%v\n", err)

	fmt.Println()
	fmt.Println("=== Composite: file system tree ===")
	root := NewDir("project")
	src := NewDir("src")
	src.Add(&File{"main.go", 1240})
	src.Add(&File{"service.go", 3200})
	src.Add(&File{"repo.go", 880})
	tests := NewDir("tests")
	tests.Add(&File{"service_test.go", 1900})
	root.Add(src)
	root.Add(tests)
	root.Add(&File{"go.mod", 180})
	root.Add(&File{"README.md", 512})
	root.Display(0)
	fmt.Printf("  total size: %d B\n", root.Size())

	fmt.Println()
	fmt.Println("=== Facade: media converter ===")
	converter := NewMediaConverter()
	converter.ConvertToMP4("video.avi", "video.mp4")
}
