//go:build sliver_controlflow_e2e

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	balancedV1Directive      = "//garble:controlflow block_splits=2 junk_jumps=2 flatten_passes=1 flatten_hardening=xor trash_blocks=0"
	controlFlowGeneratedFile = "GARBLE_controlflow.go"
)

var expectedControlFlowFunctions = map[string]string{
	"determineDirPathFilter": filepath.Join("handlers", "handlers.go"),
	"registerSliver":         filepath.Join("runner", "runner.go"),
	"randomCCDomain":         filepath.Join("transports", "transports.go"),
}

func verifyRenderedControlFlow(serverRoot string, implantName string) error {
	sourceRoot := filepath.Join(serverRoot, "slivers", targetOS, targetArch, filepath.Base(implantName), "src")
	found := map[string]string{}
	directiveCount := 0
	err := filepath.WalkDir(sourceRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && path != sourceRoot {
			switch entry.Name() {
			case "vendor", "protobuf":
				return filepath.SkipDir
			}
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			return nil
		}
		relativePath, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parse rendered source %q: %w", path, err)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Doc == nil {
				continue
			}
			for _, comment := range function.Doc.List {
				if comment.Text != balancedV1Directive {
					continue
				}
				directiveCount++
				expectedSuffix, expected := expectedControlFlowFunctions[function.Name.Name]
				if !expected {
					return fmt.Errorf("unexpected exact control-flow directive on %s in %s", function.Name.Name, relativePath)
				}
				if !pathHasSuffix(relativePath, expectedSuffix) {
					return fmt.Errorf("control-flow function %s rendered in %s, want suffix %s", function.Name.Name, relativePath, expectedSuffix)
				}
				if previousPath, duplicate := found[function.Name.Name]; duplicate {
					return fmt.Errorf("control-flow directive for %s appears in both %s and %s", function.Name.Name, previousPath, relativePath)
				}
				found[function.Name.Name] = relativePath
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("inspect rendered source for %s: %w", implantName, err)
	}
	if directiveCount != len(expectedControlFlowFunctions) {
		return fmt.Errorf("rendered source has %d exact control-flow directives, want %d", directiveCount, len(expectedControlFlowFunctions))
	}
	for function, suffix := range expectedControlFlowFunctions {
		if found[function] == "" {
			return fmt.Errorf("rendered source is missing control-flow directive for %s in *%s", function, suffix)
		}
	}
	return nil
}

func verifyGarbleControlFlowDebugDir(debugDir string) (int, error) {
	garbledRoot := filepath.Join(debugDir, "garbled")
	foundPackages := map[string]string{}
	transformedFiles := 0
	err := filepath.WalkDir(garbledRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != controlFlowGeneratedFile {
			return nil
		}
		transformedFiles++
		packageName := filepath.Base(filepath.Dir(path))
		if _, expected := map[string]struct{}{"handlers": {}, "runner": {}, "transports": {}}[packageName]; !expected {
			return fmt.Errorf("unexpected transformed control-flow package %q at %s", packageName, path)
		}
		if previousPath, duplicate := foundPackages[packageName]; duplicate {
			return fmt.Errorf("multiple %s files for package %s: %s and %s", controlFlowGeneratedFile, packageName, previousPath, path)
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.Contains(source, []byte("goto ")) {
			return fmt.Errorf("transformed source %s contains no flattened control-flow jumps", path)
		}
		if _, err := parser.ParseFile(token.NewFileSet(), path, source, parser.SkipObjectResolution); err != nil {
			return fmt.Errorf("parse transformed source %s: %w", path, err)
		}
		foundPackages[packageName] = path
		return nil
	})
	if err != nil {
		return 0, err
	}
	if transformedFiles != len(expectedControlFlowFunctions) {
		return 0, fmt.Errorf("Garble debug output has %d %s files, want %d", transformedFiles, controlFlowGeneratedFile, len(expectedControlFlowFunctions))
	}
	for _, packageName := range []string{"handlers", "runner", "transports"} {
		if foundPackages[packageName] == "" {
			return 0, fmt.Errorf("Garble debug output is missing transformed package %s", packageName)
		}
	}
	return transformedFiles, nil
}

func pathHasSuffix(path string, suffix string) bool {
	path = filepath.ToSlash(filepath.Clean(path))
	suffix = filepath.ToSlash(filepath.Clean(suffix))
	if path == suffix {
		return true
	}
	return strings.HasSuffix(path, "/"+suffix)
}
