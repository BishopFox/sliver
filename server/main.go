package main

import (
	"flag"
	"fmt"
	"os"

	"sliver/server/assets"
	"sliver/server/console"
)

var (
	sliverServerVersion = fmt.Sprintf("0.0.4 - %s", assets.GitVersion)
)

func main() {
	unpack := flag.Bool("unpack", false, "force unpack assets")
	version := flag.Bool("version", false, "print version number")
	flag.Parse()

	if *version {
		fmt.Printf("v%s\n", sliverServerVersion)
		os.Exit(0)
	}

	assets.Setup(*unpack)
	console.Start()
}
