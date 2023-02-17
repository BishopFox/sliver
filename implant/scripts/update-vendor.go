package main

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// These are the directories imports will be taken from. If the implant
// needs files outside of these dirs in the future, add the dirs to this
// slice.
var srcDirs = []string{
	"../sliver",
	"../../protobuf/commonpb",
	"../../protobuf/dnspb",
	"../../protobuf/sliverpb",
}

func fatalf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}

func getImports(imports map[string]struct{}, dir string) error {
	fset := token.NewFileSet()
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}

		for _, imp := range file.Imports {
			impPath := strings.ToLower(imp.Path.Value)
			// skip local packages, those shouldn't be vendored
			if strings.HasPrefix(impPath, `"github.com/bishopfox/sliver`) {
				continue
			}

			imports[imp.Path.Value] = struct{}{}
		}

		return nil
	})
}

func main() {
	if len(os.Args) < 2 {
		fatalf("expected temporary directory argument")
	}

	dir := os.Args[1]
	imports := make(map[string]struct{})

	// Parse Go files and build a unique list of all their 3rd party
	// imports so we can build a vendor dir from them. Stdlib packages
	// will be included as well but 'go mod vendor' will ignore them.
	for _, srcDir := range srcDirs {
		err := getImports(imports, srcDir)
		if err != nil {
			fatalf("error walking directory: %v", err)
		}
	}

	// Create a Go source file containing only import statements of all
	// the packages an implant might need, no matter what OS, transport,
	// etc is configured.
	var buf bytes.Buffer
	buf.WriteString("package sliver\n\nimport (\n")
	for imp := range imports {
		buf.WriteString("\t _ ")
		buf.WriteString(imp)
		buf.WriteByte('\n')
	}
	buf.WriteString(")\n")

	err := os.WriteFile(filepath.Join(dir, "imports.go"), buf.Bytes(), 0660)
	if err != nil {
		fatalf("error creating file: %v", err)
	}
}
