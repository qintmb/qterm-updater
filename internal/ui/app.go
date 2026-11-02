package ui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/qintmb/qterm-updater/internal/scan"
	"github.com/qintmb/qterm-updater/internal/state"
	"github.com/qintmb/qterm-updater/internal/theme"
)

// section groups sidebar entries (brewery-style).
type section int

const (
	secHome section = iota
	secPackages
	secManagers
	secTools
)

var sectionNames = map[section]string{
	secPackages: "PACKAGES",
	secManagers: "MANAGERS",
	secTools:    "TOOLS",
}

type entry struct {
	label string
	sec   section
	view  int
}

var entries = []entry{
	{"Home", secHome, viewHome},
	{"All", secPackages, viewAll},
	{"Marked", secPackages, viewMarked},
	{"brew", secManagers, viewBrew},
	{"npm", secManagers, viewNpm},
	{"pnpm", secManagers, viewPnpm},
	{"uv", secManagers, viewUv},
	{"History", secTools, viewHistory},
}

const (
	viewHome = iota
	viewAll
	viewMarked
	viewBrew
	viewNpm
	viewPnpm
	viewUv
	viewHistory
)

type focusArea int

const (
	focusSidebar focusArea = iota
	focusTable
	focusFilter
)

type upgradeMsg struct {
	pkg string
	err error
}

// App is the root Bubble Tea model.
type App struct {
	st        theme.Styles
	sel       int                // sidebar selection
	tblCur    int                // table cursor
	rows      []scan.Row         // all outdated packages
	shown     []scan.Row         // filtered/scoped rows
	filter    string             // "/" filter
	marked    map[string]bool    // package key -> marked
	focus     focusArea
	width     int
	height    int
	upgrading bool
	upgPkg    string
	upgLog    []string
	quitting  bool
}

// NewApp creates the initial model.
func NewApp(rows []scan.Row) App {
	st := theme.Default()
	a := App{
		st:     st,
		rows:   rows,
		marked: make(map[string]bool),
		focus:  focusTable,
	}
	a.applyScope()
	return a
}

func (a *App) applyScope() {
	var out []scan.Row
	curView := entries[a.sel].view

	for _, r := range a.rows {
		// Filter by view scope
		switch curView {
		case viewAll, viewHome:
			// show all
		case viewMarked:
			if !a.marked[r.Pkg] {
				continue
			}
		case viewBrew:
			if r.Manager != "brew" {
				continue
			}
		case viewNpm:
			if r.Manager != "npm" {
				continue
			}
		case viewPnpm:
			if r.Manager != "pnpm" {
				continue
			}
		case viewUv:
			if r.Manager != "uv" {
				continue
			}
		}

		// Filter by "/" search
		if a.filter != "" {
			q := strings.ToLower(a.filter)
			if !strings.Contains(strings.ToLower(r.Pkg), q) &&
				!strings.Contains(strings.ToLower(r.Desc), q) {
				continue
			}
		}

		out = append(out, r)
	}

	a.shown = out
	if a.tblCur >= len(a.shown) {
		a.tblCur = max(0, len(a.shown)-1)
	}
}

func (a App) Init() tea.Cmd { return nil }

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return a.handleKey(msg)
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		return a, nil
	case upgradeMsg:
		a.upgrading = false
		if msg.err == nil {
			a.upgLog = append(a.upgLog, fmt.Sprintf("✓ %s upgraded", msg.pkg))
			// Remove from rows
			var remaining []scan.Row
			for _, r := range a.rows {
				if r.Pkg != msg.pkg {
					remaining = append(remaining, r)
				}
			}
			a.rows = remaining
			delete(a.marked, msg.pkg)
			a.applyScope()
		} else {
			a.upgLog = append(a.upgLog, fmt.Sprintf("✗ %s failed: %v", msg.pkg, msg.err))
		}
		return a, nil
	}
	return a, nil
}

func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Filter typing mode
	if a.focus == focusFilter {
		switch msg.String() {
		case "esc", "enter":
			a.focus = focusTable
		case "backspace":
			if len(a.filter) > 0 {
				a.filter = a.filter[:len(a.filter)-1]
				a.applyScope()
			}
		default:
			if len(msg.String()) == 1 {
				a.filter += msg.String()
				a.applyScope()
			}
		}
		return a, nil
	}

	switch msg.String() {
	case "ctrl+c", "q":
		a.quitting = true
		return a, tea.Quit

	case "tab":
		if a.focus == focusSidebar {
			a.focus = focusTable
		} else {
			a.focus = focusSidebar
		}

	case "/":
		a.focus = focusFilter
		a.filter = ""
		a.applyScope()

	case "esc":
		if a.filter != "" {
			a.filter = ""
			a.applyScope()
		}

	case "1", "2", "3", "4", "5", "6", "7", "8":
		idx := int(msg.String()[0] - '1')
		if idx >= 0 && idx < len(entries) {
			a.sel = idx
			a.tblCur = 0
			a.applyScope()
			a.focus = focusTable
		}

	case "j", "down":
		if a.focus == focusSidebar {
			if a.sel < len(entries)-1 {
				a.sel++
				a.tblCur = 0
				a.applyScope()
			}
		} else {
			if a.tblCur < len(a.shown)-1 {
				a.tblCur++
			}
		}

	case "k", "up":
		if a.focus == focusSidebar {
			if a.sel > 0 {
				a.sel--
				a.tblCur = 0
				a.applyScope()
			}
		} else {
			if a.tblCur > 0 {
				a.tblCur--
			}
		}

	case " ":
		if a.focus == focusTable && len(a.shown) > 0 {
			pkg := a.shown[a.tblCur].Pkg
			a.marked[pkg] = !a.marked[pkg]
		}

	case "u":
		// Upgrade selected package
		if a.focus == focusTable && len(a.shown) > 0 && !a.upgrading {
			row := a.shown[a.tblCur]
			a.upgrading = true
			a.upgPkg = row.Pkg
			return a, doUpgrade(row)
		}

	case "U":
		// Upgrade all marked packages
		if len(a.marked) > 0 && !a.upgrading {
			for _, r := range a.rows {
				if a.marked[r.Pkg] {
					a.upgrading = true
					a.upgPkg = r.Pkg
					return a, doUpgrade(r)
				}
			}
		}
	}

	return a, nil
}

func doUpgrade(r scan.Row) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch r.Manager {
		case "brew":
			cmd = exec.Command("brew", "upgrade", r.Pkg)
		case "npm":
			cmd = exec.Command("npm", "install", "-g", r.Pkg+"@latest")
		case "pnpm":
			cmd = exec.Command("pnpm", "update", "-g", "--latest", r.Pkg)
		case "uv":
			cmd = exec.Command("uv", "tool", "upgrade", r.Pkg)
		default:
			return upgradeMsg{pkg: r.Pkg, err: fmt.Errorf("unknown manager")}
		}
		out, err := cmd.CombinedOutput()
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
		return upgradeMsg{pkg: r.Pkg, err: err}
	}
}

func (a App) View() string {
	if a.quitting {
		return ""
	}

	sidebarW := 22
	contentW := a.width - sidebarW - 3
	if contentW < 40 {
		contentW = 60
	}

	sidebar := a.renderSidebar(sidebarW)
	content := a.renderContent(contentW)

	mainPane := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
	footer := a.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, mainPane, footer)
}

func (a App) renderSidebar(width int) string {
	var b strings.Builder
	style := lipgloss.NewStyle().Width(width)

	b.WriteString(a.st.Title.Render("⟳ qterm") + "\n\n")

	for i, e := range entries {
		// Section headers
		if e.sec != secHome && (i == 0 || entries[i-1].sec != e.sec) {
			b.WriteString("\n" + a.st.Section.Render(sectionNames[e.sec]) + "\n")
		}

		// Count badges
		badge := ""
		switch e.view {
		case viewAll:
			badge = fmt.Sprintf(" %d", len(a.rows))
		case viewMarked:
			markedCount := 0
			for _, m := range a.marked {
				if m {
					markedCount++
				}
			}
			if markedCount > 0 {
				badge = fmt.Sprintf(" %d", markedCount)
			}
		}

		label := fmt.Sprintf("%d %s%s", i+1, e.label, badge)
		if i == a.sel {
			b.WriteString(a.st.SidebarSel.Render(label) + "\n")
		} else {
			b.WriteString(a.st.Sidebar.Render(label) + "\n")
		}
	}

	return style.Render(b.String())
}

func (a App) renderContent(width int) string {
	switch entries[a.sel].view {
	case viewHome:
		return a.renderHome(width)
	case viewHistory:
		return a.renderHistory(width)
	default:
		return a.renderTableAndDetails(width)
	}
}

func (a App) renderHome(width int) string {
	var b strings.Builder

	// Top tiles (brewery style)
	b.WriteString(a.st.Header.Render("HEALTH SUMMARY") + "\n\n")

	byMgr := make(map[string]int)
	for _, r := range a.rows {
		byMgr[r.Manager]++
	}

	var tiles []string
	totalTile := fmt.Sprintf("Total Outdated\n%s", a.st.TileVal.Render(fmt.Sprintf("%d", len(a.rows))))
	tiles = append(tiles, a.st.Tile.Render(totalTile))

	for mgr, count := range byMgr {
		t := fmt.Sprintf("%s\n%s", mgr, a.st.TileVal.Render(fmt.Sprintf("%d", count)))
		tiles = append(tiles, a.st.Tile.Render(t))
	}

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tiles...) + "\n\n")

	// Upgrading banner
	if a.upgrading {
		b.WriteString(a.st.Warn.Render(fmt.Sprintf("⠋ Upgrading %s...", a.upgPkg)) + "\n\n")
	}

	// Recent logs
	if len(a.upgLog) > 0 {
		b.WriteString(a.st.Header.Render("LOGS") + "\n")
		for _, l := range a.upgLog {
			b.WriteString(l + "\n")
		}
	}

	return b.String()
}

func (a App) renderTableAndDetails(width int) string {
	tableW := (width * 3) / 5
	detailsW := width - tableW - 2

	tablePane := a.renderTable(tableW)
	detailsPane := a.renderDetails(detailsW)

	return lipgloss.JoinHorizontal(lipgloss.Top, tablePane, detailsPane)
}

func (a App) renderTable(width int) string {
	var b strings.Builder

	// Filter bar
	if a.focus == focusFilter || a.filter != "" {
		b.WriteString(fmt.Sprintf("/ %s\n", a.filter))
	}

	// Header
	header := fmt.Sprintf("  %-18s %-12s %-12s", "PACKAGE", "CURRENT", "NEW")
	b.WriteString(a.st.TableHeader.Render(header) + "\n")

	if len(a.shown) == 0 {
		b.WriteString(a.st.Muted.Render("\n  No outdated packages\n"))
		return b.String()
	}

	// Rows
	for i, r := range a.shown {
		mark := " "
		if a.marked[r.Pkg] {
			mark = a.st.Mark.Render("•")
		}

		line := fmt.Sprintf("%s %-18s %-12s %-12s", mark, trunc(r.Pkg, 18), trunc(r.Cur, 12), trunc(r.New, 12))

		if i == a.tblCur && a.focus == focusTable {
			b.WriteString(a.st.TableCur.Render(line) + "\n")
		} else {
			b.WriteString(a.st.TableRow.Render(line) + "\n")
		}
	}

	return b.String()
}

func (a App) renderDetails(width int) string {
	if len(a.shown) == 0 || a.tblCur >= len(a.shown) {
		return ""
	}

	r := a.shown[a.tblCur]
	var b strings.Builder
	style := lipgloss.NewStyle().Width(width).Padding(0, 1)

	b.WriteString(a.st.Header.Render("DETAILS") + "\n\n")
	b.WriteString(fmt.Sprintf("Package:  %s\n", a.st.Title.Render(r.Pkg)))
	b.WriteString(fmt.Sprintf("Manager:  %s\n", r.Manager))
	b.WriteString(fmt.Sprintf("Current:  %s\n", r.Cur))
	b.WriteString(fmt.Sprintf("Latest:   %s\n", a.st.OK.Render(r.New)))

	if r.Desc != "" {
		b.WriteString(fmt.Sprintf("\nDescription:\n%s\n", r.Desc))
	}
	if r.Home != "" {
		b.WriteString(fmt.Sprintf("\nHomepage:\n%s\n", a.st.Muted.Render(r.Home)))
	}

	return style.Render(b.String())
}

func (a App) renderHistory(width int) string {
	h := state.LoadHistory()
	if len(h) == 0 {
		return a.st.Muted.Render("No history yet.")
	}

	var b strings.Builder
	b.WriteString(a.st.Header.Render("RECENT UPDATES") + "\n\n")

	for _, e := range h {
		status := a.st.OK.Render("✓")
		if !e.OK {
			status = a.st.Err.Render("✗")
		}
		b.WriteString(fmt.Sprintf("%s %-18s %s → %s (%s) %s\n",
			status, e.Pkg, e.From, e.To, e.Manager,
			a.st.Muted.Render(e.StartedAt.Format("01/02 15:04"))))
	}

	return b.String()
}

func (a App) renderFooter() string {
	keys := []string{
		"[1-8] Jump",
		"[Tab] Switch pane",
		"[j/k] Navigate",
		"[Space] Mark",
		"[u] Upgrade",
		"[U] Upgrade marked",
		"[/] Filter",
		"[q] Quit",
	}
	return "\n" + a.st.Footer.Render(strings.Join(keys, "  "))
}

func trunc(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return s[:maxLen]
	}
	return s[:maxLen-1] + "…"
}
