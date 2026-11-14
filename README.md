# qterm-updater

Daily package update nag for your terminal — now with a full TUI styled
after [brewery](https://github.com/dillo/brewery): three panes (sidebar,
list, details), a status bar with a mode badge, a `:` command line, a
theme picker, and a `?` help overlay.

A Go rewrite with Bubble Tea, inspired by brewery's look and feel.

## Features

- **Brewery-style TUI** — Sidebar pane with section headings and per-entry
  counts, list pane with a version table, details pane, all in rounded
  bordered panes that light up when focused.
- **Multi-manager** — brew, npm, pnpm, uv, each a sidebar entry with its
  own count.
- **Theme picker** — `t` previews 11 themes live; `enter` keeps it, `esc`
  puts the old one back. The choice is saved in the state directory.
- **Nerd Font icons** — `:icons on` draws glyphs before package names;
  `:icons off` (default) keeps the words alone.
- **Command line** — `:` runs a command with completion (`tab`) and
  session history (`↑`/`↓`).
- **Help overlay** — `?` shows the keys that work where you are.
- **Marking** — `space` marks packages, `U` upgrades them in order, `A`
  upgrades everything in the current view.
- **Jobs history** — Every update is recorded to disk and shown in History.

## Install

From source:

```sh
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
| `j/k` or arrows | Move in the sidebar or list |
| `h/l` or `tab` | Sidebar, list, details |
| `1-8` | Jump to a sidebar entry |
| `[` `]` | Previous or next entry |
| `gg` `G` | First, last row |
| `ctrl+d` `ctrl+u` | Half a page |
| `/` | Filter the list; `esc` clears |
| `space` | Mark a package |
| `u` | Upgrade the selected package |
| `U` | Upgrade marked packages |
| `A` | Upgrade all packages in the current view |
| `r` | Re-scan every manager |
| `t` | Change the theme |
| `:` | Run a command |
| `?` | Help |
| `q` / `ctrl+c` | Quit |

### Command line

| Command | Action |
|---------|--------|
| `refresh` / `update` | Re-scan every manager |
| `upgrade-all` | Upgrade everything in the current view |
| `theme [name]` | Open the picker, or switch to a named theme |
| `icons on|off` | Toggle Nerd Font icons (persisted) |
| `history` | Jump to History |
| `help` | Open the help overlay |
| `quit` / `q` | Quit |

## Settings

Settings live in the state directory (`$XDG_STATE_HOME/qterm-updater`, or
`~/.local/state/qterm-updater`): `theme` holds the chosen theme name and
`icons` the icon switch. They are written atomically, so a symlink planted
where they go is replaced, not written through.

## Sidebar

- **Home** — Health summary: counts and bars per manager, recent jobs.
- **All** — Every outdated package.
- **Marked** — Packages you marked with `space`.
- **brew / npm / pnpm / uv** — One manager's packages.
- **History** — All updates ever run, newest first.

## Architecture

```
internal/
├── scan/      — Package manager scanners (brew, npm, pnpm, uv)
├── state/     — Cache, history and settings persistence
├── jobs/      — Background upgrade queue
├── theme/     — Palettes, semantic styles, settings files
└── ui/        — Bubble Tea TUI (panes, sidebar, command line, pickers)
```

## License

MIT
