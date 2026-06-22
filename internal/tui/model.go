package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	vt "github.com/kujtimiihoxha/vimtea"

	"github.com/martins-fresh/conductor-config-tui/internal/store"
)

type focusArea int

const (
	focusConductors focusArea = iota
	focusFiles
	focusContent
)

type mode int

const (
	modeBrowse mode = iota
	modeEdit
	modeConfirm
	modeInput
)

// inputPurpose distinguishes what a text-input modal is collecting.
type inputPurpose int

const (
	inputNewConductor inputPurpose = iota
	inputNewFile
)

// confirmTarget distinguishes what a confirmation modal will act on.
type confirmTarget int

const (
	confirmDeleteFile confirmTarget = iota
	confirmDeleteConductor
)

type confirmState struct {
	target    confirmTarget
	prompt    string
	file      store.File
	conductor store.Conductor
	steps     int // remaining confirmations required (shared files need 2)
}

// Model is the root Bubble Tea model.
type Model struct {
	store      store.Store
	conductors []store.Conductor
	files      []store.File

	condCursor int
	fileCursor int

	focus focusArea
	mode  mode

	vp     viewport.Model
	editor vt.Editor // vim-style editor, non-nil only while editing
	input  textinput.Model
	help   help.Model
	keys   keyMap

	width  int
	height int
	ready  bool

	// layout dimensions computed on resize
	leftW      int
	midW       int
	rightW     int
	bodyHeight int
	paneOuter  int

	content          string
	contentPath      string
	contentTruncated bool
	editingFile      store.File

	status    string
	statusErr bool

	confirm      confirmState
	inputPurpose inputPurpose

	err error
}

// New builds the initial model and performs first discovery.
func New(s store.Store) Model {
	ti := textinput.New()
	ti.Placeholder = "name"
	ti.CharLimit = 128

	h := help.New()

	m := Model{
		store: s,
		focus: focusConductors,
		mode:  modeBrowse,
		input: ti,
		help:  h,
		keys:  defaultKeys(),
	}
	m.reload()
	m.loadContent()
	return m
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// reload re-discovers conductors and the current file list, clamping cursors.
func (m *Model) reload() {
	conductors, err := m.store.Discover()
	if err != nil {
		m.err = err
		return
	}
	m.conductors = conductors
	if m.condCursor >= len(m.conductors) {
		m.condCursor = len(m.conductors) - 1
	}
	if m.condCursor < 0 {
		m.condCursor = 0
	}
	m.reloadFiles()
}

// reloadFiles refreshes the file list for the selected conductor.
func (m *Model) reloadFiles() {
	if len(m.conductors) == 0 {
		m.files = nil
		return
	}
	files, err := m.store.ListFiles(m.conductors[m.condCursor])
	if err != nil {
		m.err = err
		m.files = nil
		return
	}
	m.files = files
	if m.fileCursor >= len(m.files) {
		m.fileCursor = len(m.files) - 1
	}
	if m.fileCursor < 0 {
		m.fileCursor = 0
	}
}

func (m *Model) currentConductor() (store.Conductor, bool) {
	if len(m.conductors) == 0 {
		return store.Conductor{}, false
	}
	return m.conductors[m.condCursor], true
}

func (m *Model) currentFile() (store.File, bool) {
	if len(m.files) == 0 || m.fileCursor < 0 || m.fileCursor >= len(m.files) {
		return store.File{}, false
	}
	return m.files[m.fileCursor], true
}

// loadContent reads the selected file into the viewport.
func (m *Model) loadContent() {
	f, ok := m.currentFile()
	if !ok {
		m.content = ""
		m.contentPath = ""
		m.contentTruncated = false
		m.vp.SetContent("")
		return
	}
	text, truncated, err := m.store.Read(f)
	if err != nil {
		m.setError(fmt.Sprintf("read failed: %v", err))
		m.content = ""
		m.vp.SetContent("")
		return
	}
	m.content = text
	m.contentPath = f.Path
	m.contentTruncated = truncated
	m.vp.SetContent(text)
	m.vp.GotoTop()
}

func (m *Model) setStatus(s string) {
	m.status = s
	m.statusErr = false
}

func (m *Model) setError(s string) {
	m.status = s
	m.statusErr = true
}
