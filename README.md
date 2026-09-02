# qterm-updater

Daily package update nag for your terminal — zsh only, zero dependencies
beyond what macOS already ships (zsh, python3) plus your package managers.

On the **first shell of the day** (Ghostty, Terminal.app, iTerm — any
terminal running zsh), it scans these managers **in the background**:

- **brew** — formulae + casks
- **npm** — global packages
- **pnpm** — global packages
- **uv** — installed tools (`uv tool`)

…then shows a notice listing every outdated package with its version
delta, description, and homepage:

```
⟳ qterm-updater  ·  12 update(s) available

  brew
  #   PACKAGE                VERSION                  DESCRIPTION
  1   ripgrep                14.1.0 → 14.2.0          Fast regex search tool
  2   ghostty                1.1.0   → 1.2.0          Terminal emulator
  3   …

  npm
  #   PACKAGE                VERSION                  DESCRIPTION
  4   claude                 2.1.240 → 2.1.241        CLI for Claude

  [Y] update now   [N] Skip   [Esc] Cancel
```

## Keys (single keypress, no Enter)

| Key | Action |
|-----|--------|
| `Y` (or Enter) | Update everything now — runs in background, shell stays usable |
| `N` | Silence until tomorrow |
| `Esc` | Hide for this shell only — the notice comes back next time you open a terminal |

Updates run `brew upgrade && brew upgrade --cask`, `npm update -g`,
`pnpm update -g --latest`, and `uv tool upgrade --all` (only for the
managers found on your machine).

## Manual use

Pressed `N` or `Esc` and changed your mind? Call it directly — the same
notice, on demand:

```
qterm-updater           # fresh scan, then the notice with [Y]/[N]/[Esc]
qterm-updater list      # show the last scan result (instant, no scan)
qterm-updater update    # update everything now, no prompt
qterm-updater reset     # clear skip + cache; nag returns next shell
qterm-updater --help
```

`qterm-updater` (no args) scans in the foreground, so it takes as long as
your managers do — `list` is the instant one.

## Install

From a clone:

```
git clone https://github.com/qintmb/qterm-updater.git
cd qterm-updater
zsh install.sh
```

Or the one-liner (once pushed):

```
curl -fsSL https://raw.githubusercontent.com/qintmb/qterm-updater/main/install.sh | zsh
```

The installer copies `qterm-updater.zsh` to
`~/.local/share/qterm-updater/` and appends a guarded `source` block to
your `~/.zshrc`. Re-running it is safe (idempotent).

## Uninstall

```
rm -rf ~/.local/share/qterm-updater ~/.local/state/qterm-updater
# remove the "# qterm-updater BEGIN … END" block from ~/.zshrc
```

## Configure

Edit the `QTERM_MANAGERS` array at the top of
`~/.local/share/qterm-updater/qterm-updater.zsh` — remove a name to stop
scanning that manager. State (scan cache, skip marker) lives under
`${XDG_STATE_HOME:-~/.local/state}/qterm-updater/`; delete it to force a
fresh scan.

## How it works

- First interactive shell of the day spawns a background scan (mkdir
  lock — only one shell scans at a time).
- Scan results are cached; every later shell that day reads the cache,
  so nothing is re-fetched.
- The nag hooks zsh's `precmd`: it fires once the prompt first renders,
  never blocking shell startup.
- Non-interactive shells, CI, and `TERM=dumb` never trigger it.
