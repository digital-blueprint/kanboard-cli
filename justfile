# kanboard-cli justfile
# Run `just` or `just --list` to see available recipes.

module     := "github.com/tu-graz/kanboard-cli"
binary     := "kanboard-cli"
version    := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
commit     := `git rev-parse --short HEAD 2>/dev/null || echo "none"`
build_date := `date -u +"%Y-%m-%dT%H:%M:%SZ"`
ldflags    := "-s -w \
  -X " + module + "/internal/version.Version=" + version + " \
  -X " + module + "/internal/version.Commit=" + commit + " \
  -X " + module + "/internal/version.Date=" + build_date

# ── Default: show available recipes ──────────────────────────────────────────
default:
    @just --list

# ── Go build & run ────────────────────────────────────────────────────────────

# Build the binary into the project root
build:
    go build -ldflags '{{ldflags}}' -o {{binary}} .

# Build and run, passing any extra arguments:  just run task list --project-id 1
run *args: build
    ./{{binary}} {{args}}

# Run tests
test:
    go test ./...

# Run tests with race detector
test-race:
    go test -race ./...

# Lint (requires golangci-lint)
lint:
    golangci-lint run ./...

# Format all Go source files
fmt:
    gofmt -w .
    goimports -w . 2>/dev/null || true

# Remove build artefacts
clean:
    rm -f {{binary}}
    rm -rf dist/

# ── Nix ───────────────────────────────────────────────────────────────────────

# Build with Nix (output symlinked to ./result)
nix-build:
    nix build .#default

# Build and run with Nix, passing extra arguments:  just nix-run version
nix-run *args:
    nix run .#default -- {{args}}

# ── Helpers ───────────────────────────────────────────────────────────────────

# Show version information from the built binary
version: build
    ./{{binary}} version

# Tidy Go modules and sync the vendor directory
vendor:
    go mod tidy
    go mod vendor
