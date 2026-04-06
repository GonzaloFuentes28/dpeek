package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

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
		fmt.Fprintf(os.Stderr, "Usage: dpeek [flags] <file>\n")
		fmt.Fprintf(os.Stderr, "       command | dpeek\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("dpeek %s\n", version)
		os.Exit(0)
	}

	args := flag.Args()

	// Check if reading from stdin
	var path string
	var stdinTmp string
	if len(args) == 0 || args[0] == "-" {
		// Check if stdin is a pipe
		info, _ := os.Stdin.Stat()
		if info.Mode()&os.ModeCharDevice != 0 {
			flag.Usage()
			os.Exit(1)
		}
		// Determine extension for format detection
		ext := ".csv" // default
		if *delimiter == "\t" {
			ext = ".tsv"
		}
		if len(args) > 0 && args[0] != "-" {
			// If a filename hint was somehow provided, use its extension
			if e := filepath.Ext(args[0]); e != "" {
				ext = e
			}
		}
		// Sniff content to detect JSON
		buf, _ := io.ReadAll(os.Stdin)
		trimmed := strings.TrimSpace(string(buf))
		if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
			ext = ".json"
			// Check if it looks like JSONL (multiple lines starting with {)
			lines := strings.SplitN(trimmed, "\n", 3)
			if len(lines) >= 2 && len(strings.TrimSpace(lines[1])) > 0 && strings.TrimSpace(lines[1])[0] == '{' {
				ext = ".jsonl"
			}
		}
		// Write to temp file
		tmp, err := os.CreateTemp("", "dpeek-*"+ext)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not create temp file: %v\n", err)
			os.Exit(1)
		}
		if _, err := tmp.Write(buf); err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not write temp file: %v\n", err)
			os.Exit(1)
		}
		tmp.Close()
		path = tmp.Name()
		stdinTmp = path
	} else if len(args) == 1 {
		path = args[0]
	} else {
		flag.Usage()
		os.Exit(1)
	}

	// Handle remote URLs
	var remoteTmp string
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		u, err := url.Parse(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid URL: %v\n", err)
			os.Exit(1)
		}
		ext := filepath.Ext(u.Path)
		if ext == "" {
			ext = ".csv" // default
		}
		fmt.Fprintf(os.Stderr, "Downloading %s...\n", path)
		resp, err := http.Get(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: download failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "Error: HTTP %d\n", resp.StatusCode)
			os.Exit(1)
		}
		tmp, err := os.CreateTemp("", "dpeek-*"+ext)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not create temp file: %v\n", err)
			os.Exit(1)
		}
		if _, err := io.Copy(tmp, resp.Body); err != nil {
			fmt.Fprintf(os.Stderr, "Error: download failed: %v\n", err)
			os.Exit(1)
		}
		tmp.Close()
		path = tmp.Name()
		remoteTmp = path
	}

	// Clean up temp files on exit
	if stdinTmp != "" {
		defer os.Remove(stdinTmp)
	}
	if remoteTmp != "" {
		defer os.Remove(remoteTmp)
	}

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
	isTempFile := stdinTmp != "" || remoteTmp != ""
	app := model.NewApp(path, format, delim, *noHeader, isTempFile)

	// Run TUI
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
