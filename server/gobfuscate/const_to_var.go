package gobfuscate

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"sort"
	"strings"
)

func stringConstsToVar(path string) error {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	set := token.NewFileSet()
	file, err := parser.ParseFile(set, path, nil, 0)
	if err != nil {
		// If the file is invalid, we do nothing.
		return nil
	}

	ctv := &constToVar{}
	for _, decl := range file.Decls {
		ast.Walk(ctv, decl)
	}
	sort.Sort(ctv)

	var resBuf bytes.Buffer
	var lastIdx int
	for _, decl := range ctv.Decls {
		start := int(decl.Pos() - 1)
		end := int(decl.End() - 1)
		resBuf.Write(contents[lastIdx:start])
		declData := contents[start:end]
		varData := strings.Replace(string(declData), "const", "var", 1)
		resBuf.WriteString(varData)
		lastIdx = end
	}
	resBuf.Write(contents[lastIdx:])

	return ioutil.WriteFile(path, resBuf.Bytes(), 0755)
}

type constToVar struct {
	Decls []*ast.GenDecl
}

func (c *constToVar) Visit(n ast.Node) ast.Visitor {
	if decl, ok := n.(*ast.GenDecl); ok {
		if decl.Tok == token.CONST {
			if constOnlyHasStrings(decl) {
				c.Decls = append(c.Decls, decl)
			}
		}
	}
	return c
}

func (c *constToVar) Len() int {
	return len(c.Decls)
}

func (c *constToVar) Swap(i, j int) {
	c.Decls[i], c.Decls[j] = c.Decls[j], c.Decls[i]
}

func (c *constToVar) Less(i, j int) bool {
	return c.Decls[i].Pos() < c.Decls[j].Pos()
}

func constOnlyHasStrings(decl *ast.GenDecl) bool {
	for _, spec := range decl.Specs {
		if cs, ok := spec.(*ast.ValueSpec); ok {
			if !specIsString(cs) {
				return false
			}
		}
	}
	return true
}

func specIsString(v *ast.ValueSpec) bool {
	if v.Type != nil {
		s, ok := v.Type.(fmt.Stringer)
		if ok && s.String() == "string" {
			return true
		}
	}
	if len(v.Values) != 1 {
		return false
	}
	return exprIsString(v.Values[0])
}

func exprIsString(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.BasicLit:
		if e.Kind == token.STRING {
			return true
		}
	case *ast.BinaryExpr:
		if e.Op == token.ADD {
			return exprIsString(e.X) || exprIsString(e.Y)
		}
		return false
	case *ast.ParenExpr:
		return exprIsString(e.X)
	}
	return false
}
