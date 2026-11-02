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

// ---- sidebar ---------------------------------------------------------------

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

// ---- table columns ---------------------------------------------------------

type col struct {
	title string
	width int
}

var pkgCols = []col{
	{"MARK", 4},
	{"PACKAGE", 24},
	{"CURRENT", 14},
	{"LATEST", 14},
}

var histCols = []col{
	{"STATUS", 6},
	{"PACKAGE", 24},
	{"VERSION", 28},
	{"MANAGER", 10},
	{"WHEN", 14},
}

// ---- focus / mode ----------------------------------------------------------

type focusArea int

const (
	focusSidebar focusArea = iota
	focusTable
	focusDetails
	focusFilter
)

// upgradeMsg is sent when an upgrade finishes.
type upgradeMsg struct {
	row scan.Row
	err error
}

// tickMsg re-renders while a job runs.
type tickMsg struct{}

// App is the root Bubble Tea model.
type App struct {
	st       theme.Styles
	sel      int             // sidebar selection
	tblCur   int             // table cursor
	histCur  int             // history cursor
	hist     []state.HistoryEntry
	rows     []scan.Row      // all outdated packages
	shown    []scan.Row      // scoped + filtered
	filter   string          // "/" filter
	filtering bool
	marked   map[string]bool // package key -> marked
	focus    focusArea
	width    int
	height   int
	running  bool            // an upgrade is in flight
	runRow   scan.Row
	logs     []string
	quitting bool
}

// NewApp creates the initial model.
func NewApp(rows []scan.Row) App {
	st := theme.Default()
	a := App{
		st:     st,
		rows:   rows,
		hist:   state.LoadHistory(),
		marked: make(map[string]bool),
		focus:  focusTable,
	}
	a.applyScope()
	return a
}

func (a App) Init() tea.Cmd { return nil }

func (a *App) applyScope() {
	var out []scan.Row
	curView := entries[a.sel].view
	for _, r := range a.rows {
		switch curView {
		case viewMarked:
			if !a.marked[r.Pkg] {
				continue
			}
		case viewBrew, viewNpm, viewPnpm, viewUv:
			want := entries[a.sel].label
			if r.Manager != want {
				continue
			}
		}
		if a.filter != "" {
			q := strings.ToLower(a.filter)
			if !strings.Contains(strings.ToLower(r.Pkg), q) &&
				!strings.Contains(strings.ToLower(r.Desc), q) &&
				!strings.Contains(strings.ToLower(r.Manager), q) {
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

func (a App) view() int { return entries[a.sel].view }

// ---- update ---------------------------------------------------------------

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		return a, nil

	case tickMsg:
		if a.running {
			return a, tea.Tick(time.Millisecond*100, func(time.Time) tea.Msg { return tickMsg{} })
		}
		return a, nil

	case upgradeMsg:
		a.running = false
		status := "✓"
		style := a.st.OK
		if msg.err != nil {
			status = "✗"
			style = a.st.Err
		}
		a.logs = append(a.logs, fmt.Sprintf("%s %s: %s → %s (%s)",
			style.Render(status), msg.row.Pkg, msg.row.Cur, msg.row.New, msg.row.Manager))
		if msg.err == nil {
			var keep []scan.Row
			for _, r := range a.rows {
				if r.Pkg != msg.row.Pkg {
					keep = append(keep, r)
				}
			}
			a.rows = keep
			delete(a.marked, msg.row.Pkg)
			a.applyScope()
		}
		a.hist = state.LoadHistory()
		return a, nil

	case tea.KeyMsg:
		return a.handleKey(msg)
	}
	return a, nil
}

func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Filter typing mode.
	if a.filtering {
		switch msg.String() {
		case "esc":
			a.filtering = false
			a.filter = ""
		case "enter":
			a.filtering = false
		case "backspace":
			if len(a.filter) > 0 {
				a.filter = a.filter[:len(a.filter)-1]
			}
		default:
			if len(msg.Runes) == 1 {
				a.filter += string(msg.Runes[0])
			}
		}
		a.applyScope()
		return a, nil
	}

	// In a list: up/down move the cursor, left/right change pane.
	switch msg.String() {
	case "ctrl+c", "q":
		a.quitting = true
		return a, tea.Quit

	case "up", "k":
		return a.move(-1)

	case "down", "j":
		return a.move(1)

	case "home", "g":
		if a.focus == focusSidebar {
			a.sel = 0
		} else {
			a.tblCur = 0
			a.histCur = 0
		}
		return a, nil

	case "end", "G":
		if a.focus == focusSidebar {
			a.sel = len(entries) - 1
		} else if a.view() == viewHistory {
			a.histCur = max(0, len(a.hist)-1)
		} else {
			a.tblCur = max(0, len(a.shown)-1)
		}
		return a, nil

	case "pgup", "ctrl+u":
		step := max(1, a.listHeight())
		return a.move(-step)

	case "pgdown", "ctrl+d":
		step := max(1, a.listHeight())
		return a.move(step)

	// left/right move focus between the three panes.
	case "left", "h":
		switch a.focus {
		case focusDetails:
			a.focus = focusTable
		case focusTable:
			a.focus = focusSidebar
		}
		return a, nil

	case "right", "l", "tab":
		switch a.focus {
		case focusSidebar:
			a.focus = focusTable
		case focusTable:
			if a.view() != viewHistory && a.view() != viewHome && len(a.shown) > 0 {
				a.focus = focusDetails
			}
		}
		return a, nil

	case "/":
		a.filtering = true
		a.filter = ""
		a.applyScope()
		return a, nil

	case "esc":
		if a.filter != "" {
			a.filter = ""
			a.applyScope()
		} else {
			a.focus = focusTable
		}
		return a, nil

	case "1", "2", "3", "4", "5", "6", "7", "8":
		idx := int(msg.String()[0] - '1')
		if idx >= 0 && idx < len(entries) {
			a.sel = idx
			a.tblCur = 0
			a.histCur = 0
			a.applyScope()
			if a.focus == focusSidebar {
				a.focus = focusTable
			}
		}
		return a, nil

	case " ":
		if a.focus == focusTable && len(a.shown) > 0 && a.tblCur < len(a.shown) {
			pkg := a.shown[a.tblCur].Pkg
			a.marked[pkg] = !a.marked[pkg]
			a.applyScope()
		}
		return a, nil

	case "enter":
		// Enter on sidebar: jump to that view and move into the table.
		if a.focus == focusSidebar {
			a.focus = focusTable
			return a, nil
		}
		// Enter on a table row: open details (same as right).
		if a.focus == focusTable && a.view() != viewHistory && len(a.shown) > 0 {
			a.focus = focusDetails
		}
		return a, nil

	case "u":
		if !a.running && a.view() != viewHistory && len(a.shown) > 0 && a.tblCur < len(a.shown) {
			r := a.shown[a.tblCur]
			a.running = true
			a.runRow = r
			return a, tea.Batch(doUpgrade(r), tea.Tick(time.Millisecond*100, func(time.Time) tea.Msg { return tickMsg{} }))
		}
		return a, nil

	case "U":
		if !a.running && len(a.marked) > 0 {
			for _, r := range a.rows {
				if a.marked[r.Pkg] {
					a.running = true
					a.runRow = r
					return a, tea.Batch(doUpgrade(r), tea.Tick(time.Millisecond*100, func(time.Time) tea.Msg { return tickMsg{} }))
				}
			}
		}
		return a, nil
	}
	return a, nil
}

// move shifts the cursor of the focused pane, or scrolls details.
func (a App) move(delta int) (tea.Model, tea.Cmd) {
	switch a.focus {
	case focusSidebar:
		a.sel = clamp(a.sel+delta, 0, len(entries)-1)
		a.tblCur = 0
		a.histCur = 0
		a.applyScope()
	case focusTable:
		if a.view() == viewHistory {
			a.histCur = clamp(a.histCur+delta, 0, max(0, len(a.hist)-1))
		} else {
			a.tblCur = clamp(a.tblCur+delta, 0, max(0, len(a.shown)-1))
		}
	}
	return a, nil
}

func (a App) listHeight() int {
	h := a.height - 8 // header + footer + padding
	if h < 5 {
		return 5
	}
	return h
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ---- actions ---------------------------------------------------------------

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
			return upgradeMsg{row: r, err: fmt.Errorf("unknown manager %q", r.Manager)}
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
		return upgradeMsg{row: r, err: err}
	}
}

// ---- view ------------------------------------------------------------------

func (a App) View() string {
	if a.quitting {
		return ""
	}

	sidebarW := 24
	contentW := a.width - sidebarW
	if contentW < 50 {
		contentW = 50
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		a.renderSidebar(sidebarW),
		a.renderContent(contentW),
	)
	status := a.renderStatus(contentW + sidebarW)
	return lipgloss.JoinVertical(lipgloss.Left, body, status)
}

func (a App) renderSidebar(width int) string {
	var b strings.Builder
	title := a.st.Title.Render("⟳ qterm-updater")
	b.WriteString(title + "\n\n")

	for i, e := range entries {
		if e.sec != secHome && (i == 0 || entries[i-1].sec != e.sec) {
			b.WriteString(a.st.Section.Render(sectionNames[e.sec]) + "\n")
		}

		badge := ""
		switch e.view {
		case viewAll:
			badge = fmt.Sprintf(" %d", len(a.rows))
		case viewMarked:
			n := 0
			for _, m := range a.marked {
				if m {
					n++
				}
			}
			badge = fmt.Sprintf(" %d", n)
		case viewBrew, viewNpm, viewPnpm, viewUv:
			n := 0
			for _, r := range a.rows {
				if r.Manager == e.label {
					n++
				}
			}
			badge = fmt.Sprintf(" %d", n)
		case viewHistory:
			if len(a.hist) > 0 {
				badge = fmt.Sprintf(" %d", len(a.hist))
			}
		}

		key := fmt.Sprintf("%d", i+1)
		label := fmt.Sprintf("%s %-10s%s", key, e.label, badge)
		if i == a.sel {
			b.WriteString(a.st.SidebarSel.Render(label) + "\n")
		} else {
			b.WriteString(a.st.Sidebar.Render(label) + "\n")
		}
	}
	// focus indicator
	if a.focus == focusSidebar {
		b.WriteString("\n" + a.st.Accent.Render("◀ focus"))
	}
	return lipgloss.NewStyle().Width(width).Padding(0, 1).Render(b.String())
}

func (a App) renderContent(width int) string {
	switch a.view() {
	case viewHome:
		return a.renderHome(width)
	case viewHistory:
		return a.renderHistory(width)
	default:
		return a.renderTableDetails(width)
	}
}

func (a App) renderHome(width int) string {
	var b strings.Builder
	b.WriteString(a.st.Header.Render("  HEALTH SUMMARY") + "\n\n")

	byMgr := map[string]int{}
	for _, r := range a.rows {
		byMgr[r.Manager]++
	}

	tileW := 18
	var tiles []string
	tiles = append(tiles, a.st.Tile.
		Width(tileW).
		Render(fmt.Sprintf("Outdated\n%s", a.st.TileVal.Render(fmt.Sprintf("%4d", len(a.rows))))))
	tiles = append(tiles, a.st.Tile.
		Width(tileW).
		Render(fmt.Sprintf("Marked\n%s", a.st.TileVal.Render(fmt.Sprintf("%4d", len(a.marked))))))
	for _, mgr := range []string{"brew", "npm", "pnpm", "uv"} {
		if n := byMgr[mgr]; n > 0 {
			tiles = append(tiles, a.st.Tile.
				Width(tileW).
				Render(fmt.Sprintf("%s\n%s", mgr, a.st.TileVal.Render(fmt.Sprintf("%4d", n)))))
		}
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tiles...) + "\n\n")

	// Upgrade banner with a spinner.
	if a.running {
		spin := spinnerFrame()
		b.WriteString("  " + a.st.Warn.Render(spin+" upgrading "+a.runRow.Pkg+" ("+a.runRow.Manager+")") + "\n\n")
	}

	// Recent log lines.
	if len(a.logs) > 0 {
		b.WriteString(a.st.Header.Render("  LOG") + "\n")
		start := max(0, len(a.logs)-8)
		for _, l := range a.logs[start:] {
			b.WriteString("  " + l + "\n")
		}
		b.WriteString("\n")
	}

	// Help hint.
	b.WriteString("  " + a.st.Muted.Render("j/k or arrows navigate · u upgrade · U upgrade marked · / filter") + "\n")
	return lipgloss.NewStyle().Width(width).Render(b.String())
}

var spinFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func spinnerFrame() string {
	return spinFrames[int(time.Now().UnixNano()/100000000)%len(spinFrames)]
}

func (a App) renderTableDetails(width int) string {
	tableW := min(width*3/5, 70)
	detailsW := width - tableW - 2
	if detailsW < 24 {
		detailsW = 24
	}
	return lipgloss.JoinHorizontal(lipgloss.Top,
		a.renderTable(tableW),
		"  ",
		a.renderDetails(detailsW),
	)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (a App) renderTable(width int) string {
	var b strings.Builder

	// Filter line.
	if a.filtering {
		b.WriteString("  /" + a.filter + "█\n")
	} else if a.filter != "" {
		b.WriteString("  " + a.st.Accent.Render("/"+a.filter) + " " + a.st.Muted.Render("(enter to edit, esc to clear)") + "\n")
	}

	// Column header.
	hdr := fmt.Sprintf("  %-*s %-*s %-*s %-*s",
		pkgCols[0].width, pkgCols[0].title,
		pkgCols[1].width, pkgCols[1].title,
		pkgCols[2].width, pkgCols[2].title,
		pkgCols[3].width, pkgCols[3].title)
	b.WriteString(a.st.TableHeader.Render(hdr) + "\n")

	if len(a.shown) == 0 {
		b.WriteString("  " + a.st.Muted.Render("no packages") + "\n")
		return lipgloss.NewStyle().Width(width).Render(b.String())
	}

	// Visible window around the cursor.
	h := a.listHeight()
	start, end := a.window(len(a.shown), a.tblCur, h)

	for i := start; i < end; i++ {
		r := a.shown[i]
		mark := " "
		if a.marked[r.Pkg] {
			mark = a.st.Mark.Render("●")
		}
		line := fmt.Sprintf("  %s %-*s %-*s %-*s",
			mark,
			pkgCols[1].width, trunc(r.Pkg, pkgCols[1].width),
			pkgCols[2].width, trunc(r.Cur, pkgCols[2].width),
			pkgCols[3].width, trunc(r.New, pkgCols[3].width))
		if i == a.tblCur {
			if a.focus == focusTable {
				b.WriteString(a.st.TableCur.Render(line) + "\n")
			} else {
				b.WriteString(a.st.TableRowDim.Render(line) + "\n")
			}
		} else {
			b.WriteString(a.st.TableRow.Render(line) + "\n")
		}
	}
	if n := len(a.shown); n > end {
		b.WriteString("  " + a.st.Muted.Render(fmt.Sprintf("↓ %d more", n-end)) + "\n")
	}
	return lipgloss.NewStyle().Width(width).Render(b.String())
}

func (a App) renderDetails(width int) string {
	var b strings.Builder
	b.WriteString(a.st.Header.Render("DETAILS") + "\n\n")
	if len(a.shown) == 0 || a.tblCur >= len(a.shown) {
		b.WriteString(a.st.Muted.Render("nothing selected"))
	} else {
		r := a.shown[a.tblCur]
		b.WriteString(fmt.Sprintf("  %s\n\n", a.st.Title.Render(r.Pkg)))
		rows := [][2]string{
			{"Manager", r.Manager},
			{"Current", r.Cur},
			{"Latest", a.st.OK.Render(r.New)},
			{"Marked", "yes"},
		}
		for _, kv := range rows {
			if kv[1] == "yes" && !a.marked[r.Pkg] {
				continue
			}
			if kv[1] == "" {
				continue
			}
			b.WriteString(fmt.Sprintf("  %-10s %s\n", kv[0]+":", kv[1]))
		}
		if r.Desc != "" {
			b.WriteString("\n  " + wrapWords(r.Desc, width-4) + "\n")
		}
		if r.Home != "" {
			b.WriteString("\n  " + a.st.Muted.Render(trunc(r.Home, width-4)) + "\n")
		}
		b.WriteString("\n  " + a.st.Muted.Render("u = upgrade this package"))
	}
	if a.focus == focusDetails {
		b.WriteString("\n\n" + a.st.Accent.Render("◀ focus"))
	}
	return lipgloss.NewStyle().Width(width).Padding(1, 1).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("8")).Render(b.String())
}

func (a App) renderHistory(width int) string {
	var b strings.Builder
	b.WriteString(a.st.Header.Render("  HISTORY") + "\n\n")

	if len(a.hist) == 0 {
		b.WriteString("  " + a.st.Muted.Render("no history yet — upgrades run from here are recorded") + "\n")
		return lipgloss.NewStyle().Width(width).Render(b.String())
	}

	hdr := fmt.Sprintf("  %-*s %-*s %-*s %-*s",
		histCols[0].width, histCols[0].title,
		histCols[1].width, histCols[1].title,
		histCols[2].width, histCols[2].title,
		histCols[3].width, histCols[3].title)
	b.WriteString(a.st.TableHeader.Render(hdr) + "\n")

	h := a.listHeight()
	start, end := a.window(len(a.hist), a.histCur, h)
	for i := start; i < end; i++ {
		e := a.hist[i]
		status := a.st.OK.Render("✓")
		if !e.OK {
			status = a.st.Err.Render("✗")
		}
		line := fmt.Sprintf("  %-*s %-*s %-*s %-*s",
			histCols[0].width, status,
			histCols[1].width, trunc(e.Pkg, histCols[1].width),
			histCols[2].width, trunc(e.From+" → "+e.To, histCols[2].width),
			histCols[3].width, e.Manager)
		if i == a.histCur && a.focus == focusTable {
			b.WriteString(a.st.TableCur.Render(line) + "\n")
		} else {
			b.WriteString(a.st.TableRow.Render(line) + "\n")
		}
	}
	if n := len(a.hist); n > end {
		b.WriteString("  " + a.st.Muted.Render(fmt.Sprintf("↓ %d more", n-end)) + "\n")
	}
	return lipgloss.NewStyle().Width(width).Render(b.String())
}

func (a App) renderStatus(width int) string {
	left := fmt.Sprintf(" %d outdated · %d marked", len(a.rows), len(a.marked))
	if a.running {
		left += " · " + a.st.Warn.Render("running "+a.runRow.Pkg)
	}
	var focus string
	switch a.focus {
	case focusSidebar:
		focus = "sidebar"
	case focusTable:
		focus = "table"
	case focusDetails:
		focus = "details"
	case focusFilter:
		focus = "filter"
	}
	right := a.st.Muted.Render(fmt.Sprintf("focus: %s", focus)) + "  "
	keys := a.st.Footer.Render("j/k↑↓ move · h/l←→ pane · 1-8 jump · space mark · u/U upgrade · / filter · q quit")
	bar := lipgloss.NewStyle().Width(width).Padding(0, 1).Render(left)
	return lipgloss.JoinVertical(lipgloss.Left, bar, keys, right)
}

// window returns [start,end) of visible rows around cursor cur.
func (a App) window(n, cur, h int) (int, int) {
	if n <= h {
		return 0, n
	}
	start := cur - h/2
	if start < 0 {
		start = 0
	}
	if start+h > n {
		start = n - h
	}
	return start, start + h
}

// ---- small helpers ---------------------------------------------------------

func trunc(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	if w <= 1 {
		return s[:w]
	}
	return s[:w-1] + "…"
}

func wrapWords(s string, w int) string {
	var b strings.Builder
	line := 0
	for _, word := range strings.Fields(s) {
		if line+len(word)+1 > w {
			b.WriteString("\n  ")
			line = 0
		} else if line > 0 {
			b.WriteString(" ")
			line++
		}
		b.WriteString(word)
		line += len(word)
	}
	return b.String()
}
