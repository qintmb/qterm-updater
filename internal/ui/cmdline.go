package ui

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/qintmb/qterm-updater/internal/state"
	"github.com/qintmb/qterm-updater/internal/theme"
)

// candidate is one completion: what it inserts, and what it shows.
type candidate struct {
	value, label string
}

// cmdline is the ":" prompt: a one-line editor that remembers the session's
// earlier commands and completes command names.
type cmdline struct {
	value  string
	cursor int         // rune index into value
	past   []string    // earlier commands, oldest first
	back   int         // how far up into past the line is; 0 is the draft
	draft  string      // what was typed before browsing past commands
	items  []candidate // the completion popup, empty when hidden
	sel    int         // the selected item, -1 for none
}

func (c *cmdline) runes() []rune { return []rune(c.value) }

// set puts s on the line with the cursor at the end.
func (c *cmdline) set(s string) {
	c.value = s
	c.cursor = len([]rune(s))
	c.back, c.draft, c.items, c.sel = 0, "", nil, -1
}

// reset blanks the line for a new one.
func (c *cmdline) reset() { c.set("") }

// remember adds a run command to the past, skipping a repeat of the last.
func (c *cmdline) remember(line string) {
	if n := len(c.past); n == 0 || c.past[n-1] != line {
		c.past = append(c.past, line)
	}
}

// older shows the command before the one on the line.
func (c *cmdline) older() {
	if len(c.past) == 0 || c.back == len(c.past) {
		return
	}
	if c.back == 0 {
		c.draft = c.value
	}
	c.back++
	c.set(c.past[len(c.past)-c.back])
}

// newer shows the command after the one on the line, and then the draft.
func (c *cmdline) newer() {
	if c.back == 0 {
		return
	}
	c.back--
	if c.back == 0 {
		c.set(c.draft)
		return
	}
	c.set(c.past[len(c.past)-c.back])
}

// insert types r at the cursor.
func (c *cmdline) insert(r rune) {
	rs := c.runes()
	rs = append(rs[:c.cursor], append([]rune{r}, rs[c.cursor:]...)...)
	c.value = string(rs)
	c.cursor++
	c.sel = -1
}

// backspace deletes the character before the cursor.
func (c *cmdline) backspace() {
	rs := c.runes()
	if c.cursor == 0 {
		return
	}
	rs = append(rs[:c.cursor-1], rs[c.cursor:]...)
	c.value, c.cursor = string(rs), c.cursor-1
	c.sel = -1
}

// left and right move the cursor inside value.
func (c *cmdline) left() {
	if c.cursor > 0 {
		c.cursor--
	}
}

func (c *cmdline) right() {
	if c.cursor < len(c.runes()) {
		c.cursor++
	}
}

// complete replaces the last word with the first completion that fits.
// Each call cycles to the next.
func (c *cmdline) complete(cands []candidate) {
	c.items = cands
	if len(cands) == 0 {
		return
	}
	c.sel = (c.sel + 1) % len(cands)
	line := c.value
	i := strings.LastIndexByte(line, ' ')
	word, base := line[i+1:], line[:i+1]
	if base == "" || strings.HasPrefix(cands[c.sel].value, word) {
		c.value = base + cands[c.sel].value
		c.cursor = len(c.runes())
	}
}

// popup renders the completions over at most w columns; empty when hidden.
func (c *cmdline) popup(st theme.Styles, w int) string {
	if len(c.items) == 0 {
		return ""
	}
	inner := min(60, max(w-4, 10))
	var lines []string
	start := max(c.sel-4, 0)
	for i := start; i < min(start+9, len(c.items)); i++ {
		it := c.items[i]
		val := ansi.Truncate(it.value, inner-2-12, "…")
		lab := ansi.Truncate(it.label, 12, "…")
		line := padRight(val, inner-2-13) + " " + st.Muted.Render(lab)
		if i == c.sel {
			line = st.Accent.Render("› ") + st.Text.Bold(true).Render(line[4:])
		} else {
			line = "  " + line
		}
		lines = append(lines, ansi.Truncate(line, inner, ""))
	}
	return box(st.P.Mauve).Render(strings.Join(lines, "\n"))
}

// view renders the prompt in w columns, with the cursor on screen.
func (c *cmdline) view(w int) string {
	rs := c.runes()
	right := min(max(c.cursor, 0), len(rs))
	room := max(w-2, 1) // the prompt and the cell under the cursor
	left := ansi.Truncate(string(rs[:right]), max(room-1, 0), "…")
	vis := displayWidth(left)
	cell := " "
	if right < len(rs) {
		cell = string(rs[right])
	}
	tail := ""
	if right+1 < len(rs) {
		roomTail := max(room-vis-displayWidth(cell), 0)
		tail = ansi.Truncate(string(rs[right+1:]), roomTail, "")
	}
	return ":" + left + lipgloss.NewStyle().Reverse(true).Render(cell) + tail
}

// displayWidth is how many cells s takes.
func displayWidth(s string) int { return ansi.StringWidth(s) }

// padRight pads s with spaces to w cells, cutting it when longer.
func padRight(s string, w int) string {
	if n := displayWidth(s); n > w {
		return ansi.Truncate(s, w, "…")
	} else if n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

// commandNames are what the ":" prompt completes on the first word.
var commandNames = []string{
	"update", "upgrade", "upgrade-all", "refresh", "theme", "theme-next", "icons",
	"history", "clear-history", "help", "quit",
}

// suggestions returns completions for line.
func suggestions(line string) []candidate {
	w := strings.ToLower(strings.TrimSpace(line))
	if w == "theme" || strings.HasPrefix(w, "theme ") {
		var cs []candidate
		for _, t := range theme.Themes {
			cs = append(cs, candidate{t.Name, t.Title})
		}
		return cs
	}
	if strings.Contains(w, " ") {
		return nil
	}
	cs := make([]candidate, 0, len(commandNames))
	for _, n := range commandNames {
		if strings.HasPrefix(n, w) {
			cs = append(cs, candidate{n, ""})
		}
	}
	return cs
}

// errUnclosedQuote is a line whose quote or backslash never closed.
var errUnclosedQuote = errors.New("unclosed quote")

// splitArgs splits a line into arguments the way a shell would, without
// running one: spaces separate, quotes group, and a backslash escapes the
// next character (outside single quotes).
func splitArgs(line string) ([]string, error) {
	var args []string
	var cur strings.Builder
	inArg, quote, escaped := false, rune(0), false
	for _, r := range line {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
		case r == '\\' && quote != '\'':
			escaped, inArg = true, true
		case quote != 0 && r == quote:
			quote = 0
		case quote != 0:
			cur.WriteRune(r)
		case r == '"' || r == '\'':
			quote, inArg = r, true
		case r == ' ' || r == '\t':
			if inArg {
				args = append(args, cur.String())
				cur.Reset()
				inArg = false
			}
		default:
			cur.WriteRune(r)
			inArg = true
		}
	}
	if quote != 0 || escaped {
		return nil, errUnclosedQuote
	}
	if inArg {
		args = append(args, cur.String())
	}
	return args, nil
}

// cmdKey handles a key in command mode: enter runs the line, esc cancels,
// tab completes, up and down browse the past, and everything else edits.
func (a App) cmdKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.mode = modeNormal
		a.cmd.set("")
		return a, nil
	case "ctrl+c":
		a.quitting = true
		return a, tea.Quit
	case "tab", "shift+tab":
		if len(a.cmd.items) == 0 {
			a.cmd.items = suggestions(a.cmd.value)
			a.cmd.sel = -1
		}
		if n := len(a.cmd.items); n > 0 {
			a.cmd.sel = (a.cmd.sel + 1) % n
			a.cmd.set(a.cmd.items[a.cmd.sel].value)
		}
		return a, nil
	case "up":
		a.cmd.older()
		return a, nil
	case "down":
		a.cmd.newer()
		return a, nil
	case "enter":
		a.mode = modeNormal
		line := strings.TrimSpace(a.cmd.value)
		a.cmd.items = nil
		if line == "" {
			return a, nil
		}
		a.cmd.remember(line)
		return a.execute(line)
	case "backspace":
		a.cmd.backspace()
	case "left":
		a.cmd.left()
	case "right":
		a.cmd.right()
	default:
		if len(msg.Runes) == 1 {
			a.cmd.insert(msg.Runes[0])
		}
	}
	if len(a.cmd.items) > 0 || strings.Contains(a.cmd.value, " ") {
		a.cmd.items = suggestions(a.cmd.value)
	}
	return a, nil
}

// execute runs a line typed at the prompt.
func (a App) execute(line string) (App, tea.Cmd) {
	args, err := splitArgs(line)
	if err != nil {
		a.flash = a.st.Err.Render("✗ " + err.Error())
		return a, nil
	}
	if len(args) == 0 {
		return a, nil
	}
	switch args[0] {
	case "q", "quit":
		a.quitting = true
		return a, tea.Quit
	case "update", "upgrade", "refresh":
		m, c := a.refresh()
		return m.(App), c
	case "upgrade-all", "update-all":
		app, c := a.upgradeAll()
		return app.(App), c
	case "history":
		app, c := a.selectEntry(entryIndex(viewHistory))
		return app, c
	case "help":
		a.help = true
	case "theme":
		if len(args) == 0 || len(args) == 1 {
			a.openThemes()
			return a, nil
		}
		if _, ok := theme.Index(args[1]); !ok {
			a.flash = a.st.Err.Render("✗ no theme called " + args[1])
			return a, nil
		}
		a = a.keepTheme(args[1])
	case "icons":
		on := len(args) > 1 && args[1] == "on"
		if len(args) > 1 && args[1] != "on" && args[1] != "off" {
			a.flash = a.st.Err.Render("✗ icons on|off")
			return a, nil
		}
		a.iconsOn = on
		a.st.Icons = iconSet(on)
		if err := theme.SaveIcons(state.Dir(), on); err != nil {
			a.flash = a.st.Err.Render("✗ couldn't save icons: " + err.Error())
			return a, nil
		}
		a.flash = a.st.OK.Render("icons " + map[bool]string{true: "on", false: "off"}[on])
	default:
		a.flash = a.st.Err.Render("✗ no command " + args[0] + " · :help lists them")
	}
	return a, nil
}

// entryIndex is the sidebar index of a view.
func entryIndex(view int) int {
	for i, e := range entries {
		if e.view == view {
			return i
		}
	}
	return 0
}

// aloneW adapts a two-value refresh to the one-value command handler.
func aloneW(m tea.Model, c tea.Cmd) (App, tea.Cmd) { return m.(App), c }
