package ui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/qintmb/qterm-updater/internal/scan"
	"github.com/qintmb/qterm-updater/internal/state"
	"github.com/qintmb/qterm-updater/internal/theme"
)

// section is a group of sidebar entries; 1-8 jump to them by position.
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

// entry is one line of the sidebar.
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

// col is a focus column.
type col int

const (
	colSidebar col = iota
	colList
	colDetails
)

// mode routes key presses.
type mode int

const (
	modeNormal mode = iota
	modeCommand
	modeFilter
)

// scanMsg carries a fresh scan's rows.
type scanMsg struct {
	rows []scan.Row
	errs map[string]error
}

// upgradeMsg is sent when an upgrade finishes.
type upgradeMsg struct {
	row scan.Row
	err error
}

// tickMsg re-renders while a job runs and advances the spinner.
type tickMsg struct{}

// App is the root Bubble Tea model.
type App struct {
	st        theme.Styles
	themeName string
	iconsOn   bool
	updatedAt time.Time // when the current rows were scanned

	sel     int
	tblCur  int
	histCur int
	rows    []scan.Row
	shown   []scan.Row
	hist    []state.HistoryEntry

	filter    string
	filtering bool
	marked    map[string]bool

	focus col
	mode  mode
	cmd   cmdline
	filt  filt

	help         bool
	themes       *themeBox
	drawerScroll int // details scroll when colDetails is focused

	running bool
	runRow  scan.Row
	logs    []string
	frame   int

	flash    string
	width    int
	height   int
	quitting bool
}

// NewApp creates the initial model, reading saved settings.
func NewApp(rows []scan.Row) App {
	dir := state.Dir()
	name := theme.LoadTheme(dir)
	st := theme.New(theme.Lookup(name).Palette)
	st.Icons = iconSet(theme.LoadIcons(dir))
	a := App{
		st:        st,
		themeName: name,
		iconsOn:   theme.LoadIcons(dir),
		rows:      rows,
		hist:      state.LoadHistory(),
		marked:    make(map[string]bool),
		focus:     colList,
		cmd:       cmdline{sel: -1},
		filt:      filt{sel: -1},
	}
	if c := state.LoadCache(); c != nil {
		a.updatedAt = c.ScannedAt
	}
	a.applyScope()
	return a
}

// iconSet picks the glyphs the app draws, or none when icons are off.
func iconSet(on bool) theme.Icons {
	if on {
		return theme.Nerd
	}
	return theme.Icons{}
}

// Init returns no command: the app starts on cached data.
func (a App) Init() tea.Cmd { return nil }

// entryView returns the current sidebar entry's view.
func (a App) entryView() int { return entries[a.sel].view }

// listHeight is the number of rows the list pane can show.
func (a App) listHeight() int {
	h := a.height - 1 - 4 // status bar, pane title, borders, padding
	if h < 5 {
		return 5
	}
	return h
}

// applyScope rebuilds a.shown for the current entry and filter.
func (a *App) applyScope() {
	var out []scan.Row
	curView := a.entryView()
	for _, r := range a.rows {
		switch curView {
		case viewMarked:
			if !a.marked[r.Pkg] {
				continue
			}
		case viewBrew, viewNpm, viewPnpm, viewUv:
			if r.Manager != entries[a.sel].label {
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

// selected returns the row under the list cursor, if the current view has one.
func (a App) selected() *scan.Row {
	if a.entryView() == viewHistory || len(a.shown) == 0 || a.tblCur >= len(a.shown) {
		return nil
	}
	return &a.shown[a.tblCur]
}

// hasDetails reports whether the current view shows a details pane.
func (a App) hasDetails() bool {
	v := a.entryView()
	return v != viewHome && v != viewHistory
}

// selectedName is the package or job the status bar names.
func (a App) selectedName() (string, bool) {
	if r := a.selected(); r != nil {
		return r.Pkg, true
	}
	if a.entryView() == viewHistory && a.histCur < len(a.hist) {
		return a.hist[a.histCur].Pkg, true
	}
	return "", false
}

// marks is how many packages are marked.
func (a App) marks() int {
	n := 0
	for _, m := range a.marked {
		if m {
			n++
		}
	}
	return n
}

// openCmdline switches to command mode with an empty prompt.
func (a *App) openCmdline() {
	a.mode = modeCommand
	a.cmd.reset()
	a.cmd.sel = -1
	a.cmd.items = nil
}

// openFilter switches to filter mode, keeping any filter already set.
func (a *App) openFilter() {
	a.mode = modeFilter
	a.filt.set(a.filter)
	a.filt.sel = -1
	a.filt.items = nil
}

// ---- update ---------------------------------------------------------------

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		return a, nil

	case tickMsg:
		a.frame++
		if a.running {
			return a, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
		}
		return a, nil

	case scanMsg:
		a.rows = msg.rows
		a.updatedAt = time.Now()
		a.applyScope()
		a.flash = a.st.Info.Render("refreshed")
		if len(msg.errs) > 0 {
			var names []string
			for n := range msg.errs {
				names = append(names, n)
			}
			a.flash = a.st.Err.Render("✗ " + strings.Join(names, ", ") + " failed")
		}
		return a, nil

	case upgradeMsg:
		a.running = false
		status, style := "✓", a.st.OK
		if msg.err != nil {
			status, style = "✗", a.st.Err
		}
		line := style.Render(status) + " " + msg.row.Pkg + " " + msg.row.Cur + " → " + msg.row.New + " (" + msg.row.Manager + ")"
		a.logs = append(a.logs, line)
		if len(a.logs) > 20 {
			a.logs = a.logs[len(a.logs)-20:]
		}
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
		a.flash = a.st.Info.Render(line)
		return a, nil

	case tea.KeyMsg:
		return a.handleKey(msg)
	}
	return a, nil
}

// handleKey routes a key press by mode and open overlay.
func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	switch a.mode {
	case modeCommand:
		return a.cmdKey(msg)
	case modeFilter:
		return a.filterKey(msg)
	}
	if a.themes != nil {
		return a.themesKey(k)
	}
	if a.help {
		if k == "esc" || k == "q" || k == "?" {
			a.help = false
		}
		return a, nil
	}
	return a.normalKey(k)
}

// normalKey handles keys in normal mode.
func (a App) normalKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "ctrl+c", "q":
		a.quitting = true
		return a, tea.Quit
	case ":":
		a.openCmdline()
		return a, nil
	case "?":
		a.help = true
		return a, nil
	case "t":
		a.openThemes()
		return a, nil
	case "r":
		return a.refresh()
	case "A":
		if !a.running && a.hasDetails() {
			return a.upgradeAll()
		}
		return a, nil
	case "U":
		if !a.running && a.marks() > 0 {
			return a.upgradeMarked()
		}
		return a, nil
	case "/":
		a.openFilter()
		return a, nil
	case "h", "left":
		switch a.focus {
		case colList:
			a.focus = colSidebar
		case colDetails:
			a.focus = colList
		}
		return a, nil
	case "l", "right", "enter", "tab":
		switch a.focus {
		case colSidebar:
			a.focus = colList
		case colList:
			if a.hasDetails() {
				a.focus = colDetails
			}
		}
		return a, nil
	case "[":
		return a.selectEntry((a.sel + len(entries) - 1) % len(entries))
	case "]":
		return a.selectEntry((a.sel + 1) % len(entries))
	case "1", "2", "3", "4", "5", "6", "7", "8":
		idx := int(k[0] - '1')
		if idx < len(entries) {
			return a.selectEntry(idx)
		}
		return a, nil
	}
	if a.sidebarOrListKey(k) {
		return a.moveKey(k)
	}
	return a, nil
}

// sidebarOrListKey reports whether k is a movement key for sidebar or list.
func (a App) sidebarOrListKey(k string) bool {
	switch k {
	case "j", "down", "k", "up", "home", "gg", "G", "end", "ctrl+d", "ctrl+u", "pgup", "pgdown":
		return true
	}
	return false
}

// selectEntry shows entry i and returns the command to run, if any.
func (a App) selectEntry(i int) (App, tea.Cmd) {
	a.sel = i
	a.tblCur, a.histCur = 0, 0
	a.applyScope()
	if a.focus == colSidebar {
		a.focus = colList
	}
	return a, nil
}

// moveKey handles j/k and friends on the focused column.
func (a App) moveKey(k string) (tea.Model, tea.Cmd) {
	switch a.focus {
	case colSidebar:
		switch k {
		case "j", "down":
			a.sel = min(a.sel+1, len(entries)-1)
		case "k", "up":
			a.sel = max(a.sel-1, 0)
		case "home", "gg":
			a.sel = 0
		case "G", "end":
			a.sel = len(entries) - 1
		}
		a.tblCur, a.histCur = 0, 0
		a.applyScope()
	case colList:
		n := len(a.shown)
		if a.entryView() == viewHistory {
			n = len(a.hist)
		}
		switch k {
		case "j", "down":
			a.tblCur = min(a.tblCur+1, max(n-1, 0))
		case "k", "up":
			a.tblCur = max(a.tblCur-1, 0)
		case "home", "gg":
			a.tblCur = 0
		case "G", "end":
			a.tblCur = max(n-1, 0)
		case "ctrl+d", "pgdown":
			a.tblCur = min(a.tblCur+a.listHeight(), max(n-1, 0))
		case "ctrl+u", "pgup":
			a.tblCur = max(a.tblCur-a.listHeight(), 0)
		}
		a.histCur = a.tblCur
	case colDetails:
		switch k {
		case "j", "down", "ctrl+d", "pgdown":
			a.drawerScroll++
		case "k", "up", "ctrl+u", "pgup":
			a.drawerScroll = max(a.drawerScroll-1, 0)
		}
	}
	return a, nil
}

// refresh re-scans every manager.
func (a App) refresh() (tea.Model, tea.Cmd) {
	return a, func() tea.Msg {
		res, errs := scan.Run(context.Background(), scan.All())
		return scanMsg{rows: res.Rows, errs: errs}
	}
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

// clamp keeps v in [lo,hi].
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// trunc cuts s to w cells, adding an ellipsis when cut.
func trunc(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	if w <= 1 {
		return s[:w]
	}
	return s[:w-1] + "…"
}
