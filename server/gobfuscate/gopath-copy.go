package gobfuscate

import (
	"errors"
	"go/build"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/refactor/importgraph"
)

// CopyGopath - Creates a new Gopath with a copy of a package
// and all of its dependencies.
func CopyGopath(ctx build.Context, packageName string, newGopath string, keepTests bool) bool {
	if ctx.GOPATH == "" {
		obfuscateLog.Warn("GOPATH not set.")
	}
	forward, _, errs := importgraph.Build(&ctx)
	if _, ok := forward[packageName]; !ok {
		obfuscateLog.Errorf("Failed to build import graph: %s", packageName)
		if err, ok := errs[packageName]; ok {
			obfuscateLog.Errorf(" -> Error for package: %s", err)
		}
		return false
	}
	allDeps := forward.Search(packageName)

	for dep := range allDeps {
		err := copyDep(dep, ctx.GOPATH, newGopath, keepTests)
		if err != nil {
			obfuscateLog.Errorf("Failed to copy %s: %s\n", dep, err)
			return false
		}
	}

	if !keepTests {
		ctx.GOPATH = newGopath
		forward, _, errs = importgraph.Build(&ctx)
		if _, ok := forward[packageName]; !ok {
			obfuscateLog.Errorf("Failed to re-build import graph: %s", packageName)
			if err, ok := errs[packageName]; ok {
				obfuscateLog.Errorf(" -> Error for package: %s", err)
			}
			return false
		}
		allDeps = forward.Search(packageName)
	}

	if err := removeUnusedPkgs(newGopath, allDeps); err != nil {
		obfuscateLog.Errorf("Failed to prune sub-packages: %s", err)
		return false
	}
	return true
}

func copyDep(packagePath, oldGopath, newGopath string, keepTests bool) error {
	oldPath := filepath.Join(oldGopath, "src", packagePath)
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return nil
	}
	newPath := filepath.Join(newGopath, "src", packagePath)
	if _, err := os.Stat(newPath); err == nil {
		os.RemoveAll(newPath)
	}
	createDir(newPath)
	return filepath.Walk(oldPath, func(source string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		base, err := filepath.Rel(oldGopath, source)
		if err != nil {
			return err
		}
		newPath := filepath.Join(newGopath, base)
		if info.IsDir() {
			return createDir(newPath)
		}
		if !keepTests && strings.HasSuffix(source, "_test.go") {
			return nil
		}
		newFile, err := os.Create(newPath)
		if err != nil {
			return err
		}
		defer newFile.Close()
		oldFile, err := os.Open(source)
		if err != nil {
			return err
		}
		defer oldFile.Close()
		_, err = io.Copy(newFile, oldFile)
		return err
	})
}

func removeUnusedPkgs(gopath string, deps map[string]bool) error {
	srcDir := filepath.Join(gopath, "src")
	return filepath.Walk(srcDir, func(sub string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		if !containsDep(gopath, sub, deps) {
			os.RemoveAll(sub)
			return filepath.SkipDir
		}
		return nil
	})
}

func containsDep(gopath, dir string, deps map[string]bool) bool {
	for dep := range deps {
		depDir := filepath.Clean(filepath.Join(gopath, "src", dep))
		if strings.HasPrefix(depDir, filepath.Clean(dir)) {
			return true
		}
	}
	return false
}

func createDir(dir string) error {
	if info, err := os.Stat(dir); err == nil {
		if info.IsDir() {
			return nil
		}
		return errors.New("file already exists: " + dir)
	}
	if filepath.Dir(dir) != dir {
		parent := filepath.Dir(dir)
		if err := createDir(parent); err != nil {
			return err
		}
	}
	return os.Mkdir(dir, 0755)
}
