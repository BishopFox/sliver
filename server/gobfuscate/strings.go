package gobfuscate

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	insecureRand "math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	canaryPrefix = "can://"
)

// ObfuscateStrings - Obfuscate strings in a given gopath, skips canaries
func ObfuscateStrings(gopath string) error {
	return filepath.Walk(gopath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) != GoExtension || info.IsDir() {
			return nil
		}
		if err := stringConstsToVar(path); err != nil {
			return err
		}

		set := token.NewFileSet()
		file, err := parser.ParseFile(set, path, nil, 0)
		if err != nil {
			return nil
		}
		contents, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		obfuscator := &stringObfuscator{Contents: contents}
		for _, decl := range file.Decls {
			ast.Walk(obfuscator, decl)
		}
		newCode, err := obfuscator.Obfuscate()
		if err != nil {
			return err
		}
		return ioutil.WriteFile(path, newCode, 0755)
	})
}

type stringObfuscator struct {
	Contents []byte
	Nodes    []*ast.BasicLit
}

func (s *stringObfuscator) Visit(n ast.Node) ast.Visitor {
	if lit, ok := n.(*ast.BasicLit); ok {
		if lit.Kind == token.STRING {
			s.Nodes = append(s.Nodes, lit)
		}
		return nil
	} else if decl, ok := n.(*ast.GenDecl); ok {
		if decl.Tok == token.CONST || decl.Tok == token.IMPORT {
			return nil
		}
	} else if _, ok := n.(*ast.StructType); ok {
		// Avoid messing with annotation strings.
		return nil
	}
	return s
}

func (s *stringObfuscator) Obfuscate() ([]byte, error) {
	sort.Sort(s)

	parsed := make([]string, s.Len())
	for i, n := range s.Nodes {
		var err error
		parsed[i], err = strconv.Unquote(n.Value)
		if err != nil {
			return nil, err
		}
	}

	var lastIndex int
	var result bytes.Buffer
	data := s.Contents
	for i, node := range s.Nodes {
		strVal := parsed[i]
		if strings.HasPrefix(strVal, canaryPrefix) {
			startIdx := node.Pos() - 1
			endIdx := node.End() - 1
			result.Write(data[lastIndex:startIdx])
			canary := fmt.Sprintf("\"http://%s\"", strVal[len(canaryPrefix):])
			result.Write([]byte(canary))
			lastIndex = int(endIdx)
		} else {
			startIdx := node.Pos() - 1
			endIdx := node.End() - 1
			result.Write(data[lastIndex:startIdx])
			result.Write(obfuscatedStringCode(strVal))
			lastIndex = int(endIdx)
		}
	}
	result.Write(data[lastIndex:])
	return result.Bytes(), nil
}

func (s *stringObfuscator) Len() int {
	return len(s.Nodes)
}

func (s *stringObfuscator) Swap(i, j int) {
	s.Nodes[i], s.Nodes[j] = s.Nodes[j], s.Nodes[i]
}

func (s *stringObfuscator) Less(i, j int) bool {
	return s.Nodes[i].Pos() < s.Nodes[j].Pos()
}

func obfuscatedStringCode(str string) []byte {
	var res bytes.Buffer
	res.WriteString("(func() string {\n")
	res.WriteString("mask := []byte(\"")
	mask := make([]byte, len(str))
	for i := range mask {
		mask[i] = byte(insecureRand.Intn(256))
		res.WriteString(fmt.Sprintf("\\x%02x", mask[i]))
	}
	res.WriteString("\")\nmaskedStr := []byte(\"")
	for i, x := range []byte(str) {
		res.WriteString(fmt.Sprintf("\\x%02x", x^mask[i]))
	}
	res.WriteString("\")\nres := make([]byte, ")
	res.WriteString(strconv.Itoa(len(mask)))
	res.WriteString(`)
        for i, m := range mask {
            res[i] = m ^ maskedStr[i]
        }
        return string(res)
        }())`)
	return res.Bytes()
}
