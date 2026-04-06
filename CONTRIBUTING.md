# Contributing to dpeek

Thanks for your interest in contributing! Here's how to get started.

## Development setup

```bash
git clone https://github.com/GonzaloFuentes28/dpeek.git
cd dpeek
make build    # build binary
make test     # run tests
make lint     # run linter (requires golangci-lint)
```

Requires Go 1.24+.

## Making changes

1. Fork the repo and create a branch from `main`
2. Make your changes
3. Add tests for new functionality
4. Run `make test` and `make lint` to verify
5. Open a pull request against `main`

## Project structure

```
cmd/dpeek/main.go       Entry point, flags, stdin handling
internal/
  csv/reader.go          CSV/TSV parsing, DataSet, chunked loading
  json/tree.go           JSON/JSONL tree, chunked loading
  sql/engine.go          SQLite in-memory engine for SQL queries
  detect/detect.go       File format detection
  model/
    app.go               Top-level bubbletea model
    csv_model.go          CSV table view
    json_model.go         JSON tree view
    search.go             Search state (shared)
    filter.go             Row filtering (CSV)
    sort.go               Column sorting (CSV)
    stats.go              Column statistics + charts
    undo.go               Generic undo/redo stack
    help.go               Help overlay
  style/theme.go          Lipgloss styles
```

## Guidelines

- Keep PRs focused — one feature or fix per PR
- Follow existing code patterns and style
- Write tests for new features
- Update help text, man page (`dpeek.1`), and README for user-facing changes
- No external dependencies unless strictly necessary

## Reporting issues

Open an issue at https://github.com/GonzaloFuentes28/dpeek/issues with:
- What you expected vs what happened
- Steps to reproduce
- dpeek version (`dpeek --version`)
- OS and terminal

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
