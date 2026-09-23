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
# Skip in CI / dumb terminal / when re-sourced. We still define the manual CLI
# in those cases so `qterm-updater list` / `update` work in scripts.
if [[ -n "$CI" || "$TERM" == "dumb" ]] || (( ${+functions[qterm-updater]} )); then
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
# the file referenced by $_QTERM_RESULT_TMP. A failed manager leaves at most
# the rows it had emitted — one command can never blank out the whole table.

_qterm_scan_brew() {
  if ! command -v brew >/dev/null 2>&1; then return 0; fi
  local stage="$_QTERM_RESULT_TMP.brew"
  : > "$stage"
  local tmpout="$QTERM_DIR/brew.outdated.json" tmpinfo="$QTERM_DIR/brew.info.json"
  brew outdated --json=v2 > "$tmpout" 2>/dev/null || { rm -f "$tmpout"; return 0; }
  # Fetch desc/homepage for the ~1000 OUTDATED packages in one call — the full
  # --installed set would hang on Intel Macs (251+ formulae, several minutes).
  brew info --json=v2 $(< <(python3 -c 'import json,sys
d=json.load(open(sys.argv[1]))
print(" ".join(f["name"] for sec in ("formulae","casks") for f in d.get(sec,[])))' "$tmpout")) > "$tmpinfo" 2>/dev/null || true
  /usr/bin/python3 - "$tmpout" "$tmpinfo" <<'PY' >> "$stage"
import json, sys
try: outdated = json.load(open(sys.argv[1]))
except Exception: sys.exit(0)
info = {}
try:
    d = json.load(open(sys.argv[2]))
    for sec in ("formulae", "casks"):
        for f in d.get(sec, []):
            desc = (f.get("desc") or "").replace("\t", " ")
            home = f.get("homepage") or ""
            info[f["name"]] = (desc, home)
except Exception:
    pass
rows = []
for sec in ("formulae", "casks"):
    for f in outdated.get(sec, []):
        cur = (f.get("installed_versions") or [f.get("installed_version") or "?"])[-1]
        name = f["name"]
        new = f.get("current_version", "?")
        desc, home = info.get(name, ("", ""))
        rows.append("\t".join(["brew", name, cur, new, desc, home]))
print("\n".join(rows))
PY
  rm -f "$tmpout" "$tmpinfo"
  cat "$stage" >> "$_QTERM_RESULT_TMP"
  rm -f "$stage"
}
_qterm_scan_npm() {
  if ! command -v npm >/dev/null 2>&1; then return 0; fi
  local stage="$_QTERM_RESULT_TMP.npm"
  : > "$stage"          # truncate — otherwise a stale file doubles every row
  # Capture stdout even when npm exits 1 (outdated → exit 1)
  local out
  out=$(npm outdated -g --json 2>&1) || true
  [[ -z "$out" ]] && return 0
  # First non-`{` line might be a warning; strip leading junk
  local i
  i=$(print -r -- "$out" | awk 'index($0,"{"){print NR; exit}')
  [[ -n "$i" ]] && out=$(print -r -- "$out" | tail -n +"$i")
  /usr/bin/python3 - "$out" <<'PY' >> "$stage" 2>/dev/null
import json, sys
raw = sys.argv[1]
i = raw.find("{")
j = raw.rfind("}")
if i < 0 or j < 0: sys.exit(0)
try: d = json.loads(raw[i:j+1])
except Exception: sys.exit(0)
for name, v in d.items():
    print("\t".join([name, v.get("current", "?"), v.get("latest", "?")]))
PY
  # Enrich with description/homepage via one `npm view` per pkg.
  local name cur lat desc home
  while IFS=$'\t' read -r name cur lat; do
    [[ -z "$name" ]] && continue
    desc=$(npm view "$name" description 2>/dev/null)
    home=$(npm view "$name" homepage 2>/dev/null)
    desc="${desc//$'\t'/ }"
    printf 'npm\t%s\t%s\t%s\t%s\t%s\n' "$name" "$cur" "$lat" "$desc" "$home" >> "$_QTERM_RESULT_TMP"
  done < "$stage"
  rm -f "$stage"
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
        print("pnpm\t%s\t%s\t%s\t\t" % (cols[0], cols[1], cols[2]))
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
    print("\t".join(["uv", name, ver, lat, desc, home]))
PY
}

_qterm_scan() {
  # Writes to a per-PID scratch file, then atomically swaps it in as the
  # cached result. Safe in the foreground (manual `qterm-updater`) and from
  # the boot-time background job: a scan that finishes while a NEWER scan
  # has already completed abandons its result instead of clobbering it.
  local started
  started=$(date +%s)
  _QTERM_RESULT_TMP="$QTERM_RESULT.scan.$$"
  : > "$_QTERM_RESULT_TMP"
  local mgr
  for mgr in "${QTERM_MANAGERS[@]}"; do
    "_qterm_scan_${mgr}" 2>/dev/null
  done
  grep -v '^[[:space:]]*$' "$_QTERM_RESULT_TMP" > "${_QTERM_RESULT_TMP}.f" 2>/dev/null
  mv -f "${_QTERM_RESULT_TMP}.f" "$_QTERM_RESULT_TMP"
  if [[ -f "$QTERM_LAST" ]] && (( $(cat "$QTERM_LAST" 2>/dev/null || echo 0) > started )); then
    rm -f "$_QTERM_RESULT_TMP"
    return 0
  fi
  mv -f "$_QTERM_RESULT_TMP" "$QTERM_RESULT"
  date +%s > "$QTERM_LAST"
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
# Truncate to a display width, appending "…" when cut.
_qterm_trunc() {
  local s="$1" w="$2"
  (( ${#s} <= w )) && { printf '%s' "$s"; return; }
  printf '%s…' "${s[1,$((w-1))]}"
}

_qterm_render_notice() {
  local total=0
  total=$(grep -c . "$QTERM_RESULT" 2>/dev/null) || total=0
  (( total > 0 )) || return 1

  # Column widths, adapted to the terminal but clamped to sane bounds.
  local cols=${COLUMNS:-100}
  (( cols < 60 )) && cols=60
  local w_pkg=24 w_ver=11 w_desc
  # 2 indent + 2 num + 2 gap + pkg + 2 gap + (ver→ver) + 2 gap + desc
  w_desc=$(( cols - 2 - 2 - 2 - w_pkg - 2 - (w_ver * 2 + 3) - 2 ))
  (( w_desc < 16 )) && w_desc=16
  (( w_desc > 60 )) && w_desc=60

  printf '\n\033[1;36m⟳ qterm-updater\033[0m  \033[2m·\033[0m  \033[33m%d\033[0m update(s) available\n\n' "$total"

  local n=0 cur_mgr="" mgr pkg cur new desc home
  while IFS=$'\t' read -r mgr pkg cur new desc home; do
    [[ -z "$mgr" ]] && continue
    if [[ "$mgr" != "$cur_mgr" ]]; then
      cur_mgr="$mgr"
      # Section header with a rule that spans the table.
      printf '\n  \033[1;35m%s\033[0m\n' "$mgr"
      printf '  \033[2m%-2s  %-*s  %-*s  %s\033[0m\n' \
        '#' "$w_pkg" 'PACKAGE' "$(( w_ver * 2 + 3 ))" 'VERSION' 'DESCRIPTION'
    fi
    (( n++ ))
    printf '  \033[2m%-2s\033[0m  \033[36m%-*s\033[0m  \033[2m%*s → %-*s\033[0m  %s\n' \
      "$n" \
      "$w_pkg" "$(_qterm_trunc "$pkg" $w_pkg)" \
      "$w_ver" "$(_qterm_trunc "$cur" $w_ver)" \
      "$w_ver" "$(_qterm_trunc "$new" $w_ver)" \
      "$(_qterm_trunc "$desc" $w_desc)"
  done < "$QTERM_RESULT"

  printf '\n'
  # Footer key hint only appears when the caller is about to prompt.
  if [[ "${_QTERM_PROMPT:-0}" == "1" ]]; then
    printf '  \033[1;32m[Y]\033[0m update now   \033[1;32m[N]\033[0m Skip   \033[1;32m[Esc]\033[0m Cancel\n'
  fi
  return 0
}

_qterm_fmtsecs() {
  # 3754 -> "1h02m", 125 -> "2m05s", 7 -> "7s"
  local s=$1
  (( s >= 3600 )) && { printf '%dh%02dm' $(( s / 3600 )) $(( (s % 3600) / 60 )); return; }
  (( s >= 60 ))   && { printf '%dm%02ds' $(( s / 60 )) $(( s % 60 )); return; }
  printf '%ds' "$s"
}

_qterm_apply_updates() {
  # Update one package at a time so progress is visible: a live spinner line
  # per package (name, version bump, elapsed), then a ✓/✗ line. Raw output
  # goes to a log file; on failure the last lines are shown.
  local log="$QTERM_DIR/update.log" n=0 ok=0 fail=0 start st
  : > "$log"
  local -a frames=('⠋' '⠙' '⠹' '⠸' '⠼' '⠴' '⠦' '⠧' '⠇' '⠏')
  local -i fi=0

  if [[ ! -s "$QTERM_RESULT" ]]; then
    # No scan data — fall back to whole-manager upgrades.
    printf '\n\033[1;36m⟳ Updating %s…\033[0m\n' "${QTERM_MANAGERS[*]}"
    local mgr
    for mgr in "${QTERM_MANAGERS[@]}"; do
      command -v "$mgr" >/dev/null 2>&1 || continue
      printf '\n\033[1;35m── %s ──\033[0m\n' "$mgr"
      start=$SECONDS
      case "$mgr" in
        brew) brew update && brew upgrade && brew upgrade --cask ;;
        npm)  npm update -g ;;
        pnpm) pnpm update -g --latest ;;
        uv)   uv tool upgrade --all ;;
      esac
      printf '\033[2m%s done in %s.\033[0m\n' "$mgr" "$(_qterm_fmtsecs $(( SECONDS - start )))"
    done
    printf '\n\033[1;32m✓ all updates finished.\033[0m Re-scanning to verify…\n\n'
    rm -f "$QTERM_SKIP"
    _qterm_scan
    return
  fi

  local total
  total=$(grep -c . "$QTERM_RESULT" 2>/dev/null) || total=0
  printf '\n\033[1;36m⟳ Updating %d package(s), one by one — full log: \033[2m%s\033[0m\n\n' "$total" "$log"

  local mgr cur_mgr="" pkg cur new desc home
  while IFS=$'\t' read -r mgr pkg cur new desc home; do
    [[ -z "$mgr" ]] && continue
    if [[ "$mgr" != "$cur_mgr" ]]; then
      [[ -n "$cur_mgr" ]] && printf '\n'
      cur_mgr="$mgr"
      printf '\033[1;35m── %s ──\033[0m\n' "$mgr"
      [[ "$mgr" == brew ]] && { printf '  \033[2mbrew update…\033[0m'; brew update >> "$log" 2>&1; printf '\r\033[2K  \033[2mbrew update ok\033[0m\n'; }
    fi
    command -v "$mgr" >/dev/null 2>&1 || continue
    (( n++ ))
    local -a cmd
    case "$mgr" in
      brew) cmd=(brew upgrade "$pkg") ;;
      npm)  cmd=(npm install -g "${pkg}@latest") ;;
      pnpm) cmd=(pnpm update -g --latest "$pkg") ;;
      uv)   cmd=(uv tool upgrade "$pkg") ;;
    esac
    start=$SECONDS
    "$cmd[@]" >> "$log" 2>&1 < /dev/null &
    local pid=$!
    if [[ -t 1 ]]; then
      while kill -0 "$pid" 2>/dev/null; do
        printf '\r  [%2d/%2d] \033[36m%s\033[0m %-24s %s → %s \033[2m%s\033[0m  ' \
          "$n" "$total" "${frames[$(( fi % 10 + 1 ))]}" \
          "$(_qterm_trunc "$pkg" 24)" "$(_qterm_trunc "$cur" 11)" "$(_qterm_trunc "$new" 11)" \
          "$(_qterm_fmtsecs $(( SECONDS - start )))"
        (( fi++ ))
        sleep 0.2
      done
    else
      printf '  [%2d/%2d] %s %s → %s…\n' "$n" "$total" "$pkg" "$cur" "$new"
    fi
    wait "$pid"; st=$?
    if (( st == 0 )); then
      (( ok++ ))
      printf '\r\033[2K  [%2d/%2d] \033[32m✓\033[0m %-24s %s → %s \033[2m(%s)\033[0m\n' \
        "$n" "$total" "$(_qterm_trunc "$pkg" 24)" "$(_qterm_trunc "$cur" 11)" "$(_qterm_trunc "$new" 11)" \
        "$(_qterm_fmtsecs $(( SECONDS - start )))"
    else
      (( fail++ ))
      printf '\r\033[2K  [%2d/%2d] \033[31m✗\033[0m %-24s %s → %s \033[2m(%s)\033[0m\n' \
        "$n" "$total" "$(_qterm_trunc "$pkg" 24)" "$(_qterm_trunc "$cur" 11)" "$(_qterm_trunc "$new" 11)" \
        "$(_qterm_fmtsecs $(( SECONDS - start )))"
      tail -n 4 "$log" | sed 's/^/        /'
    fi
  done < "$QTERM_RESULT"

  printf '\n\033[1;32m✓ %d updated\033[0m' "$ok"
  (( fail > 0 )) && printf '\033[1;31m, %d failed\033[0m' "$fail"
  printf ' \033[2m— log: %s\033[0m\nRe-scanning to verify…\n\n' "$log"
  rm -f "$QTERM_SKIP"
  _qterm_scan
}

# ---- nag (called by precmd) -------------------------------------------------
# Per-shell "already shown" flag — a shell variable, NOT a state file, so the
# notice reappears in every newly opened terminal after [Esc].
typeset -g _QTERM_SHOWN=0

qterm_updater_nag() {
  (( _QTERM_SHOWN )) && return 0
  _qterm_is_skipped_today && return 0
  [[ -f "$QTERM_RESULT" ]] || return 0
  _QTERM_PROMPT=1 _qterm_render_notice || return 0
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

_qterm_bg_scan() {
  # Boot-time background scan, SIGHUP-proof (&! = bg + disown, so closing the
  # terminal mid-scan no longer kills it) and guarded by a stale-tolerant lock:
  # a lock left behind by a killed shell used to block every future scan.
  if ! mkdir "$QTERM_DIR/lock" 2>/dev/null; then
    local mtime
    mtime=$(stat -f %m "$QTERM_DIR/lock" 2>/dev/null) || \
      mtime=$(stat -c %Y "$QTERM_DIR/lock" 2>/dev/null) || return 0
    (( $(date +%s) - mtime < 1800 )) && return 0
    rm -rf "$QTERM_DIR/lock"
    mkdir "$QTERM_DIR/lock" 2>/dev/null || return 0
  fi
  # The scan owns the lock: it is disowned, so it may outlive this shell, and
  # a trap here would free the lock while the scan is still running.
  ( _qterm_scan; rmdir "$QTERM_DIR/lock" 2>/dev/null ) &!
}

# ---- boot -------------------------------------------------------------------
# 1) If we need to scan, do it in the background (non-blocking).
# 2) Wire the nag into precmd so the notice appears after the prompt renders.
# 3) Drop scratch files from scans that were killed mid-run.
if _qterm_needs_scan; then
  _qterm_bg_scan
fi
find "$QTERM_DIR" -name 'result.scan.*' -mtime +0 -delete 2>/dev/null

# ---- manual CLI ---------------------------------------------------------------
# qterm-updater            — scan now (fresh, blocking) then show the notice
# qterm-updater list       — show cached notice (no scan)
# qterm-updater update     — update all managers now (no prompt)
# qterm-updater reset      — forget skip + cache (nag re-appears next shell)

_qterm_scan_with_spinner() {
  # Run the scan in the background while a spinner ticks on this line.
  # Works because every scanner only appends to $_QTERM_RESULT_TMP.
  local label="Scanning ${QTERM_MANAGERS[*]:gs/ / \/ /}"
  _qterm_scan &
  local pid=$!

  # Skip the spinner entirely when stdout is not a tty (pipes, CI).
  if [[ ! -t 1 ]]; then
    printf '%s…\n' "$label"
    wait "$pid"
    return
  fi

  local -a frames=('⠋' '⠙' '⠹' '⠸' '⠼' '⠴' '⠦' '⠧' '⠇' '⠏')
  local i=0 elapsed
  tput civis 2>/dev/null  # hide cursor
  while kill -0 "$pid" 2>/dev/null; do
    elapsed=$(( SECONDS ))
    printf '\r  \033[36m%s\033[0m %s… \033[2m%ds\033[0m  ' \
      "${frames[$(( i % 10 + 1 ))]}" "$label" "$elapsed"
    (( i++ ))
    sleep 0.1
  done
  wait "$pid"
  printf '\r\033[2K\033[0m'  # clear spinner line, restore cursor
  tput cnorm 2>/dev/null
}
qterm-updater() {
  local cmd="${1:-check}"
  case "$cmd" in
    -h|--help|help)
      printf 'usage: qterm-updater [check|list|update|reset]\n'
      printf '  check    Scan now (blocking) and show the notice (default)\n'
      printf '  list     Show the most recent cached notice (no scan)\n'
      printf '  update   Update all managers immediately, no prompt\n'
      printf '  reset    Clear skip + cache — nag re-appears next shell\n'
      ;;
    check)
      _qterm_scan_with_spinner
      _QTERM_PROMPT=1 _qterm_render_notice || {
        printf 'Everything is up to date.\n'
        return 0
      }
      printf '\n'
      local key
      key=$(_qterm_key_reader)
      case "$key" in
        yes) _qterm_apply_updates ;;
        *)   printf '  (no action taken)\n' ;;
      esac
      printf '\n'
      ;;
    list)
      if [[ ! -s "$QTERM_RESULT" ]]; then
        printf 'No scan yet. Run: qterm-updater\n'
        return 0
      fi
      _qterm_render_notice || printf 'Everything up to date (per last scan).\n'
      ;;
    update)
      _qterm_apply_updates
      ;;
    reset)
      rm -f "$QTERM_SKIP" "$QTERM_RESULT" "$QTERM_LAST"
      printf 'Cleared. The nag will re-scan on the next shell.\n'
      ;;
    *)
      printf 'usage: qterm-updater [check|list|update|reset]\n'
      ;;
  esac
}

# ---- interactive shell wiring (notice + precmd hook) --------------------------
if [[ -o interactive ]] && [[ -t 1 ]]; then
  autoload -U add-zsh-hook 2>/dev/null
  add-zsh-hook precmd qterm_updater_nag 2>/dev/null || precmd_functions+=(qterm_updater_nag)
fi
