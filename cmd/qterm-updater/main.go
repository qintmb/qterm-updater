package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/qintmb/qterm-updater/internal/scan"
	"github.com/qintmb/qterm-updater/internal/state"
	"github.com/qintmb/qterm-updater/internal/ui"
)

func main() {
	ctx := context.Background()
	
	// Check if we have a recent cache.
	cache := state.LoadCache()
	if cache == nil || time.Since(cache.ScannedAt) > 24*time.Hour {
		// Run a fresh scan.
		fmt.Println("Scanning package managers...")
		result, errs := scan.Run(ctx, scan.All())
		if len(errs) > 0 {
			for mgr, err := range errs {
				fmt.Fprintf(os.Stderr, "%s: %v\n", mgr, err)
			}
		}
		cache = &state.Cache{
			ScannedAt: time.Now(),
			Rows:      result.Rows,
		}
		_ = state.Save(*cache)
	}
	
	// Launch TUI.
	app := ui.NewApp(cache.Rows)
	if _, err := tea.NewProgram(app).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
