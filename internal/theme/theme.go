package theme

import "github.com/charmbracelet/lipgloss"

// Styles groups the colors used across the TUI.
type Styles struct {
	Title    lipgloss.Style
	Muted    lipgloss.Style
	Accent   lipgloss.Style
	OK       lipgloss.Style
	Warn     lipgloss.Style
	Err      lipgloss.Style
	Selected lipgloss.Style
	Header   lipgloss.Style
	Border   lipgloss.Style
}

// Default returns the default dark-terminal palette.
func Default() Styles {
	base := lipgloss.NewStyle()
	return Styles{
		Title:    base.Bold(true).Foreground(lipgloss.Color("6")),
		Muted:    base.Foreground(lipgloss.Color("8")),
		Accent:   base.Foreground(lipgloss.Color("5")),
		OK:       base.Foreground(lipgloss.Color("2")),
		Warn:     base.Foreground(lipgloss.Color("3")),
		Err:      base.Foreground(lipgloss.Color("1")),
		Selected: base.Foreground(lipgloss.Color("15")).Background(lipgloss.Color("4")),
		Header:   base.Bold(true).Foreground(lipgloss.Color("7")),
		Border:   base.Foreground(lipgloss.Color("8")),
	}
}
