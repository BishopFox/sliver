package main

import (
	"fmt"
	bar "foo/bar"
	_ "foo/bar/baz"
	_ "foo/bar/qux"
	_ "foo/barista"
	"other/pkg"
)

const example = "foo/bar/baz"

func main() {
	fmt.Println(bar.Name, example)
}
