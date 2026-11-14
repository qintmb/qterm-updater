package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/qintmb/qterm-updater/internal/state"
	"github.com/qintmb/qterm-updater/internal/theme"
)

// themesFooter says how to use the picker.
const themesFooter = "j/k preview · enter keep · esc cancel"

// themeBox is the theme picker. The theme under the cursor is in use while
// it is open: enter keeps it, esc puts back the one from before.
type themeBox struct {
	cursor int    // index into theme.Themes
	was    string // the theme in use when the picker opened
}

// openThemes opens the picker on the theme in use.
func (a *App) openThemes() {
	i, _ := theme.Index(a.themeName) // UseTheme only keeps names it knows
	a.themes = &themeBox{cursor: i, was: a.themeName}
}

// key handles a key press while the picker is open. done is true when it
// should close, and keep then says whether the theme under the cursor stays.
func (b *themeBox) key(k string) (done, keep bool) {
	n := len(theme.Themes)
	switch k {
	case "esc", "q":
		return true, false
	case "enter":
		return true, true
	case "j", "down":
		b.cursor = (b.cursor + 1) % n
	case "k", "up":
		b.cursor = (b.cursor + n - 1) % n
	case "G", "end":
		b.cursor = n - 1
	case "g", "home":
		b.cursor = 0
	}
	return false, false
}

// view renders the picker, at most maxW columns wide.
func (b *themeBox) view(st theme.Styles, maxW int) string {
	inner := min(ansi.StringWidth(themesFooter), max(maxW-4, 10))
	lines := []string{st.Title.Render("Theme"), ""}
	for i, t := range theme.Themes {
		label := t.Title
		if t.Name == b.was {
			label += " ✓"
		}
		mark := "  "
		if i == b.cursor {
			mark, label = st.Accent.Render("› "), st.Accent.Bold(true).Render(label)
		}
		lines = append(lines, ansi.Truncate(mark+label, inner, "…"))
	}
	lines = append(lines, "", st.Muted.Render(ansi.Truncate(themesFooter, inner, "…")))
	return box(st.P.Mauve).Render(strings.Join(lines, "\n"))
}

// themesKey handles a key while the picker is open: a move switches to the
// theme under the cursor, enter keeps it, esc puts the old one back.
func (a App) themesKey(k string) (tea.Model, tea.Cmd) {
	b := a.themes
	before := b.cursor
	done, keep := b.key(k)
	switch {
	case done && keep:
		a.themes = nil
		return a.keepTheme(theme.Themes[b.cursor].Name), nil
	case done:
		a.themes = nil
		return a.useTheme(b.was), nil
	case b.cursor != before:
		return a.useTheme(theme.Themes[b.cursor].Name), nil
	}
	return a, nil
}

// useTheme switches to the theme called name. It is not saved: keepTheme
// does that.
func (a App) useTheme(name string) App {
	t := theme.Lookup(name)
	st := theme.New(t.Palette)
	st.Icons = a.st.Icons
	a.st = st
	a.themeName = t.Name
	return a
}

// keepTheme switches to name and saves it for next run.
func (a App) keepTheme(name string) App {
	a = a.useTheme(name)
	if err := theme.SaveTheme(state.Dir(), a.themeName); err != nil {
		a.flash = a.st.Err.Render("✗ theme: " + a.themeName + "; couldn't save it: " + err.Error())
		return a
	}
	a.flash = a.st.OK.Render("theme: " + theme.Lookup(a.themeName).Title)
	return a
}
