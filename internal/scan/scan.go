package scan

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
)

// Row is one outdated package from one manager.
type Row struct {
	Manager string
	Pkg     string
	Cur     string
	New     string
	Desc    string
	Home    string
}

// Scanner runs one manager's outdated query.
type Scanner interface {
	Name() string
	// Scan returns rows for this manager. It must be safe to call with a
	// cancelled context (it should return promptly).
	Scan(ctx context.Context) ([]Row, error)
}

func has(ctx context.Context, bin string) bool {
	return exec.CommandContext(ctx, "which", bin).Run() == nil
}

// ---- brew ----

type Brew struct{}

func (Brew) Name() string { return "brew" }

func (Brew) Scan(ctx context.Context) ([]Row, error) {
	if !has(ctx, "brew") {
		return nil, nil
	}
	out, err := exec.CommandContext(ctx, "brew", "outdated", "--json=v2").Output()
	if err != nil {
		return nil, err
	}
	var d struct {
		Formulae []struct {
			Name              string   `json:"name"`
			InstalledVersions []string `json:"installed_versions"`
			CurrentVersion    string   `json:"current_version"`
		} `json:"formulae"`
		Casks []struct {
			Name              string   `json:"name"`
			InstalledVersions []string `json:"installed_versions"`
			CurrentVersion    string   `json:"current_version"`
		} `json:"casks"`
	}
	if err := json.Unmarshal(out, &d); err != nil {
		return nil, err
	}
	var rows []Row
	for _, f := range d.Formulae {
		cur := "?"
		if n := len(f.InstalledVersions); n > 0 {
			cur = f.InstalledVersions[n-1]
		}
		rows = append(rows, Row{Manager: "brew", Pkg: f.Name, Cur: cur, New: f.CurrentVersion})
	}
	for _, c := range d.Casks {
		cur := "?"
		if n := len(c.InstalledVersions); n > 0 {
			cur = c.InstalledVersions[n-1]
		}
		rows = append(rows, Row{Manager: "brew", Pkg: c.Name, Cur: cur, New: c.CurrentVersion})
	}
	return rows, nil
}

// ---- npm ----

type Npm struct{}

func (Npm) Name() string { return "npm" }

func (Npm) Scan(ctx context.Context) ([]Row, error) {
	if !has(ctx, "npm") {
		return nil, nil
	}
	out, err := exec.CommandContext(ctx, "npm", "outdated", "-g", "--json").Output()
	if err != nil && !strings.Contains(string(out), "{") {
		// npm exits 1 when outdated; only error when no JSON at all.
		return nil, err
	}
	var d map[string]struct {
		Current string `json:"current"`
		Latest  string `json:"latest"`
	}
	if err := json.Unmarshal(out, &d); err != nil {
		return nil, err
	}
	var rows []Row
	for name, v := range d {
		rows = append(rows, Row{Manager: "npm", Pkg: name, Cur: v.Current, New: v.Latest})
	}
	return rows, nil
}

// ---- pnpm ----

type Pnpm struct{}

func (Pnpm) Name() string { return "pnpm" }

func (Pnpm) Scan(ctx context.Context) ([]Row, error) {
	if !has(ctx, "pnpm") {
		return nil, nil
	}
	out, err := exec.CommandContext(ctx, "pnpm", "outdated", "-g").Output()
	if err != nil && len(out) == 0 {
		return nil, err
	}
	var rows []Row
	for _, line := range strings.Split(string(out), "\n") {
		cols := strings.Fields(line)
		if len(cols) < 3 {
			continue
		}
		rows = append(rows, Row{Manager: "pnpm", Pkg: cols[0], Cur: cols[1], New: cols[2]})
	}
	return rows, nil
}

// ---- uv ----

type Uv struct{}

func (Uv) Name() string { return "uv" }

func (Uv) Scan(ctx context.Context) ([]Row, error) {
	if !has(ctx, "uv") {
		return nil, nil
	}
	out, err := exec.CommandContext(ctx, "uv", "tool", "list").Output()
	if err != nil {
		return nil, err
	}
	var rows []Row
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.Contains(fields[1], "v") {
			continue
		}
		name := fields[0]
		cur := strings.TrimPrefix(fields[1], "v")
		// Ask PyPI for the latest version and description.
		latest, desc, home := pypiInfo(ctx, name)
		if latest == "" || latest == cur {
			continue
		}
		rows = append(rows, Row{Manager: "uv", Pkg: name, Cur: cur, New: latest, Desc: desc, Home: home})
	}
	return rows, nil
}
