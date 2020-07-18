package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

var (
	protoPath = flag.String("proto-path",
		"/usr/share/xcb", "path to directory of X protocol XML files")
	gofmt = flag.Bool("gofmt", true,
		"When disabled, gofmt will not be run before outputting Go code")
)

func usage() {
	basename := os.Args[0]
	if lastSlash := strings.LastIndex(basename, "/"); lastSlash > -1 {
		basename = basename[lastSlash+1:]
	}
	log.Printf("Usage: %s [flags] xml-file", basename)
	flag.PrintDefaults()
	os.Exit(1)
}

func init() {
	log.SetFlags(0)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		log.Printf("A single XML protocol file can be processed at once.")
		flag.Usage()
	}

	// Read the single XML file into []byte
	xmlBytes, err := ioutil.ReadFile(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}

	// Initialize the buffer, parse it, and filter it through gofmt.
	c := newContext()
	c.Morph(xmlBytes)

	if !*gofmt {
		c.out.WriteTo(os.Stdout)
	} else {
		cmdGofmt := exec.Command("gofmt")
		cmdGofmt.Stdin = c.out
		cmdGofmt.Stdout = os.Stdout
		cmdGofmt.Stderr = os.Stderr
		err = cmdGofmt.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
}
