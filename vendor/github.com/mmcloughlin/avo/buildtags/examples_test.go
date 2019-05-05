package buildtags_test

import (
	"fmt"

	"github.com/mmcloughlin/avo/buildtags"
)

func ExampleParseConstraint() {
	c, err := buildtags.ParseConstraint("a,!b c")
	fmt.Print(c.GoString())
	fmt.Println(err)
	// Output:
	// // +build a,!b c
	// <nil>
}
