# dpeek

Interactive terminal data viewer and editor for CSV, TSV, JSON, and JSONL files.

![Go 1.22+](https://img.shields.io/badge/go-1.22%2B-blue)
![License: MIT](https://img.shields.io/badge/license-MIT-green)

## What it does

`dpeek` opens data files in an interactive TUI — like htop, but for your data. Navigate, search, filter, sort, edit, and save, all from the terminal.

**CSV/TSV mode** — scrollable table with cell editing:
```
┌─ data.csv ─────────────────────────────────────────────┐
│       │ name       │ age │ city       │ salary │ active │
│───────┼────────────┼─────┼────────────┼────────┼────────│
│     1 │ Alice      │  32 │ Madrid     │  45000 │ true   │
│     2 │▶Bob        │  28 │ Barcelona  │  52000 │ false  │
│     3 │ Carol      │  45 │ Valencia   │  61000 │ true   │
├────────────────────────────────────────────────────────-┤
│ 20 rows  │  5 cols  │  SORT: name ▲                    │
│ F1 Help  F2 Save  F3 Search  F4 Filter  F5 Sort  F10 Q│
└────────────────────────────────────────────────────────-┘
```

**JSON mode** — collapsible tree with color-coded types:
```
▼ {
    ▼ address: {
        city: "Madrid"
        country: "Spain"
      ▼ coordinates: {
            lat: 40.4168
            lng: -3.7038
        }
    }
    ▶ employees: [3 items]
    ▶ tags: [3 items]
    metadata: null
  }
```

## Installation

### With Homebrew (macOS/Linux)

```bash
brew tap GonzaloFuentes28/tap
brew install dpeek
```

### From source

```bash
git clone https://github.com/GonzaloFuentes28/dpeek.git
cd dpeek
make install
```

Requires Go 1.22+.

## Usage

```bash
# Open a CSV file
dpeek data.csv

# Open a TSV file
dpeek data.tsv

# Open a JSON file
dpeek config.json

# Open a JSONL file
dpeek logs.jsonl

# Custom delimiter
dpeek data.txt --delimiter ";"

# CSV without header row
dpeek data.csv --no-header
```

## Keyboard shortcuts

### CSV/TSV mode

| Key | Action |
|-----|--------|
| `↑↓←→` / `hjkl` | Navigate cells |
| `PgUp` / `PgDn` | Scroll page |
| `Home` / `End` | Jump to first/last cell |
| `Enter` | Edit selected cell |
| `Tab` | Confirm edit, move to next cell |
| `Esc` | Cancel edit / clear search/filter |
| `F1` | Help |
| `F2` / `Ctrl+S` | Save file |
| `F3` / `/` | Search all cells |
| `F4` | Filter rows |
| `F5` | Sort by current column |
| `F6` | Column statistics |
| `n` / `N` | Next/previous search match |
| `F10` / `q` | Quit |

### JSON mode

| Key | Action |
|-----|--------|
| `↑↓` / `jk` | Navigate nodes |
| `←` / `h` | Collapse or go to parent |
| `→` / `l` | Expand or go to child |
| `Enter` / `Space` | Toggle expand/collapse or edit leaf |
| `Tab` | Confirm edit, jump to next leaf |
| `e` | Expand all |
| `c` | Collapse all |
| `F1` | Help |
| `F2` / `Ctrl+S` | Save file |
| `F3` / `/` | Search keys and values |
| `n` / `N` | Next/previous match |
| `F10` / `q` | Quit |

## Supported formats

| Format | Extensions | Features |
|--------|-----------|----------|
| CSV | `.csv` | Full: view, edit, search, filter, sort, stats, save |
| TSV | `.tsv` | Full: same as CSV with tab delimiter |
| JSON | `.json` | View, edit leaf values, search, save |
| JSONL | `.jsonl`, `.ndjson` | View, edit, search (each line as array item) |

## License

MIT
