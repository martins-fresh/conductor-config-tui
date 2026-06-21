# Build brief: conductor-config-tui

A terminal UI (TUI) for **viewing and CRUD-editing the agent-deck conductor config files**.

## Stack
- Go (module already initialized: `github.com/martins-fresh/conductor-config-tui`, go 1.26).
- Charm stack: `bubbletea` (model), `bubbles` (list/textarea/viewport/help), `lipgloss` (styling). Add them with `go get`.
- Single static binary. `go build -o bin/cctui .`

## What it manages
Conductor configs live under `~/.agent-deck/conductor/`. Expand `~` at runtime; allow `--root <dir>` to override (default `$HOME/.agent-deck/conductor`).

Layout:
- **Per-conductor dirs** (each contains a `CLAUDE.md`): `core/`, `adverse-events/`, `ahpra/`, `drugbook/`, `github/`, `imgproxy/`, `telehealth/`, `treatments/`, `us-bugs/`, … (discover dynamically: a subdir is a "conductor" iff it has `CLAUDE.md`).
  - Common files per conductor: `CLAUDE.md`, `POLICY.md`, `LEARNINGS.md`, `state.json`, `task-log.md`, `meta.json` (some optional; some have extras like `heartbeat.sh`, `POLICY.draft.md`).
- **Shared top-level files**: `CLAUDE.md`, `POLICY.md`, `LEARNINGS.md`, `task-log.md` directly under the root.

## UX (suggested 3-pane / drill-down)
1. **Conductors list** (left): discovered conductor dirs + a "(shared)" entry for top-level files.
2. **Files list** (middle): config files in the selected conductor. Mark which are editable vs read-only.
3. **Content view/edit** (right): viewport for reading; `e` opens an inline `textarea` editor (or `$EDITOR` if set — your call, inline preferred for the TUI feel); `s` saves.

Include a help bar (bubbles/help) with keybindings.

## CRUD — apply judgment per file type
- **Read**: every file. Pretty-render markdown-ish (at least monospace + scroll); for `*.json`, pretty-print and validate on save.
- **Update (edit + save)**: `CLAUDE.md`, `POLICY.md`, `LEARNINGS.md`, `task-log.md`, `meta.json`, `state.json`. For JSON files, reject save on invalid JSON with a clear error; write atomically (temp file + rename) to avoid corrupting `state.json`.
- **Create**:
  - New conductor: scaffold a dir with starter `CLAUDE.md` + `POLICY.md` (minimal templates).
  - New file within a conductor (e.g. add a missing `POLICY.md`).
- **Delete**: individual files and whole conductor dirs — **always behind a confirmation modal**. Never delete without an explicit y/N confirm. Do not offer delete for the shared top-level files unless confirmed twice (these are high-value).

## Safety
- Atomic writes for all saves (write `.tmp`, fsync, rename).
- Confirmation modal for every destructive action.
- Never follow symlinks out of the root.
- Show the absolute path of whatever is focused.

## Deliverable
- Working `go build` producing `bin/cctui`.
- A short demo in the README (keybindings + screenshot/asciinema optional).
- Open a PR against `main` of this repo when done. Commit and push freely (no ask-before-commit gate).

## Process
- Work on branch `feature/initial-tui` (you are already in a worktree on it).
- Run `gofmt`/`go vet` before committing. Keep commits scoped.
- If you hit a genuine design fork you can't resolve from this brief, message the conductor (session `conductor-core`); otherwise proceed.
