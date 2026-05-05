// FILE: book/part6_production_engineering/chapter94_cicd/examples/02_goreleaser/main.go
// CHAPTER: 94 — CI/CD for Go Services
// TOPIC: GoReleaser configuration — cross-compilation, Docker publishing,
//        changelogs, SBOM, and release automation.
//
// Run:
//   go run ./book/part6_production_engineering/chapter94_cicd/examples/02_goreleaser

package main

import (
	"fmt"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// GORELEASER CONFIG REFERENCE
// ─────────────────────────────────────────────────────────────────────────────

const goreleaserConfig = `# .goreleaser.yaml
version: 2

before:
  hooks:
    - go mod tidy
    - go generate ./...

builds:
  - id: app
    main: ./cmd/app
    binary: app
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ignore:
      - goos: windows
        goarch: arm64
    ldflags:
      - -s -w
      - -X main.Version={{.Version}}
      - -X main.Commit={{.Commit}}
      - -X main.BuildTime={{.Date}}
    flags:
      - -trimpath

archives:
  - id: app
    format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: 'checksums.txt'
  algorithm: sha256

sboms:
  - artifacts: archive   # generates SBOM for each archive

changelog:
  sort: asc
  use: github
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^chore:'
  groups:
    - title: 'New Features'
      regexp: '^feat'
    - title: 'Bug Fixes'
      regexp: '^fix'
    - title: 'Performance'
      regexp: '^perf'

dockers:
  - id: amd64
    goos: linux
    goarch: amd64
    image_templates:
      - "ghcr.io/myorg/app:{{ .Tag }}-amd64"
    use: buildx
    build_flag_templates:
      - "--platform=linux/amd64"
      - "--label=org.opencontainers.image.version={{ .Version }}"
      - "--label=org.opencontainers.image.revision={{ .Commit }}"
  - id: arm64
    goos: linux
    goarch: arm64
    image_templates:
      - "ghcr.io/myorg/app:{{ .Tag }}-arm64"
    use: buildx
    build_flag_templates:
      - "--platform=linux/arm64"

docker_manifests:
  - name_template: "ghcr.io/myorg/app:{{ .Tag }}"
    image_templates:
      - "ghcr.io/myorg/app:{{ .Tag }}-amd64"
      - "ghcr.io/myorg/app:{{ .Tag }}-arm64"
  - name_template: "ghcr.io/myorg/app:latest"
    image_templates:
      - "ghcr.io/myorg/app:{{ .Tag }}-amd64"
      - "ghcr.io/myorg/app:{{ .Tag }}-arm64"

release:
  github:
    owner: myorg
    name: app
  draft: false
  prerelease: auto   # marks as pre-release if tag contains '-'`

// ─────────────────────────────────────────────────────────────────────────────
// RELEASE ARTIFACT SIMULATION
// ─────────────────────────────────────────────────────────────────────────────

type ReleaseArtifact struct {
	Name    string
	Type    string
	SizeKB  int
	Purpose string
}

func generateArtifacts(version string) []ReleaseArtifact {
	return []ReleaseArtifact{
		{fmt.Sprintf("app_%s_linux_amd64.tar.gz", version), "archive", 7200, "Linux x86-64 binary"},
		{fmt.Sprintf("app_%s_linux_arm64.tar.gz", version), "archive", 6800, "Linux ARM64 binary"},
		{fmt.Sprintf("app_%s_darwin_amd64.tar.gz", version), "archive", 7100, "macOS Intel binary"},
		{fmt.Sprintf("app_%s_darwin_arm64.tar.gz", version), "archive", 6900, "macOS Apple Silicon binary"},
		{fmt.Sprintf("app_%s_windows_amd64.zip", version), "archive", 7400, "Windows x86-64 binary"},
		{"checksums.txt", "checksum", 1, "SHA256 checksums for all archives"},
		{fmt.Sprintf("app_%s_linux_amd64.tar.gz.sbom.json", version), "sbom", 85, "Software Bill of Materials (SPDX)"},
		{fmt.Sprintf("ghcr.io/myorg/app:%s", version), "docker", 0, "Multi-arch container image"},
	}
}

func printArtifacts(artifacts []ReleaseArtifact) {
	fmt.Printf("  %-50s  %-10s  %8s  %s\n", "Artifact", "Type", "Size", "Purpose")
	fmt.Printf("  %s\n", strings.Repeat("-", 95))
	for _, a := range artifacts {
		size := fmt.Sprintf("%dKB", a.SizeKB)
		if a.SizeKB == 0 {
			size = "~9MB"
		}
		fmt.Printf("  %-50s  %-10s  %8s  %s\n", a.Name, a.Type, size, a.Purpose)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// CHANGELOG SIMULATION
// ─────────────────────────────────────────────────────────────────────────────

type CommitEntry struct {
	Hash    string
	Type    string
	Message string
}

var mockCommits = []CommitEntry{
	{"a1b2c3d", "feat", "add payment webhook retry logic"},
	{"b2c3d4e", "fix", "handle nil pointer in order processor"},
	{"c3d4e5f", "perf", "replace O(n²) dedup with map lookup"},
	{"d4e5f6g", "fix", "correct timezone handling in report generator"},
	{"e5f6g7h", "feat", "add OpenTelemetry tracing to HTTP handlers"},
	{"f6g7h8i", "docs", "update deployment guide for Kubernetes 1.29"},
	{"g7h8i9j", "chore", "upgrade Go 1.23 → 1.24"},
	{"h8i9j0k", "test", "add integration tests for payment service"},
}

func generateChangelog(commits []CommitEntry) string {
	var sb strings.Builder
	groups := map[string][]CommitEntry{
		"feat": {},
		"fix":  {},
		"perf": {},
	}
	groupNames := map[string]string{
		"feat": "New Features",
		"fix":  "Bug Fixes",
		"perf": "Performance",
	}
	for _, c := range commits {
		if _, ok := groups[c.Type]; ok {
			groups[c.Type] = append(groups[c.Type], c)
		}
	}
	for _, typ := range []string{"feat", "fix", "perf"} {
		entries := groups[typ]
		if len(entries) == 0 {
			continue
		}
		fmt.Fprintf(&sb, "### %s\n", groupNames[typ])
		for _, e := range entries {
			fmt.Fprintf(&sb, "* %s (%s)\n", e.Message, e.Hash)
		}
		fmt.Fprintln(&sb)
	}
	return sb.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 94: GoReleaser ===")
	fmt.Println()

	fmt.Println("--- .goreleaser.yaml ---")
	fmt.Println(goreleaserConfig)
	fmt.Println()

	fmt.Println("--- Release artifacts for v2.4.1 ---")
	artifacts := generateArtifacts("2.4.1")
	printArtifacts(artifacts)
	fmt.Println()

	fmt.Println("--- Generated changelog ---")
	fmt.Println("## What's Changed")
	fmt.Println()
	fmt.Println(generateChangelog(mockCommits))

	fmt.Println("--- CLI usage ---")
	fmt.Println(`  # Dry run (no publish)
  goreleaser release --snapshot --clean

  # Full release from tag
  git tag v2.4.1
  git push origin v2.4.1
  goreleaser release --clean

  # Local build only
  goreleaser build --single-target --snapshot`)
}
