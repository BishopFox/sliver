package goname

import (
	"errors"
	"io/fs"
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
