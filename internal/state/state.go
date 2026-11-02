package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/qintmb/qterm-updater/internal/scan"
)

// Dir returns the state directory, creating it if needed.
func Dir() string {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		base = filepath.Join(os.Getenv("HOME"), ".local", "state")
	}
	dir := filepath.Join(base, "qterm-updater")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// Cache is the on-disk scan result.
type Cache struct {
	ScannedAt time.Time      `json:"scanned_at"`
	Rows      []scan.Row     `json:"rows"`
	Errors    map[string]string `json:"errors,omitempty"`
}

// LoadCache reads the cached scan, or nil when there is none.
func LoadCache() *Cache {
	b, err := os.ReadFile(filepath.Join(Dir(), "cache.json"))
	if err != nil {
		return nil
	}
	var c Cache
	if json.Unmarshal(b, &c) != nil {
		return nil
	}
	return &c
}

// Save writes the cache atomically.
func Save(c Cache) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	dst := filepath.Join(Dir(), "cache.json")
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

// SkipUntil stores the "silence until" timestamp for the daily nag.
func SkipUntil() (time.Time, bool) {
	b, err := os.ReadFile(filepath.Join(Dir(), "skip_until"))
	if err != nil {
		return time.Time{}, false
	}
	var t time.Time
	if json.Unmarshal(b, &t) != nil {
		return time.Time{}, false
	}
	return t, true
}

// SetSkipUntil writes the "silence until" timestamp.
func SetSkipUntil(t time.Time) error {
	b, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(Dir(), "skip_until"), b, 0o644)
}

// ClearSkip removes the skip marker.
func ClearSkip() { _ = os.Remove(filepath.Join(Dir(), "skip_until")) }

// History is one finished job.
type HistoryEntry struct {
	StartedAt time.Time     `json:"started_at"`
	EndedAt   time.Time     `json:"ended_at"`
	Pkg       string        `json:"pkg"`
	Manager   string        `json:"manager"`
	From      string        `json:"from"`
	To        string        `json:"to"`
	OK        bool          `json:"ok"`
	Output    string        `json:"output,omitempty"`
}

// LoadHistory reads all finished jobs, newest first.
func LoadHistory() []HistoryEntry {
	b, err := os.ReadFile(filepath.Join(Dir(), "history.json"))
	if err != nil {
		return nil
	}
	var h []HistoryEntry
	if json.Unmarshal(b, &h) != nil {
		return nil
	}
	return h
}

// AppendHistory adds one entry at the front, capped at 500.
func AppendHistory(e HistoryEntry) {
	h := LoadHistory()
	h = append([]HistoryEntry{e}, h...)
	if len(h) > 500 {
		h = h[:500]
	}
	b, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(Dir(), "history.json"), b, 0o644)
}
