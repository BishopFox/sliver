package gotypes_test

import (
	"fmt"

	"github.com/mmcloughlin/avo/gotypes"
)

func ExampleParseSignature() {
	s, err := gotypes.ParseSignature("func(s string, n int) string")
	fmt.Println(s)
	fmt.Println(err)
	// Output:
	// (s string, n int) string
	// <nil>
}
