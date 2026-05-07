package configs

import (
	"os"
	"path/filepath"
	"strings"
)

func legacyBackupPath(legacyPath string) string {
	dir := filepath.Dir(legacyPath)
	base := filepath.Base(legacyPath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(dir, "."+name+".json-old")
}

func renameLegacyConfig(legacyPath string) error {
	if _, err := os.Stat(legacyPath); os.IsNotExist(err) {
		return nil
	}
	backupPath := legacyBackupPath(legacyPath)
	if _, err := os.Stat(backupPath); err == nil {
		if err := os.Remove(backupPath); err != nil {
			return err
		}
	}
	return os.Rename(legacyPath, backupPath)
}
