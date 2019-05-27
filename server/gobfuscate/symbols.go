package gobfuscate

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/refactor/importgraph"
)

// IgnoreMethods - Methods to skip when obfuscating
var IgnoreMethods = map[string]bool{
	"main":      true,
	"init":      true,
	"RunSliver": true,
}

// SkipRenames - Skip renaming these symbols
var SkipRenames = map[string]bool{
	"_":          true,
	"int32ptr":   true,
	"atomicLock": true,
	"grow":       true,
}

type symbolRenameReq struct {
	OldName string
	NewName string
}

// ObfuscateSymbols - Obfuscate binary symbols
func ObfuscateSymbols(ctx build.Context, gopath string, enc *Encrypter) error {
	renames, err := topLevelRenames(gopath, enc)
	if err != nil {
		return fmt.Errorf("top-level renames: %s", err)
	}
	if err := runRenames(ctx, gopath, renames); err != nil {
		return fmt.Errorf("top-level renaming: %s", err)
	}
	renames, err = methodRenames(ctx, gopath, enc)
	if err != nil {
		return fmt.Errorf("method renames: %s", err)
	}
	if err := runRenames(ctx, gopath, renames); err != nil {
		return fmt.Errorf("method renaming: %s", err)
	}
	return nil
}

func runRenames(ctx build.Context, gopath string, renames []symbolRenameReq) error {
	ctx.GOPATH = gopath
	for _, r := range renames {
		parts := strings.Split(r.OldName, ".") // OldName contains the full obfuscated path
		symbol := parts[len(parts)-1]
		if _, ok := SkipRenames[symbol]; ok {
			obfuscateLog.Infof("Skipping rename of %s", symbol)
			continue
		}
		obfuscateLog.Infof("Rename %s -> %s", symbol, r.NewName)
		if err := Rename(&ctx, "", r.OldName, r.NewName); err != nil {
			return err
		}
	}
	return nil
}

func topLevelRenames(gopath string, enc *Encrypter) ([]symbolRenameReq, error) {
	srcDir := filepath.Join(gopath, "src")
	res := map[symbolRenameReq]int{}
	addRes := func(pkgPath, name string) {
		prefix := "\"" + pkgPath + "\"."
		oldName := prefix + name
		newName := enc.Encrypt(name)
		res[symbolRenameReq{oldName, newName}]++
	}
	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && containsUnsupportedCode(path) {
			return filepath.SkipDir
		}
		if filepath.Ext(path) != GoExtension {
			return nil
		}
		pkgPath, err := filepath.Rel(srcDir, filepath.Dir(path))
		if err != nil {
			return err
		}
		set := token.NewFileSet()
		file, err := parser.ParseFile(set, path, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if !IgnoreMethods[d.Name.Name] && d.Recv == nil {
					addRes(pkgPath, d.Name.Name)
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						addRes(pkgPath, spec.Name.Name)
					case *ast.ValueSpec:
						for _, name := range spec.Names {
							addRes(pkgPath, name.Name)
						}
					}
				}
			}
		}
		return nil
	})
	return singleRenames(res), err
}

func methodRenames(ctx build.Context, gopath string, enc *Encrypter) ([]symbolRenameReq, error) {
	exclude, err := interfaceMethods(ctx, gopath)
	if err != nil {
		return nil, err
	}

	srcDir := filepath.Join(gopath, "src")
	res := map[symbolRenameReq]int{}
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && containsUnsupportedCode(path) {
			return filepath.SkipDir
		}
		if filepath.Ext(path) != GoExtension {
			return nil
		}
		pkgPath, err := filepath.Rel(srcDir, filepath.Dir(path))
		if err != nil {
			return err
		}
		set := token.NewFileSet()
		file, err := parser.ParseFile(set, path, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			d, ok := decl.(*ast.FuncDecl)
			if !ok || exclude[d.Name.Name] || d.Recv == nil {
				continue
			}
			prefix := "\"" + pkgPath + "\"."
			for _, rec := range d.Recv.List {
				s, ok := rec.Type.(fmt.Stringer)
				if !ok {
					continue
				}
				oldName := prefix + s.String() + "." + d.Name.Name
				newName := enc.Encrypt(d.Name.Name)
				res[symbolRenameReq{oldName, newName}]++
			}
		}
		return nil
	})
	return singleRenames(res), err
}

func interfaceMethods(ctx build.Context, gopath string) (map[string]bool, error) {
	ctx.GOPATH = gopath
	forward, backward, _ := importgraph.Build(&ctx)
	pkgs := map[string]bool{}
	for _, m := range []importgraph.Graph{forward, backward} {
		for x := range m {
			pkgs[x] = true
		}
	}
	res := map[string]bool{}
	for pkgName := range pkgs {
		pkg, err := ctx.Import(pkgName, gopath, 0)
		if err != nil {
			return nil, fmt.Errorf("import %s: %s", pkgName, err)
		}
		for _, fileName := range pkg.GoFiles {
			sourcePath := filepath.Join(pkg.Dir, fileName)
			set := token.NewFileSet()
			file, err := parser.ParseFile(set, sourcePath, nil, 0)
			if err != nil {
				return nil, err
			}
			for _, decl := range file.Decls {
				d, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				for _, spec := range d.Specs {
					spec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					t, ok := spec.Type.(*ast.InterfaceType)
					if !ok {
						continue
					}
					for _, field := range t.Methods.List {
						for _, name := range field.Names {
							res[name.Name] = true
						}
					}
				}
			}
		}
	}
	return res, nil
}

// singleRenames removes any rename requests which appear
// more than one time.
// This is necessary because of build constraints, which
// the refactoring API doesn't seem to properly support.
func singleRenames(multiset map[symbolRenameReq]int) []symbolRenameReq {
	var res []symbolRenameReq
	for x, count := range multiset {
		if count == 1 {
			res = append(res, x)
		}
	}
	return res
}

// containsUnsupportedCode checks if a source directory
// contains assembly or CGO code, neither of which are
// supported by the refactoring API.
func containsUnsupportedCode(dir string) bool {
	return containsAssembly(dir) || containsCGO(dir)
}

// containsAssembly checks if a source directory contains
// any assembly files.
// We cannot rename symbols in assembly-filled directories
// because of limitations of the refactoring API.
func containsAssembly(dir string) bool {
	contents, _ := ioutil.ReadDir(dir)
	for _, item := range contents {
		if filepath.Ext(item.Name()) == ".s" {
			return true
		}
	}
	return false
}

// containsCGO checks if a package relies on CGO.
// We cannot rename symbols in packages that use CGO due
// to limitations of the refactoring API.
func containsCGO(dir string) bool {
	listing, err := ioutil.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, item := range listing {
		if filepath.Ext(item.Name()) == GoExtension {
			path := filepath.Join(dir, item.Name())
			set := token.NewFileSet()
			file, err := parser.ParseFile(set, path, nil, 0)
			if err != nil {
				return false
			}
			for _, spec := range file.Imports {
				if spec.Path.Value == `"C"` {
					return true
				}
			}
		}
	}
	return false
}
