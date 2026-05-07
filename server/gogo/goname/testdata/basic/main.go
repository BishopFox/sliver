package main

import (
	"fmt"
	"example.com/old"
	oldpkg "example.com/old/pkg/foo"
	_ "example.com/oldish/pkg"
)

const example = "example.com/old/pkg/foo"

func main() {
	fmt.Println(oldpkg.Name, example)
}
