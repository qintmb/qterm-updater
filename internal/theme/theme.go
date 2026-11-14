// Package theme maps semantic UI roles to colors. Views use Styles only,
// never raw colors, so adding a palette touches one file.
package theme

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Palette holds the color slots the TUI uses.
type Palette struct {
	Mantle, Crust            lipgloss.Color
	Surface0, Surface1       lipgloss.Color
	Overlay0, Subtext0, Text lipgloss.Color
	Mauve, Blue, Teal, Green lipgloss.Color
	Yellow, Peach, Red, Pink lipgloss.Color
}

// Macchiato is Catppuccin Macchiato.
var Macchiato = Palette{
	Mantle: "#1e2030", Crust: "#181926",
	Surface0: "#363a4f", Surface1: "#494d64",
	Overlay0: "#6e738d", Subtext0: "#a5adcb", Text: "#cad3f5",
	Mauve: "#c6a0f6", Blue: "#8aadf4", Teal: "#8bd5ca", Green: "#a6da95",
	Yellow: "#eed49f", Peach: "#f5a97f", Red: "#ed8796", Pink: "#f5bde6",
}

// Icons are glyphs drawn before some words. With the zero Icons the words
// stand alone, which suits terminals without a Nerd Font.
type Icons struct {
	Formula, Cask, Pinned    string
	Started, Stopped, Failed string
}

// Nerd are Nerd Font (v3) glyphs.
var Nerd = Icons{
	Formula: "\uf0fc",
	Cask:    "\U000f003b",
	Pinned:  "\U000f0403",
	Started: "\U000f040a",
	Stopped: "\U000f04db",
	Failed:  "\U000f0028",
}

// With puts icon before word, or word alone when icon is empty.
func With(icon, word string) string {
	if icon == "" {
		return word
	}
	return icon + " " + word
}

// Styles are the semantic styles views render with.
type Styles struct {
	P Palette

	Text, Muted, Accent, Title lipgloss.Style
	OK, Warn, Err, Info        lipgloss.Style

	Pane, PaneFocused           lipgloss.Style
	PaneTitle, PaneTitleFocused lipgloss.Style

	TabActive                    lipgloss.Style
	ChipWarn, ChipErr, ChipStale lipgloss.Style
	StatusBar                    lipgloss.Style

	TableHeader, TableCell, TableSelected lipgloss.Style

	Icons Icons
}

// New builds the styles for p.
func New(p Palette) Styles {
	fg := func(c lipgloss.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c) }
	pane := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(p.Surface1).Padding(0, 1)
	return Styles{
		P:    p,
		Text: fg(p.Text), Muted: fg(p.Overlay0), Accent: fg(p.Mauve), Title: fg(p.Blue).Bold(true),
		OK: fg(p.Green), Warn: fg(p.Yellow), Err: fg(p.Red), Info: fg(p.Teal),

		Pane:             pane,
		PaneFocused:      pane.BorderForeground(p.Mauve),
		PaneTitle:        fg(p.Subtext0),
		PaneTitleFocused: fg(p.Mauve).Bold(true),

		TabActive: lipgloss.NewStyle().Foreground(p.Crust).Background(p.Mauve).Bold(true),
		ChipWarn:  chip(p.Yellow, p.Crust),
		ChipErr:   chip(p.Red, p.Crust),
		ChipStale: chip(p.Peach, p.Crust),
		StatusBar: lipgloss.NewStyle().Foreground(p.Subtext0).Background(p.Mantle),

		TableHeader: fg(p.Subtext0).Bold(true).Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).BorderForeground(p.Surface1),
		TableCell:     lipgloss.NewStyle().Padding(0, 1),
		TableSelected: lipgloss.NewStyle().Foreground(p.Text).Background(p.Surface0).Bold(true),
	}
}

// chip is a small solid badge.
func chip(bg, fg lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(fg).Background(bg).Bold(true).Padding(0, 1)
}

// Mode is the status-bar badge style for revert an editor-style mode name.
func (s Styles) Mode(mode string) lipgloss.Style {
	c := s.P.Mauve
	switch mode {
	case "COMMAND":
		c = s.P.Blue
	case "FILTER":
		c = s.P.Teal
	case "RUNNING":
		c = s.P.Peach
	case "INSERT":
		c = s.P.Green
	}
	return lipgloss.NewStyle().Foreground(s.P.Crust).Background(c).Bold(true).Padding(0, 1)
}

// Theme is a palette and the names the TUI knows it by.
type Theme struct {
	Name, Title string
	Palette     Palette
}

// Themes are the themes the picker offers, in its order; the first is the default.
var Themes = []Theme{
	{"catppuccin-macchiato", "Catppuccin Macchiato", Macchiato},
	{"catppuccin-mocha", "Catppuccin Mocha", Palette{
		Mantle: "#181825", Crust: "#11111b",
		Surface0: "#313244", Surface1: "#45475a",
		Overlay0: "#6c7086", Subtext0: "#a6adc8", Text: "#cdd6f4",
		Mauve: "#cba6f7", Blue: "#89b4fa", Teal: "#94e2d5", Green: "#a6e3a1",
		Yellow: "#f9e2af", Peach: "#fab387", Red: "#f38ba8", Pink: "#f5c2e7",
	}},
	{"catppuccin-frappe", "Catppuccin Frappe", Palette{
		Mantle: "#292c3c", Crust: "#232634",
		Surface0: "#414559", Surface1: "#51576d",
		Overlay0: "#737994", Subtext0: "#a5adce", Text: "#c6d0f5",
		Mauve: "#ca9ee6", Blue: "#8caaee", Teal: "#81c8be", Green: "#a6d189",
		Yellow: "#e5c890", Peach: "#ef9f76", Red: "#e78284", Pink: "#f4b8e4",
	}},
	{"dracula", "Dracula", Palette{
		Mantle: "#21222c", Crust: "#191a21",
		Surface0: "#44475a", Surface1: "#6272a4",
		Overlay0: "#6272a4", Subtext0: "#f8f8f2", Text: "#f8f8f2",
		Mauve: "#ff79c6", Blue: "#bd93f9", Teal: "#8be9fd", Green: "#50fa7b",
		Yellow: "#f1fa8c", Peach: "#ffb86c", Red: "#ff5555", Pink: "#ff79c6",
	}},
	{"nord", "Nord", Palette{
		Mantle: "#2e3440", Crust: "#2e3440",
		Surface0: "#3b4252", Surface1: "#434c5e",
		Overlay0: "#616e88", Subtext0: "#d8dee9", Text: "#eceff4",
		Mauve: "#b48ead", Blue: "#81a1c1", Teal: "#8fbcbb", Green: "#a3be8c",
		Yellow: "#ebcb8b", Peach: "#d08770", Red: "#bf616a", Pink: "#b48ead",
	}},
	{"gruvbox-dark", "Gruvbox Dark", Palette{
		Mantle: "#1d2021", Crust: "#1d2021",
		Surface0: "#3c3836", Surface1: "#504945",
		Overlay0: "#928374", Subtext0: "#d5c4a1", Text: "#ebdbb2",
		Mauve: "#d3869b", Blue: "#83a598", Teal: "#8ec07c", Green: "#b8bb26",
		Yellow: "#fabd2f", Peach: "#fe8019", Red: "#fb4934", Pink: "#d3869b",
	}},
	{"tokyo-night", "Tokyo Night", Palette{
		Mantle: "#16161e", Crust: "#15161e",
		Surface0: "#292e42", Surface1: "#414868",
		Overlay0: "#565f89", Subtext0: "#a9b1d6", Text: "#c0caf5",
		Mauve: "#bb9af7", Blue: "#7aa2f7", Teal: "#73daca", Green: "#9ece6a",
		Yellow: "#e0af68", Peach: "#ff9e64", Red: "#f7768e", Pink: "#bb9af7",
	}},
	{"one-dark", "One Dark", Palette{
		Mantle: "#21252b", Crust: "#181a1f",
		Surface0: "#3e4451", Surface1: "#4b5263",
		Overlay0: "#5c6370", Subtext0: "#9da5b4", Text: "#abb2bf",
		Mauve: "#c678dd", Blue: "#61afef", Teal: "#56b6c2", Green: "#98c379",
		Yellow: "#e5c07b", Peach: "#d19a66", Red: "#e06c75", Pink: "#c678dd",
	}},
	{"rose-pine", "Rose Pine", Palette{
		Mantle: "#191724", Crust: "#191724",
		Surface0: "#403d52", Surface1: "#524f67",
		Overlay0: "#6e6a86", Subtext0: "#908caa", Text: "#e0def4",
		Mauve: "#c4a7e7", Blue: "#9ccfd8", Teal: "#ebbcba", Green: "#31748f",
		Yellow: "#f6c177", Peach: "#f6c177", Red: "#eb6f92", Pink: "#ebbcba",
	}},
	{"kanagawa", "Kanagawa", Palette{
		Mantle: "#16161d", Crust: "#16161d",
		Surface0: "#2d4f67", Surface1: "#54546d",
		Overlay0: "#727169", Subtext0: "#c8c093", Text: "#dcd7ba",
		Mauve: "#957fb8", Blue: "#7e9cd8", Teal: "#7aa89f", Green: "#98bb6c",
		Yellow: "#e6c384", Peach: "#ffa066", Red: "#e46876", Pink: "#d27e99",
	}},
	{"everforest-dark", "Everforest Dark", Palette{
		Mantle: "#232a2e", Crust: "#232a2e",
		Surface0: "#3d484d", Surface1: "#4f585e",
		Overlay0: "#859289", Subtext0: "#9da9a0", Text: "#d3c6aa",
		Mauve: "#d699b6", Blue: "#7fbbb3", Teal: "#83c092", Green: "#a7c080",
		Yellow: "#dbbc7f", Peach: "#e69875", Red: "#e67e80", Pink: "#d699b6",
	}},
	{"solarized-dark", "Solarized Dark", Palette{
		Mantle: "#002b36", Crust: "#002b36",
		Surface0: "#073642", Surface1: "#586e75",
		Overlay0: "#586e75", Subtext0: "#839496", Text: "#93a1a1",
		Mauve: "#6c71c4", Blue: "#268bd2", Teal: "#2aa198", Green: "#859900",
		Yellow: "#b58900", Peach: "#cb4b16", Red: "#dc322f", Pink: "#d33682",
	}},
}

// Lookup returns the theme called name, or the default when there is none.
func Lookup(name string) Theme {
	if i, ok := Index(name); ok {
		return Themes[i]
	}
	return Themes[0]
}

// Index returns where the theme called name is in Themes.
func Index(name string) (int, bool) {
	i := slices.IndexFunc(Themes, func(t Theme) bool { return t.Name == name })
	return i, i >= 0
}

// LoadTheme returns the name of the theme saved in dir, or the default's.
func LoadTheme(dir string) string {
	if dir == "" {
		return Themes[0].Name
	}
	b, err := os.ReadFile(filepath.Join(dir, "theme"))
	if err != nil {
		return Themes[0].Name
	}
	return Lookup(strings.TrimSpace(string(b))).Name
}

// SaveTheme keeps name in dir for the next run.
func SaveTheme(dir, name string) error {
	return writeAtomic(dir, "theme", name+"\n")
}

const iconsFile = "icons"
const iconsOn = "on\n"

// LoadIcons reports whether Nerd Font icons were saved as on in dir.
func LoadIcons(dir string) bool {
	if dir == "" {
		return false
	}
	b, err := os.ReadFile(filepath.Join(dir, iconsFile))
	return err == nil && strings.TrimSpace(string(b)) == "on"
}

// SaveIcons keeps the icons switch in dir for the next run.
func SaveIcons(dir string, on bool) error {
	v := "off\n"
	if on {
		v = iconsOn
	}
	return writeAtomic(dir, iconsFile, v)
}

// writeAtomic writes a small file and renames it into place, so a symlink
// planted where the file goes is replaced rather than written through.
func writeAtomic(dir, name, body string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, "."+name+"-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(body); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), filepath.Join(dir, name))
}
