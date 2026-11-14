package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// filt is the "/" prompt's state.
type filt struct {
	value string
	items []candidate
	sel   int
}

// filterKey handles a key in filter mode: the line edits, and the list
// narrows as it is typed. esc clears and closes, enter keeps.
func (a App) filterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.mode = modeNormal
		a.filter = ""
		a.applyScope()
		return a, nil
	case "enter":
		a.mode = modeNormal
		a.filter = a.filt.value
		a.applyScope()
		return a, nil
	case "ctrl+c":
		a.quitting = true
		return a, tea.Quit
	case "backspace":
		a.filt.backspace()
	default:
		if len(msg.Runes) == 1 {
			a.filt.insert(msg.Runes[0])
		}
	}
	a.filter = a.filt.value
	a.applyScope()
	return a, nil
}

// set puts s on the line.
func (f *filt) set(s string) { f.value = s }

// insert types r.
func (f *filt) insert(r rune) { f.value += string(r) }

// backspace deletes the last rune.
func (f *filt) backspace() {
	rs := []rune(f.value)
	if len(rs) > 0 {
		f.value = string(rs[:len(rs)-1])
	}
}
