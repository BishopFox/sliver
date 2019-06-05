package gobfuscate

import (
	"bytes"
	"errors"
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// GoExtension - Go src code file extension
const GoExtension = ".go"

// ObfuscatePackageNames - Obfuscate package names
func ObfuscatePackageNames(ctx build.Context, gopath string, enc *Encrypter) error {

	ctx.GOPATH = gopath

	level := 1
	srcDir := filepath.Join(gopath, "src")

	doneChan := make(chan struct{})
	defer close(doneChan)

	for {
		resChan := make(chan string)
		go func() {
			scanLevel(srcDir, level, resChan, doneChan)
			close(resChan)
		}()
		var gotAny bool
		for dirPath := range resChan {
			gotAny = true
			if containsCGO(dirPath) {
				continue
			}
			isMain := isMainPackage(dirPath)
			encPath := encryptPackageName(dirPath, enc)
			srcPkg, err := filepath.Rel(srcDir, dirPath)
			if err != nil {
				return err
			}
			dstPkg, err := filepath.Rel(srcDir, encPath)
			if err != nil {
				return err
			}
			if err := MovePackage(&ctx, srcPkg, dstPkg, ""); err != nil {
				return fmt.Errorf("package move: %s", err)
			}
			if isMain {
				if err := makeMainPackage(encPath); err != nil {
					return fmt.Errorf("make main package %s: %s", encPath, err)
				}
			}
		}
		if !gotAny {
			break
		}
		level++
	}

	return nil
}

func scanLevel(dir string, depth int, res chan<- string, done <-chan struct{}) {
	if depth == 0 {
		select {
		case res <- dir:
		case <-done:
			return
		}
		return
	}
	listing, _ := ioutil.ReadDir(dir)
	for _, item := range listing {
		if item.IsDir() {
			scanLevel(filepath.Join(dir, item.Name()), depth-1, res, done)
		}
		select {
		case <-done:
			return
		default:
		}
	}
}

func encryptPackageName(dir string, enc *Encrypter) string {
	subDir, base := filepath.Split(dir)
	return filepath.Join(subDir, enc.Encrypt(base))
}

func isMainPackage(dir string) bool {
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
			contents, err := ioutil.ReadFile(path)
			if err != nil {
				return false
			}
			fields := strings.Fields(string(contents[int(file.Package)-1:]))
			if len(fields) < 2 {
				return false
			}
			return fields[1] == "main"
		}
	}
	return false
}

func makeMainPackage(dir string) error {
	listing, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, item := range listing {
		if filepath.Ext(item.Name()) != GoExtension {
			continue
		}
		path := filepath.Join(dir, item.Name())
		set := token.NewFileSet()
		file, err := parser.ParseFile(set, path, nil, 0)
		if err != nil {
			return err
		}
		contents, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		pkgNameIdx := int(file.Package) + len("package") - 1
		prePkg := contents[:pkgNameIdx]
		postPkg := string(contents[pkgNameIdx:])

		fields := strings.Fields(postPkg)
		if len(fields) < 1 {
			return errors.New("no fields after package keyword")
		}
		packageName := fields[0]

		var newData bytes.Buffer
		newData.Write(prePkg)
		newData.WriteString(strings.Replace(postPkg, packageName, "main", 1))

		if err := ioutil.WriteFile(path, newData.Bytes(), item.Mode()); err != nil {
			return err
		}
	}
	return nil
}
