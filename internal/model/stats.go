package model

import (
	"fmt"
	"math"
	"sort"
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
	NumericVals []float64          // raw numeric values for histogram
	FreqMap     map[string]int     // value frequencies for bar chart
}

// histBucket represents one bar in a histogram.
type histBucket struct {
	Label string
	Count int
}

// freqEntry represents a value and its count for frequency charts.
type freqEntry struct {
	Value string
	Count int
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
	freqMap := make(map[string]int)
	var numericVals []float64
	intCount, floatCount, boolCount, emptyCount := 0, 0, 0, 0

	for _, row := range data.Rows {
		val := cellValue(row, col)
		if val == "" {
			emptyCount++
			continue
		}
		unique[val] = struct{}{}
		freqMap[val]++

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

	stats.FreqMap = freqMap

	// Numeric stats
	if len(numericVals) > 0 {
		stats.HasNumeric = true
		stats.NumericVals = numericVals
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

	// Chart
	if cs.HasNumeric && len(cs.NumericVals) > 0 {
		lines = append(lines, "")
		lines = append(lines, titleStyle.Render("Distribution"))
		lines = append(lines, renderHistogram(cs.NumericVals, cs.InferType == "integer")...)
	} else if len(cs.FreqMap) > 0 && !cs.HasNumeric {
		lines = append(lines, "")
		lines = append(lines, titleStyle.Render("Top Values"))
		lines = append(lines, renderFreqChart(cs.FreqMap)...)
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

const (
	chartWidth   = 30
	histBuckets  = 8
	freqMaxItems = 10
)

var barChars = []rune{'░', '▓'}

func renderHistogram(vals []float64, isInt bool) []string {
	if len(vals) == 0 {
		return nil
	}

	minV, maxV := vals[0], vals[0]
	for _, v := range vals {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}

	numBuckets := histBuckets
	if maxV == minV {
		numBuckets = 1
	}

	bucketSize := (maxV - minV) / float64(numBuckets)
	if bucketSize == 0 {
		bucketSize = 1
	}

	counts := make([]int, numBuckets)
	for _, v := range vals {
		idx := int((v - minV) / bucketSize)
		if idx >= numBuckets {
			idx = numBuckets - 1
		}
		counts[idx]++
	}

	maxCount := 0
	for _, c := range counts {
		if c > maxCount {
			maxCount = c
		}
	}

	labelStyle := lipgloss.NewStyle().Foreground(style.ColorDim).Width(14).Align(lipgloss.Right)
	barStyle := lipgloss.NewStyle().Foreground(style.ColorHeader)
	countStyle := lipgloss.NewStyle().Foreground(style.ColorNormal)

	var lines []string
	for i, c := range counts {
		lo := minV + float64(i)*bucketSize
		hi := lo + bucketSize
		var label string
		if isInt {
			label = fmt.Sprintf("%d-%d", int64(lo), int64(hi))
		} else {
			label = fmt.Sprintf("%.1f-%.1f", lo, hi)
		}

		barLen := 0
		if maxCount > 0 {
			barLen = c * chartWidth / maxCount
		}
		if c > 0 && barLen == 0 {
			barLen = 1
		}
		bar := strings.Repeat("▓", barLen) + strings.Repeat("░", chartWidth-barLen)
		lines = append(lines, labelStyle.Render(label)+" "+barStyle.Render(bar)+" "+countStyle.Render(fmt.Sprintf("%d", c)))
	}
	return lines
}

func renderFreqChart(freqMap map[string]int) []string {
	entries := make([]freqEntry, 0, len(freqMap))
	for v, c := range freqMap {
		entries = append(entries, freqEntry{Value: v, Count: c})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Count > entries[j].Count
	})

	limit := freqMaxItems
	if len(entries) < limit {
		limit = len(entries)
	}
	entries = entries[:limit]

	maxCount := 0
	maxLabelLen := 0
	for _, e := range entries {
		if e.Count > maxCount {
			maxCount = e.Count
		}
		if len(e.Value) > maxLabelLen {
			maxLabelLen = len(e.Value)
		}
	}
	if maxLabelLen > 16 {
		maxLabelLen = 16
	}

	labelStyle := lipgloss.NewStyle().Foreground(style.ColorDim).Width(maxLabelLen + 2).Align(lipgloss.Right)
	barStyle := lipgloss.NewStyle().Foreground(style.ColorHeader)
	countStyle := lipgloss.NewStyle().Foreground(style.ColorNormal)

	var lines []string
	for _, e := range entries {
		label := e.Value
		if len(label) > 16 {
			label = label[:15] + "…"
		}

		barLen := 0
		if maxCount > 0 {
			barLen = e.Count * chartWidth / maxCount
		}
		if e.Count > 0 && barLen == 0 {
			barLen = 1
		}
		bar := strings.Repeat("▓", barLen) + strings.Repeat("░", chartWidth-barLen)
		lines = append(lines, labelStyle.Render(label)+" "+barStyle.Render(bar)+" "+countStyle.Render(fmt.Sprintf("%d", e.Count)))
	}
	return lines
}
