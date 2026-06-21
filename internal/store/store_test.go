package store

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// makeRoot builds a synthetic conductor root and returns a Store.
func makeRoot(t *testing.T) Store {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "CLAUDE.md"), "# shared\n")
	mustWrite(t, filepath.Join(root, "POLICY.md"), "# shared policy\n")
	for _, name := range []string{"core", "telehealth"} {
		dir := filepath.Join(root, name)
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		mustWrite(t, filepath.Join(dir, "CLAUDE.md"), "# "+name+"\n")
		mustWrite(t, filepath.Join(dir, "meta.json"), `{"name":"`+name+`"}`)
		mustWrite(t, filepath.Join(dir, "heartbeat.sh"), "#!/bin/sh\n")
	}
	// A non-conductor dir (no CLAUDE.md) should be ignored.
	if err := os.Mkdir(filepath.Join(root, "__pycache__"), 0o755); err != nil {
		t.Fatal(err)
	}
	return New(root)
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscover(t *testing.T) {
	s := makeRoot(t)
	cs, err := s.Discover()
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 3 {
		t.Fatalf("expected shared + 2 conductors, got %d: %+v", len(cs), cs)
	}
	if !cs[0].Shared || cs[0].Name != SharedName {
		t.Errorf("first entry should be shared, got %+v", cs[0])
	}
	if cs[1].Name != "core" || cs[2].Name != "telehealth" {
		t.Errorf("conductors not sorted: %s, %s", cs[1].Name, cs[2].Name)
	}
}

func TestListFilesSharedAndConductor(t *testing.T) {
	s := makeRoot(t)
	cs, _ := s.Discover()

	shared, err := s.ListFiles(cs[0])
	if err != nil {
		t.Fatal(err)
	}
	// Only the canonical shared files that exist (CLAUDE.md, POLICY.md).
	if len(shared) != 2 {
		t.Fatalf("expected 2 shared files, got %d: %+v", len(shared), shared)
	}

	core, err := s.ListFiles(cs[1])
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]File{}
	for _, f := range core {
		got[f.Name] = f
	}
	if !got["CLAUDE.md"].Editable {
		t.Error("CLAUDE.md should be editable")
	}
	if !got["meta.json"].Editable || got["meta.json"].Kind != KindJSON {
		t.Error("meta.json should be editable JSON")
	}
	if got["heartbeat.sh"].Editable {
		t.Error("heartbeat.sh should be read-only")
	}
}

func TestSaveJSONValidation(t *testing.T) {
	s := makeRoot(t)
	cs, _ := s.Discover()
	files, _ := s.ListFiles(cs[1])
	var meta File
	for _, f := range files {
		if f.Name == "meta.json" {
			meta = f
		}
	}

	if err := s.Save(meta, []byte("{not json")); err == nil {
		t.Fatal("expected invalid JSON to be rejected")
	}
	// Original content must be untouched after a rejected save.
	data, _ := os.ReadFile(meta.Path)
	if string(data) != `{"name":"core"}` {
		t.Fatalf("file mutated after rejected save: %s", data)
	}

	if err := s.Save(meta, []byte(`{"name":"core","x":2}`)); err != nil {
		t.Fatalf("valid JSON save failed: %v", err)
	}
	data, _ = os.ReadFile(meta.Path)
	if string(data) != `{"name":"core","x":2}` {
		t.Fatalf("save did not persist: %s", data)
	}
}

func TestAtomicWritePreservesMode(t *testing.T) {
	s := makeRoot(t)
	cs, _ := s.Discover()
	files, _ := s.ListFiles(cs[1])
	var claude File
	for _, f := range files {
		if f.Name == "CLAUDE.md" {
			claude = f
		}
	}
	if err := os.Chmod(claude.Path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(claude, []byte("# updated\n")); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(claude.Path)
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("expected mode preserved 0600, got %o", info.Mode().Perm())
	}
}

func TestScaffoldAndCreateAndDelete(t *testing.T) {
	s := makeRoot(t)

	c, err := s.ScaffoldConductor("newcond")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(c.Dir, "CLAUDE.md")); err != nil {
		t.Error("scaffold missing CLAUDE.md")
	}
	if _, err := os.Stat(filepath.Join(c.Dir, "POLICY.md")); err != nil {
		t.Error("scaffold missing POLICY.md")
	}

	// Duplicate should fail.
	if _, err := s.ScaffoldConductor("newcond"); err == nil {
		t.Error("expected duplicate conductor to fail")
	}

	f, err := s.CreateFile(c, "LEARNINGS.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(f.Path); err != nil {
		t.Error("created file missing")
	}

	if err := s.Delete(f); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(f.Path); !os.IsNotExist(err) {
		t.Error("file not deleted")
	}

	if err := s.DeleteConductor(c); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(c.Dir); !os.IsNotExist(err) {
		t.Error("conductor dir not deleted")
	}
}

func TestInvalidNamesRejected(t *testing.T) {
	s := makeRoot(t)
	for _, bad := range []string{"", ".", "..", "a/b", "../escape", ".hidden"} {
		if _, err := s.ScaffoldConductor(bad); err == nil {
			t.Errorf("expected %q to be rejected", bad)
		}
	}
}

func TestSymlinkEscapeRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks unreliable on windows")
	}
	s := makeRoot(t)
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.md")
	mustWrite(t, secret, "top secret\n")

	link := filepath.Join(s.Root, "core", "escape.md")
	if err := os.Symlink(secret, link); err != nil {
		t.Fatal(err)
	}
	f := File{Name: "escape.md", Path: link, Kind: KindMarkdown, Editable: true}

	if _, _, err := s.Read(f); err == nil {
		t.Error("expected read through escaping symlink to be rejected")
	}
	if err := s.Save(f, []byte("pwned")); err == nil {
		t.Error("expected save through escaping symlink to be rejected")
	}
}

func TestReadPrettyPrintsJSON(t *testing.T) {
	s := makeRoot(t)
	cs, _ := s.Discover()
	files, _ := s.ListFiles(cs[1])
	var meta File
	for _, f := range files {
		if f.Name == "meta.json" {
			meta = f
		}
	}
	out, truncated, err := s.Read(meta)
	if err != nil {
		t.Fatal(err)
	}
	if truncated {
		t.Error("small file should not be truncated")
	}
	if out == `{"name":"core"}` {
		t.Error("expected JSON to be pretty-printed")
	}
}
