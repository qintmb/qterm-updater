# qterm-updater

Daily package update nag for your terminal — now with a full TUI.

A Go rewrite with Bubble Tea v2, inspired by [brewery](https://github.com/dillo/brewery).

## Features

- **Sidebar navigation** — Jump keys 1-4, j/k or arrow keys
- **Interactive table** — All outdated packages with version delta
- **Details panel** — Description and homepage for each package
- **Jobs history** — Track all updates ever run
- **Multi-manager** — brew, npm, pnpm, uv

## Install

From source:

```
git clone https://github.com/qintmb/qterm-updater.git
cd qterm-updater
go build -o ~/.local/bin/qterm-updater ./cmd/qterm-updater
```

## Usage

Run the TUI:

```
qterm-updater
```

### Keys

| Key | Action |
|-----|--------|
| `j/k` or arrows | Navigate sidebar |
| `1-4` | Jump to section |
| `tab` | Toggle sidebar/table focus |
| `q` or `ctrl+c` | Quit |

### Sidebar

- **Home** — Health summary with counts per manager
- **All** — Every outdated package
- **Outdated** — Same as All (filterable in future)
- **History** — All updates ever run

## Architecture

```
internal/
├── scan/      — Package manager scanners (brew, npm, pnpm, uv)
├── state/     — Cache and history persistence
├── jobs/      — Background upgrade queue
├── theme/     — Lipgloss styles
└── ui/        — Bubble Tea TUI (sidebar, table, home, history)
```

## vs zsh version

The original zsh version was a simple nag script. This Go version adds:

- Full interactive TUI
- Persistent history
- Parallel scanning
- Better error handling
- Theme support

## License

MIT
