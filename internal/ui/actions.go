package ui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/qintmb/qterm-updater/internal/scan"
	"github.com/qintmb/qterm-updater/internal/state"
	"github.com/qintmb/qterm-updater/internal/theme"
)

// upgradeCmd is the command that upgrades r.
func upgradeCmd(r scan.Row) []string {
	switch r.Manager {
	case "brew":
		return []string{"brew", "upgrade", r.Pkg}
	case "npm":
		return []string{"npm", "install", "-g", r.Pkg + "@latest"}
	case "pnpm":
		return []string{"pnpm", "update", "-g", "--latest", r.Pkg}
	case "uv":
		return []string{"uv", "tool", "upgrade", r.Pkg}
	}
	return nil
}

// doUpgrade runs one upgrade outside the TUI's event loop.
func doUpgrade(r scan.Row) tea.Cmd {
	return func() tea.Msg {
		args := upgradeCmd(r)
		if args == nil {
			return upgradeMsg{row: r, err: fmt.Errorf("unknown manager %q", r.Manager)}
		}
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		state.AppendHistory(state.HistoryEntry{
			StartedAt: time.Now(),
			EndedAt:   time.Now(),
			Pkg:       r.Pkg,
			Manager:   r.Manager,
			From:      r.Cur,
			To:        r.New,
			OK:        err == nil,
			Output:    string(out),
		})
		return upgradeMsg{row: r, err: err}
	}
}

// upgradeAll upgrades every row in the current view, one at a time.
func (a App) upgradeAll() (tea.Model, tea.Cmd) {
	if len(a.shown) == 0 {
		return a, nil
	}
	a.runRow = a.shown[0]
	a.running = true
	return a, tea.Batch(doUpgrade(a.runRow), tick())
}

// upgradeMarked upgrades a marked row, if any.
func (a App) upgradeMarked() (tea.Model, tea.Cmd) {
	for _, r := range a.rows {
		if a.marked[r.Pkg] {
			a.runRow = r
			a.running = true
			return a, tea.Batch(doUpgrade(r), tick())
		}
	}
	return a, nil
}

// tick re-renders the spinner.
func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

// kv renders aligned "label  value" pairs, skipping empty values.
func kv(st theme.Styles, pairs ...string) string {
	w := 10
	for i := 0; i < len(pairs); i += 2 {
		w = max(w, len(pairs[i])+1)
	}
	var b strings.Builder
	for i := 0; i+1 < len(pairs); i += 2 {
		if pairs[i+1] == "" {
			continue
		}
		b.WriteString(st.Muted.Render(padRight(pairs[i]+":", w)))
		b.WriteString(pairs[i+1])
		b.WriteString("\n")
	}
	return b.String()
}

// wrapWords wraps s at w cells, indenting after the first line.
func wrapWords(s string, w int) string {
	var lines []string
	var cur strings.Builder
	for _, word := range strings.Fields(s) {
		if cur.Len() > 0 && cur.Len()+1+len(word) > w {
			lines = append(lines, cur.String())
			cur.Reset()
		}
		if cur.Len() > 0 {
			cur.WriteString(" ")
		}
		cur.WriteString(word)
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return strings.Join(lines, "\n  ")
}
