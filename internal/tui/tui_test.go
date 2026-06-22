package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/martins-fresh/conductor-config-tui/internal/store"
)

func testModel(t *testing.T) Model {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("# shared\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	core := filepath.Join(root, "core")
	if err := os.Mkdir(core, 0o755); err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	for i := 1; i <= 200; i++ {
		sb.WriteString("line ")
		sb.WriteByte(byte('0' + i%10))
		sb.WriteByte('\n')
	}
	if err := os.WriteFile(filepath.Join(core, "CLAUDE.md"), []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	m := New(store.New(root))
	// Give it a size so layout (and the viewport) is initialised.
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 110, Height: 30})
	return nm.(Model)
}

func kr(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestScrollMovesThumb(t *testing.T) {
	m := testModel(t)
	// Select core, focus the content pane.
	for _, k := range []tea.KeyMsg{kr("j"), kr("l"), kr("l")} {
		nm, _ := m.Update(k)
		m = nm.(Model)
	}
	if m.focus != focusContent {
		t.Fatalf("expected content focus, got %v", m.focus)
	}
	if m.vp.TotalLineCount() < 100 {
		t.Fatalf("content not loaded into viewport: %d lines", m.vp.TotalLineCount())
	}
	topThumb := strings.Index(buildScrollbar(m.bodyHeight, m.vp.TotalLineCount(), m.vp.Height, m.vp.YOffset, true), "█")

	for i := 0; i < 50; i++ {
		nm, _ := m.Update(kr("j"))
		m = nm.(Model)
	}
	if m.vp.YOffset == 0 {
		t.Fatal("viewport did not scroll on j")
	}
	scrolledThumb := strings.Index(buildScrollbar(m.bodyHeight, m.vp.TotalLineCount(), m.vp.Height, m.vp.YOffset, true), "█")
	if scrolledThumb <= topThumb {
		t.Fatalf("thumb did not move down: top=%d scrolled=%d", topThumb, scrolledThumb)
	}
}

func TestScrollbarThumbSize(t *testing.T) {
	// A tall file in a short window yields a small thumb; everything-fits yields
	// a full-height thumb.
	small := buildScrollbar(10, 200, 10, 0, false)
	if got := strings.Count(small, "█"); got == 0 || got >= 10 {
		t.Fatalf("expected partial thumb, got %d of 10", got)
	}
	full := buildScrollbar(10, 5, 10, 0, false)
	if got := strings.Count(full, "█"); got != 10 {
		t.Fatalf("expected full thumb when all content fits, got %d", got)
	}
}

func TestReadOnlyFileRejectsEdit(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("# shared\n"), 0o644)
	core := filepath.Join(root, "core")
	_ = os.Mkdir(core, 0o755)
	_ = os.WriteFile(filepath.Join(core, "CLAUDE.md"), []byte("# core\n"), 0o644)
	_ = os.WriteFile(filepath.Join(core, "run.sh"), []byte("#!/bin/sh\n"), 0o755)

	m := New(store.New(root))
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 110, Height: 30})
	m = nm.(Model)
	// Move to core, into files.
	for _, k := range []tea.KeyMsg{kr("j"), kr("l")} {
		x, _ := m.Update(k)
		m = x.(Model)
	}
	// Select run.sh (read-only) and attempt edit.
	for {
		f, _ := m.currentFile()
		if f.Name == "run.sh" {
			break
		}
		x, _ := m.Update(kr("j"))
		m = x.(Model)
	}
	x, _ := m.Update(kr("e"))
	m = x.(Model)
	if m.mode == modeEdit {
		t.Fatal("read-only file should not enter edit mode")
	}
	if !m.statusErr {
		t.Fatal("expected an error status for read-only edit attempt")
	}
}
