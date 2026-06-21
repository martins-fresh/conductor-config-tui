package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Update satisfies tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.resize()
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case modeEdit:
			return m.updateEdit(msg)
		case modeInput:
			return m.updateInput(msg)
		case modeConfirm:
			return m.updateConfirm(msg)
		default:
			return m.updateBrowse(msg)
		}
	}
	return m, nil
}

// resize recomputes layout dimensions for all sub-components.
func (m *Model) resize() {
	if m.width == 0 || m.height == 0 {
		return
	}
	helpHeight := lipglossHeight(m.help.View(m.keys))
	// header(1) + status(1) + help
	paneOuter := m.height - 2 - helpHeight
	if paneOuter < 5 {
		paneOuter = 5
	}
	paneInner := paneOuter - 2  // rounded border top+bottom
	bodyHeight := paneInner - 1 // pane title line
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	leftW, midW, rightW := paneWidths(m.width)

	m.leftW, m.midW, m.rightW = leftW, midW, rightW
	m.bodyHeight = bodyHeight
	m.paneOuter = paneOuter

	m.vp.Width = rightW - 2
	m.vp.Height = bodyHeight
	m.vp.SetContent(m.content)

	m.ta.SetWidth(rightW - 2)
	m.ta.SetHeight(bodyHeight)

	m.help.Width = m.width
}

// paneWidths splits total width across the three panes (outer widths including
// borders).
func paneWidths(total int) (left, mid, right int) {
	left = 26
	mid = 30
	if total < left+mid+24 {
		// Shrink proportionally on narrow terminals.
		left = total * 26 / 80
		mid = total * 30 / 80
	}
	if left < 12 {
		left = 12
	}
	if mid < 14 {
		mid = 14
	}
	right = total - left - mid
	if right < 16 {
		right = 16
	}
	return left, mid, right
}

func (m Model) updateBrowse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		m.resize()
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		m.reload()
		m.loadContent()
		m.setStatus("refreshed")
		return m, nil

	case key.Matches(msg, m.keys.Tab):
		m.focus = (m.focus + 1) % 3
		return m, nil

	case key.Matches(msg, m.keys.Left):
		if m.focus > focusConductors {
			m.focus--
		}
		return m, nil

	case key.Matches(msg, m.keys.Right):
		if m.focus < focusContent {
			m.focus++
		}
		return m, nil

	case key.Matches(msg, m.keys.Up):
		m.moveUp()
		return m, nil

	case key.Matches(msg, m.keys.Down):
		m.moveDown()
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		return m.handleEnter()

	case key.Matches(msg, m.keys.Edit):
		return m.startEdit()

	case key.Matches(msg, m.keys.NewFile):
		return m.startInput(inputNewFile)

	case key.Matches(msg, m.keys.NewCond):
		return m.startInput(inputNewConductor)

	case key.Matches(msg, m.keys.Delete):
		return m.startDelete()
	}

	// Forward scrolling keys to the viewport when content is focused.
	if m.focus == focusContent {
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) moveUp() {
	switch m.focus {
	case focusConductors:
		if m.condCursor > 0 {
			m.condCursor--
			m.fileCursor = 0
			m.reloadFiles()
			m.loadContent()
		}
	case focusFiles:
		if m.fileCursor > 0 {
			m.fileCursor--
			m.loadContent()
		}
	case focusContent:
		m.vp.ScrollUp(1)
	}
}

func (m *Model) moveDown() {
	switch m.focus {
	case focusConductors:
		if m.condCursor < len(m.conductors)-1 {
			m.condCursor++
			m.fileCursor = 0
			m.reloadFiles()
			m.loadContent()
		}
	case focusFiles:
		if m.fileCursor < len(m.files)-1 {
			m.fileCursor++
			m.loadContent()
		}
	case focusContent:
		m.vp.ScrollDown(1)
	}
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusConductors:
		m.reloadFiles()
		m.loadContent()
		m.focus = focusFiles
	case focusFiles:
		m.loadContent()
		m.focus = focusContent
	}
	return m, nil
}

func (m Model) startEdit() (tea.Model, tea.Cmd) {
	f, ok := m.currentFile()
	if !ok {
		m.setError("no file selected")
		return m, nil
	}
	if !f.Editable {
		m.setError(fmt.Sprintf("%s is read-only", f.Name))
		return m, nil
	}
	// Ensure we are editing what is shown.
	if m.contentPath != f.Path {
		m.loadContent()
	}
	m.editingFile = f
	m.ta.SetValue(m.content)
	m.ta.CursorStart()
	m.ta.Focus()
	m.mode = modeEdit
	m.focus = focusContent
	m.setStatus(fmt.Sprintf("editing %s — ctrl+s save, esc cancel", f.Name))
	return m, nil
}

func (m Model) updateEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Save):
		content := []byte(m.ta.Value())
		if err := m.store.Save(m.editingFile, content); err != nil {
			m.setError(fmt.Sprintf("save failed: %v", err))
			return m, nil
		}
		m.ta.Blur()
		m.mode = modeBrowse
		m.reloadFiles()
		m.loadContent()
		m.setStatus(fmt.Sprintf("saved %s", m.editingFile.Name))
		return m, nil

	case key.Matches(msg, m.keys.Cancel):
		m.ta.Blur()
		m.mode = modeBrowse
		m.setStatus("edit cancelled")
		return m, nil
	}
	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	return m, cmd
}

func (m Model) startInput(p inputPurpose) (tea.Model, tea.Cmd) {
	if p == inputNewFile {
		c, ok := m.currentConductor()
		if !ok {
			m.setError("no conductor selected")
			return m, nil
		}
		if c.Shared {
			m.setError("create files inside a conductor, not (shared)")
			return m, nil
		}
	}
	m.inputPurpose = p
	m.input.SetValue("")
	switch p {
	case inputNewFile:
		m.input.Placeholder = "POLICY.md"
	case inputNewConductor:
		m.input.Placeholder = "new-conductor"
	}
	m.input.Focus()
	m.mode = modeInput
	return m, nil
}

func (m Model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.input.Blur()
		m.mode = modeBrowse
		m.setStatus("cancelled")
		return m, nil

	case msg.Type == tea.KeyEnter:
		name := m.input.Value()
		m.input.Blur()
		m.mode = modeBrowse
		return m.submitInput(name)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) submitInput(name string) (tea.Model, tea.Cmd) {
	switch m.inputPurpose {
	case inputNewConductor:
		c, err := m.store.ScaffoldConductor(name)
		if err != nil {
			m.setError(fmt.Sprintf("create failed: %v", err))
			return m, nil
		}
		m.reload()
		m.selectConductor(c.Name)
		m.reloadFiles()
		m.loadContent()
		m.setStatus(fmt.Sprintf("created conductor %s", c.Name))
	case inputNewFile:
		c, ok := m.currentConductor()
		if !ok {
			return m, nil
		}
		f, err := m.store.CreateFile(c, name)
		if err != nil {
			m.setError(fmt.Sprintf("create failed: %v", err))
			return m, nil
		}
		m.reloadFiles()
		m.selectFile(f.Name)
		m.loadContent()
		m.setStatus(fmt.Sprintf("created %s", f.Name))
	}
	return m, nil
}

func (m *Model) selectConductor(name string) {
	for i, c := range m.conductors {
		if c.Name == name {
			m.condCursor = i
			return
		}
	}
}

func (m *Model) selectFile(name string) {
	for i, f := range m.files {
		if f.Name == name {
			m.fileCursor = i
			return
		}
	}
}

func (m Model) startDelete() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusConductors:
		c, ok := m.currentConductor()
		if !ok {
			return m, nil
		}
		if c.Shared {
			m.setError("the shared entry cannot be deleted")
			return m, nil
		}
		m.confirm = confirmState{
			target:    confirmDeleteConductor,
			conductor: c,
			prompt:    fmt.Sprintf("Delete conductor %q and ALL its files?", c.Name),
			steps:     1,
		}
		m.mode = modeConfirm
	case focusFiles, focusContent:
		f, ok := m.currentFile()
		if !ok {
			m.setError("no file selected")
			return m, nil
		}
		c, _ := m.currentConductor()
		steps := 1
		prompt := fmt.Sprintf("Delete file %q?", f.Name)
		if c.Shared {
			// Shared top-level files are high-value: require a double confirm.
			steps = 2
			prompt = fmt.Sprintf("Delete SHARED file %q? (requires double confirm)", f.Name)
		}
		m.confirm = confirmState{
			target: confirmDeleteFile,
			file:   f,
			prompt: prompt,
			steps:  steps,
		}
		m.mode = modeConfirm
	}
	return m, nil
}

func (m Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.confirm.steps--
		if m.confirm.steps > 0 {
			m.confirm.prompt = fmt.Sprintf("Are you SURE? Confirm again to delete %q.", confirmName(m.confirm))
			return m, nil
		}
		return m.performDelete()
	case "n", "N", "esc":
		m.mode = modeBrowse
		m.setStatus("delete cancelled")
		return m, nil
	}
	return m, nil
}

func confirmName(c confirmState) string {
	if c.target == confirmDeleteConductor {
		return c.conductor.Name
	}
	return c.file.Name
}

func (m Model) performDelete() (tea.Model, tea.Cmd) {
	m.mode = modeBrowse
	switch m.confirm.target {
	case confirmDeleteConductor:
		if err := m.store.DeleteConductor(m.confirm.conductor); err != nil {
			m.setError(fmt.Sprintf("delete failed: %v", err))
			return m, nil
		}
		m.reload()
		m.loadContent()
		m.focus = focusConductors
		m.setStatus(fmt.Sprintf("deleted conductor %s", m.confirm.conductor.Name))
	case confirmDeleteFile:
		if err := m.store.Delete(m.confirm.file); err != nil {
			m.setError(fmt.Sprintf("delete failed: %v", err))
			return m, nil
		}
		m.reloadFiles()
		m.loadContent()
		if m.focus == focusContent {
			m.focus = focusFiles
		}
		m.setStatus(fmt.Sprintf("deleted %s", m.confirm.file.Name))
	}
	return m, nil
}
