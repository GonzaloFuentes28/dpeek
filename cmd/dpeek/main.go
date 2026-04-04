package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/GonzaloFuentes28/dpeek/internal/detect"
	"github.com/GonzaloFuentes28/dpeek/internal/model"
)

var version = "dev"

func main() {
	delimiter := flag.String("delimiter", "", "Column delimiter (default: auto-detect from extension)")
	noHeader := flag.Bool("no-header", false, "Treat the first row as data, not a header")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "dpeek — Interactive terminal data viewer for CSV, TSV, and JSON\n\n")
		fmt.Fprintf(os.Stderr, "Usage: dpeek [flags] <file>\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("dpeek %s\n", version)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	path := args[0]

	// Check file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: file not found: %s\n", path)
		os.Exit(1)
	}

	// Parse delimiter
	var delimRune rune
	if *delimiter != "" {
		runes := []rune(*delimiter)
		if len(runes) != 1 {
			fmt.Fprintf(os.Stderr, "Error: delimiter must be a single character\n")
			os.Exit(1)
		}
		delimRune = runes[0]
	}

	// Detect format
	format, delim := detect.DetectFormat(path, delimRune)
	if format == detect.FormatUnknown {
		fmt.Fprintf(os.Stderr, "Error: unsupported file format. Supported: .csv, .tsv, .json, .jsonl\n")
		os.Exit(1)
	}

	// Create app
	app := model.NewApp(path, format, delim, *noHeader)

	// Run TUI
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
