package goname

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/mod/modfile"
)

// Result captures changes made by RenameModule.
type Result struct {
	OldModule      string
	NewModule      string
	GoModUpdated   bool
	FilesUpdated   int
	ImportsUpdated int
}

// RenameModule updates go.mod and rewrites import paths within the module tree.
func RenameModule(dir, newModule string) (*Result, error) {
	newModule = strings.TrimSpace(newModule)
	if newModule == "" {
		return nil, errors.New("new module path is required")
	}

	goModPath := filepath.Join(dir, "go.mod")
	modBytes, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, fmt.Errorf("read go.mod: %w", err)
	}

	modFile, err := modfile.Parse(goModPath, modBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("parse go.mod: %w", err)
	}
	if modFile.Module == nil || modFile.Module.Mod.Path == "" {
		return nil, errors.New("go.mod is missing a module path")
	}

	oldModule := modFile.Module.Mod.Path
	result := &Result{
		OldModule: oldModule,
		NewModule: newModule,
	}

	if oldModule == newModule {
		return result, nil
	}

	if err := modFile.AddModuleStmt(newModule); err != nil {
		return result, fmt.Errorf("update module path: %w", err)
	}
	formatted, err := modFile.Format()
	if err != nil {
		return result, fmt.Errorf("format go.mod: %w", err)
	}
	if err := writeFilePreservePerm(goModPath, formatted); err != nil {
		return result, fmt.Errorf("write go.mod: %w", err)
	}
	result.GoModUpdated = true

	if err := rewriteImports(dir, oldModule, newModule, result); err != nil {
		return result, err
	}

	return result, nil
}

func rewriteImports(dir, oldModule, newModule string, result *Result) error {
	return filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == "vendor" || name == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		changed, importsUpdated, err := rewriteFileImports(path, oldModule, newModule)
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

func rewriteFileImports(path, oldModule, newModule string) (bool, int, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return false, 0, fmt.Errorf("parse %s: %w", path, err)
	}

	changed := false
	importsUpdated := 0
	for _, imp := range file.Imports {
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return false, 0, fmt.Errorf("parse import in %s: %w", path, err)
		}

		if !matchesModule(importPath, oldModule) {
			continue
		}

		newPath := newModule + strings.TrimPrefix(importPath, oldModule)
		if newPath == importPath {
			continue
		}

		imp.Path.Value = strconv.Quote(newPath)
		changed = true
		importsUpdated++
	}

	if !changed {
		return false, 0, nil
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return false, 0, fmt.Errorf("format %s: %w", path, err)
	}

	if err := writeFilePreservePerm(path, buf.Bytes()); err != nil {
		return false, 0, fmt.Errorf("write %s: %w", path, err)
	}

	return true, importsUpdated, nil
}

func matchesModule(importPath, modulePath string) bool {
	return importPath == modulePath || strings.HasPrefix(importPath, modulePath+"/")
}

func writeFilePreservePerm(path string, data []byte) error {
	perm := fs.FileMode(0644)
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
	}
	return os.WriteFile(path, data, perm)
}
