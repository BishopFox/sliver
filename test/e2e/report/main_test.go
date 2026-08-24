package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	e2ecoverage "github.com/bishopfox/sliver/test/e2e/coverage"
)

func TestRunWritesReportsAndFailsForFailedRecords(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := e2ecoverage.Target{OS: "linux", Arch: "386"}
	record := e2ecoverage.Record{
		TargetOS:    target.OS,
		TargetArch:  target.Arch,
		Transport:   "http",
		ImplantMode: "beacon",
		GRPCMethod:  "Download",
		Scenario:    "file, byte/line limits, and recursive directory",
		Status:      e2ecoverage.StatusFail,
		Detail:      "fixture mismatch",
	}
	if _, err := e2ecoverage.WriteTargetReports(filepath.Join(root, "input", "artifact"), target, []e2ecoverage.Record{record}); err != nil {
		t.Fatalf("WriteTargetReports() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"-input", filepath.Join(root, "input"),
		"-output", filepath.Join(root, "output"),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("run() code = %d, want 1; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "coverage contains 1 failed record") {
		t.Fatalf("stderr = %q, want failed record diagnostic", stderr.String())
	}
	if !strings.Contains(stderr.String(), "NOT RUN required cell") {
		t.Fatalf("stderr = %q, want NOT RUN diagnostic", stderr.String())
	}
	for _, name := range []string{e2ecoverage.GlobalJSONFilename, e2ecoverage.GlobalMarkdownFilename, e2ecoverage.CommandMarkdownFilename} {
		if _, err := os.Stat(filepath.Join(root, "output", name)); err != nil {
			t.Fatalf("output %s was not written: %v", name, err)
		}
	}
	if !strings.Contains(stdout.String(), filepath.Join(root, "output", e2ecoverage.CommandMarkdownFilename)) {
		t.Fatalf("stdout = %q, want command Markdown path", stdout.String())
	}
}

func TestRunWritesAllNotRunReportsForEmptyInput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "input"), 0o755); err != nil {
		t.Fatalf("Mkdir(input) error = %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"-input", filepath.Join(root, "input"),
		"-output", filepath.Join(root, "output"),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("run() code = %d, want 1; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "coverage contains 1734 NOT RUN required cell(s)") {
		t.Fatalf("stderr = %q, want all-NOT-RUN diagnostic", stderr.String())
	}
	for _, name := range []string{e2ecoverage.GlobalJSONFilename, e2ecoverage.GlobalMarkdownFilename, e2ecoverage.CommandMarkdownFilename} {
		if _, err := os.Stat(filepath.Join(root, "output", name)); err != nil {
			t.Fatalf("output %s was not written: %v", name, err)
		}
	}
}

func TestRunAllowsSyntheticPlatformSkips(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeCompleteComprehensiveReports(t, filepath.Join(root, "input"), false)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"-input", filepath.Join(root, "input"),
		"-output", filepath.Join(root, "output"),
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run() code = %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want no gate failures", stderr.String())
	}
	markdown, err := os.ReadFile(filepath.Join(root, "output", e2ecoverage.GlobalMarkdownFilename))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(markdown), "Expected platform skips") {
		t.Fatalf("summary omitted synthetic platform skips: %s", markdown)
	}
	if !strings.Contains(string(markdown), "| 0 | 570 | 0 |") {
		t.Fatalf("summary did not distinguish zero recorded skips from expected skips: %s", markdown)
	}
	commandMarkdown, err := os.ReadFile(filepath.Join(root, "output", e2ecoverage.CommandMarkdownFilename))
	if err != nil {
		t.Fatalf("ReadFile(command Markdown) error = %v", err)
	}
	if !strings.Contains(string(commandMarkdown), "| Ping | ✅ |") {
		t.Fatalf("command Markdown omitted passing Ping coverage: %s", commandMarkdown)
	}
}

func TestRunFailsForRecordedSkipOnSupportedCell(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	skippedIdentity := writeCompleteComprehensiveReports(t, filepath.Join(root, "input"), true)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"-input", filepath.Join(root, "input"),
		"-output", filepath.Join(root, "output"),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("run() code = %d, want 1; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	diagnostic := stderr.String()
	if !strings.Contains(diagnostic, "coverage contains 1 recorded skipped result(s) for catalog-supported cells") {
		t.Fatalf("stderr = %q, want recorded skip diagnostic", diagnostic)
	}
	if !strings.Contains(diagnostic, skippedIdentity.String()) {
		t.Fatalf("stderr = %q, want skipped identity %q", diagnostic, skippedIdentity)
	}
	if strings.Contains(diagnostic, "NOT RUN") || strings.Contains(diagnostic, "failed record") {
		t.Fatalf("stderr = %q, want recorded skip as the only gate failure", diagnostic)
	}
}

func writeCompleteComprehensiveReports(t *testing.T, root string, recordOneSkip bool) e2ecoverage.Identity {
	t.Helper()
	dimensions := e2ecoverage.ComprehensiveDimensions()
	var skippedIdentity e2ecoverage.Identity
	for _, target := range dimensions.Targets {
		records := make([]e2ecoverage.Record, 0)
		for _, expectation := range dimensions.Commands {
			if !supportsTarget(expectation, target) {
				continue
			}
			for _, transport := range dimensions.Transports {
				for _, mode := range dimensions.ImplantModes {
					record := e2ecoverage.Record{
						TargetOS:    target.OS,
						TargetArch:  target.Arch,
						Transport:   transport,
						ImplantMode: mode,
						GRPCMethod:  expectation.GRPCMethod,
						Scenario:    expectation.Scenario,
						Status:      e2ecoverage.StatusPass,
					}
					if recordOneSkip && skippedIdentity == (e2ecoverage.Identity{}) {
						record.Status = e2ecoverage.StatusSkip
						record.Detail = "fixture unavailable"
						skippedIdentity = record.Identity()
					}
					records = append(records, record)
				}
			}
		}
		dir := filepath.Join(root, target.OS+"-"+target.Arch)
		if _, err := e2ecoverage.WriteTargetReports(dir, target, records); err != nil {
			t.Fatalf("WriteTargetReports(%s/%s) error = %v", target.OS, target.Arch, err)
		}
	}
	return skippedIdentity
}

func supportsTarget(expectation e2ecoverage.CommandExpectation, target e2ecoverage.Target) bool {
	for _, supported := range expectation.SupportedTargets {
		if supported == target {
			return true
		}
	}
	return false
}
