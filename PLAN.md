# Port brewery look & feel ke qterm-updater

Referensi: https://github.com/dillo/brewery (clone di /tmp/brewery)

## Yang ditiru (sistem inti)

1. **theme** — palette Catppuccin + 11 tema, `theme.New(palette)`, slot semantik:
   Text/Muted/Accent/Title/OK/Warn/Err/Info, Pane/PaneFocused, PaneTitle*,
   TabActive, Chip*, StatusBar, TableHeader/TableCell/TableSelected scrub-clean, Icons.
2. **theme picker + persist** — `t` buka picker, j/k preview live, enter simpan,
   esc balik. Disimpan `~/.config/qterm-updater/theme`. `:theme <name>` juga jalan.
3. **icons on/off** — `Nerd` glyph set vs zero value. `:icons on|off`, persist.
4. **command line `:`** — mode COMMAND, enter eksekusi, esc batal, tab completion,
   ↑↓ history sesi. Command: `update`/`upgrade-all`, `refresh`, `theme`, `icons`, `quit`.
5. **help overlay `?`** — dikelompokkan per konteks (sidebar/list/details + Everywhere).
6. **layout brewery** — sidebar pane berjudul `qterm-updater`, list pane, details pane,
   semua rounded border, pane fokus berwarna Mauve, title line di dalam border.
7. **status bar** — badge mode (NORMAL/COMMAND/FILTER/INSERT/RUNNING) + breadcrumb
   `entry › pkg` + `· N marked`, kanan: flash, spinner job, hint key.
8. **keymap vim** — h/l kolom, j/k baris, gg/G, ctrl+d/u, `/` filter, `[ ]` prev/next entry,
   space mark, `u` upgrade, `A` upgrade all.

## Tidak diambil (nama saja)

Drawer PTY emulator (x/vt), services, discover+analytics, notes, tap health, reinstall,
maintenance, sizes, vuln scan, goreleaser, golden/screenshot test. Itu domain brew
spesifik; qterm-updater multi-manager.

## File baru/ubah

```
internal/theme/theme.go     -> rewrite: Palette, Themes[11], Styles (semantic), New, Lookup, Index, Save/LoadTheme, Icons
internal/settings/settings.go (baru) -> dir, load/save theme + icons
internal/ui/app.go          -> rewrite: model dgn mode/focus/panes/pickers, layout pane()
internal/ui/sidebar.go      (baru)  -> sidebar pane + entry/badge
internal/ui/pane.go         (baru)  -> pane(), box(), overlay(), truncation, wrap (port components.go)
internal/ui/help.go         (baru)  -> overlay ? , groups kontekstual
internal/ui/themes.go       (baru)  -> picker t
internal/ui/cmdline.go      (baru)  -> ":" prompt + tokenizer + completion + registry command
internal/ui/status.go       (baru)  -> status bar badge/breadcrumb/right
internal/ui/app_test.go     (baru)  -> self-check: splitArgs, pane fit, overlay, keymap filter
```

## Verifikasi

`go build ./...` → `go vet ./...` → `go test ./...`
