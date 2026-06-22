// Package store handles discovery and safe CRUD of conductor config files
// living under the agent-deck conductor root.
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MaxReadBytes caps how much of a file is loaded into the content view. Some
// files under the root (e.g. bridge.log) are multi-megabyte; we never want to
// pull all of that into memory for display.
const MaxReadBytes int64 = 512 * 1024

// SharedName is the display label for the synthetic conductor representing the
// shared top-level config files.
const SharedName = "(shared)"

// sharedFiles are the canonical config files that live directly under the root.
// We intentionally surface only these (not bridge.py, *.log, *.bak, …) so the
// shared view stays focused on editable configuration.
var sharedFiles = []string{"CLAUDE.md", "POLICY.md", "LEARNINGS.md", "task-log.md", "RESYNC.md"}

// Kind classifies a file for rendering and validation purposes.
type Kind int

const (
	KindMarkdown Kind = iota
	KindJSON
	KindOther
)

// File is a single config file within a conductor.
type File struct {
	Name     string
	Path     string
	Kind     Kind
	Editable bool
	Size     int64
}

// Conductor is a discovered conductor directory, or the synthetic shared entry.
type Conductor struct {
	Name   string // display name; SharedName for the shared entry
	Dir    string // absolute directory path
	Shared bool
}

// Store is rooted at a conductor config directory.
type Store struct {
	Root string
}

// New returns a Store rooted at root (already expanded/absolute).
func New(root string) Store { return Store{Root: root} }

// editableExt reports whether a file extension is editable. Markdown and JSON
// configs are editable; scripts, logs, and binaries are read-only.
func editableExt(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".json":
		return true
	default:
		return false
	}
}

func kindOf(name string) Kind {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md":
		return KindMarkdown
	case ".json":
		return KindJSON
	default:
		return KindOther
	}
}

// Discover returns the shared entry followed by conductor directories (those
// containing a CLAUDE.md), sorted by name.
func (s Store) Discover() ([]Conductor, error) {
	entries, err := os.ReadDir(s.Root)
	if err != nil {
		return nil, fmt.Errorf("reading root %s: %w", s.Root, err)
	}

	conductors := []Conductor{{Name: SharedName, Dir: s.Root, Shared: true}}

	var dirs []Conductor
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		dir := filepath.Join(s.Root, e.Name())
		if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err != nil {
			continue // a conductor iff it has CLAUDE.md
		}
		dirs = append(dirs, Conductor{Name: e.Name(), Dir: dir})
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })

	return append(conductors, dirs...), nil
}

// ListFiles returns the config files for a conductor. For the shared entry it
// returns only the canonical top-level config files that exist; for a real
// conductor it returns every regular, non-hidden file in the directory.
func (s Store) ListFiles(c Conductor) ([]File, error) {
	if c.Shared {
		var files []File
		for _, name := range sharedFiles {
			p := filepath.Join(s.Root, name)
			info, err := os.Stat(p)
			if err != nil || info.IsDir() {
				continue
			}
			files = append(files, fileFrom(name, p, info.Size()))
		}
		return files, nil
	}

	entries, err := os.ReadDir(c.Dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", c.Dir, err)
	}
	var files []File
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fileFrom(e.Name(), filepath.Join(c.Dir, e.Name()), info.Size()))
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	return files, nil
}

func fileFrom(name, path string, size int64) File {
	return File{
		Name:     name,
		Path:     path,
		Kind:     kindOf(name),
		Editable: editableExt(name),
		Size:     size,
	}
}

// Read loads up to MaxReadBytes of the file. The returned bool reports whether
// the content was truncated. JSON files are pretty-printed when valid.
func (s Store) Read(f File) (string, bool, error) {
	if err := s.checkWithinRoot(f.Path); err != nil {
		return "", false, err
	}
	fh, err := os.Open(f.Path)
	if err != nil {
		return "", false, err
	}
	defer fh.Close()

	buf := make([]byte, MaxReadBytes+1)
	n, err := readFull(fh, buf)
	if err != nil {
		return "", false, err
	}
	truncated := int64(n) > MaxReadBytes
	if truncated {
		n = int(MaxReadBytes)
	}
	data := buf[:n]

	if f.Kind == KindJSON && !truncated {
		if pretty, ok := prettyJSON(data); ok {
			return pretty, false, nil
		}
	}
	return string(data), truncated, nil
}

// Save atomically writes content to path: validate (JSON), write to a temp file
// in the same directory, fsync, then rename over the target.
func (s Store) Save(f File, content []byte) error {
	if err := s.checkWithinRoot(f.Path); err != nil {
		return err
	}
	if f.Kind == KindJSON {
		if err := validateJSON(content); err != nil {
			return err
		}
	}
	return atomicWrite(f.Path, content)
}

// Delete removes a single file after verifying it is within the root.
func (s Store) Delete(f File) error {
	if err := s.checkWithinRoot(f.Path); err != nil {
		return err
	}
	return os.Remove(f.Path)
}

// DeleteConductor removes an entire conductor directory tree. The shared entry
// can never be deleted.
func (s Store) DeleteConductor(c Conductor) error {
	if c.Shared {
		return fmt.Errorf("refusing to delete the shared root")
	}
	if err := s.checkWithinRoot(c.Dir); err != nil {
		return err
	}
	if filepath.Clean(c.Dir) == filepath.Clean(s.Root) {
		return fmt.Errorf("refusing to delete the root directory")
	}
	return os.RemoveAll(c.Dir)
}

// ScaffoldConductor creates a new conductor directory with starter CLAUDE.md
// and POLICY.md files. Returns the created directory path.
func (s Store) ScaffoldConductor(name string) (Conductor, error) {
	name = strings.TrimSpace(name)
	if err := validName(name); err != nil {
		return Conductor{}, err
	}
	dir := filepath.Join(s.Root, name)
	if _, err := os.Stat(dir); err == nil {
		return Conductor{}, fmt.Errorf("%q already exists", name)
	}
	if err := os.Mkdir(dir, 0o755); err != nil {
		return Conductor{}, err
	}
	claude := fmt.Sprintf("# %s conductor\n\nDescribe this conductor's responsibilities here.\n", name)
	policy := fmt.Sprintf("# %s policy\n\nDescribe the operating policy for this conductor here.\n", name)
	if err := atomicWrite(filepath.Join(dir, "CLAUDE.md"), []byte(claude)); err != nil {
		return Conductor{}, err
	}
	if err := atomicWrite(filepath.Join(dir, "POLICY.md"), []byte(policy)); err != nil {
		return Conductor{}, err
	}
	return Conductor{Name: name, Dir: dir}, nil
}

// CreateFile creates a new (empty or minimally-templated) file within a
// conductor. Returns the created File.
func (s Store) CreateFile(c Conductor, name string) (File, error) {
	name = strings.TrimSpace(name)
	if err := validName(name); err != nil {
		return File{}, err
	}
	path := filepath.Join(c.Dir, name)
	if err := s.checkWithinRoot(path); err != nil {
		return File{}, err
	}
	if _, err := os.Stat(path); err == nil {
		return File{}, fmt.Errorf("%q already exists", name)
	}
	var seed []byte
	switch kindOf(name) {
	case KindMarkdown:
		title := strings.TrimSuffix(name, filepath.Ext(name))
		seed = []byte(fmt.Sprintf("# %s\n\n", title))
	case KindJSON:
		seed = []byte("{}\n")
	}
	if err := atomicWrite(path, seed); err != nil {
		return File{}, err
	}
	info, _ := os.Stat(path)
	var size int64
	if info != nil {
		size = info.Size()
	}
	return fileFrom(name, path, size), nil
}

// validName rejects empty names and any path separators / traversal.
func validName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if strings.ContainsAny(name, "/\\") || name == "." || name == ".." {
		return fmt.Errorf("name must not contain path separators")
	}
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("name must not start with a dot")
	}
	return nil
}

// checkWithinRoot ensures path resolves to a location inside the root, refusing
// to follow symlinks out of it. The path itself need not exist (its parent is
// resolved in that case).
func (s Store) checkWithinRoot(path string) error {
	root, err := filepath.EvalSymlinks(s.Root)
	if err != nil {
		return err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		// Path may not exist yet; resolve its parent instead.
		parent, perr := filepath.EvalSymlinks(filepath.Dir(path))
		if perr != nil {
			return fmt.Errorf("cannot resolve %s: %w", path, err)
		}
		resolved = filepath.Join(parent, filepath.Base(path))
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path %s escapes the conductor root", path)
	}
	return nil
}
