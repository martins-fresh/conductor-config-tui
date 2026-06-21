package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func lipglossHeight(s string) int { return lipgloss.Height(s) }

// View satisfies tea.Model.
func (m Model) View() string {
	if !m.ready {
		return "loading…"
	}
	if m.err != nil {
		return fmt.Sprintf("\n  error: %v\n\n  press q to quit\n", m.err)
	}

	header := m.headerView()
	panes := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.conductorsPane(),
		m.filesPane(),
		m.contentPane(),
	)
	status := m.statusView()
	helpView := m.help.View(m.keys)

	base := lipgloss.JoinVertical(lipgloss.Left, header, panes, status, helpView)

	switch m.mode {
	case modeConfirm:
		return m.overlay(m.confirmModal())
	case modeInput:
		return m.overlay(m.inputModal())
	}
	return base
}

// overlay centers a modal box over the full screen.
func (m Model) overlay(box string) string {
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) headerView() string {
	title := titleStyle.Render("baton")
	sub := lipgloss.NewStyle().Foreground(colorMuted).Render(" · conductor config")
	path := m.focusedPath()
	left := title + sub
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(path)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + pathStyle.Render(path)
}

// focusedPath returns the absolute path of whatever is currently focused.
func (m Model) focusedPath() string {
	switch m.focus {
	case focusConductors:
		if c, ok := m.currentConductor(); ok {
			return c.Dir
		}
	case focusFiles, focusContent:
		if f, ok := m.currentFile(); ok {
			return f.Path
		}
		if c, ok := m.currentConductor(); ok {
			return c.Dir
		}
	}
	return m.store.Root
}

func (m Model) conductorsPane() string {
	var lines []string
	for i, c := range m.conductors {
		label := c.Name
		line := truncate(label, m.leftW-2)
		if i == m.condCursor {
			line = selectedItemStyle.Width(m.leftW - 2).Render(truncate(label, m.leftW-2))
		} else {
			line = itemStyle.Render(line)
		}
		lines = append(lines, line)
	}
	return m.paneBox("Conductors", lines, m.leftW, m.focus == focusConductors, m.condCursor)
}

func (m Model) filesPane() string {
	var lines []string
	for i, f := range m.files {
		marker := " "
		if !f.Editable {
			marker = lipgloss.NewStyle().Foreground(colorReadOnly).Render("·")
		}
		label := marker + " " + f.Name
		if i == m.fileCursor {
			lines = append(lines, selectedItemStyle.Width(m.midW-2).Render(truncate(label, m.midW-2)))
		} else if f.Editable {
			lines = append(lines, itemStyle.Render(truncate(label, m.midW-2)))
		} else {
			lines = append(lines, readOnlyItemStyle.Render(truncate(label, m.midW-2)))
		}
	}
	if len(lines) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render("(no files)"))
	}
	return m.paneBox("Files", lines, m.midW, m.focus == focusFiles, m.fileCursor)
}

func (m Model) contentPane() string {
	title := "Content"
	var body, bar string
	active := m.focus == focusContent
	if m.mode == modeEdit {
		title = "Editing"
		body = m.ta.View()
		// Approximate the visible window from the cursor line so the thumb
		// tracks roughly where we are in the file.
		total := m.ta.LineCount()
		visible := m.ta.Height()
		offset := m.ta.Line() - visible/2
		bar = buildScrollbar(m.bodyHeight, total, visible, offset, true)
	} else {
		body = m.vp.View()
		bar = buildScrollbar(m.bodyHeight, m.vp.TotalLineCount(), m.vp.Height, m.vp.YOffset, active)
		if f, ok := m.currentFile(); ok {
			badge := badgeEditStyle.Render("editable")
			if !f.Editable {
				badge = badgeROStyle.Render("read-only")
			}
			title = "Content " + badge
			if m.contentTruncated {
				title += lipgloss.NewStyle().Foreground(colorError).Render(" (truncated)")
			}
		}
	}
	titleStyleSel := paneTitleStyle
	if active {
		titleStyleSel = paneTitleActiveStyle
	}
	header := titleStyleSel.Render(title)
	bodyWithBar := lipgloss.JoinHorizontal(lipgloss.Top, body, " ", bar)
	inner := lipgloss.JoinVertical(lipgloss.Left, header, bodyWithBar)

	box := paneStyle
	if active {
		box = paneActiveStyle
	}
	return box.Width(m.rightW - 2).Height(m.paneOuter - 2).Render(inner)
}

// scrollbarCols is the width reserved on the right of the content pane: one gap
// column plus the one-column bar.
const scrollbarCols = 2

// buildScrollbar renders a vertical scrollbar of the given height. The thumb's
// size reflects the fraction of content visible and its position reflects how
// far the view is scrolled.
func buildScrollbar(height, total, visible, offset int, active bool) string {
	if height <= 0 {
		return ""
	}
	thumbStyle := scrollThumbStyle
	if active {
		thumbStyle = scrollThumbActiveStyle
	}

	// Everything fits: the whole bar is the thumb.
	if total <= 0 || visible <= 0 || total <= visible {
		var b strings.Builder
		for i := 0; i < height; i++ {
			if i > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(thumbStyle.Render("█"))
		}
		return b.String()
	}

	thumb := int(roundDiv(height*visible, total))
	if thumb < 1 {
		thumb = 1
	}
	if thumb > height {
		thumb = height
	}

	maxOffset := total - visible
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	pos := int(roundDiv((height-thumb)*offset, maxOffset))
	if pos < 0 {
		pos = 0
	}
	if pos > height-thumb {
		pos = height - thumb
	}

	var b strings.Builder
	for i := 0; i < height; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		if i >= pos && i < pos+thumb {
			b.WriteString(thumbStyle.Render("█"))
		} else {
			b.WriteString(scrollTrackStyle.Render("│"))
		}
	}
	return b.String()
}

// roundDiv returns a/b rounded to the nearest integer (b > 0).
func roundDiv(a, b int) int {
	if b == 0 {
		return 0
	}
	return (a + b/2) / b
}

// paneBox renders a titled, bordered list pane with a scrolling window around
// the cursor.
func (m Model) paneBox(title string, lines []string, outerW int, active bool, cursor int) string {
	titleSt := paneTitleStyle
	if active {
		titleSt = paneTitleActiveStyle
	}

	visible := windowLines(lines, cursor, m.bodyHeight)
	// Pad to fill the body height so all panes are the same height.
	for len(visible) < m.bodyHeight {
		visible = append(visible, "")
	}

	body := strings.Join(visible, "\n")
	inner := lipgloss.JoinVertical(lipgloss.Left, titleSt.Render(title), body)

	box := paneStyle
	if active {
		box = paneActiveStyle
	}
	return box.Width(outerW - 2).Height(m.paneOuter - 2).Render(inner)
}

// windowLines returns a slice of at most height lines scrolled so cursor stays
// visible.
func windowLines(lines []string, cursor, height int) []string {
	if len(lines) <= height {
		return lines
	}
	start := cursor - height/2
	if start < 0 {
		start = 0
	}
	if start+height > len(lines) {
		start = len(lines) - height
	}
	return lines[start : start+height]
}

func (m Model) statusView() string {
	if m.status == "" {
		return statusStyle.Render(" ")
	}
	st := statusStyle
	if m.statusErr {
		st = statusErrStyle
	} else {
		st = statusOKStyle
	}
	return st.Render("» " + m.status)
}

func (m Model) confirmModal() string {
	title := modalDangerStyle.Render("⚠  Confirm delete")
	prompt := m.confirm.prompt
	hint := helpStyle.Render("y = yes    n / esc = cancel")
	box := lipgloss.JoinVertical(lipgloss.Left, title, "", prompt, "", hint)
	return modalStyle.BorderForeground(colorError).Render(box)
}

func (m Model) inputModal() string {
	var title string
	switch m.inputPurpose {
	case inputNewConductor:
		title = modalTitleStyle.Render("New conductor")
	case inputNewFile:
		c, _ := m.currentConductor()
		title = modalTitleStyle.Render("New file in " + c.Name)
	}
	hint := helpStyle.Render("enter = create    esc = cancel")
	box := lipgloss.JoinVertical(lipgloss.Left, title, "", m.input.View(), "", hint)
	return modalStyle.Render(box)
}

// truncate shortens s to fit width display cells, adding an ellipsis.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	// Trim rune by rune until it fits with the ellipsis.
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}
