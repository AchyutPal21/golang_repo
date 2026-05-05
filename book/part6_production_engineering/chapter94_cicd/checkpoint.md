# Chapter 94 Checkpoint — CI/CD for Go Services

## Concepts to know

- [ ] What is the purpose of `-race` in `go test`? Why enable it in CI but not production builds?
- [ ] What does `hashFiles('**/go.sum')` accomplish in a GitHub Actions cache key?
- [ ] What is `golangci-lint` and which linters are most important for a production Go service?
- [ ] What is `govulncheck`? When in the pipeline should it run?
- [ ] What does `goreleaser` automate that `go build` alone cannot?
- [ ] What is a multi-arch Docker image? How is it created with `buildx`?
- [ ] What is an SBOM? Why is it increasingly required?
- [ ] What is the difference between staging and canary deployments?
- [ ] Name three things that should never be committed to version control.

## Code exercises

### 1. Pipeline stage runner

Write a `Stage` struct with `Name string` and `Run func() error`. Write a `Pipeline` that runs stages sequentially, prints `[PASS]`/`[FAIL]` for each, stops on first failure, and reports total elapsed time.

### 2. Coverage gate

Write a function `checkCoverage(coverageOutput string, threshold float64) error` that parses `go tool cover -func` output and returns an error if total coverage is below the threshold.

### 3. Build matrix

Write a `BuildMatrix` that generates all combinations of `GOOS` × `GOARCH` from two lists, runs a (simulated) `go build` for each, and reports which combinations succeeded and failed.

## Quick reference

```yaml
# .github/workflows/ci.yml skeleton
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with: { go-version: '1.24' }
    - uses: actions/cache@v4
      with:
        path: ~/go/pkg/mod
        key: go-${{ hashFiles('**/go.sum') }}
    - run: go vet ./...
    - run: go test -race -count=1 -coverprofile=coverage.out ./...
    - run: go tool cover -func=coverage.out
    - run: go build ./...
```

## Expected answers

1. `-race` detects data races at runtime. It costs ~10× memory/CPU so it's impractical in production but essential in CI where tests are run once.
2. `hashFiles` creates a unique key for the exact module set. Cache is reused when `go.sum` hasn't changed, saving 30–90s of `go mod download`.
3. `golangci-lint` runs 50+ linters. Most important: `staticcheck`, `errcheck`, `gosec`, `exhaustive`, `noctx`.
4. `govulncheck` checks your code against the Go vulnerability database. Run after `go build`, before deploying.
5. `goreleaser` handles multi-arch cross-compilation, packaging (tar/zip), checksums, GitHub Release creation, Docker image publishing, and changelogs.
6. Multi-arch image: a single manifest tag that serves different binaries by platform. Created with `docker buildx build --platform linux/amd64,linux/arm64 --push`.
7. SBOM (Software Bill of Materials) lists all dependencies. Required by US executive order EO 14028 and many enterprise procurement policies.
8. Staging: identical to prod, full deploy, used for manual/integration testing. Canary: route a small % (1–5%) of production traffic to the new version.
9. API keys/tokens, `.env` files with secrets, private keys, database passwords.
