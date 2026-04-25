# Chapter 5 — Exercises

## Exercise 5.1 — Inspect a binary

**Goal.** Read a Go binary's embedded build info.

**Task.** Run:

```bash
go run ./examples/01_build_inspector $(which gopls)
go run ./examples/01_build_inspector $(which go)
```

For each, identify (a) the Go version it was compiled with, (b) its
main module path, (c) any interesting build settings (`-trimpath`,
`vcs.revision`, etc.). Compare to the same info via `go version -m
$(which gopls)`.

**Acceptance.** You can read the build info and explain every field.

---

## Exercise 5.2 — Binary diet

**Goal.** Quantify the size impact of `-ldflags='-s -w'` and
`-trimpath`.

**Task.** From the chapter folder, run:

```bash
go build -o /tmp/app-default ./examples/01_build_inspector
go build -ldflags='-s -w' -o /tmp/app-stripped ./examples/01_build_inspector
go build -ldflags='-s -w' -trimpath -o /tmp/app-trimmed ./examples/01_build_inspector
ls -lh /tmp/app-*
```

**Acceptance.** Record the sizes. Most-stripped is typically ~30%
smaller. Optionally compress one with `upx --best /tmp/app-stripped`
and observe further shrinkage.

---

## Exercise 5.3 ★ — Cross-compile a binary

**Goal.** Build a binary for a platform you're not running on.

**Task.** From the chapter folder:

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build \
    -trimpath -ldflags='-s -w' \
    -o /tmp/app-linux-arm64 \
    ./examples/01_build_inspector

file /tmp/app-linux-arm64
```

The `file` output should say "ELF 64-bit LSB executable, ARM aarch64,
... statically linked". Confirm the binary is *static* (no
"dynamically linked" in the output).

**Acceptance.** A static ARM64 Linux binary on disk, regardless of
your host OS/arch.

---

## Exercise 5.4 ★ — Stamp a version

**Goal.** Practice the production-grade version-stamping pattern.

**Task.** Build `examples/03_ldflags_version` with custom values:

```bash
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo nogit)
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
go build \
    -ldflags="-s -w -X main.version=v0.1.0 -X main.commit=$COMMIT -X main.date=$DATE" \
    -o /tmp/vapp \
    ./examples/03_ldflags_version
/tmp/vapp
```

The output should show your custom version, commit, and date.

**Stretch.** Wrap the command in a `release.sh` script that builds
for `linux/amd64`, `linux/arm64`, and `darwin/arm64`, with the same
stamps applied to all three.
