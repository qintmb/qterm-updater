package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// statusLeft is the mode's badge, the open entry and what is selected or
// marked.
func (a App) statusLeft() string {
	st := a.st
	badge := "NORMAL"
	switch {
	case a.mode == modeCommand:
		badge = "COMMAND"
	case a.mode == modeFilter:
		badge = "FILTER"
	case a.running:
		badge = "RUNNING"
	}
	left := st.Mode(badge).Render(badge) + " " + entries[a.sel].label
	if name, ok := a.selectedName(); ok {
		left += " › " + name
	}
	if n := a.marks(); n > 0 {
		left += fmt.Sprintf(" · %d marked", n)
	}
	return left
}

// statusRight is the flash and the running job.
func (a App) statusRight() string {
	st := a.st
	var right []string
	if a.flash != "" {
		right = append(right, a.flash)
	}
	if a.running {
		right = append(right, st.Info.Render(spinner(a.frame)+" "+a.runRow.Pkg+" ("+a.runRow.Manager+")"))
	}
	return strings.Join(right, "  ")
}

// actionHints are the keys that do something here.
func (a App) actionHints() string {
	switch {
	case a.entryView() == viewHistory:
		return "j/k move · 1-8 jump · r refresh"
	case a.focus == colDetails:
		return "u upgrade · space mark · U marked · A all"
	case a.focus == colSidebar:
		return "j/k choose · l list · / filter · t theme · : command · ? help · q quit"
	}
	return "enter actions · u upgrade · space mark · / filter · t theme · : command · ? help · q quit"
}

// statusBar is one line across the screen: left, hints, right.
func (a App) statusBar() string {
	st := a.st
	left := a.statusLeft()
	r := a.statusRight()
	room := a.width - 1
	hints := st.Muted.Render(a.actionHints())
	if r != "" {
		hints = r + "  " + hints
	}
	if ansi.StringWidth(left)+1+ansi.StringWidth(stripANSI(hints)) > room {
		left = ansi.Truncate(left, max(room-1-ansi.StringWidth(stripANSI(hints)), 0), "…")
	}
	gap := max(room-ansi.StringWidth(stripANSI(left))-ansi.StringWidth(stripANSI(hints)), 1)
	line := ansi.Truncate(left+strings.Repeat(" ", gap)+hints+" ", a.width, "…")
	return st.StatusBar.Width(a.width).Render(line)
}

// stripANSI drops escape sequences before a width is measured. lipgloss has
// no exported width for a string it did not render.
func stripANSI(s string) string { return ansi.Strip(s) }

// spinner is the frame shown while a job runs.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func spinner(n int) string { return spinnerFrames[n%len(spinnerFrames)] }
