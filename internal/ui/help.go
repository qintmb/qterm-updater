package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// helpGroup is one section of the ? overlay: a heading and its keys.
type helpGroup struct {
	title string
	keys  [][2]string // key, what it does
}

// helpKeyW is the width of the key column.
const helpKeyW = 11

// helpGroups are the keys that work where the user is now: the focused
// column's first, then everywhere's.
func (a App) helpGroups() []helpGroup {
	var here helpGroup
	switch a.focus {
	case colSidebar:
		here = helpGroup{"Sidebar", [][2]string{
			{"j k", "choose"}, {"gg G", "first, last"}, {"[ ]", "previous, next"},
			{"l enter", "go to the list"}, {"1-8", "jump to an entry"},
		}}
	case colDetails:
		here = helpGroup{"Details", [][2]string{
			{"j k", "scroll"}, {"h", "back to the list"}, {"u", "upgrade"},
			{"space", "mark"}, {"U", "upgrade marked"}, {"A", "upgrade all in view"},
		}}
	default:
		here = helpGroup{"List", [][2]string{
			{"j k", "move"}, {"gg G", "top, bottom"}, {"ctrl+d u", "half a page"},
			{"l tab", "details, then j/k scroll"}, {"/", "filter by name"},
			{"space", "mark for several"}, {"U", "upgrade marked"}, {"A", "upgrade all in view"},
		}}
		if a.entryView() == viewHistory {
			here = helpGroup{"List", [][2]string{
				{"j k", "move"}, {"gg G", "top, bottom"}, {"ctrl+d u", "half a page"},
				{"h", "back to the sidebar"},
			}}
		}
	}
	return []helpGroup{here, {"Everywhere", [][2]string{
		{"h l", "sidebar, list, details"}, {"1-8", "jump to a sidebar entry"},
		{"[ ]", "previous or next entry"}, {":", "run a command"}, {"t", "change the theme"},
		{"r", "refresh"}, {"/", "filter, or clear one"}, {"?", "these keys"},
		{"q", "quit"},
	}}}
}

// helpWidth is how wide the overlay draws.
const helpWidth = 58

// helpView renders the ? overlay in at most maxW x maxH cells.
func (a App) helpView(maxW, maxH int) string {
	st := a.st
	inner := min(helpWidth, max(maxW-4, 24))
	var lines []string
	for i, g := range a.helpGroups() {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, st.Accent.Bold(true).Render(g.title))
		for _, k := range g.keys {
			key := st.Title.Render(k[0]) + strings.Repeat(" ", max(helpKeyW-ansi.StringWidth(k[0]), 1))
			lines = append(lines, ansi.Truncate(key+k[1], inner, "…"))
		}
	}
	lines = append(lines, "", st.Muted.Render("any key closes this"))
	body := strings.Join(lines, "\n")
	return box(st.P.Mauve).Render(body)
}
