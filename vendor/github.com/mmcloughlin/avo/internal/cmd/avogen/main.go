// Command avogen generates code based on the instruction database.
package main

import (
	"flag"
	"go/build"
	"log"
	"os"
	"path/filepath"

	"github.com/mmcloughlin/avo/internal/gen"
	"github.com/mmcloughlin/avo/internal/inst"
	"github.com/mmcloughlin/avo/internal/load"
	"github.com/mmcloughlin/avo/printer"
)

var generators = map[string]gen.Builder{
	"asmtest":    gen.NewAsmTest,
	"godata":     gen.NewGoData,
	"godatatest": gen.NewGoDataTest,
	"ctors":      gen.NewCtors,
	"ctorstest":  gen.NewCtorsTest,
	"build":      gen.NewBuild,
	"mov":        gen.NewMOV,
}

// Command-line flags.
var (
	bootstrap = flag.Bool("bootstrap", false, "regenerate instruction list from original data")
	datadir   = flag.String(
		"data",
		filepath.Join(build.Default.GOPATH, "src/github.com/mmcloughlin/avo/internal/data"),
		"path to data directory",
	)
	output = flag.String("output", "", "path to output file (default stdout)")
)

func main() {
	log.SetPrefix("avogen: ")
	log.SetFlags(0)
	flag.Parse()

	// Build generator.
	t := flag.Arg(0)
	builder := generators[t]
	if builder == nil {
		log.Fatalf("unknown generator type '%s'", t)
	}

	g := builder(printer.NewArgvConfig())

	// Determine output writer.
	w := os.Stdout
	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		w = f
	}

	// Load instructions.
	is := inst.Instructions
	if *bootstrap {
		log.Printf("bootstrap: loading instructions from data directory %s", *datadir)
		l := load.NewLoaderFromDataDir(*datadir)
		r, err := l.Load()
		if err != nil {
			log.Fatal(err)
		}
		is = r
	}

	// Generate output.
	b, err := g.Generate(is)
	if err != nil {
		log.Fatal(err)
	}

	// Write.
	if _, err := w.Write(b); err != nil {
		log.Fatal(err)
	}
}
