# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## What this is

A Go TUI tool for interactively viewing and editing CSV, TSV, JSON, and JSONL files in the terminal. Built with the Charm ecosystem (bubbletea, bubbles, lipgloss). Distributed as a single binary via Homebrew.

## Commands

```bash
# Build
make build              # builds ./dpeek binary

# Run
make run ARGS="file.csv" # go run ./cmd/dpeek file.csv
dpeek file.csv           # after make install
dpeek file.json

# Test
make test               # go test ./... -v -count=1

# Lint
make lint               # golangci-lint run ./...

# Install to GOPATH/bin
make install
```

## Architecture

The codebase follows standard Go project layout:

```
cmd/dpeek/main.go           — Entry point, flag parsing, launches bubbletea
internal/
  detect/detect.go          — File format detection from extension
  csv/reader.go             — CSV/TSV parsing, DataSet struct, Save(), ColumnStats
  json/tree.go              — JSON/JSONL parsing into collapsible Node tree, serialization, Save()
  model/
    app.go                  — Top-level bubbletea model, dispatches to CSV or JSON sub-model
    csv_model.go            — CSV table view: navigation, editing, rendering
    json_model.go           — JSON tree view: expand/collapse, editing, rendering
    help.go                 — Help overlay rendering
    search.go               — Search state (shared between CSV and JSON modes)
    filter.go               — Row filtering (CSV only)
    sort.go                 — Column sorting (CSV only)
    stats.go                — Column statistics (CSV only)
  style/theme.go            — Lipgloss styles, status bar, F-key bar
```

### Key patterns

- **bubbletea Model pattern**: App → CSVModel / JSONModel. App handles quit keys and window resize, delegates everything else to the active sub-model.
- **Overlay system**: `overlay` enum (none/help/stats) controls what's rendered on top of the main view.
- **Filter-aware navigation**: CSVModel uses `filter.MapRow()` to translate visible row indices to actual data row indices.
- **JSON tree**: `Node.VisibleNodes()` flattens the tree respecting expand/collapse state. Cursor indexes into this flat list.

### Dependencies

- `github.com/charmbracelet/bubbletea` — TUI framework
- `github.com/charmbracelet/bubbles` — text input component
- `github.com/charmbracelet/lipgloss` — styling
- Everything else is Go stdlib (encoding/csv, encoding/json, flag, sort)

## Build system

- Go 1.22+, module path `github.com/GonzaloFuentes28/dpeek`
- Version injected via ldflags: `-X main.version=$(VERSION)`
- GoReleaser for cross-compilation and Homebrew formula generation
- CI: GitHub Actions runs `golangci-lint` and `go test` on push/PR

## Testing

Tests are unit tests using Go's `testing` package. Test fixtures live in `testdata/`.
Run with `make test` or `go test ./... -v`.
