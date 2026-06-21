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
| `e` | Edit the selected file (inline editor) |
| `ctrl+s` | Save (in edit mode) |
| `esc` | Cancel edit / dismiss modal / go back |
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
