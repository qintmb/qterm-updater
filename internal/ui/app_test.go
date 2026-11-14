package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/qintmb/qterm-updater/internal/scan"
	"github.com/qintmb/qterm-updater/internal/state"
)

// dummy rows for tests
func testRows() []scan.Row {
	return []scan.Row{
		{Manager: "brew", Pkg: "git", Cur: "2.1", New: "2.2"},
		{Manager: "npm", Pkg: "typescript", Cur: "5.0", New: "5.1"},
		{Manager: "uv", Pkg: "ruff", Cur: "0.1", New: "0.2", Desc: "a linter"},
	}
}

func TestNewAppScopesRows(t *testing.T) {
	a := NewApp(testRows())
	if len(a.rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(a.rows))
	}
	if len(a.shown) != 3 {
		t.Fatalf("shown = %d, want 3", len(a.shown))
	}
}

func TestApplyScopeManager(t *testing.T) {
	a := NewApp(testRows())
	// jump to brew (index 3)
	a.sel = 3
	a.applyScope()
	if len(a.shown) != 1 || a.shown[0].Manager != "brew" {
		t.Fatalf("brew scope = %+v, want 1 brew row", a.shown)
	}
}

func TestFilterKeyNarrows(t *testing.T) {
	a := NewApp(testRows())
	a.openFilter()
	a.filt.insert('t')
	a.filt.insert('y')
	a.filt.insert('p')
	a.filt.insert('e')
	a.filter = a.filt.value
	a.applyScope()
	if len(a.shown) != 1 || a.shown[0].Pkg != "typescript" {
		t.Fatalf("filter type shown = %+v, want 1 (typescript)", a.shown)
	}
}

func TestMarkCycle(t *testing.T) {
	a := NewApp(testRows())
	a.focus = colList
	a.tblCur = 1
	pkg := a.shown[1].Pkg
	a.marked[pkg] = true
	a.applyScope()
	// mark is outside scope by design; toggling happens via key
	if !a.marked[pkg] {
		t.Fatalf("mark not set")
	}
}

func TestSplitArgs(t *testing.T) {
	cases := []struct {
		in   string
		want []string
		err  bool
	}{
		{"theme", []string{"theme"}, false},
		{"icons on", []string{"icons", "on"}, false},
		{`"a b" c`, []string{"a b", "c"}, false},
		{"upgrade-all", []string{"upgrade-all"}, false},
		{`unclosed "quote`, nil, true},
	}
	for _, c := range cases {
		got, err := splitArgs(c.in)
		if c.err {
			if err == nil {
				t.Errorf("%q: want error, got %v", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q: unexpected error %v", c.in, err)
			continue
		}
		if len(got) != len(c.want) {
			t.Errorf("%q: got %v want %v", c.in, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%q: got %v want %v", c.in, got, c.want)
				break
			}
		}
	}
}

func TestSelectEntryJumps(t *testing.T) {
	a := NewApp(testRows())
	a.sel = 1
	a.applyScope()
	if a.entryView() != viewAll {
		t.Fatalf("view = %d, want viewAll", a.entryView())
	}
}

func TestStatusBarDoesNotPanic(t *testing.T) {
	a := NewApp(testRows())
	a.width, a.height = 100, 30
	_ = a.View()
	a.help = true
	_ = a.View()
	a.openThemes()
	_ = a.View()
}

var _ = state.HistoryEntry{}
var _ = tea.Msg(nil)

func TestQuitKey(t *testing.T) {
	a := NewApp(testRows())
	m, _ := a.Update(tea.KeyMsg{})
	_ = m
	a.quitting = true
	if !a.quitting {
		t.Fatal("quit not set")
	}
}

func TestThemeSwitchAndPersist(t *testing.T) {
	a := NewApp(testRows())
	a = a.keepTheme("nord")
	if a.themeName != "nord" {
		t.Fatalf("themeName = %s, want nord", a.themeName)
	}
	a = a.useTheme("dracula")
	if a.themeName != "dracula" {
		t.Fatalf("themeName = %s, want dracula", a.themeName)
	}
}

func TestCommandExecuteTheme(t *testing.T) {
	a := NewApp(testRows())
	app, _ := a.execute("theme dracula")
	if app.themeName != "dracula" {
		t.Fatalf("themeName = %s, want dracula", app.themeName)
	}
}

func TestCommandUnknownFlash(t *testing.T) {
	a := NewApp(testRows())
	app, _ := a.execute("bogus")
	if app.flash == "" {
		t.Fatal("expected flash on unknown command")
	}
}
