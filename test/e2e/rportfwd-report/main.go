package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	coverage "github.com/bishopfox/sliver/test/e2e/coverage"
	rportfwdcoverage "github.com/bishopfox/sliver/test/e2e/rportfwdcoverage"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("sliver-rportfwd-report", flag.ContinueOnError)
	flags.SetOutput(stderr)
	input := flags.String("input", ".", "directory recursively containing per-target rportfwd coverage JSON reports")
	output := flags.String("output", ".", "directory for aggregate rportfwd coverage reports")
	targetOS := flags.String("target-os", "", "verify only the named target operating system")
	targetArch := flags.String("target-arch", "", "verify only the named target architecture")
	transports := flags.String("transports", strings.Join(rportfwdcoverage.Transports(), ","), "comma-separated transports required in target verification mode")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected positional arguments: %v\n", flags.Args())
		return 2
	}
	if (*targetOS == "") != (*targetArch == "") {
		fmt.Fprintln(stderr, "-target-os and -target-arch must be provided together")
		return 2
	}

	if *targetOS != "" {
		selected, err := parseTransports(*transports)
		if err != nil {
			fmt.Fprintf(stderr, "parse transports: %v\n", err)
			return 2
		}
		target := coverage.Target{OS: *targetOS, Arch: *targetArch}
		if !supportedTarget(target) {
			fmt.Fprintf(stderr, "target %s/%s is outside the rportfwd matrix\n", target.OS, target.Arch)
			return 2
		}
		path := filepath.Join(*input, rportfwdcoverage.TargetJSONFilename(target))
		report, err := rportfwdcoverage.LoadTargetReport(path)
		if err != nil {
			fmt.Fprintf(stderr, "load target rportfwd coverage report: %v\n", err)
			return 1
		}
		if report.Target != target {
			fmt.Fprintf(
				stderr,
				"target report identity mismatch: got %s/%s, want %s/%s\n",
				report.Target.OS,
				report.Target.Arch,
				target.OS,
				target.Arch,
			)
			return 1
		}
		if err := report.ValidateComplete(selected); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(
			stdout,
			"Verified %s/%s rportfwd coverage: %d transports x %d scenarios passed\n",
			target.OS,
			target.Arch,
			len(selected),
			len(rportfwdcoverage.Scenarios()),
		)
		return 0
	}

	report, err := rportfwdcoverage.AggregateDirectory(*input)
	if err != nil {
		fmt.Fprintf(stderr, "aggregate rportfwd coverage reports: %v\n", err)
		return 1
	}
	paths, err := rportfwdcoverage.WriteGlobalReports(*output, report)
	if err != nil {
		fmt.Fprintf(stderr, "write rportfwd coverage reports: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Wrote %s\nWrote %s\n", paths.JSON, paths.Markdown)
	if err := report.GateError(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func parseTransports(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, fmt.Errorf("at least one transport is required")
	}
	parts := strings.Split(value, ",")
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
		if parts[index] == "" {
			return nil, fmt.Errorf("transport list contains an empty value")
		}
	}
	return rportfwdcoverage.NormalizeTransports(parts)
}

func supportedTarget(target coverage.Target) bool {
	for _, candidate := range rportfwdcoverage.Targets() {
		if candidate == target {
			return true
		}
	}
	return false
}
