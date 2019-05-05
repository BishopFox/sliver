// Command asmdecl reports mismatches between assembly files and Go declarations.
//
// Standalone version of the static analyzer in go vet.
package main

import (
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() { singlechecker.Main(asmdecl.Analyzer) }
