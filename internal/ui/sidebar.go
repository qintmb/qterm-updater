package ui

import (
	"fmt"
	"strings"
)

// sidebarView renders the entries, a heading per section, each with its count.
func (a App) sidebarView() string {
	var b strings.Builder
	for i, e := range entries {
		if e.sec != secHome && (i == 0 || entries[i-1].sec != e.sec) {
			b.WriteString(a.st.Muted.Bold(true).Render(sectionNames[e.sec]) + "\n")
		}
		label := fmt.Sprintf("%d %s", i+1, e.label)
		if n := a.badgeCount(e.view); n > 0 {
			label = padRight(label, 13) + a.st.Muted.Render(fmt.Sprint(n))
		}
		if i == a.sel {
			b.WriteString(a.st.TabActive.Render(" "+label+" ") + "\n")
		} else {
			b.WriteString(a.st.Text.Render(" "+label) + "\n")
		}
	}
	return b.String()
}

// badgeCount is the count shown beside an entry.
func (a App) badgeCount(view int) int {
	switch view {
	case viewAll:
		return len(a.rows)
	case viewMarked:
		return a.marks()
	case viewBrew, viewNpm, viewPnpm, viewUv:
		n := 0
		for _, r := range a.rows {
			if r.Manager == managerOf(view) {
				n++
			}
		}
		return n
	case viewHistory:
		return len(a.hist)
	}
	return 0
}

// managerOf returns the manager name a manager view filters on.
func managerOf(view int) string {
	switch view {
	case viewBrew:
		return "brew"
	case viewNpm:
		return "npm"
	case viewPnpm:
		return "pnpm"
	case viewUv:
		return "uv"
	}
	return ""
}
