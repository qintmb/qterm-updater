#!/usr/bin/env zsh
# qterm-updater — daily package update nag for your terminal.
#
# On the first shell of the day, scans brew / npm -g / pnpm -g / uv tools
# in the background and shows a notice with per-package details.
#   [Y] = update now (no Enter needed)   [N] = skip today   [Esc] = cancel (shown again next shell)
#
# Install: run install.sh, or add `source ~/.local/share/qterm-updater/qterm-updater.zsh`
# at the end of your .zshrc.

# ---- guards -----------------------------------------------------------------
# Skip in non-interactive, CI, dumb terminal, or when re-sourced in the same shell.
if [[ ! -o interactive ]] || [[ -n "$CI" || "$TERM" == "dumb" || ! -t 1 ]]; then
  return 0
fi
if (( ${+functions[qterm_updater_nag]} )); then
  return 0
fi

# ---- config -----------------------------------------------------------------
QTERM_DIR="${XDG_STATE_HOME:-$HOME/.local/state}/qterm-updater"
QTERM_RESULT="$QTERM_DIR/result"        # scan output (tsv: manager pkg cur new desc home)
QTERM_LAST="$QTERM_DIR/last_check"      # epoch seconds of last successful scan
QTERM_SKIP="$QTERM_DIR/skip_today"      # epoch seconds until which [N] silences the nag
mkdir -p "$QTERM_DIR" 2>/dev/null

# Managers to scan. Comment out any line to skip that manager.
typeset -ga QTERM_MANAGERS=(brew npm pnpm uv)

# ---- helpers ----------------------------------------------------------------
_qterm_now() { date +%s }

_qterm_key_reader() {
  # Read a single keypress (Y/N/Esc/Enter) without waiting for Enter.
  local key
  if read -k 1 -s key 2>/dev/null; then
    case "$key" in
      $'\e')               print esc ;;
      $'\r'|$'\n'|'y'|'Y') print yes ;;
      'n'|'N'|'q'|'Q')     print no ;;
      *)                   print other ;;
    esac
  else
    print no
  fi
}

# ---- scan functions ---------------------------------------------------------
# Each writes tsv rows (manager<TAB>pkg<TAB>cur<TAB>new<TAB>desc<TAB>home) to
# the file referenced by $_QTERM_RESULT_TMP. Failed/unavailable managers are
# silent — `command -v` guards ensure we never error in a user's shell.

_qterm_scan_brew() {
  if ! command -v brew >/dev/null 2>&1; then return 0; fi
  local tmpjson="$QTERM_DIR/brew.json" namest="$QTERM_DIR/brew-names.tsv"
  brew outdated --json=v2 > "$tmpjson" 2>/dev/null || return 0
  # namest: name<TAB>current<TAB>new
  /usr/bin/python3 - "$tmpjson" > "$namest" <<'PY'
import json, sys
d = json.load(open(sys.argv[1]))
for sec in ("formulae", "casks"):
    for f in d.get(sec, []):
        cur = (f.get("installed_versions") or [f.get("installed_version") or "?"])[-1]
        print("\t".join([f["name"], cur, f.get("current_version", "?")]))
PY
  [[ -s "$namest" ]] || return 0
  # Batch-fetch desc/homepage via a single `brew info --json=v2` call
  local pkg_list
  pkg_list=$(cut -f1 "$namest" | paste -sd' ' -)
  brew info --json=v2 $pkg_list > "$tmpjson" 2>/dev/null || true
  /usr/bin/python3 - "$tmpjson" "$namest" <<'PY' >> "$_QTERM_RESULT_TMP"
import json, sys
info = {}
try:
    d = json.load(open(sys.argv[1]))
    for sec in ("formulae", "casks"):
        for f in d.get(sec, []):
            desc = (f.get("desc") or "").replace("\t", " ")
            home = f.get("homepage") or ""
            info[f["name"]] = (desc, home)
except Exception:
    pass
for line in open(sys.argv[2]):
    parts = line.rstrip("\n").split("\t")
    if len(parts) < 3: continue
    name, cur, new = parts[0], parts[1], parts[2]
    desc, home = info.get(name, ("", ""))
    print("\t".join(["brew", name, cur, new, desc, home]))
PY
}
_qterm_scan_npm() {
  if ! command -v npm >/dev/null 2>&1; then return 0; fi
  # Capture stdout even when npm exits 1 (outdated → exit 1)
  local out
  out=$(npm outdated -g --json 2>&1) || true
  [[ -z "$out" ]] && return 0
  # First non-`{` line might be a warning; strip leading junk
  local i j
  i=$(print -r -- "$out" | awk 'index($0,"{"){print NR; exit}')
  [[ -n "$i" ]] && out=$(print -r -- "$out" | tail -n +"$i")
  /usr/bin/python3 - "$out" <<'PY' >> "$_QTERM_RESULT_TMP.tmp" 2>/dev/null
import json, sys
raw = sys.argv[1]
i = raw.find("{")
j = raw.rfind("}")
if i < 0 or j < 0: sys.exit(0)
try: d = json.loads(raw[i:j+1])
except Exception: sys.exit(0)
for name, v in d.items():
    cur = v.get("current", "?")
    lat = v.get("latest", "?")
    print("\t".join([name, cur, lat]))
PY
  # Enrich with description/homepage via one `npm view` per pkg.
  # `npm view pkg description` prints just the string.
  local name cur lat desc home
  while IFS=$'\t' read -r name cur lat; do
    desc=$(npm view "$name" description 2>/dev/null)
    home=$(npm view "$name" homepage 2>/dev/null)
    desc="${desc//$'\t'/ }"
    printf 'npm\t%s\t%s\t%s\t%s\t%s\n' "$name" "$cur" "$lat" "$desc" "$home" >> "$_QTERM_RESULT_TMP"
  done < "$_QTERM_RESULT_TMP.tmp"
  rm -f "$_QTERM_RESULT_TMP.tmp"
}

_qterm_scan_pnpm() {
  if ! command -v pnpm >/dev/null 2>&1; then return 0; fi
  pnpm outdated -g 2>/dev/null | /usr/bin/python3 -c '
import sys, re
for line in sys.stdin:
    line = line.rstrip()
    if not line: continue
    if line.startswith(("┌", "├", "└", "─")) or line.startswith("Package"): continue
    cols = [c.strip() for c in line.split() if c.strip()]
    if len(cols) >= 3 and re.match(r"^@?[\w./-]+(?:/[\w.-]+)?$", cols[0]):
        print("%s\t%s\t%s\t\t" % (cols[0], cols[1], cols[2]))
' >> "$_QTERM_RESULT_TMP" 2>/dev/null
}

_qterm_scan_uv() {
  if ! command -v uv >/dev/null 2>&1; then return 0; fi
  local listing
  listing=$(uv tool list 2>/dev/null | awk '/^[a-zA-Z0-9_.-]+ v?[0-9]/')
  [[ -z "$listing" ]] && return 0
  /usr/bin/python3 - "$listing" <<'PY' >> "$_QTERM_RESULT_TMP" 2>/dev/null
import sys, json, urllib.request, re
ver_re = re.compile(r"^([a-zA-Z0-9_.-]+)\s+v?(\S+)")
tools = []
for line in sys.argv[1].splitlines():
    m = ver_re.match(line)
    if m: tools.append((m.group(1), m.group(2)))
for name, ver in tools:
    try:
        with urllib.request.urlopen(f"https://pypi.org/pypi/{name}/json", timeout=5) as r:
            d = json.load(r)
    except Exception:
        continue
    info = d.get("info", {})
    lat = info.get("version", "?")
    if lat == ver: continue
    desc = (info.get("summary") or "").replace("\t", " ")
    home = info.get("home_page") or ""
    print(f"{name}\t{ver}\t{lat}\t{desc}\t{home}")
PY
}

_qterm_scan() {
  # Background subshell. Writes to tmp, then atomically renames to result.
  _QTERM_RESULT_TMP="$QTERM_RESULT.scan.$$"
  : > "$_QTERM_RESULT_TMP"
  local mgr
  for mgr in "${QTERM_MANAGERS[@]}"; do
    "qterm_scan_${mgr}" 2>/dev/null
  done
  # Filter empty lines
  grep -v '^$' "$_QTERM_RESULT_TMP" 2>/dev/null > "$QTERM_RESULT" || mv -f "$_QTERM_RESULT_TMP" "$QTERM_RESULT"
  rm -f "$_QTERM_RESULT_TMP"
  print -r -- "$(date +%s)" > "$QTERM_LAST"
}

_qterm_needs_scan() {
  [[ ! -f "$QTERM_RESULT" ]] && return 0
  local last today
  last=$(date -r "$QTERM_LAST" +%Y-%m-%d 2>/dev/null) || last=0
  today=$(date +%Y-%m-%d)
  [[ "$last" != "$today" ]]
}

_qterm_is_skipped_today() {
  [[ -f "$QTERM_SKIP" ]] || return 1
  local until now
  until=$(cat "$QTERM_SKIP" 2>/dev/null) || return 1
  now=$(_qterm_now)
  (( now < until ))
}

# ---- display ----------------------------------------------------------------
_qterm_render_notice() {
  local total=0 mgr
  while IFS=$'\t' read -r mgr _ _ _ _ _; do
    (( total++ ))
  done < "$QTERM_RESULT"
  (( total > 0 )) || return 1

  printf '\n'
  printf '\033[1;36m⟳ qterm-updater\033[0m — \033[33m%d\033[0m package update(s) available\n' "$total"
  printf '\n'

  local cur_mgr=""
  local pkg cur new desc home
  while IFS=$'\t' read -r mgr pkg cur new desc home; do
    if [[ "$mgr" != "$cur_mgr" ]]; then
      cur_mgr="$mgr"
      printf '  \033[1m%s\033[0m\n' "$mgr"
    fi
    printf '    \033[36m%s\033[0m  %s → \033[33m%s\033[0m\n' "$pkg" "$cur" "$new"
    if [[ -n "$desc" ]]; then
      printf '        \033[2m%s\033[0m\n' "$desc"
    fi
    if [[ -n "$home" ]]; then
      printf '        \033[2;37m%s\033[0m\n' "$home"
    fi
  done < "$QTERM_RESULT"
  printf '\n'
  printf '  \033[1;32m[Y]\033[0m update now   \033[1;32m[N]\033[0m Skip   \033[1;32m[Esc]\033[0m Cancel\n'
  return 0
}

_qterm_apply_updates() {
  printf '\n\033[1;36m⟳ Updating…\033[0m\n'
  (
    local mgr
    for mgr in "${QTERM_MANAGERS[@]}"; do
      case "$mgr" in
        brew) [[ -x "$(command -v brew)" ]] && { brew update >/dev/null 2>&1 && brew upgrade >/dev/null 2>&1 && brew upgrade --cask >/dev/null 2>&1; } ;;
        npm)  [[ -x "$(command -v npm)"  ]] && npm update -g >/dev/null 2>&1 ;;
        pnpm) [[ -x "$(command -v pnpm)" ]] && pnpm update -g --latest >/dev/null 2>&1 ;;
        uv)   [[ -x "$(command -v uv)"   ]] && uv tool upgrade --all >/dev/null 2>&1 ;;
      esac
    done
    print -r -- " done. Re-scan tomorrow."
  ) &
  rm -f "$QTERM_SKIP"
}

# ---- nag (called by precmd) -------------------------------------------------
# Per-shell "already shown" flag — a shell variable, NOT a state file, so the
# notice reappears in every newly opened terminal after [Esc].
typeset -g _QTERM_SHOWN=0

qterm_updater_nag() {
  (( _QTERM_SHOWN )) && return 0
  _qterm_is_skipped_today && return 0
  [[ -f "$QTERM_RESULT" ]] || return 0
  _qterm_render_notice || return 0
  printf '\n'

  local key
  key=$(_qterm_key_reader)
  case "$key" in
    yes)
      _qterm_apply_updates
      _QTERM_SHOWN=1
      ;;
    no)
      # Skip until next local midnight
      local next_midnight
      if [[ "$(uname)" == "Darwin" ]]; then
        next_midnight=$(date -j -f "%H:%M:%S" "00:00:00" "+%s" 2>/dev/null)
        (( next_midnight += 86400 ))
      else
        next_midnight=$(date -d "tomorrow 00:00" +%s 2>/dev/null)
      fi
      [[ -n "$next_midnight" ]] && printf '%s\n' "$next_midnight" > "$QTERM_SKIP"
      _QTERM_SHOWN=1
      printf '  (skipped for today)\n'
      ;;
    esc|*)
      # Esc: hide in this shell only — the next shell nags again from cache
      _QTERM_SHOWN=1
      printf '  (cancelled — will ask again in the next shell)\n'
      ;;
  esac
  printf '\n'
}

# ---- boot -------------------------------------------------------------------
# 1) If we need to scan, do it in the background (non-blocking).
# 2) Wire the nag into precmd so the notice appears after the prompt renders.
if _qterm_needs_scan; then
  if mkdir "$QTERM_DIR/lock" 2>/dev/null; then
    ( _qterm_scan & ) &>/dev/null
    trap "rmdir '$QTERM_DIR/lock' 2>/dev/null" EXIT INT HUP
  fi
fi

# Hook into precmd (after Powerlevel10k / Starship finish rendering).
autoload -U add-zsh-hook 2>/dev/null
add-zsh-hook precmd qterm_updater_nag 2>/dev/null || precmd_functions+=(qterm_updater_nag)
