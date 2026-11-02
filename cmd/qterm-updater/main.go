package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/qintmb/qterm-updater/internal/scan"
	"github.com/qintmb/qterm-updater/internal/state"
	"github.com/qintmb/qterm-updater/internal/ui"
)

// version is stamped by the build: -ldflags "-X main.version=...".
var version = "dev"

func main() {
	var (
		list    = flag.Bool("list", false, "print the cached scan as text and exit")
		refresh = flag.Bool("refresh", false, "force a fresh scan before starting")
		showVer = flag.Bool("version", false, "print the version and exit")
	)
	flag.Parse()

	if *showVer {
		fmt.Println("qterm-updater", version)
		return
	}

	cache := state.LoadCache()
	if *refresh || cache == nil || time.Since(cache.ScannedAt) > 24*time.Hour {
		fmt.Fprintln(os.Stderr, "Scanning package managers...")
		res, errs := scan.Run(context.Background(), scan.All())
		for mgr, err := range errs {
			fmt.Fprintf(os.Stderr, "  %s: %v\n", mgr, err)
		}
		cache = &state.Cache{ScannedAt: time.Now(), Rows: res.Rows}
		if err := state.Save(*cache); err != nil {
			fmt.Fprintf(os.Stderr, "  cache write failed: %v\n", err)
		}
	}

	if *list {
		printList(cache)
		return
	}

	out, err := tea.NewProgram(ui.NewApp(cache.Rows), tea.WithAltScreen()).Run()
	_ = out
	if err != nil {
		fmt.Fprintf(os.Stderr, "qterm-updater: %v\n", err)
		os.Exit(1)
	}
}

func printList(c *state.Cache) {
	if len(c.Rows) == 0 {
		fmt.Println("Everything is up to date.")
		return
	}
	fmt.Printf("%d update(s) available — scanned %s\n\n", len(c.Rows), c.ScannedAt.Format("2006-01-02 15:04"))
	w := tabwriter.NewWriter(os.Stdout, 2, 2, 2, ' ', 0)
	fmt.Fprintln(w, "MANAGER\tPACKAGE\tCURRENT\tLATEST\tDESCRIPTION")
	for _, r := range c.Rows {
		desc := r.Desc
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.Manager, r.Pkg, r.Cur, r.New, desc)
	}
	_ = w.Flush()
}
