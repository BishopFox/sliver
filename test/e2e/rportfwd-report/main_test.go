package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	coverage "github.com/bishopfox/sliver/test/e2e/coverage"
	rportfwdcoverage "github.com/bishopfox/sliver/test/e2e/rportfwdcoverage"
)

func TestRunTargetVerification(t *testing.T) {
	root := t.TempDir()
	target := coverage.Target{OS: "linux", Arch: "amd64"}
	if _, err := rportfwdcoverage.WriteTargetReports(root, target, targetRecords(target, []string{"mtls", "http"})); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	status := run([]string{
		"-input", root,
		"-target-os", target.OS,
		"-target-arch", target.Arch,
		"-transports", "http, mtls",
	}, &stdout, &stderr)
	if status != 0 {
		t.Fatalf("run() status = %d, stderr = %s", status, stderr.String())
	}
	if !strings.Contains(stdout.String(), "2 transports x 9 scenarios passed") {
		t.Fatalf("stdout = %q, want verification summary", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	status = run([]string{
		"-input", root,
		"-target-os", target.OS,
		"-target-arch", target.Arch,
		"-transports", "mtls,wg,http",
	}, &stdout, &stderr)
	if status != 1 || !strings.Contains(stderr.String(), "9 NOT RUN") {
		t.Fatalf("incomplete run() = %d, stderr = %q, want nine missing wg cells", status, stderr.String())
	}
}

func TestRunAggregateWritesDiagnosticsBeforeGate(t *testing.T) {
	t.Run("complete", func(t *testing.T) {
		root := t.TempDir()
		input := filepath.Join(root, "input")
		writeAllTargets(t, input)
		output := filepath.Join(root, "output")
		var stdout, stderr bytes.Buffer
		if status := run([]string{"-input", input, "-output", output}, &stdout, &stderr); status != 0 {
			t.Fatalf("run() status = %d, stderr = %s", status, stderr.String())
		}
		for _, name := range []string{rportfwdcoverage.GlobalJSONFilename, rportfwdcoverage.GlobalMarkdownFilename} {
			if _, err := os.Stat(filepath.Join(output, name)); err != nil {
				t.Fatalf("missing aggregate artifact %s: %v", name, err)
			}
		}
	})

	t.Run("missing", func(t *testing.T) {
		root := t.TempDir()
		output := filepath.Join(root, "output")
		var stdout, stderr bytes.Buffer
		status := run([]string{"-input", root, "-output", output}, &stdout, &stderr)
		if status != 1 || !strings.Contains(stderr.String(), "162 NOT RUN") {
			t.Fatalf("run() status = %d, stderr = %q, want complete NOT RUN gate", status, stderr.String())
		}
		if _, err := os.Stat(filepath.Join(output, rportfwdcoverage.GlobalJSONFilename)); err != nil {
			t.Fatalf("aggregate JSON was not written before gate failure: %v", err)
		}
	})
}

func TestRunRejectsMalformedInputWithoutPublishingAggregate(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input")
	if err := os.MkdirAll(input, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(input, "rportfwd-coverage-linux-amd64.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "output")
	var stdout, stderr bytes.Buffer
	status := run([]string{"-input", input, "-output", output}, &stdout, &stderr)
	if status != 1 || !strings.Contains(stderr.String(), "aggregate rportfwd coverage reports") {
		t.Fatalf("run() status = %d, stderr = %q, want aggregate error", status, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(output, rportfwdcoverage.GlobalJSONFilename)); !os.IsNotExist(err) {
		t.Fatalf("malformed input published aggregate JSON: stat error = %v", err)
	}
}

func TestRunFlagValidation(t *testing.T) {
	tests := [][]string{
		{"-target-os", "linux"},
		{"-target-arch", "amd64"},
		{"-target-os", "linux", "-target-arch", "amd64", "-transports", ""},
		{"-target-os", "linux", "-target-arch", "amd64", "-transports", "bogus"},
		{"-target-os", "../../tmp", "-target-arch", "amd64"},
		{"unexpected"},
	}
	for _, arguments := range tests {
		var stdout, stderr bytes.Buffer
		if status := run(arguments, &stdout, &stderr); status != 2 {
			t.Fatalf("run(%v) status = %d, stderr = %q, want usage failure", arguments, status, stderr.String())
		}
	}
}

func writeAllTargets(t *testing.T, root string) {
	t.Helper()
	for _, target := range rportfwdcoverage.Targets() {
		directory := filepath.Join(root, fmt.Sprintf("artifact-%s-%s", target.OS, target.Arch))
		if _, err := rportfwdcoverage.WriteTargetReports(directory, target, targetRecords(target, rportfwdcoverage.Transports())); err != nil {
			t.Fatalf("WriteTargetReports(%s/%s) error = %v", target.OS, target.Arch, err)
		}
	}
}

func targetRecords(target coverage.Target, transports []string) []rportfwdcoverage.Record {
	records := make([]rportfwdcoverage.Record, 0, len(transports)*len(rportfwdcoverage.Scenarios()))
	for _, transport := range transports {
		for _, scenario := range rportfwdcoverage.Scenarios() {
			records = append(records, rportfwdcoverage.Record{
				Target:    target,
				Transport: transport,
				Scenario:  scenario,
				Status:    coverage.StatusPass,
			})
		}
	}
	return records
}
