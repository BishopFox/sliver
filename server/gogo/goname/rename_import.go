package goname

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ImportResult captures changes made by RenameImport.
type ImportResult struct {
	OldImportPrefix string
	NewImportPrefix string
	FilesUpdated    int
	ImportsUpdated  int
}

// RenameImport rewrites import paths that match oldImportPrefix within the directory tree.
func RenameImport(dir, oldImportPrefix, newImportPrefix string) (*ImportResult, error) {
	oldImportPrefix = strings.TrimSpace(oldImportPrefix)
	newImportPrefix = strings.TrimSpace(newImportPrefix)
	if oldImportPrefix == "" {
		return nil, errors.New("old import prefix is required")
	}
	if newImportPrefix == "" {
		return nil, errors.New("new import prefix is required")
	}

	result := &ImportResult{
		OldImportPrefix: oldImportPrefix,
		NewImportPrefix: newImportPrefix,
	}

	if err := rewriteImportPrefixes(dir, oldImportPrefix, newImportPrefix, result); err != nil {
		return result, err
	}
	if result.ImportsUpdated == 0 {
		return result, errors.New("no imports updated (check old/new import prefixes)")
	}
	if err := renameImportDir(dir, oldImportPrefix, newImportPrefix); err != nil {
		return result, err
	}

	return result, nil
}

func rewriteImportPrefixes(dir, oldImportPrefix, newImportPrefix string, result *ImportResult) error {
	return filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == "vendor" || strings.HasPrefix(name, ".") {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		changed, importsUpdated, err := rewriteFileImports(path, oldImportPrefix, newImportPrefix)
		if err != nil {
			return err
		}
		if changed {
			result.FilesUpdated++
			result.ImportsUpdated += importsUpdated
		}
		return nil
	})
}

func renameImportDir(dir, oldImportPrefix, newImportPrefix string) error {
	oldParts := splitImportPrefix(oldImportPrefix)
	newParts := splitImportPrefix(newImportPrefix)
	if len(oldParts) == 0 || len(newParts) == 0 {
		return nil
	}

	for i := 0; i < len(oldParts); i++ {
		suffixParts := oldParts[i:]
		oldDir := filepath.Join(append([]string{dir}, suffixParts...)...)
		info, err := os.Stat(oldDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if !info.IsDir() {
			continue
		}

		if len(newParts) < len(suffixParts) {
			return fmt.Errorf("new import prefix has fewer path segments than old import prefix")
		}
		newSuffixParts := newParts[len(newParts)-len(suffixParts):]
		newDir := filepath.Join(append([]string{dir}, newSuffixParts...)...)

		if oldDir == newDir {
			return nil
		}

		if _, err := os.Stat(newDir); err == nil {
			return fmt.Errorf("rename import dir: target exists: %s", newDir)
		} else if !os.IsNotExist(err) {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(newDir), 0700); err != nil {
			return err
		}
		return os.Rename(oldDir, newDir)
	}

	return nil
}

func splitImportPrefix(prefix string) []string {
	cleaned := path.Clean(strings.TrimSpace(prefix))
	if cleaned == "." || cleaned == "/" {
		return nil
	}
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "" {
		return nil
	}
	return strings.Split(cleaned, "/")
}
