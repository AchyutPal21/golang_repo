# Chapter 3 — Installing Go and Setting Up Your Environment

> **Reading time:** ~22 minutes (5,500 words). **Code:** 2 runnable
> programs (~140 lines). **Target Go version:** 1.22+.
>
> A short, practical chapter. You'll install Go on your operating
> system, configure your shell, set up your editor, and verify the
> whole stack with a single command. By the end, when something
> "doesn't work" later, you'll know exactly which knob to turn.

---

## 1. Concept Introduction

A working Go install is four things on disk:

1. The **toolchain itself** — the `go` binary plus the standard
   library, typically at `/usr/local/go` or `/opt/homebrew/Cellar/go`
   or `C:\Program Files\Go`.
2. A **GOPATH** — your personal workspace, defaulting to `~/go`,
   which holds the module cache and installed tool binaries.
3. A **shell setup** — `$PATH` entries that let you type `go` and
   `gopls` and `golangci-lint` without absolute paths.
4. An **editor setup** — VS Code, GoLand, Vim, or whichever, talking
   to `gopls` (the official language server).

Get all four right once and you'll never think about them again. Get
any one wrong and you'll spend an afternoon debugging instead of
coding.

> **Working definition:** "Setting up Go" means installing the
> toolchain at a known path, putting `go` and `~/go/bin` on `$PATH`,
> and pointing your editor at `gopls`. Everything else is preference.

---

## 2. Why This Exists

Most install guides on the internet teach you a single happy path
("run this script and pray"). When something goes wrong — and it
will, because operating systems differ — you have nothing to fall
back on. This chapter gives you the *model* of a Go install, so the
specific commands are interchangeable. You'll know what each step is
*for*, which means you can adapt to the next OS, the next package
manager, the next CI runner without re-learning.

It also exists because *senior engineers don't tolerate broken
installs*. The single most common reason juniors get stuck is some
small `$PATH` or `$GOPATH` issue that wasn't caught early. We'll
catch them all here.

---

## 3. Problem It Solves

Specific install-time problems this chapter prevents:

1. **"`go` command not found"** — toolchain not on `$PATH`.
2. **"`gopls` not found, install it"** — `~/go/bin` not on `$PATH`.
3. **"`go: GOPATH entry contains \"src/github.com\"…"** — leftover
   GOPATH-mode setup from pre-2018 tutorials.
4. **Wrong Go version** — distro packages on Linux are sometimes
   years behind. Ubuntu 22.04 ships Go 1.18; you want 1.22+.
5. **Editor doesn't know about your code** — `gopls` not started, or
   started in the wrong directory.
6. **Mixed installs** — Homebrew Go on macOS plus a tarball install,
   plus an `asdf` version. Whichever is first on `$PATH` wins, and
   it's probably not the one you think.
7. **CI vs. local divergence** — CI on Go 1.22, your laptop on
   Go 1.20, surprise build differences.

---

## 4. Historical Context

Go's install story has been simple from day one. The toolchain has
always shipped as a self-contained tarball you extract somewhere on
disk. There has never been a separate "JDK" and "JRE" split. There
has never been a runtime to install on the target machine.

A few historical notes:

* **Go 1.0 (2012)** introduced the now-standard install layout:
  `/usr/local/go/{bin,src,pkg}`. Almost every Linux/macOS install
  guide since has copied this.
* **Go 1.5 (2015)** removed the C-based bootstrap. Before this, you
  needed a C compiler to build Go from source. Today, the toolchain
  is self-hosted: a Go N-1 toolchain compiles Go N.
* **Go 1.11 (2018)** introduced modules, which decoupled "where on
  disk your code lives" from "what its import path is." Pre-1.11,
  every project had to live under `$GOPATH/src/<full-import-path>`,
  which produced absurd layouts like
  `~/go/src/github.com/foo/bar/...` for every personal project.
* **Go 1.16 (2021)** made modules mandatory. GOPATH-mode is gone
  except for legacy edge cases.
* **Go 1.21 (2023)** introduced `GOTOOLCHAIN` — the toolchain can
  now download and use a different Go version than the one that
  shipped with your install, on a per-module basis. This means a
  single Go install can build any project, regardless of its `go.mod`
  directive.

The `GOTOOLCHAIN=auto` default since 1.21 is genuinely game-changing
for fleets: you no longer need to coordinate Go versions across
machines. Each repo declares its needs in `go.mod`; each machine
honors them by downloading the requested toolchain.

---

## 5. How Industry Uses It

The industry-standard install pattern in 2026:

* **Personal laptops:** install via the official tarball (Linux/macOS)
  or the official MSI (Windows). Symlink `/usr/local/go/bin/go` into
  `/usr/local/bin` (or use the installer, which does this for you).
  Add `~/go/bin` to `$PATH` in `~/.bashrc` or `~/.zshrc`.
* **macOS specifically:** Homebrew (`brew install go`) is widely used
  and totally fine; it lags the official release by hours-to-days.
* **Linux specifically:** the official tarball, **not** the distro
  package. Ubuntu, Debian, Fedora packages routinely ship Go versions
  that are 6–24 months behind. Use the tarball.
* **Multi-version setups:** `gvm` (legacy) or — increasingly — just
  let `GOTOOLCHAIN=auto` handle it (1.21+). For development across
  many Go versions, `asdf-go` is popular.
* **CI:** GitHub Actions' `actions/setup-go@v5` with `go-version-file:
  go.mod`. This pins the CI to whatever your repo declares. CircleCI,
  GitLab, Buildkite all have equivalent first-party setup actions.
* **Containers:** the official `golang:1.22` image, or
  `golang:1.22-alpine` for smaller layers. For final stages,
  `gcr.io/distroless/static` (covered in Chapter 94).
* **Editors:** VS Code with the Go extension dominates by a mile.
  GoLand is preferred at some shops where everyone already has the
  JetBrains license. Neovim with `nvim-lspconfig` + `gopls` is the
  power-user path.

---

## 6. Real-World Production Use Cases

**Onboarding script.** A Go-first company hands new engineers a
single shell script:

```bash
brew install go             # macOS
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/gopls@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
code --install-extension golang.go
```

Five commands, total install time under 10 minutes, every engineer's
laptop is configured identically. The script is checked into a
"toolbox" repo and updated as the toolchain evolves.

**Cross-platform releases.** A team building a CLI tool used by
end-users on macOS, Linux, and Windows lets `goreleaser` build all
nine targets (3 OSes × 3 architectures) from a single Go install on
one CI machine. No SDK juggling. No cross-toolchain configuration.

**Reproducible production builds.** A regulated industry (banking,
healthcare) wants reproducible binaries: byte-for-byte identical given
the same source. Go achieves this with `-trimpath -ldflags='-s -w
-buildid='` and a pinned toolchain via `GOTOOLCHAIN`. Auditors verify
by re-running the build and comparing SHA-256 hashes.

**Air-gapped enterprise install.** A team in a network-restricted
environment hosts an internal Go SDK mirror plus an Athens module
proxy. Engineers `export GOPROXY=https://athens.internal,direct` and
`export GOSUMDB=sum.internal`. Builds work without public-internet
access.

**Multi-version dev across legacy + new services.** A platform team
maintains a 2018-vintage service still on Go 1.16 alongside green-
field services on Go 1.22. Pre-1.21 they had to script `gvm` switches
in their dev loop. Post-1.21 with `GOTOOLCHAIN=auto`, each repo's
`go.mod` declares its required version and the toolchain handles the
rest.

---

## 7. Beginner-Friendly Explanation

If you've never installed a programming language before:

1. Programming languages are usually delivered as a *toolchain*: a
   bundle of programs (compiler, linker, etc.) plus a *standard
   library* (a folder of pre-written code your programs can use).
2. To install the toolchain, you download a bundle and extract it into
   a directory. Then you add that directory's `bin/` to your shell's
   `PATH` so the shell can find the programs.
3. The Go toolchain happens to also include a *workspace* convention:
   it expects to keep downloaded libraries and your installed tools
   in `~/go/`. You don't have to think about this directory; the
   toolchain manages it. You just need to add `~/go/bin/` to `PATH`
   too, so any tool you install with `go install` is runnable.
4. Once that's set up, you write code in any folder you like. Your
   editor (VS Code, GoLand, etc.) needs an *extension* that talks to
   `gopls`, the language server. With that, you get autocomplete,
   inline errors, jump-to-definition, etc.

That's the whole thing. The exact commands depend on your OS, but
the *concepts* are the same on all of them.

> **Coming From Python —** No `pip`, no `venv`. The Go toolchain
> handles dependency caching itself; there's nothing per-project
> about an install. Set up once, use forever.

> **Coming From JavaScript —** No `nvm`, usually. `GOTOOLCHAIN=auto`
> on 1.21+ does what `nvm` does, but driven by `go.mod` instead of
> a per-project `.nvmrc`.

> **Coming From Java —** No `JAVA_HOME`. The Go toolchain finds
> itself via `$PATH`; there's no equivalent of the JDK/JRE split.

> **Coming From C++ —** No "Visual Studio install + Build Tools +
> Windows SDK + vcpkg." The Go install is one tarball.

---

## 8. Deep Technical Explanation

Now we walk it OS by OS.

### 8.1. Linux (Ubuntu/Debian/Fedora/Arch)

The right way:

```bash
# 1. Download the official tarball. Update the version as needed.
curl -LO https://go.dev/dl/go1.22.5.linux-amd64.tar.gz

# 2. Verify the checksum (paranoid but cheap).
sha256sum go1.22.5.linux-amd64.tar.gz
# Compare against the value at https://go.dev/dl/

# 3. Replace any existing /usr/local/go install.
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.5.linux-amd64.tar.gz

# 4. Add to PATH in your shell rc (~/.bashrc or ~/.zshrc).
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc

# 5. Verify.
go version
```

The wrong way (which 90% of tutorials still teach):

```bash
sudo apt install golang  # ← do not do this
```

Reason: distro packages lag. As of 2026, Debian stable ships Go 1.19,
Ubuntu LTS 22.04 ships Go 1.18. Both are unsupported. The official
tarball is current and removes the version-mismatch class of problems.

For ARM (Raspberry Pi, AWS Graviton, modern Apple Silicon Linux VMs):
substitute `linux-arm64` for `linux-amd64` in the URL.

### 8.2. macOS

Two equally valid paths.

**Option A — Homebrew:**

```bash
brew install go
```

Homebrew installs to `/opt/homebrew/Cellar/go/<version>` (Apple
Silicon) or `/usr/local/Cellar/go/<version>` (Intel), with a symlink
in `/opt/homebrew/bin/go` (or `/usr/local/bin/go`). `brew upgrade go`
moves you forward.

**Option B — Official package:**

Download `go1.22.5.darwin-arm64.pkg` (Apple Silicon) or
`go1.22.5.darwin-amd64.pkg` (Intel) from `https://go.dev/dl/`,
double-click, follow the prompts. Installs to `/usr/local/go` and
adds `/usr/local/go/bin` to `/etc/paths.d/go`.

In both cases, add `~/go/bin` to `$PATH`:

```bash
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.zshrc
source ~/.zshrc
```

### 8.3. Windows

Download `go1.22.5.windows-amd64.msi` from `https://go.dev/dl/`.
Run it. The installer puts the toolchain at `C:\Program Files\Go`
and adds it to your user `PATH`. You'll need to log out/in for
`PATH` changes to take effect in new shells.

For PowerShell developers, add `~/go/bin` to PATH in your profile:

```powershell
# In $PROFILE:
$env:Path += ";$HOME\go\bin"
```

For Windows Subsystem for Linux (WSL): treat it as Linux. Install
the Linux tarball inside WSL. Don't try to share a Go install between
the Windows host and the WSL guest.

### 8.4. Verifying the install

A correctly-installed Go has these visible signs:

```bash
$ go version
go version go1.22.5 linux/amd64

$ go env GOROOT
/usr/local/go

$ go env GOPATH
/home/yourname/go

$ which go
/usr/local/go/bin/go

$ which gopls       # may say "not found" until you install it
```

If any of these are unexpected — wrong version, weird `GOROOT`,
multiple `go` binaries on `$PATH` — fix it before writing any code.

### 8.5. Installing the daily-driver tools

Three tools that every Go engineer should have on `$PATH`:

```bash
go install golang.org/x/tools/gopls@latest                              # editor LSP
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest   # linter
go install golang.org/x/vuln/cmd/govulncheck@latest                     # vulnerabilities
```

After running these, `which gopls` should resolve. If not, your
`~/go/bin` (or `$GOBIN`) is not on `$PATH`. Fix the shell rc.

Optional but useful tools:

```bash
go install github.com/go-delve/delve/cmd/dlv@latest       # debugger
go install github.com/goreleaser/goreleaser@latest         # release tooling
go install golang.org/x/perf/cmd/benchstat@latest          # benchmark comparison
go install honnef.co/go/tools/cmd/staticcheck@latest       # extra static analyser
```

### 8.6. Editor setup

#### VS Code

1. Install VS Code.
2. Install the "Go" extension (publisher: Go Team at Google).
3. Open any `.go` file. The extension will prompt you to install
   missing tools (`gopls`, `dlv`, `golangci-lint`). Accept.
4. Verify: open a `.go` file, hover over a standard library
   identifier — you should see its doc comment in a tooltip.

The extension reads settings from `.vscode/settings.json`. A useful
project-local config:

```jsonc
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "go.formatTool": "goimports",
  "[go]": {
    "editor.formatOnSave": true,
    "editor.codeActionsOnSave": {
      "source.organizeImports": "explicit"
    }
  }
}
```

#### GoLand

GoLand auto-detects the toolchain from `$PATH`. If you have multiple
installs, configure under *Settings → Go → GOROOT*. Everything else
just works.

#### Neovim (LSP path)

```lua
-- with nvim-lspconfig
require'lspconfig'.gopls.setup{
  settings = {
    gopls = {
      analyses = { unusedparams = true },
      staticcheck = true,
    },
  },
}
```

Combine with `null-ls` or `conform.nvim` to wire up `gofmt`/`goimports`
on save.

### 8.7. The `GOTOOLCHAIN` modern model (1.21+)

`GOTOOLCHAIN=auto` (the default since 1.21) makes the Go binary on
your `$PATH` a *bootstrap*: when it sees a `go.mod` requiring a
newer Go version, it downloads that version under
`$GOMODCACHE/golang.org/toolchain/...` and runs the build with it.
This means you don't have to upgrade your system Go to keep up with
projects that use newer language features.

Settings worth knowing:

* `GOTOOLCHAIN=auto` — default. Use the version in `go.mod` if newer,
  else use the system toolchain.
* `GOTOOLCHAIN=local` — never download; always use the system
  toolchain. Use this in air-gapped environments.
* `GOTOOLCHAIN=go1.22.5` — pin to a specific version regardless of
  what `go.mod` says. Useful in CI when you want absolute determinism.
* `GOTOOLCHAIN=path` — never download; refuse if the requested
  version isn't on `$PATH`. Ultra-strict.

In CI, prefer `GOTOOLCHAIN=local` plus a pinned setup-go version.
Avoid the toolchain auto-download in environments where you want
network-free, deterministic builds.

---

## 9. Internal Working (How Go Handles It)

* The `go` binary is a **driver**: it parses your subcommand
  (`run`, `build`, `test`, etc.), then `exec`s into one of the
  internal tools (`compile`, `link`, `vet`, ...) located under
  `$GOROOT/pkg/tool/$GOOS_$GOARCH/`.
* The **standard library source** lives at `$GOROOT/src/`. When
  you `import "fmt"`, the toolchain finds the package by consulting
  `$GOROOT/src/fmt/`.
* The **module cache** at `$GOMODCACHE` (defaults to
  `$GOPATH/pkg/mod/`) holds downloaded modules, read-only. Never
  edit files there; copy them out if you need to read them.
* The **build cache** at `$GOCACHE` (`~/.cache/go-build/` on Linux)
  holds compiled package objects keyed by content hash. Cold builds
  populate it; warm builds reuse it.
* `gopls` is itself a Go binary that runs as a long-lived process
  alongside your editor, communicating over stdin/stdout with the
  Language Server Protocol (LSP). Each time you open a file, your
  editor sends a `textDocument/didOpen` message; `gopls` responds
  with diagnostics, completions, and hovers.

If you ever wonder "where did this come from," the answer is almost
always one of those four directories. `find $(go env GOROOT) $(go
env GOPATH) -name 'go.mod' 2>/dev/null` is a useful diagnostic.

---

## 10. Syntax Breakdown

Not applicable for this chapter because the topic is install/setup,
not language syntax. Move to Section 11.

---

## 11. Multiple Practical Examples

### Example 1 — `examples/01_install_self_check`

```bash
go run ./examples/01_install_self_check
```

Validates your install. Checks that `go` is on `$PATH`, that
`~/go/bin` is on `$PATH`, that `gopls` and `golangci-lint` are
present, that the toolchain version is recent enough. Prints a
checklist and exits non-zero if anything fails. Use it as the last
step of any onboarding script.

### Example 2 — `examples/02_editor_smoketest`

```bash
go run ./examples/02_editor_smoketest
```

A small program with deliberate bait for editor features: a typo in
a function name (your editor should underline it), an unused import
(your editor should auto-remove on save), a doc comment on an
exported function (your editor should render it in a hover). Open
the file in your editor and check each one. If any of them don't
work, your editor isn't talking to `gopls`.

---

## 12. Good vs Bad Examples

**Good shell rc (`~/.bashrc` or `~/.zshrc`):**

```bash
export PATH=$PATH:/usr/local/go/bin
export PATH=$PATH:$HOME/go/bin
# GOPATH defaults to $HOME/go; do not override unless you have a reason.
# GOROOT is auto-detected; do not set unless you have a reason.
```

**Bad shell rc (yes, this is from a real onboarding doc circa 2018):**

```bash
export GOROOT=/usr/local/go
export GOPATH=$HOME/Code/go
export GOBIN=$GOPATH/bin
export PATH=$GOROOT/bin:$GOPATH/bin:$PATH
mkdir -p $GOPATH/src/github.com/$USER
cd $GOPATH/src/github.com/$USER
```

What's wrong:

1. `GOROOT` does not need to be set; the toolchain finds itself.
2. Overriding `GOPATH` to a non-default path causes confusion every
   time someone copy-pastes a tutorial command. Stick to `~/go`.
3. `GOBIN` is redundant; it defaults to `$GOPATH/bin`.
4. The `mkdir -p $GOPATH/src/...` is GOPATH-mode legacy from
   pre-modules. **Delete this from any rc you find with it.**
5. `cd` in a shell rc is a footgun: every new shell drops you in a
   surprise directory.

**Good Go-version-pinning in CI (GitHub Actions):**

```yaml
- uses: actions/setup-go@v5
  with:
    go-version-file: go.mod
    cache: true
```

**Bad:**

```yaml
- uses: actions/setup-go@v5
  with:
    go-version: 1.22  # drifts when 1.23 ships
```

The first uses the version in `go.mod` as the source of truth. The
second hard-codes a version that will be out of date in six months,
producing CI/local divergence.

---

## 13. Common Mistakes

1. **Using a distro package.** Symptom: `go version` says 1.18
   but you needed 1.21+. Fix: use the official tarball.
2. **Forgetting `~/go/bin` in `$PATH`.** Symptom: `gopls` not
   found. Fix: add to shell rc.
3. **Copy-pasting `GOPATH` overrides from old tutorials.**
   Symptom: nothing works as documented. Fix: unset `GOPATH` if
   you set it to a non-default value, restart the shell.
4. **Two Go installs on `$PATH`** (e.g. Homebrew + tarball).
   Symptom: `which -a go` shows two paths. Fix: remove one.
5. **Setting `GOROOT` by hand.** Almost always wrong on modern
   installs. The toolchain locates itself.
6. **Editing files inside the module cache.** Symptom: changes
   silently revert because the cache is read-only on a clean
   install. Fix: never modify cache; clone the upstream repo if
   you need to patch a dep.
7. **Not restarting the shell after editing rc.** Symptom: edits
   "didn't take." Fix: `exec $SHELL -l` or open a new terminal.
8. **Letting your editor install `gopls` from a stale extension
   prompt.** Symptom: out-of-date language server, weird false
   positives. Fix: `go install golang.org/x/tools/gopls@latest`
   from your terminal, restart the editor.
9. **Using `sudo go install`.** Never. `go install` writes to
   `$GOBIN` which is in your user home. `sudo` will install into
   root's home and confuse everything.
10. **Running `go mod tidy` from `~/`.** Symptom: confusing errors
    because `~/` is not a module. Run `go` commands from inside a
    project directory.

---

## 14. Debugging Tips

* **`go env`** as the first move on any "weird" issue. Half of all
  install problems are visible there.
* **`which -a go`** to find duplicate installs.
* **`echo $PATH | tr ':' '\n'`** to read `$PATH` as a list. Look for
  `/usr/local/go/bin` and `~/go/bin`.
* **`gopls version`** to confirm `gopls` is reachable and current.
* **`go version -m $(which gopls)`** prints the dependencies a
  binary was built against. Useful when something behaves oddly.
* **`code --status`** in VS Code prints the running extension
  versions; check the Go extension is current.
* **Reinstalling the toolchain** is fast (one tarball) and is the
  right answer to "I think my Go install is corrupted."

---

## 15. Performance Considerations

The toolchain is fast; install setup doesn't move the needle. Two
small wins worth knowing:

* **Pre-warm the module cache** in CI by running `go mod download`
  in a separate cached step. Subsequent `go build` and `go test`
  steps then hit the cache.
* **Use the build cache.** GitHub Actions' `actions/setup-go@v5`
  enables it via `cache: true` (defaults to true on v5+). On a typical
  service this halves CI time.
* **`GOMAXPROCS`** — on container environments without a CFS-aware
  Go runtime (rare on 1.21+), set `GOMAXPROCS` to match the cgroup
  CPU limit. The runtime now does this automatically; if you see
  weird scheduling on 1.21+ it's almost always something else.

---

## 16. Security Considerations

* **Verify download checksums.** `https://go.dev/dl/` lists SHA-256
  hashes. Compare them after download. Yes, even from `https://`.
* **Don't `curl | sh`.** A surprising number of "install Go" guides
  pipe a remote script into `sh`. Don't. Download, inspect, run.
* **Pin `actions/setup-go` to a major version**, e.g. `@v5`. Avoid
  `@main`, which is mutable.
* **Audit installed tools.** `go install` runs arbitrary Go code from
  arbitrary modules. Treat each `go install` line in your onboarding
  doc as a supply-chain decision. A compromised `gopls` would have
  shell access on every developer's laptop.
* **Restrict CI write access.** A CI job that can `go install` from
  the public proxy can also `go install` a malicious module. Use
  `GOPROXY` pinning or vendoring for build steps that handle secrets.

---

## 17. Senior Engineer Best Practices

1. **Always use the official tarball or installer**, not the distro
   package, on Linux/macOS.
2. **Don't override `GOPATH` or `GOROOT`** unless you have a
   specific reason. Defaults are correct.
3. **Add the canonical PATH lines once** to `~/.bashrc`/`~/.zshrc`
   and forget about them.
4. **Pin Go version in `go.mod`**: `go 1.22.0`, not `go 1.22`.
5. **Pin CI Go version via `go-version-file: go.mod`**, not a
   hard-coded number.
6. **Install `gopls`, `golangci-lint`, `govulncheck` on day one.**
7. **Re-run `go install ...@latest` for tools every quarter** to
   stay current.
8. **Document your team's onboarding script in a repo**, not a wiki.
   Keep it under version control.
9. **Test the onboarding script on a fresh VM annually.** Bit-rot
   happens fast.
10. **Have a "no distro packages" policy** in writing. New hires
    keep installing `apt install golang-go`; head it off.

---

## 18. Interview Questions

1. *(junior)* Where does `go install` write its output by default?
2. *(junior)* What environment variable points to the Go toolchain
   itself?
3. *(mid)* What is `gopls`?
4. *(mid)* Why do most senior engineers avoid distro Go packages?
5. *(senior)* What does `GOTOOLCHAIN=auto` do?
6. *(senior)* On a fresh laptop, walk me through your Go install
   from zero to "I can run my team's code."
7. *(senior)* What's wrong with `export GOPATH=$HOME/Code/go` in a
   shell rc?
8. *(staff)* You're standing up a new dev VM for a regulated
   environment with no public internet. How do you provision Go,
   `gopls`, and the team's modules?

---

## 19. Interview Answers

1. **`$GOBIN`** if set, otherwise **`$GOPATH/bin`** (which defaults
   to `~/go/bin`). It needs to be on `$PATH` for installed tools to
   be runnable.

2. **`GOROOT`**, but you should almost never set it manually. The
   toolchain locates itself via the `go` binary's path, walking up
   the directory tree. If you find yourself setting `GOROOT`,
   something's wrong.

3. **`gopls`** is the official Go language server, maintained by the
   Go team. It implements LSP, so any LSP-compatible editor (VS Code,
   GoLand, Neovim) can use it. It provides completions, hovers,
   diagnostics, refactors. It runs as a long-lived process per
   workspace.

4. They lag the upstream release. Ubuntu LTS, Debian stable, even
   Fedora typically ship Go versions 6–24 months behind. Modern
   features (generics in 1.18, structured logging in 1.21,
   `GOTOOLCHAIN` in 1.21) won't be available. The official tarball
   is one curl command and removes the version-mismatch class of
   problems.

5. **`GOTOOLCHAIN=auto`** (the default since 1.21) makes the
   installed `go` binary a bootstrap. When it sees a `go.mod`
   requiring a newer Go version, it transparently downloads that
   version under `$GOMODCACHE/golang.org/toolchain` and uses it for
   the build. This decouples "what version of Go is installed on
   my laptop" from "what version of Go this project needs."

6. Walk through it: download official tarball for OS/arch, extract
   to `/usr/local/go`, add `/usr/local/go/bin` and `~/go/bin` to
   `$PATH` in shell rc, restart shell, `go version` to verify,
   `go install gopls@latest`, `go install golangci-lint@latest`,
   `go install govulncheck@latest`, install editor extension, clone
   team repo, `go build ./...` to populate caches, done.

7. It's a non-default path. Tutorials, copy-pasted commands, and
   teammates' troubleshooting suggestions all assume `$GOPATH=$HOME/go`.
   Override that and every step that mentions `~/go/...` becomes a
   mental translation. Plus, you've now got modules and the build
   cache split across two places that nobody ever cleans up.

8. Mirror the official Go SDK download internally (a self-hosted
   tarball at a known URL). Mirror `proxy.golang.org` with Athens or
   JFrog Artifactory; mirror the checksum DB or run `GOSUMDB=off`
   with strict internal review. Bake all of it into a base image
   plus a provisioning script. Set `GOTOOLCHAIN=local` in CI to
   prevent surprise downloads. Pre-install `gopls`, `golangci-lint`,
   `govulncheck` from the internal proxy. Document the precise
   versions of everything in a "golden image" repo.

---

## 20. Hands-On Exercises

**Exercise 3.1 — Editor smoke test.** Open
`examples/02_editor_smoketest/main.go` in your editor. Verify
that:

* hovering over `fmt.Println` shows its doc comment;
* the deliberate typo `Pintln` is underlined as an error;
* the deliberate unused import is highlighted;
* `gopls` is responding (status bar in VS Code shows "Go" with a
  spinner that resolves to a checkmark).

If any of these fail, your editor isn't talking to `gopls`. Fix
before continuing.

**Exercise 3.2 — Onboarding script.** Write a shell script that
performs the install and tool setup from scratch on a fresh laptop
of your OS. Test it in a VM or container. Commit it to a
`team-toolbox` repo (you can fake the repo locally).

**Acceptance.** A new VM, run the script, end up with a working Go
install plus `gopls`, `golangci-lint`, `govulncheck` on `$PATH`.

**Exercise 3.3 ★ — Multi-version sanity.** With a 1.22+ system
install, create two projects: one with `go 1.21` in `go.mod`, one
with `go 1.22` in `go.mod`. Build both. Verify via the build output
that the toolchain resolution worked as expected. Switch
`GOTOOLCHAIN=local` and observe the behavior change.

---

## 21. Mini Project Tasks

**Task — Team install-doctor.** Extend `examples/01_install_self_check`
into a `doctor` command for your team:

* Detects the OS and recommends OS-specific install steps.
* Verifies `go`, `gopls`, `golangci-lint`, `govulncheck` are on
  `$PATH` and their versions are above team-specified minimums.
* Prints a colored summary (green check / red cross per item).
* Exits non-zero on any failure.

This is the kind of internal tool that lives in every Go shop's
toolbox repo.

---

## 22. Chapter Summary

* A working Go install is four things: the toolchain, a GOPATH, a
  shell `PATH` that points to both, and an editor talking to
  `gopls`.
* Use the official tarball or installer, not your distro package.
  Stay within one major version of latest.
* The defaults for `GOPATH` (`~/go`) and `GOROOT` (auto-detected)
  are correct; don't override them.
* Add `~/go/bin` to `$PATH` so `go install`-ed tools are findable.
* Install `gopls`, `golangci-lint`, `govulncheck` on day one.
* `GOTOOLCHAIN=auto` (1.21+) handles per-project version pinning
  without `gvm`-style version managers.
* In CI, pin Go via `go-version-file: go.mod`, cache the module
  and build caches between runs.

Updated working definition: *Setting up Go means dropping the
official toolchain at a known location, putting `go` and
`~/go/bin` on `$PATH`, installing `gopls` plus your linter and
vuln scanner, and pointing your editor at `gopls`. Configure once;
forget for years.*

---

## 23. Advanced Follow-up Concepts

* **`GOTOOLCHAIN` proposal** (`golang.org/issue/57001`) — the design
  document for the per-module toolchain switching introduced in 1.21.
* **`go.dev/dl/`** — official downloads, checksums, and changelog.
  Subscribe via RSS.
* **`gopls` changelog** at `golang.org/x/tools/gopls` — read the
  release notes for new editor features.
* **`golangci-lint` documentation** — the linter list and config
  reference.
* **`go.dev/blog/path-tools`** (2022) — Russ Cox on the design of
  the module-aware toolchain dispatch.
* **Eli Bendersky, "Setting up Go on macOS"** — a long-running blog
  series with up-to-date recipes for each macOS major release.

> **End of Chapter 3.** Move on to [Chapter 4 — The Go Workspace and
> Project Structure](../chapter04_workspace_structure/README.md).
