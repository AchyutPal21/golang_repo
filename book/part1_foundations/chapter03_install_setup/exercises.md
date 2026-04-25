# Chapter 3 — Exercises

## Exercise 3.1 — Editor smoke test

**Goal.** Confirm `gopls` is alive in your editor.

**Task.** Open
[`examples/02_editor_smoketest/main.go`](examples/02_editor_smoketest/main.go)
in your editor. Verify all four signals from the file's comments
(hover on `fmt.Println`, hover on `Greet`, gofmt-on-save, etc.). If
any fails, fix your editor setup before continuing.

**Acceptance.** Every signal works.

---

## Exercise 3.2 — Onboarding script

**Goal.** Reproduce your install from zero on a fresh machine.

**Task.** Write a shell script (`onboard.sh` for Linux/macOS,
`onboard.ps1` for Windows) that:

1. Downloads the official Go tarball for the OS/arch.
2. Verifies the SHA-256 against the published value.
3. Extracts to `/usr/local/go` (or equivalent).
4. Adds the canonical `PATH` lines to the shell rc.
5. `go install`s `gopls`, `golangci-lint`, `govulncheck`.
6. Runs the install self-check from
   [`examples/01_install_self_check`](examples/01_install_self_check/main.go).

Test it in a clean VM, container, or with `dotfiles`-style isolation.

**Acceptance.** A new machine, run the script, go from "no Go" to
"book examples build."

---

## Exercise 3.3 ★ — Multi-version sanity (`GOTOOLCHAIN`)

**Goal.** Internalize the 1.21+ toolchain dispatch model.

**Task.** With a 1.22+ system install, create two throwaway
projects:

```bash
mkdir /tmp/proj-old && cd /tmp/proj-old && go mod init example.com/old
# Edit go.mod: change "go 1.22" to "go 1.21"
echo 'package main\nfunc main(){}' > main.go
go build .

mkdir /tmp/proj-new && cd /tmp/proj-new && go mod init example.com/new
# go.mod stays at "go 1.22"
echo 'package main\nfunc main(){}' > main.go
go build .
```

Both should build. Now switch:

```bash
GOTOOLCHAIN=local go build .   # in proj-old
GOTOOLCHAIN=local go build .   # in proj-new
```

Read the docs for `GOTOOLCHAIN` (`go help toolchain`) and write a
one-paragraph explanation of when the auto-download happens and when
it doesn't.

**Acceptance.** Your written explanation matches the actual behavior
you observed.
