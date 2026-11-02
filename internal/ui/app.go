package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/qintmb/qterm-updater/internal/scan"
	"github.com/qintmb/qterm-updater/internal/state"
	"github.com/qintmb/qterm-updater/internal/theme"
)

// section groups sidebar entries.
type section int

const (
	secHome section = iota
	secPackages
	secTools
)

var sectionNames = map[section]string{
	secPackages: "PACKAGES",
	secTools:    "TOOLS",
}

// entry is one line in the sidebar.
type entry struct {
	label string
	sec   section
	view  int
}

var entries = []entry{
	{"Home", secHome, viewHome},
	{"All", secPackages, viewAll},
	{"Outdated", secPackages, viewOutdated},
	{"History", secTools, viewHistory},
}

const (
	viewHome = iota
	viewAll
	viewOutdated
	viewHistory
)

// App is the root Bubble Tea model.
type App struct {
	st       theme.Styles
	sel      int          // selected sidebar entry
	rows     []scan.Row   // scan result
	filter   string       // "/" filter
	marked   map[int]bool // rows picked with space
	focus    focusArea    // where keys go
	width    int
	height   int
	quitting bool
}

type focusArea int

const (
	focusSidebar focusArea = iota
	focusTable
	focusFilter
)

// NewApp creates the initial model.
func NewApp(rows []scan.Row) App {
	st := theme.Default()
	return App{
		st:     st,
		rows:   rows,
		marked: make(map[int]bool),
		focus:  focusTable,
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
	}
	return a, nil
}

func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	case "1", "2", "3", "4":
		idx := int(msg.String()[0] - '1')
		if idx >= 0 && idx < len(entries) {
			a.sel = idx
			a.focus = focusTable
		}
	case "j", "down":
		if a.focus == focusSidebar && a.sel < len(entries)-1 {
			a.sel++
		}
	case "k", "up":
		if a.focus == focusSidebar && a.sel > 0 {
			a.sel--
		}
	}
	return a, nil
}

func (a App) View() string {
	if a.quitting {
		return ""
	}
	sidebar := a.renderSidebar()
	content := a.renderContent()
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

func (a App) renderSidebar() string {
	var b strings.Builder
	for i, e := range entries {
		style := a.st.Muted
		if i == a.sel {
			style = a.st.Selected
		}
		if e.sec == secHome && i > 0 {
			// First entry is Home, no section header.
		} else if name, ok := sectionNames[e.sec]; ok && (i == 0 || entries[i-1].sec != e.sec) {
			b.WriteString(a.st.Header.Render(name) + "\n")
		}
		b.WriteString(style.Render("  "+e.label) + "\n")
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (a App) renderContent() string {
	switch entries[a.sel].view {
	case viewHome:
		return a.renderHome()
	case viewAll, viewOutdated:
		return a.renderTable()
	case viewHistory:
		return a.renderHistory()
	default:
		return ""
	}
}

func (a App) renderHome() string {
	var b strings.Builder
	b.WriteString(a.st.Title.Render("qterm-updater") + "\n\n")
	b.WriteString(fmt.Sprintf("%d outdated packages\n", len(a.rows)))
	byMgr := make(map[string]int)
	for _, r := range a.rows {
		byMgr[r.Manager]++
	}
	for mgr, n := range byMgr {
		b.WriteString(fmt.Sprintf("  %s: %d\n", mgr, n))
	}
	b.WriteString("\nPress Enter to update all, or use the sidebar.\n")
	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (a App) renderTable() string {
	var b strings.Builder
	b.WriteString(a.st.Header.Render("PACKAGE                 CURRENT        NEW           DESCRIPTION") + "\n")
	for _, r := range a.rows {
		line := fmt.Sprintf("%-24s %-14s %-13s %s", r.Pkg, r.Cur, r.New, r.Desc)
		b.WriteString(line + "\n")
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (a App) renderHistory() string {
	h := state.LoadHistory()
	if len(h) == 0 {
		return a.st.Muted.Render("No history yet.")
	}
	var b strings.Builder
	b.WriteString(a.st.Header.Render("RECENT UPDATES") + "\n")
	for _, e := range h {
		status := a.st.OK.Render("✓")
		if !e.OK {
			status = a.st.Err.Render("✗")
		}
		b.WriteString(fmt.Sprintf("%s %s %s → %s (%s)\n", status, e.Pkg, e.From, e.To, e.Manager))
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}
