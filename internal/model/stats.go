package model

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	csvpkg "github.com/GonzaloFuentes28/dpeek/internal/csv"
	"github.com/GonzaloFuentes28/dpeek/internal/style"
)

// ColumnStats holds computed statistics for a single column.
type ColumnStats struct {
	Name        string
	InferType   string // "integer", "float", "boolean", "string", "mixed"
	TotalCount  int
	EmptyCount  int
	UniqueCount int
	MinVal      string
	MaxVal      string
	Mean        float64
	HasNumeric  bool
}

// ComputeStats calculates statistics for a column.
func ComputeStats(data *csvpkg.DataSet, col int) ColumnStats {
	stats := ColumnStats{
		Name:       data.Headers[col],
		TotalCount: data.RowCount(),
	}

	if data.RowCount() == 0 || col >= data.ColCount() {
		stats.InferType = "empty"
		return stats
	}

	unique := make(map[string]struct{})
	var numericVals []float64
	intCount, floatCount, boolCount, emptyCount := 0, 0, 0, 0

	for _, row := range data.Rows {
		val := cellValue(row, col)
		if val == "" {
			emptyCount++
			continue
		}
		unique[val] = struct{}{}

		lower := strings.ToLower(val)
		if lower == "true" || lower == "false" {
			boolCount++
			continue
		}

		if f, err := strconv.ParseFloat(val, 64); err == nil {
			numericVals = append(numericVals, f)
			if _, intErr := strconv.ParseInt(val, 10, 64); intErr == nil {
				intCount++
			} else {
				floatCount++
			}
		}
	}

	stats.EmptyCount = emptyCount
	stats.UniqueCount = len(unique)

	nonEmpty := stats.TotalCount - emptyCount
	if nonEmpty == 0 {
		stats.InferType = "empty"
		return stats
	}

	// Type inference
	switch {
	case intCount == nonEmpty:
		stats.InferType = "integer"
	case intCount+floatCount == nonEmpty:
		stats.InferType = "float"
	case boolCount == nonEmpty:
		stats.InferType = "boolean"
	case intCount+floatCount > 0 && boolCount > 0:
		stats.InferType = "mixed"
	default:
		stats.InferType = "string"
	}

	// Numeric stats
	if len(numericVals) > 0 {
		stats.HasNumeric = true
		minV, maxV := numericVals[0], numericVals[0]
		sum := 0.0
		for _, v := range numericVals {
			sum += v
			if v < minV {
				minV = v
			}
			if v > maxV {
				maxV = v
			}
		}
		stats.Mean = sum / float64(len(numericVals))

		if stats.InferType == "integer" {
			stats.MinVal = fmt.Sprintf("%d", int64(minV))
			stats.MaxVal = fmt.Sprintf("%d", int64(maxV))
		} else {
			stats.MinVal = fmt.Sprintf("%.2f", minV)
			stats.MaxVal = fmt.Sprintf("%.2f", maxV)
		}
	} else {
		// String min/max (lexicographic)
		first := true
		for v := range unique {
			if first {
				stats.MinVal = v
				stats.MaxVal = v
				first = false
				continue
			}
			if strings.ToLower(v) < strings.ToLower(stats.MinVal) {
				stats.MinVal = v
			}
			if strings.ToLower(v) > strings.ToLower(stats.MaxVal) {
				stats.MaxVal = v
			}
		}
	}

	return stats
}

// RenderStats renders a stats panel for display.
func RenderStats(cs ColumnStats, width, height int) string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(style.ColorHeader).
		Padding(1, 2)

	titleStyle := lipgloss.NewStyle().
		Foreground(style.ColorHeader).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(style.ColorFKeyLabel).
		Bold(true).
		Width(12)

	valueStyle := lipgloss.NewStyle().
		Foreground(style.ColorNormal)

	var lines []string
	lines = append(lines, titleStyle.Render(fmt.Sprintf("Column: %s", cs.Name)))
	lines = append(lines, "")
	lines = append(lines, labelStyle.Render("Type")+valueStyle.Render(cs.InferType))
	lines = append(lines, labelStyle.Render("Total")+valueStyle.Render(fmt.Sprintf("%d", cs.TotalCount)))
	lines = append(lines, labelStyle.Render("Empty")+valueStyle.Render(fmt.Sprintf("%d", cs.EmptyCount)))
	lines = append(lines, labelStyle.Render("Unique")+valueStyle.Render(fmt.Sprintf("%d", cs.UniqueCount)))
	lines = append(lines, labelStyle.Render("Min")+valueStyle.Render(cs.MinVal))
	lines = append(lines, labelStyle.Render("Max")+valueStyle.Render(cs.MaxVal))

	if cs.HasNumeric && !math.IsNaN(cs.Mean) {
		lines = append(lines, labelStyle.Render("Mean")+valueStyle.Render(fmt.Sprintf("%.2f", cs.Mean)))
	}

	lines = append(lines, "")
	lines = append(lines, style.DimStyle.Render("Press F6 or Esc to close"))

	content := boxStyle.Render(strings.Join(lines, "\n"))

	// Center in viewport
	contentWidth := lipgloss.Width(content)
	contentHeight := lipgloss.Height(content)

	padLeft := ""
	if width > contentWidth {
		padLeft = strings.Repeat(" ", (width-contentWidth)/2)
	}

	padTop := ""
	if height > contentHeight {
		padTop = strings.Repeat("\n", (height-contentHeight)/2)
	}

	var centeredLines []string
	for _, line := range strings.Split(content, "\n") {
		centeredLines = append(centeredLines, padLeft+line)
	}

	return padTop + strings.Join(centeredLines, "\n")
}
