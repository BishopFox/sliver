package main

import (
	"fmt"
	"os"

	"github.com/bishopfox/sliver/util/assets"
)

func main() {
	if err := assets.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
