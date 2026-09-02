#!/usr/bin/env zsh
# install.sh — install qterm-updater from the cloned repo.
# Usage: zsh install.sh

set -e

REPO_DIR="${0:A:h}"   # directory holding this script
SRC="$REPO_DIR/qterm-updater.zsh"
DST_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/qterm-updater"
DST="$DST_DIR/qterm-updater.zsh"
ZSHRC="${ZDOTDIR:-$HOME}/.zshrc"

if [[ ! -f "$SRC" ]]; then
  print -r -- "error: $SRC not found" >&2
  exit 1
fi

# 1) Copy script
mkdir -p "$DST_DIR"
cp -f "$SRC" "$DST"

# 2) Ensure source line in .zshrc (idempotent — guarded by markers)
mkdir -p "$(dirname "$ZSHRC")"
touch "$ZSHRC"
if ! grep -q '# qterm-updater BEGIN' "$ZSHRC"; then
  cat >> "$ZSHRC" <<'EOF'

# qterm-updater BEGIN
if [[ -r "$HOME/.local/share/qterm-updater/qterm-updater.zsh" ]]; then
  source "$HOME/.local/share/qterm-updater/qterm-updater.zsh"
fi
# qterm-updater END
EOF
fi

print -r -- ""
print -r -- "  ✓ installed to $DST"
print -r -- "  ✓ sourced from $ZSHRC"
print -r -- ""
print -r -- "Restart your terminal (or: source $ZSHRC) to start seeing"
print -r -- "the daily update nag. Force a check anytime with: rm -f"
print -r -- "${XDG_STATE_HOME:-$HOME/.local/state}/qterm-updater/last_check"
