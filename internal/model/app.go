package model

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	csvpkg "github.com/GonzaloFuentes28/dpeek/internal/csv"
	"github.com/GonzaloFuentes28/dpeek/internal/detect"
	jsonpkg "github.com/GonzaloFuentes28/dpeek/internal/json"
	"github.com/GonzaloFuentes28/dpeek/internal/style"
)

// App is the top-level bubbletea model that dispatches to CSV or JSON mode.
type App struct {
	mode      detect.Format
	csvModel  CSVModel
	jsonModel JSONModel
	width     int
	height    int
	err       error
	quitPending bool // true when user pressed q with unsaved changes
}

// NewApp creates a new App model by loading the file and detecting its format.
func NewApp(path string, format detect.Format, delimiter rune, noHeader bool, isTempFile bool) App {
	switch format {
	case detect.FormatCSV, detect.FormatTSV:
		// Use chunked loading for files larger than 5MB
		const chunkThreshold = 5 * 1024 * 1024
		info, _ := os.Stat(path)
		if info != nil && info.Size() > chunkThreshold {
			data, cr, err := csvpkg.LoadChunk(path, delimiter, !noHeader, csvpkg.DefaultChunkSize)
			if err != nil {
				return App{err: err}
			}
			if cr != nil {
				cm := NewCSVModelChunked(data, cr)
				cm.isTempFile = isTempFile
				return App{
					mode:     format,
					csvModel: cm,
				}
			}
			// File fully read within first chunk
			cm := NewCSVModel(data)
			cm.isTempFile = isTempFile
			return App{
				mode:     format,
				csvModel: cm,
			}
		}
		data, err := csvpkg.Load(path, delimiter, !noHeader)
		if err != nil {
			return App{err: err}
		}
		cm := NewCSVModel(data)
		cm.isTempFile = isTempFile
		return App{
			mode:     format,
			csvModel: cm,
		}

	case detect.FormatJSON:
		root, err := jsonpkg.Parse(path)
		if err != nil {
			return App{err: err}
		}
		jm := NewJSONModel(root, path, false)
		jm.isTempFile = isTempFile
		return App{
			mode:      format,
			jsonModel: jm,
		}

	case detect.FormatJSONL:
		const chunkThreshold = 5 * 1024 * 1024
		info, _ := os.Stat(path)
		if info != nil && info.Size() > chunkThreshold {
			root, cr, err := jsonpkg.ParseJSONLChunk(path, jsonpkg.DefaultJSONLChunkSize)
			if err != nil {
				return App{err: err}
			}
			if cr != nil {
				jm := NewJSONModelChunked(root, path, cr)
				jm.isTempFile = isTempFile
				return App{
					mode:      format,
					jsonModel: jm,
				}
			}
			jm := NewJSONModel(root, path, true)
			jm.isTempFile = isTempFile
			return App{
				mode:      format,
				jsonModel: jm,
			}
		}
		root, err := jsonpkg.ParseJSONL(path)
		if err != nil {
			return App{err: err}
		}
		jm := NewJSONModel(root, path, true)
		jm.isTempFile = isTempFile
		return App{
			mode:      format,
			jsonModel: NewJSONModel(root, path, true),
		}

	default:
		return App{err: fmt.Errorf("format %s not yet supported", format)}
	}
}

// Init implements tea.Model.
func (a App) Init() tea.Cmd {
	switch a.mode {
	case detect.FormatCSV, detect.FormatTSV:
		return a.csvModel.Init()
	case detect.FormatJSON, detect.FormatJSONL:
		return a.jsonModel.Init()
	}
	return nil
}

// Update implements tea.Model.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case tea.KeyMsg:
		// Handle quit confirmation
		if a.quitPending {
			switch msg.String() {
			case "q", "f10", "esc":
				return a, tea.Quit
			default:
				a.quitPending = false
				a.setStatus("")
			}
			return a, nil
		}

		switch msg.String() {
		case "ctrl+c":
			return a, tea.Quit
		case "f10", "q":
			if a.hasActiveInput() || a.hasDismissableState() {
				break
			}
			if a.isModified() {
				a.quitPending = true
				a.setStatus(style.ErrorStatusStyle.Render("Unsaved changes! Press q again to quit, any other key to cancel"))
				return a, nil
			}
			return a, tea.Quit
		case "esc":
			if a.hasDismissableState() {
				break // let sub-model handle it
			}
			if a.isModified() {
				a.quitPending = true
				a.setStatus(style.ErrorStatusStyle.Render("Unsaved changes! Press Esc again to quit, any other key to cancel"))
				return a, nil
			}
			return a, tea.Quit
		}
	}

	switch a.mode {
	case detect.FormatCSV, detect.FormatTSV:
		var cmd tea.Cmd
		a.csvModel, cmd = a.csvModel.Update(msg)
		return a, cmd
	case detect.FormatJSON, detect.FormatJSONL:
		var cmd tea.Cmd
		a.jsonModel, cmd = a.jsonModel.Update(msg)
		return a, cmd
	}

	return a, nil
}

func (a App) isModified() bool {
	switch a.mode {
	case detect.FormatCSV, detect.FormatTSV:
		return a.csvModel.undo.IsModified()
	case detect.FormatJSON, detect.FormatJSONL:
		return a.jsonModel.undo.IsModified()
	}
	return false
}

func (a *App) setStatus(msg string) {
	switch a.mode {
	case detect.FormatCSV, detect.FormatTSV:
		a.csvModel.statusMsg = msg
	case detect.FormatJSON, detect.FormatJSONL:
		a.jsonModel.statusMsg = msg
	}
}

func (a App) hasDismissableState() bool {
	switch a.mode {
	case detect.FormatCSV, detect.FormatTSV:
		return a.csvModel.HasDismissableState()
	case detect.FormatJSON, detect.FormatJSONL:
		return a.jsonModel.HasDismissableState()
	}
	return false
}

func (a App) hasActiveInput() bool {
	switch a.mode {
	case detect.FormatCSV, detect.FormatTSV:
		return a.csvModel.HasActiveInput()
	case detect.FormatJSON, detect.FormatJSONL:
		return a.jsonModel.HasActiveInput()
	}
	return false
}

// View implements tea.Model.
func (a App) View() string {
	if a.err != nil {
		return "Error: " + a.err.Error() + "\n\nPress q to quit."
	}

	switch a.mode {
	case detect.FormatCSV, detect.FormatTSV:
		return a.csvModel.View()
	case detect.FormatJSON, detect.FormatJSONL:
		return a.jsonModel.View()
	default:
		return "Unsupported format\n\nPress q to quit."
	}
}
