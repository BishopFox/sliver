package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	e2ecoverage "github.com/bishopfox/sliver/test/e2e/coverage"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("sliver-e2e-report", flag.ContinueOnError)
	flags.SetOutput(stderr)
	input := flags.String("input", ".", "directory recursively containing per-target coverage JSON reports")
	output := flags.String("output", ".", "directory for coverage-summary.json, coverage-summary.md, and command-coverage.md")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected positional arguments: %v\n", flags.Args())
		return 2
	}

	report, err := e2ecoverage.AggregateDirectory(*input, e2ecoverage.ComprehensiveDimensions())
	if err != nil {
		fmt.Fprintf(stderr, "aggregate coverage reports: %v\n", err)
		return 1
	}
	paths, err := e2ecoverage.WriteGlobalReports(*output, report)
	if err != nil {
		fmt.Fprintf(stderr, "write coverage reports: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Wrote %s\nWrote %s\nWrote %s\n", paths.JSON, paths.Markdown, paths.CommandMarkdown)

	failed := report.FailedRecords()
	skipped := report.RecordedSkipRecords()
	missing := report.NotRunIdentities()
	if len(failed) > 0 {
		fmt.Fprintf(stderr, "coverage contains %d failed record(s):\n", len(failed))
		for _, record := range limitedRecords(failed, 50) {
			fmt.Fprintf(stderr, "- %s\n", record.Identity())
		}
		if len(failed) > 50 {
			fmt.Fprintf(stderr, "- ... and %d more\n", len(failed)-50)
		}
	}
	if len(skipped) > 0 {
		fmt.Fprintf(stderr, "coverage contains %d recorded skipped result(s) for catalog-supported cells:\n", len(skipped))
		for _, record := range limitedRecords(skipped, 50) {
			fmt.Fprintf(stderr, "- %s\n", record.Identity())
		}
		if len(skipped) > 50 {
			fmt.Fprintf(stderr, "- ... and %d more\n", len(skipped)-50)
		}
	}
	if len(missing) > 0 {
		fmt.Fprintf(stderr, "coverage contains %d NOT RUN required cell(s):\n", len(missing))
		for _, identity := range limitedIdentities(missing, 50) {
			fmt.Fprintf(stderr, "- %s\n", identity)
		}
		if len(missing) > 50 {
			fmt.Fprintf(stderr, "- ... and %d more\n", len(missing)-50)
		}
	}
	if len(failed) > 0 || len(skipped) > 0 || len(missing) > 0 {
		return 1
	}
	return 0
}

func limitedRecords(records []e2ecoverage.Record, limit int) []e2ecoverage.Record {
	if len(records) <= limit {
		return records
	}
	return records[:limit]
}

func limitedIdentities(identities []e2ecoverage.Identity, limit int) []e2ecoverage.Identity {
	if len(identities) <= limit {
		return identities
	}
	return identities[:limit]
}
