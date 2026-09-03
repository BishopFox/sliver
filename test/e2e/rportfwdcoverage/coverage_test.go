package rportfwdcoverage_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	coverage "github.com/bishopfox/sliver/test/e2e/coverage"
	rportfwdcoverage "github.com/bishopfox/sliver/test/e2e/rportfwdcoverage"
)

func TestFixedSecurityCatalog(t *testing.T) {
	wantScenarios := []string{
		"invalid-destination",
		"invalid-session-stop-isolation",
		"metadata-authority",
		"bidirectional-echo",
		"sequential-connections",
		"concurrent-connections",
		"established-before-stop",
		"stop-closes-established",
		"disconnect-closes-established",
	}
	gotScenarios := rportfwdcoverage.Scenarios()
	if strings.Join(gotScenarios, ",") != strings.Join(wantScenarios, ",") {
		t.Fatalf("Scenarios() = %v, want %v", gotScenarios, wantScenarios)
	}
	if got := len(rportfwdcoverage.Targets()) * len(rportfwdcoverage.Transports()) * len(gotScenarios); got != rportfwdcoverage.RequiredCombinations {
		t.Fatalf("matrix size = %d, RequiredCombinations = %d", got, rportfwdcoverage.RequiredCombinations)
	}
	if rportfwdcoverage.RequiredCombinations != 162 {
		t.Fatalf("RequiredCombinations = %d, want 162", rportfwdcoverage.RequiredCombinations)
	}
}

func TestRecorderCompleteValidationAndDuplicateRejection(t *testing.T) {
	target := coverage.Target{OS: "linux", Arch: "amd64"}
	recorder, err := rportfwdcoverage.NewRecorder(target)
	if err != nil {
		t.Fatalf("NewRecorder() error = %v", err)
	}
	for _, transport := range []string{rportfwdcoverage.TransportMTLS, rportfwdcoverage.TransportHTTP} {
		for _, scenario := range rportfwdcoverage.Scenarios() {
			if err := recorder.Add(rportfwdcoverage.Observation{
				Transport: transport,
				Scenario:  scenario,
				Status:    coverage.StatusPass,
				Duration:  time.Millisecond,
			}); err != nil {
				t.Fatalf("Add(%s, %s) error = %v", transport, scenario, err)
			}
		}
	}
	if err := recorder.ValidateComplete([]string{"http", "mtls"}); err != nil {
		t.Fatalf("ValidateComplete() error = %v", err)
	}
	if err := recorder.Add(rportfwdcoverage.Observation{
		Transport: rportfwdcoverage.TransportMTLS,
		Scenario:  rportfwdcoverage.ScenarioBidirectionalEcho,
		Status:    coverage.StatusPass,
	}); err == nil || !strings.Contains(err.Error(), "duplicate record identity") {
		t.Fatalf("duplicate Add() error = %v, want duplicate diagnostic", err)
	}
	if err := recorder.ValidateComplete([]string{"mtls"}); err == nil || !strings.Contains(err.Error(), "unselected transport") {
		t.Fatalf("narrow ValidateComplete() error = %v, want unselected-transport diagnostic", err)
	}
}

func TestValidateCompleteReportsFailuresAndMissingCells(t *testing.T) {
	target := coverage.Target{OS: "windows", Arch: "arm64"}
	records := []rportfwdcoverage.Record{
		{
			Target:    target,
			Transport: rportfwdcoverage.TransportWG,
			Scenario:  rportfwdcoverage.ScenarioBidirectionalEcho,
			Status:    coverage.StatusFail,
			Detail:    "relay closed early",
		},
	}
	report := rportfwdcoverage.TargetReport{
		SchemaVersion: rportfwdcoverage.SchemaVersion,
		Kind:          rportfwdcoverage.TargetReportKind,
		Target:        target,
		Records:       records,
	}
	err := report.ValidateComplete([]string{"wg"})
	if err == nil || !strings.Contains(err.Error(), "1 FAIL, 8 NOT RUN") {
		t.Fatalf("ValidateComplete() error = %v, want one failure and eight missing cells", err)
	}
}

func TestTargetReportWriteLoadIsDeterministicAndSupportsPartialReports(t *testing.T) {
	target := coverage.Target{OS: "darwin", Arch: "arm64"}
	recorder, err := rportfwdcoverage.NewRecorder(target)
	if err != nil {
		t.Fatalf("NewRecorder() error = %v", err)
	}
	if err := recorder.Add(rportfwdcoverage.Observation{
		Transport: rportfwdcoverage.TransportHTTP,
		Scenario:  rportfwdcoverage.ScenarioStopClosesEstablished,
		Status:    coverage.StatusFail,
		Duration:  2 * time.Second,
		Detail:    "listener remained reachable",
	}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	root := t.TempDir()
	first, err := recorder.Write(filepath.Join(root, "first"))
	if err != nil {
		t.Fatalf("Write(first) error = %v", err)
	}
	second, err := recorder.Write(filepath.Join(root, "second"))
	if err != nil {
		t.Fatalf("Write(second) error = %v", err)
	}
	firstJSON, err := os.ReadFile(first.JSON)
	if err != nil {
		t.Fatal(err)
	}
	secondJSON, err := os.ReadFile(second.JSON)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstJSON, secondJSON) {
		t.Fatalf("JSON output is not deterministic\nfirst: %s\nsecond: %s", firstJSON, secondJSON)
	}
	loaded, err := rportfwdcoverage.LoadTargetReport(first.JSON)
	if err != nil {
		t.Fatalf("LoadTargetReport() error = %v", err)
	}
	if loaded.Target != target || len(loaded.Records) != 1 {
		t.Fatalf("loaded report = %+v, want target and one record", loaded)
	}

	emptyRecorder, err := rportfwdcoverage.NewRecorder(coverage.Target{OS: "windows", Arch: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	emptyPaths, err := emptyRecorder.Write(filepath.Join(root, "empty"))
	if err != nil {
		t.Fatalf("Write(empty) error = %v", err)
	}
	if _, err := rportfwdcoverage.LoadTargetReport(emptyPaths.JSON); err != nil {
		t.Fatalf("LoadTargetReport(empty) error = %v", err)
	}
}

func TestLoadTargetReportRejectsMalformedJSONContracts(t *testing.T) {
	targetJSON := `{"schema_version":1,"kind":"sliver-e2e-rportfwd-target-coverage","target":{"os":"linux","arch":"amd64"},"records":[]}`
	tests := []struct {
		name string
		data string
		want string
	}{
		{name: "unknown", data: strings.Replace(targetJSON, `"records":[]`, `"records":[],"extra":true`, 1), want: "unknown field"},
		{name: "duplicate case folded", data: strings.Replace(targetJSON, `"kind":`, `"Kind":"shadow","kind":`, 1), want: "duplicate JSON field"},
		{name: "missing", data: strings.Replace(targetJSON, `,"records":[]`, ``, 1), want: "missing field"},
		{name: "null records", data: strings.Replace(targetJSON, `"records":[]`, `"records":null`, 1), want: "records must be an array"},
		{name: "trailing", data: targetJSON + `{}`, want: "trailing JSON"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "rportfwd-coverage-linux-amd64.json")
			if err := os.WriteFile(path, []byte(test.data), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := rportfwdcoverage.LoadTargetReport(path); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("LoadTargetReport() error = %v, want diagnostic containing %q", err, test.want)
			}
		})
	}
}

func TestAggregateCompleteMatrixAndWriteReports(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input")
	writeCompleteMatrix(t, input)

	report, err := rportfwdcoverage.AggregateDirectory(input)
	if err != nil {
		t.Fatalf("AggregateDirectory() error = %v", err)
	}
	if report.Summary.Required != rportfwdcoverage.RequiredCombinations || report.Summary.Pass != rportfwdcoverage.RequiredCombinations {
		t.Fatalf("summary = %+v, want %d required passes", report.Summary, rportfwdcoverage.RequiredCombinations)
	}
	if report.Summary.NotRun != 0 || report.Summary.Fail != 0 {
		t.Fatalf("summary = %+v, want no failures or missing cells", report.Summary)
	}
	if err := report.GateError(); err != nil {
		t.Fatalf("GateError() = %v, want nil", err)
	}
	paths, err := rportfwdcoverage.WriteGlobalReports(filepath.Join(root, "output"), report)
	if err != nil {
		t.Fatalf("WriteGlobalReports() error = %v", err)
	}
	for _, path := range []string{paths.JSON, paths.Markdown} {
		if info, err := os.Stat(path); err != nil || info.Size() == 0 {
			t.Fatalf("artifact %s stat = (%v, %v), want nonempty file", path, info, err)
		}
	}
}

func TestAggregateRetainsFailuresAndMissingTargetsForGate(t *testing.T) {
	root := t.TempDir()
	target := coverage.Target{OS: "linux", Arch: "amd64"}
	records := completeTargetRecords(target)
	records[0].Status = coverage.StatusFail
	records[0].Detail = "metadata mismatch"
	if _, err := rportfwdcoverage.WriteTargetReports(root, target, records); err != nil {
		t.Fatal(err)
	}

	report, err := rportfwdcoverage.AggregateDirectory(root)
	if err != nil {
		t.Fatalf("AggregateDirectory() error = %v", err)
	}
	if report.Summary.Fail != 1 || report.Summary.NotRun != rportfwdcoverage.RequiredCombinations-len(records) {
		t.Fatalf("summary = %+v, want one failure and all other targets NOT RUN", report.Summary)
	}
	if len(report.FailedRecords()) != 1 || len(report.NotRunIdentities()) != report.Summary.NotRun {
		t.Fatalf("failure/missing accessors disagree with summary %+v", report.Summary)
	}
	if err := report.GateError(); err == nil || !strings.Contains(err.Error(), "1 FAIL") || !strings.Contains(err.Error(), "135 NOT RUN") {
		t.Fatalf("GateError() = %v, want explicit fail and NOT RUN counts", err)
	}
}

func TestAggregateRejectsDuplicateTargetReportsAndMisnamedArtifacts(t *testing.T) {
	target := coverage.Target{OS: "linux", Arch: "arm64"}
	t.Run("duplicate target", func(t *testing.T) {
		root := t.TempDir()
		for _, directory := range []string{"one", "two"} {
			if _, err := rportfwdcoverage.WriteTargetReports(filepath.Join(root, directory), target, nil); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := rportfwdcoverage.AggregateDirectory(root); err == nil || !strings.Contains(err.Error(), "duplicate target report") {
			t.Fatalf("AggregateDirectory() error = %v, want duplicate-target diagnostic", err)
		}
	})

	t.Run("misnamed", func(t *testing.T) {
		root := t.TempDir()
		paths, err := rportfwdcoverage.WriteTargetReports(root, target, nil)
		if err != nil {
			t.Fatal(err)
		}
		wrong := filepath.Join(root, "rportfwd-coverage-wrong.json")
		if err := os.Rename(paths.JSON, wrong); err != nil {
			t.Fatal(err)
		}
		if _, err := rportfwdcoverage.AggregateDirectory(root); err == nil || !strings.Contains(err.Error(), "must be named") {
			t.Fatalf("AggregateDirectory() error = %v, want filename diagnostic", err)
		}
	})
}

func writeCompleteMatrix(t *testing.T, root string) {
	t.Helper()
	for _, target := range rportfwdcoverage.Targets() {
		directory := filepath.Join(root, fmt.Sprintf("artifact-%s-%s", target.OS, target.Arch))
		if _, err := rportfwdcoverage.WriteTargetReports(directory, target, completeTargetRecords(target)); err != nil {
			t.Fatalf("WriteTargetReports(%s/%s) error = %v", target.OS, target.Arch, err)
		}
	}
}

func completeTargetRecords(target coverage.Target) []rportfwdcoverage.Record {
	records := make([]rportfwdcoverage.Record, 0, len(rportfwdcoverage.Transports())*len(rportfwdcoverage.Scenarios()))
	for _, transport := range rportfwdcoverage.Transports() {
		for _, scenario := range rportfwdcoverage.Scenarios() {
			records = append(records, rportfwdcoverage.Record{
				Target:    target,
				Transport: transport,
				Scenario:  scenario,
				Status:    coverage.StatusPass,
				Duration:  time.Millisecond,
			})
		}
	}
	return records
}
