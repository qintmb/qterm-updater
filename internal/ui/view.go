package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// View renders the screen: the panes under the status bar, with any open
// overlay drawn over both.
func (a App) View() string {
	if a.quitting {
		return ""
	}
	if a.width == 0 || a.height == 0 {
		return "loading…"
	}
	body := a.body()
	screen := lipgloss.JoinVertical(lipgloss.Left, body, a.statusBar())
	screen = clipLines(screen, a.height)

	switch {
	case a.themes != nil:
		screen = a.over(screen, a.themes.view(a.st, a.width))
	case a.help:
		screen = a.over(screen, a.helpView(a.width, a.height))
	case a.mode == modeCommand && len(a.cmd.items) > 0:
		screen = a.over(screen, a.cmd.popup(a.st, a.width))
	}
	return screen
}

// body is the area above the status bar.
func (a App) body() string {
	h := a.bodyHeight()
	sideW := min(max(a.width/6, 20), 28)
	listW := max((a.width-sideW)*45/100, 28)
	detailW := a.width - sideW - listW
	if !a.hasDetails() || detailW < 24 {
		listW = a.width - sideW
		detailW = 0
	}
	parts := []string{pane(a.st, "qterm-updater", a.sidebarView(), sideW, h, a.focus == colSidebar)}
	switch a.entryView() {
	case viewHome:
		parts = append(parts, pane(a.st, "Home", a.homeView(sideW+listW+detailW), a.width-sideW, h, a.focus == colList))
	default:
		parts = append(parts, pane(a.st, entries[a.sel].label, a.listView(listW), listW, h, a.focus == colList))
		if detailW > 0 {
			parts = append(parts, pane(a.st, "Details", a.detailsView(detailW), detailW, h, a.focus == colDetails))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// bodyHeight is the space above the status bar.
func (a App) bodyHeight() int { return max(a.height-1, 3) }

// clipLines keeps at most the first n lines of s.
func clipLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	return strings.Join(lines[:min(len(lines), max(n, 0))], "\n")
}

// over draws pop at the bottom of screen, centred.
func (a App) over(screen, pop string) string {
	h := lipgloss.Height(screen)
	y := max(h-1-lipgloss.Height(pop), 0)
	lines := strings.Split(screen, "\n")
	box := strings.Split(pop, "\n")
	for i, l := range box {
		row := i + y
		if row >= len(lines) {
			break
		}
		lines[row] = strings.Repeat(" ", max((a.width-ansi.StringWidth(l))/2, 0)) + l
	}
	return strings.Join(lines, "\n")
}
