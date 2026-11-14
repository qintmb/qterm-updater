package ui

import (
	"strings"
	"testing"
)

func TestSmokeViews(t *testing.T) {
	a := NewApp(testRows())
	a.width, a.height = 120, 40
	for i := range entries {
		a.sel = i
		a.applyScope()
		out := a.View()
		if out == "" {
			t.Fatalf("entry %d empty view", i)
		}
		if strings.Contains(out, "loading") {
			t.Fatalf("entry %d stuck loading", i)
		}
	}
	// command popup
	a.openCmdline()
	a.cmd.value = "theme"
	a.cmd.items = suggestions("theme")
	a.cmd.sel = 0
	_ = a.View()
	// filter bar
	a.openFilter()
	_ = a.View()
	// help
	a.help = true
	_ = a.View()
	// theme picker
	a.help = false
	a.openThemes()
	_ = a.View()
	t.Log("smoke OK")
}
