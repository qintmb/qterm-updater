package theme

import "github.com/charmbracelet/lipgloss"

// Styles groups the colors used across the TUI.
type Styles struct {
	Title      lipgloss.Style
	Muted      lipgloss.Style
	Accent     lipgloss.Style
	OK         lipgloss.Style
	Warn       lipgloss.Style
	Err        lipgloss.Style
	Sidebar    lipgloss.Style
	SidebarSel lipgloss.Style
	Section    lipgloss.Style
	Header     lipgloss.Style
	TableHeader lipgloss.Style
	TableRow    lipgloss.Style
	TableRowDim lipgloss.Style
	TableCur    lipgloss.Style
	Mark       lipgloss.Style
	Footer     lipgloss.Style
	Tile       lipgloss.Style
	TileVal    lipgloss.Style
	Border     lipgloss.Style
}

// Default returns the default dark-terminal palette.
func Default() Styles {
	base := lipgloss.NewStyle()
	return Styles{
		Title:       base.Bold(true).Foreground(lipgloss.Color("6")),
		Muted:       base.Foreground(lipgloss.Color("8")),
		Accent:      base.Foreground(lipgloss.Color("5")),
		OK:          base.Foreground(lipgloss.Color("2")),
		Warn:        base.Foreground(lipgloss.Color("3")),
		Err:         base.Foreground(lipgloss.Color("1")),
		Sidebar:     base.Padding(0, 2).Foreground(lipgloss.Color("7")),
		SidebarSel:  base.Padding(0, 2).Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("6")),
		Section:     base.Padding(0, 2).Bold(true).Foreground(lipgloss.Color("8")),
		Header:      base.Bold(true).Foreground(lipgloss.Color("7")),
		TableHeader: base.Bold(true).Foreground(lipgloss.Color("8")),
		TableRow:    base.Padding(0, 1).Foreground(lipgloss.Color("7")),
		TableRowDim: base.Padding(0, 1).Foreground(lipgloss.Color("8")),
		TableCur:    base.Padding(0, 1).Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("4")),
		Mark:        base.Foreground(lipgloss.Color("3")),
		Footer:      base.Padding(0, 1).Foreground(lipgloss.Color("8")),
		Tile:        base.Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("8")),
		TileVal:     base.Bold(true).Foreground(lipgloss.Color("6")),
		Border:      base.Foreground(lipgloss.Color("8")),
	}
}
