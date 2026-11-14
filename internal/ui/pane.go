package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/qintmb/qterm-updater/internal/theme"
)

// pane draws a rounded box of exactly w x h cells with a title on its first
// line. The body is clipped to the box before the border goes on, so long
// content never wraps or pushes the bottom border out of view.
func pane(st theme.Styles, title, body string, w, h int, focused bool) string {
	box, ts := st.Pane, st.PaneTitle
	if focused {
		box, ts = st.PaneFocused, st.PaneTitleFocused
	}
	iw, ih := paneInner(w, h)
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	if len(lines) > ih {
		keep := max(ih-1, 0)
		lines = append(lines[:keep], st.Muted.Render(fmt.Sprintf("… %d more", len(lines)-keep)))
	}
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, iw, "…")
	}
	inner := ts.Render(ansi.Truncate(title, iw, "…")) + "\n" + strings.Join(lines, "\n")
	return box.Width(w).Height(h).MaxWidth(w).MaxHeight(h).Render(inner)
}

// paneInner returns the content area of a pane of size w x h, excluding the
// border, horizontal padding and title line.
func paneInner(w, h int) (int, int) {
	return max(w-4, 0), max(h-3, 0)
}

// box frames a box drawn over the screen, such as a picker or a prompt, in a
// rounded border of the given color.
func box(c lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(c).Padding(0, 1)
}
