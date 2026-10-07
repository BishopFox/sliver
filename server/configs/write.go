package configs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// renameConfigFile is replaceable by tests to exercise failed commits.
var renameConfigFile = os.Rename

// atomicWriteConfig replaces a config only after its complete contents reach disk.
// Existing symlinks keep pointing at the same target rather than being replaced.
func atomicWriteConfig(path string, data []byte) error {
	target := path
	info, err := os.Lstat(path)
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		target, err = filepath.EvalSymlinks(path)
		if err != nil {
			return fmt.Errorf("resolve config symlink %s: %w", path, err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("inspect config %s: %w", path, err)
	}

	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config directory %s: %w", dir, err)
	}
	temp, err := os.CreateTemp(dir, "."+filepath.Base(target)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	defer os.Remove(temp.Name())
	defer temp.Close()

	if err := temp.Chmod(0600); err != nil {
		return fmt.Errorf("set temporary config permissions: %w", err)
	}
	n, err := temp.Write(data)
	if err != nil {
		return fmt.Errorf("write temporary config: %w", err)
	}
	if n != len(data) {
		return io.ErrShortWrite
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("sync temporary config: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary config: %w", err)
	}
	if err := renameConfigFile(temp.Name(), target); err != nil {
		return fmt.Errorf("replace config %s: %w", path, err)
	}
	return nil
}
