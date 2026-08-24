package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	shellcodecoverage "github.com/bishopfox/sliver/test/e2e/shellcodecoverage"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("sliver-shellcode-report", flag.ContinueOnError)
	flags.SetOutput(stderr)
	input := flags.String("input", ".", "directory recursively containing per-target shellcode coverage JSON reports")
	output := flags.String("output", ".", "directory for shellcode-coverage.json and shellcode-coverage.md")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected positional arguments: %v\n", flags.Args())
		return 2
	}

	report, err := shellcodecoverage.AggregateDirectory(*input)
	if err != nil {
		fmt.Fprintf(stderr, "aggregate shellcode coverage reports: %v\n", err)
		return 1
	}
	paths, err := shellcodecoverage.WriteGlobalReports(*output, report)
	if err != nil {
		fmt.Fprintf(stderr, "write shellcode coverage reports: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Wrote %s\nWrote %s\n", paths.JSON, paths.Markdown)

	if err := report.GateError(); err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	return 0
}
