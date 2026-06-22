# baton

A Bubble Tea TUI for viewing and CRUD-editing the **agent-deck conductor config files**.

`baton` discovers conductor directories under `~/.agent-deck/conductor/` (any
subdir with a `CLAUDE.md` is a conductor) plus a synthetic **(shared)** entry for
the top-level config files, and gives you a three-pane drill-down to read, edit,
create, and delete them safely.

```
┌ Conductors ┐┌ Files ──────┐┌ Content  editable ───────────────┐
│ (shared)   ││   CLAUDE.md ││ # core conductor                 │
│ core       ││   LEARNINGS ││                                  │
│ telehealth ││ · heartbeat ││ Describe responsibilities here.  │
│ …          ││   meta.json ││                                  │
└────────────┘└─────────────┘└──────────────────────────────────┘
» saved meta.json
↑/k up · ↓/j down · ←/h focus left · →/l focus right · enter open · e edit · ? help · q quit
```

(`·` marks a read-only file.)

## Build & run

```sh
go build -o bin/baton .
./bin/baton                       # uses $HOME/.agent-deck/conductor
./bin/baton --root /path/to/dir   # override the root
```

Single static binary, no runtime dependencies.

## Keybindings

| Key | Action |
| --- | --- |
| `↑`/`k`, `↓`/`j` | Move within the focused pane (scrolls content pane) |
| `←`/`h`, `→`/`l` | Move focus between panes |
| `tab` | Cycle panes |
| `enter` | Drill in (conductor → files → content) |
| `e` | Edit the selected file (opens the vim editor) |
| `esc` | Dismiss a modal / go back |
| `n` | New file in the selected conductor |
| `N` | New conductor (scaffolds `CLAUDE.md` + `POLICY.md`) |
| `d` | Delete the focused file or conductor (confirmation required) |
| `r` | Refresh from disk |
| `?` | Toggle full help |
| `q` / `ctrl+c` | Quit |

## CRUD behaviour

- **Read** — every file. JSON is pretty-printed; large files are truncated for
  display (capped at 512 KiB).
- **Edit** — Markdown (`.md`) and JSON (`.json`) files are editable; scripts,
  logs and other files are read-only (shown with a `·` marker and `read-only`
  badge). Invalid JSON is **rejected on save** with an error.

### Editing (vim)

Pressing `e` on an editable file opens a [vimtea](https://github.com/kujtimiihoxha/vimtea)
modal editor with normal/insert/visual/command modes and the usual vim motions
(`h j k l`, `w b e`, `0 $`, `gg G`), edits (`i a o O`, `x`, `dd`, `yy`/`p`),
and `u`/`ctrl+r` undo/redo. To leave or save:

| Command / key | Action |
| --- | --- |
| `i` / `esc` | Enter insert mode / return to normal mode |
| `ctrl+s` | Save and stay in the editor |
| `:w` | Save |
| `:q` / `:q!` | Quit the editor (back to browsing) |
| `:wq` / `:x` | Save and quit |

Saves go through the same atomic write + JSON validation as everything else; a
rejected save keeps you in the editor so you can fix it.

> **Line numbers:** vimtea has no option to disable its line-number gutter, so
> the digits are painted in the terminal background colour to make them
> invisible. A 4-column blank margin remains on the left — that's a vimtea
> limitation, not a setting.
- **Create** — new conductor (scaffolds starter `CLAUDE.md` + `POLICY.md`) or a
  new file within a conductor.
- **Delete** — files and whole conductor dirs, always behind a confirmation
  modal. Shared top-level files require a **double** confirm; the `(shared)`
  entry itself cannot be deleted.

## Safety

- **Atomic writes**: every save goes to a temp file in the same directory, is
  `fsync`'d, then renamed over the target (the directory is fsync'd too) so an
  interrupted save never corrupts `state.json`.
- **Confirmation modal** for every destructive action.
- **No symlink escape**: reads, writes and deletes are refused if the resolved
  path falls outside the conductor root.
- The absolute path of whatever is focused is shown in the header.

## Development

```sh
gofmt -l .      # formatting check
go vet ./...    # static analysis
go test ./...   # unit tests (store package: atomicity, JSON validation,
                # symlink-escape rejection, scaffold/create/delete)
```
