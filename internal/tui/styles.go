package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var (
	colorAccent   = lipgloss.Color("205") // pink
	colorMuted    = lipgloss.Color("242")
	colorBorder   = lipgloss.Color("240")
	colorActive   = lipgloss.Color("205")
	colorText     = lipgloss.Color("252")
	colorError    = lipgloss.Color("203")
	colorOK       = lipgloss.Color("78")
	colorReadOnly = lipgloss.Color("245")
	colorBg       = lipgloss.Color("236")
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	paneActiveStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorActive)

	paneTitleStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Bold(true).
			Padding(0, 1)

	paneTitleActiveStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true).
				Padding(0, 1)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("231")).
				Background(colorAccent).
				Bold(true)

	itemStyle = lipgloss.NewStyle().Foreground(colorText)

	readOnlyItemStyle = lipgloss.NewStyle().Foreground(colorReadOnly)

	pathStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	statusStyle = lipgloss.NewStyle().Foreground(colorMuted)

	statusErrStyle = lipgloss.NewStyle().Foreground(colorError).Bold(true)

	statusOKStyle = lipgloss.NewStyle().Foreground(colorOK).Bold(true)

	badgeROStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("235")).
			Background(colorReadOnly).
			Padding(0, 1)

	badgeEditStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("235")).
			Background(colorOK).
			Padding(0, 1)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(1, 3)

	modalTitleStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	modalDangerStyle = lipgloss.NewStyle().
				Foreground(colorError).
				Bold(true)

	helpStyle = lipgloss.NewStyle().Foreground(colorMuted)

	scrollThumbStyle       = lipgloss.NewStyle().Foreground(colorMuted)
	scrollThumbActiveStyle = lipgloss.NewStyle().Foreground(colorAccent)
	scrollTrackStyle       = lipgloss.NewStyle().Foreground(colorBorder)
)

// gutterStyle hides the vim editor's line-number gutter as much as styling
// allows: vimtea always renders a 4-column gutter with no way to disable it, so
// we paint the digits in the terminal's actual background colour to make them
// invisible. The 4-column blank margin still remains — that is a vimtea
// limitation. Falls back to a dark grey if the background can't be queried.
func gutterStyle() lipgloss.Style {
	bg := termenv.DefaultOutput().BackgroundColor()
	if bg == nil {
		return lipgloss.NewStyle().Foreground(colorBg)
	}
	hex := termenv.ConvertToRGB(bg).Hex()
	if hex == "" {
		return lipgloss.NewStyle().Foreground(colorBg)
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
}
