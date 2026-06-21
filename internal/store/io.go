package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// readFull reads from r into buf until buf is full or EOF, returning the number
// of bytes read. Unlike io.ReadFull it does not error on a short final read.
func readFull(r io.Reader, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := r.Read(buf[total:])
		total += n
		if err == io.EOF {
			return total, nil
		}
		if err != nil {
			return total, err
		}
		if n == 0 {
			return total, nil
		}
	}
	return total, nil
}

// prettyJSON pretty-prints data with two-space indentation. The bool reports
// whether the input was valid JSON.
func prettyJSON(data []byte) (string, bool) {
	var out bytes.Buffer
	if err := json.Indent(&out, data, "", "  "); err != nil {
		return "", false
	}
	return out.String(), true
}

// validateJSON returns a descriptive error if data is not valid JSON.
func validateJSON(data []byte) error {
	if len(bytes.TrimSpace(data)) == 0 {
		return fmt.Errorf("JSON file cannot be empty")
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

// atomicWrite writes data to path durably: a temp file in the same directory is
// written, fsync'd, and renamed over the target so a crash never leaves a
// partially-written config.
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".baton-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Clean up the temp file on any error path.
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// Preserve the existing file mode if present.
	if info, err := os.Stat(path); err == nil {
		_ = os.Chmod(tmpName, info.Mode())
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	// fsync the directory so the rename is durable.
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}
