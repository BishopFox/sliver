package coverage

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAggregateDirectoryBuildsCrossProduct(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	linux := Record{
		TargetOS:    "linux",
		TargetArch:  "amd64",
		Transport:   "mtls",
		ImplantMode: "session",
		GRPCMethod:  "/sliverpb.Sliver/Ls",
		Scenario:    "basic listing",
		Status:      StatusPass,
		Duration:    300 * time.Millisecond,
	}
	windows := linux
	windows.TargetOS = "windows"
	windows.Transport = "http"
	windows.Status = StatusSkip
	windows.Detail = "not supported by fixture"
	writeTestTargetReport(t, filepath.Join(root, "linux-artifact"), Target{OS: "linux", Arch: "amd64"}, []Record{linux})
	writeTestTargetReport(t, filepath.Join(root, "windows-artifact"), Target{OS: "windows", Arch: "amd64"}, []Record{windows})

	dimensions := Dimensions{
		Targets: []Target{
			{OS: "linux", Arch: "amd64"},
			{OS: "windows", Arch: "amd64"},
		},
		Transports:   []string{"mtls", "http"},
		ImplantModes: []string{"session"},
	}
	report, err := AggregateDirectory(root, dimensions)
	if err != nil {
		t.Fatalf("AggregateDirectory() error = %v", err)
	}
	if got, want := len(report.Matrix), 1; got != want {
		t.Fatalf("matrix rows = %d, want %d", got, want)
	}
	if got, want := len(report.Matrix[0].Cells), 4; got != want {
		t.Fatalf("matrix cells = %d, want %d", got, want)
	}
	if report.Summary.Recorded != 2 || report.Summary.Pass != 1 || report.Summary.Skip != 1 || report.Summary.Fail != 0 || report.Summary.NotRun != 2 || report.Summary.TotalCells != 4 {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
	if failed := report.FailedRecords(); len(failed) != 0 {
		t.Fatalf("FailedRecords() = %v, want none", failed)
	}

	output := filepath.Join(root, "summary")
	paths, err := WriteGlobalReports(output, report)
	if err != nil {
		t.Fatalf("WriteGlobalReports() error = %v", err)
	}
	firstJSON := readTestFile(t, paths.JSON)
	firstMarkdown := readTestFile(t, paths.Markdown)
	if _, err := WriteGlobalReports(output, report); err != nil {
		t.Fatalf("second WriteGlobalReports() error = %v", err)
	}
	if !bytes.Equal(firstJSON, readTestFile(t, paths.JSON)) {
		t.Fatal("global JSON report changed for identical input")
	}
	if !bytes.Equal(firstMarkdown, readTestFile(t, paths.Markdown)) {
		t.Fatal("global Markdown report changed for identical input")
	}
	markdown := string(firstMarkdown)
	if !strings.Contains(markdown, "## mtls / session") || !strings.Contains(markdown, "NOT RUN") {
		t.Fatalf("global Markdown is missing cross-product coverage: %s", markdown)
	}
	malformed := report
	malformed.Summary.NotRun++
	if _, err := WriteGlobalReports(filepath.Join(root, "malformed"), malformed); err == nil || !strings.Contains(err.Error(), "summary does not match") {
		t.Fatalf("WriteGlobalReports(malformed) error = %v, want summary validation", err)
	}
}

func TestAggregateCatalogCreatesNotRunAndExpectedPlatformSkip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	linuxTarget := Target{OS: "linux", Arch: "amd64"}
	windowsTarget := Target{OS: "windows", Arch: "amd64"}
	writeTestTargetReport(t, root, linuxTarget, nil)
	expectation := CommandExpectation{
		Command:           Command{GRPCMethod: "Chmod", Scenario: "fixture mode"},
		SupportedTargets:  []Target{linuxTarget},
		UnsupportedReason: "supported only on Linux",
	}
	dimensions := Dimensions{
		Targets:      []Target{linuxTarget, windowsTarget},
		Transports:   []string{TransportMTLS},
		ImplantModes: []string{ImplantModeSession},
		Commands:     []CommandExpectation{expectation},
	}

	report, err := AggregateDirectory(root, dimensions)
	if err != nil {
		t.Fatalf("AggregateDirectory() error = %v", err)
	}
	if len(report.Matrix) != 1 || len(report.Matrix[0].Cells) != 2 {
		t.Fatalf("unexpected matrix shape: %+v", report.Matrix)
	}
	if got := report.Matrix[0].Cells[0].Status; got != MatrixStatusNotRun {
		t.Fatalf("supported absent cell status = %q, want %q", got, MatrixStatusNotRun)
	}
	synthetic := report.Matrix[0].Cells[1]
	if synthetic.Status != string(StatusSkip) || synthetic.Recorded || synthetic.Detail != expectation.UnsupportedReason {
		t.Fatalf("synthetic platform skip = %+v", synthetic)
	}
	if report.Summary.Recorded != 0 || report.Summary.Skip != 1 || report.Summary.NotRun != 1 || report.Summary.TotalCells != 2 {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
	missing := report.NotRunIdentities()
	if len(missing) != 1 || missing[0].TargetOS != "linux" || missing[0].GRPCMethod != "Chmod" {
		t.Fatalf("NotRunIdentities() = %+v", missing)
	}

	linuxRecord := Record{
		TargetOS:    linuxTarget.OS,
		TargetArch:  linuxTarget.Arch,
		Transport:   TransportMTLS,
		ImplantMode: ImplantModeSession,
		GRPCMethod:  expectation.GRPCMethod,
		Scenario:    expectation.Scenario,
		Status:      StatusPass,
	}
	writeTestTargetReport(t, root, linuxTarget, []Record{linuxRecord})
	report, err = AggregateDirectory(root, dimensions)
	if err != nil {
		t.Fatalf("AggregateDirectory() after record error = %v", err)
	}
	if report.Summary.Recorded != 1 || report.Summary.Pass != 1 || report.Summary.Skip != 1 || report.Summary.NotRun != 0 {
		t.Fatalf("unexpected complete summary: %+v", report.Summary)
	}
	paths, err := WriteGlobalReports(filepath.Join(root, "summary"), report)
	if err != nil {
		t.Fatalf("WriteGlobalReports() error = %v", err)
	}
	if markdown := string(readTestFile(t, paths.Markdown)); !strings.Contains(markdown, "Expected platform skips") || !strings.Contains(markdown, expectation.UnsupportedReason) {
		t.Fatalf("Markdown omitted expected skip reason: %s", markdown)
	}
}

func TestAggregateEmptyDirectoryCreatesComprehensiveNotRunReport(t *testing.T) {
	t.Parallel()

	report, err := AggregateDirectory(t.TempDir(), ComprehensiveDimensions())
	if err != nil {
		t.Fatalf("AggregateDirectory() error = %v", err)
	}
	if report.Summary.Recorded != 0 || report.Summary.NotRun != 1320 || report.Summary.Skip != 696 || report.Summary.TotalCells != 2016 {
		t.Fatalf("unexpected empty-input summary: %+v", report.Summary)
	}
	if got := len(report.NotRunIdentities()); got != 1320 {
		t.Fatalf("NotRunIdentities() count = %d, want 1320", got)
	}
	if _, err := AggregateDirectory(t.TempDir(), Dimensions{}); err == nil || !strings.Contains(err.Error(), "no per-target coverage JSON reports") {
		t.Fatalf("AggregateDirectory() with inferred empty dimensions error = %v, want no-reports error", err)
	}
}

func TestRecordedSkipRecordsExcludesSyntheticPlatformSkips(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	linuxTarget := Target{OS: "linux", Arch: "amd64"}
	windowsTarget := Target{OS: "windows", Arch: "amd64"}
	expectation := CommandExpectation{
		Command:           Command{GRPCMethod: "Chmod", Scenario: "fixture mode"},
		SupportedTargets:  []Target{linuxTarget},
		UnsupportedReason: "supported only on Linux",
	}
	recordedSkip := Record{
		TargetOS:    linuxTarget.OS,
		TargetArch:  linuxTarget.Arch,
		Transport:   TransportMTLS,
		ImplantMode: ImplantModeSession,
		GRPCMethod:  expectation.GRPCMethod,
		Scenario:    expectation.Scenario,
		Status:      StatusSkip,
		Detail:      "fixture unavailable",
	}
	writeTestTargetReport(t, root, linuxTarget, []Record{recordedSkip})

	report, err := AggregateDirectory(root, Dimensions{
		Targets:      []Target{linuxTarget, windowsTarget},
		Transports:   []string{TransportMTLS},
		ImplantModes: []string{ImplantModeSession},
		Commands:     []CommandExpectation{expectation},
	})
	if err != nil {
		t.Fatalf("AggregateDirectory() error = %v", err)
	}
	skipped := report.RecordedSkipRecords()
	if len(skipped) != 1 || skipped[0].Identity() != recordedSkip.Identity() {
		t.Fatalf("RecordedSkipRecords() = %+v, want only %+v", skipped, recordedSkip.Identity())
	}
	if report.Summary.Skip != 2 {
		t.Fatalf("summary skips = %d, want one recorded and one synthetic", report.Summary.Skip)
	}
	if synthetic := report.Matrix[0].Cells[1]; synthetic.Recorded || synthetic.Status != string(StatusSkip) {
		t.Fatalf("synthetic platform skip = %+v", synthetic)
	}

	paths, err := WriteGlobalReports(filepath.Join(root, "summary"), report)
	if err != nil {
		t.Fatalf("WriteGlobalReports() error = %v", err)
	}
	markdown := string(readTestFile(t, paths.Markdown))
	if !strings.Contains(markdown, "| 1 | 0 | 0 | 1 | 1 | 0 | 2 |") {
		t.Fatalf("Markdown did not distinguish recorded and expected skips: %s", markdown)
	}
	if !strings.Contains(markdown, "Every result in this section fails the comprehensive coverage gate.") {
		t.Fatalf("Markdown did not surface recorded skip as a gate failure: %s", markdown)
	}
}

func TestAggregateStrictCatalogRejectsIdentityDrift(t *testing.T) {
	t.Parallel()

	target := Target{OS: "linux", Arch: "amd64"}
	expectation := CommandExpectation{
		Command:          Command{GRPCMethod: "Ping", Scenario: "nonce"},
		SupportedTargets: []Target{target},
	}
	dimensions := Dimensions{
		Targets:      []Target{target},
		Transports:   []string{TransportMTLS},
		ImplantModes: []string{ImplantModeSession},
		Commands:     []CommandExpectation{expectation},
	}
	record := Record{
		TargetOS:    target.OS,
		TargetArch:  target.Arch,
		Transport:   TransportMTLS,
		ImplantMode: "sessoin",
		GRPCMethod:  expectation.GRPCMethod,
		Scenario:    expectation.Scenario,
		Status:      StatusPass,
	}
	root := t.TempDir()
	writeTestTargetReport(t, root, target, []Record{record})
	if _, err := AggregateDirectory(root, dimensions); err == nil || !strings.Contains(err.Error(), "unexpected implant mode") {
		t.Fatalf("AggregateDirectory() error = %v, want strict mode rejection", err)
	}
}

func TestAggregateDirectoryRejectsDuplicateIdentityAcrossReports(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	record := Record{
		TargetOS:    "darwin",
		TargetArch:  "arm64",
		Transport:   "wg",
		ImplantMode: "beacon",
		GRPCMethod:  "/sliverpb.Sliver/Pwd",
		Scenario:    "working directory",
		Status:      StatusPass,
	}
	target := Target{OS: record.TargetOS, Arch: record.TargetArch}
	writeTestTargetReport(t, filepath.Join(root, "artifact-a"), target, []Record{record})
	writeTestTargetReport(t, filepath.Join(root, "artifact-b"), target, []Record{record})

	if _, err := AggregateDirectory(root, Dimensions{}); err == nil || !strings.Contains(err.Error(), "duplicate record identity") {
		t.Fatalf("AggregateDirectory() error = %v, want duplicate identity", err)
	}
}

func TestAggregateDirectoryRejectsMalformedRequiredIdentity(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "coverage-linux-amd64.json")
	data := []byte(`{
  "schema_version": 1,
  "kind": "sliver-e2e-target-coverage",
  "target": {"os": "linux", "arch": "amd64"},
  "records": [{
    "target_os": "linux",
    "target_arch": "amd64",
    "transport": "",
    "implant_mode": "session",
    "grpc_method": "/sliverpb.Sliver/Ls",
    "scenario": "basic listing",
    "status": "pass",
    "duration_ns": 0,
    "detail": ""
  }]
}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := AggregateDirectory(root, Dimensions{}); err == nil || !strings.Contains(err.Error(), "transport is required") {
		t.Fatalf("AggregateDirectory() error = %v, want malformed transport", err)
	}
}

func TestComprehensiveDimensions(t *testing.T) {
	t.Parallel()

	dimensions := ComprehensiveDimensions()
	if len(dimensions.Targets) != 6 {
		t.Fatalf("targets = %d, want 6", len(dimensions.Targets))
	}
	if got := strings.Join(dimensions.Transports, ","); got != "mtls,wg,http" {
		t.Fatalf("transports = %q", got)
	}
	if got := strings.Join(dimensions.ImplantModes, ","); got != "session,beacon" {
		t.Fatalf("implant modes = %q", got)
	}
	if len(dimensions.Commands) != 56 {
		t.Fatalf("commands = %d, want 56", len(dimensions.Commands))
	}
	for _, expectation := range dimensions.Commands {
		if len(expectation.SupportedTargets) == 0 {
			t.Fatalf("catalog entry has no explicit targets: %+v", expectation)
		}
	}
}

func TestComprehensiveCatalogOPFORSupport(t *testing.T) {
	t.Parallel()

	windowsAMD64 := Target{OS: "windows", Arch: "amd64"}
	windowsARM64 := Target{OS: "windows", Arch: "arm64"}
	opforScenarios := map[string]bool{
		"OPFOR Cat CNA reads isolated test file":                 true,
		"OPFOR FirefoxDump CNA finds no host profiles":           true,
		"OPFOR callback preserves ordered typed binary channels": true,
		"typed BOF partial output retained on callback error":    true,
		"malformed BOF returns bounded loader error":             true,
		"finite BOF deadline returns and target recovers":        true,
		"OPFOR FindDotnet CNA read-only process inventory":       true,
		"OPFOR FindSysmon CNA read-only registry probe":          true,
	}

	found := map[string]bool{}
	for _, expectation := range ComprehensiveCatalog() {
		_, isOPFOR := opforScenarios[expectation.Scenario]
		if !isOPFOR {
			continue
		}
		found[expectation.Scenario] = true
		if expectation.GRPCMethod != "CallExtension" {
			t.Errorf("OPFOR scenario %q method = %q, want CallExtension", expectation.Scenario, expectation.GRPCMethod)
		}
		if !expectation.supports(windowsAMD64) {
			t.Errorf("OPFOR scenario %q does not support windows/amd64", expectation.Scenario)
		}
		if expectation.supports(windowsARM64) {
			t.Errorf("OPFOR scenario %q unexpectedly supports windows/arm64", expectation.Scenario)
		}
		if !strings.Contains(expectation.UnsupportedReason, "Windows") && !strings.Contains(expectation.UnsupportedReason, "windows") {
			t.Errorf("OPFOR scenario %q skip reason does not explain its Windows boundary: %q", expectation.Scenario, expectation.UnsupportedReason)
		}
		if !strings.Contains(expectation.UnsupportedReason, "arm64") {
			t.Errorf("OPFOR scenario %q skip reason does not explain windows/arm64: %q", expectation.Scenario, expectation.UnsupportedReason)
		}
	}
	if len(found) != len(opforScenarios) {
		t.Fatalf("found %d OPFOR scenarios, want %d: %#v", len(found), len(opforScenarios), found)
	}
}

func writeTestTargetReport(t *testing.T, dir string, target Target, records []Record) {
	t.Helper()
	if _, err := WriteTargetReports(dir, target, records); err != nil {
		t.Fatalf("WriteTargetReports() error = %v", err)
	}
}
