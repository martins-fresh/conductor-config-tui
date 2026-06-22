// Command baton is a terminal UI for viewing and CRUD-editing the agent-deck
// conductor config files.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/martins-fresh/conductor-config-tui/internal/store"
	"github.com/martins-fresh/conductor-config-tui/internal/tui"
)

func main() {
	defaultRoot := filepath.Join(homeDir(), ".agent-deck", "conductor")
	root := flag.String("root", defaultRoot, "conductor config root directory")
	flag.Parse()

	resolved, err := expand(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "baton: %v\n", err)
		os.Exit(1)
	}

	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "baton: conductor root %q is not a directory\n", resolved)
		os.Exit(1)
	}

	s := store.New(resolved)
	p := tea.NewProgram(tui.New(s), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "baton: %v\n", err)
		os.Exit(1)
	}
}

// expand resolves a leading ~ and makes the path absolute.
func expand(path string) (string, error) {
	if path == "~" {
		return homeDir(), nil
	}
	if len(path) >= 2 && path[:2] == "~/" {
		path = filepath.Join(homeDir(), path[2:])
	}
	return filepath.Abs(path)
}

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return h
}
