package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/qintmb/qterm-updater/internal/theme"
)

// col describes a table column. Width 0 means "take the remaining space".
type tableCol struct {
	title string
	width int
}

var pkgCols = []tableCol{
	{"", 2},
	{"PACKAGE", 22},
	{"CURRENT", 12},
	{"LATEST", 12},
	{"MANAGER", 0},
}

var histCols = []tableCol{
	{"", 2},
	{"PACKAGE", 20},
	{"VERSION", 0},
	{"MANAGER", 8},
	{"WHEN", 16},
}

// widths fits cols into w, giving the flexible ones what is left.
func widths(cols []tableCol, w int) []int {
	ws := make([]int, len(cols))
	fixed := 0
	for i, c := range cols {
		if c.width > 0 {
			ws[i] = max(c.width, ansi.StringWidth(c.title))
			fixed += ws[i] + 2
		}
	}
	spare := max(w-fixed, 8)
	for i, c := range cols {
		if c.width == 0 {
			ws[i] = spare
			spare = 0
		}
	}
	return ws
}

// rowLine renders cells into w cells with 2-space gaps.
func rowLine(cells []string, ws []int) string {
	var b strings.Builder
	for i, c := range cells {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(padRight(c, ws[i]))
	}
	return ansi.Truncate(b.String(), total(ws), "…")
}

func total(ws []int) int {
	n := 0
	for _, w := range ws {
		n += w + 2
	}
	return n - 2
}

// mark is the cell a marked row shows.
func (a App) mark(pkg string) string {
	if a.marked[pkg] {
		return a.st.Warn.Render("●")
	}
	return " "
}

// listView draws the current entry's rows inside width w columns.
func (a App) listView(w int) string {
	if a.entryView() == viewHistory {
		return a.historyView(w)
	}
	ws := widths(pkgCols, w)
	var b strings.Builder
	if a.filter != "" && a.mode != modeFilter {
		b.WriteString(a.st.Muted.Render("/"+a.filter) + "\n")
	}
	b.WriteString(a.st.TableHeader.Render(rowLine([]string{"", "PACKAGE", "CURRENT", "LATEST", "MANAGER"}, ws)) + "\n")
	if len(a.shown) == 0 {
		b.WriteString(a.st.Muted.Render("  nothing to update"))
		return b.String()
	}
	h := a.listHeight() - 2
	if a.filter != "" {
		h--
	}
	start, end := a.window(len(a.shown), a.tblCur, h)
	for i := start; i < end; i++ {
		r := a.shown[i]
		cells := []string{a.mark(r.Pkg), r.Pkg, r.Cur, r.New, r.Manager}
		line := rowLine(cells, ws)
		if i == a.tblCur {
			b.WriteString(a.st.TableSelected.Render(line) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}
	if n := len(a.shown); n > end {
		b.WriteString(a.st.Muted.Render(fmt.Sprintf("  … %d more", n-end)))
	}
	return b.String()
}

// historyView draws the finished jobs.
func (a App) historyView(w int) string {
	ws := widths(histCols, w)
	var b strings.Builder
	b.WriteString(a.st.TableHeader.Render(rowLine([]string{"", "PACKAGE", "VERSION", "MANAGER", "WHEN"}, ws)) + "\n")
	if len(a.hist) == 0 {
		b.WriteString(a.st.Muted.Render("  no history yet"))
		return b.String()
	}
	h := a.listHeight() - 2
	start, end := a.window(len(a.hist), a.histCur, h)
	for i := start; i < end; i++ {
		e := a.hist[i]
		status := a.st.OK.Render("✓")
		if !e.OK {
			status = a.st.Err.Render("✗")
		}
		cells := []string{status, e.Pkg, e.From + " → " + e.To, e.Manager, when(e.StartedAt)}
		line := rowLine(cells, ws)
		if i == a.histCur {
			b.WriteString(a.st.TableSelected.Render(line) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}
	if n := len(a.hist); n > end {
		b.WriteString(a.st.Muted.Render(fmt.Sprintf("  … %d more", n-end)))
	}
	return b.String()
}

// when is a job's time as the list shows it.
func when(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}

// homeView is the health summary: counts per manager, recent jobs and the
// keys that work here.
func (a App) homeView(w int) string {
	var b strings.Builder
	b.WriteString(a.st.Title.Render("Health") + "\n\n")
	byMgr := map[string]int{}
	for _, r := range a.rows {
		byMgr[r.Manager]++
	}
	managers := []string{"brew", "npm", "pnpm", "uv"}
	for _, m := range managers {
		line := padRight(theme.With(a.st.Icons.Formula, m), 12) + statsBar(byMgr[m], len(a.rows), a.st) +
			"  " + a.st.Muted.Render(fmt.Sprint(byMgr[m]))
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")
	b.WriteString(a.st.Muted.Render(fmt.Sprintf("%d outdated · %d marked", len(a.rows), a.marks())))
	if !a.updatedAt.IsZero() {
		b.WriteString(a.st.Muted.Render(" · scanned " + a.updatedAt.Format("15:04")))
	}
	b.WriteString("\n")
	if len(a.logs) > 0 {
		b.WriteString("\n" + a.st.Title.Render("Recent jobs") + "\n")
		start := max(len(a.logs)-6, 0)
		for _, l := range a.logs[start:] {
			b.WriteString(ansi.Truncate(l, max(w-2, 8), "…") + "\n")
		}
	}
	return b.String()
}

// statsBar is a small bar for n of total.
func statsBar(n, total int, st theme.Styles) string {
	const cells = 10
	fill := 0
	if total > 0 {
		fill = n * cells / total
	}
	return st.Accent.Render(strings.Repeat("▮", fill)) + st.Muted.Render(strings.Repeat("▯", cells-fill))
}

// detailsView is the selected package: versions, descriptionUS, homepage.
func (a App) detailsView(w int) string {
	r := a.selected()
	if r == nil {
		return a.st.Muted.Render("nothing selected")
	}
	inner := max(w-4, 8)
	var b strings.Builder
	b.WriteString(a.st.Title.Render(ansi.Truncate(r.Pkg, inner, "…")) + "\n\n")
	pairs := []string{"Manager", r.Manager, "Current", r.Cur, "Latest", r.New, "Marked", ""}
	b.WriteString(kv(a.st, pairs...))
	if r.Desc != "" {
		b.WriteString("\n" + a.st.Text.Render(wrapWords(r.Desc, inner)) + "\n")
	}
	if r.Home != "" {
		b.WriteString("\n" + a.st.Muted.Render(ansi.Truncate(r.Home, inner, "…")) + "\n")
	}
	b.WriteString("\n" + a.st.Muted.Render("u upgrade · space mark"))
	return b.String()
}
