package coverage

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRecorderWritesDeterministicTargetReports(t *testing.T) {
	t.Parallel()

	recorder, err := NewRecorder(Target{OS: "linux", Arch: "amd64"})
	if err != nil {
		t.Fatalf("NewRecorder() error = %v", err)
	}
	observations := []Observation{
		{
			Transport:   "mtls",
			ImplantMode: "session",
			GRPCMethod:  "/sliverpb.Sliver/Ls",
			Scenario:    "recursive listing",
			Status:      StatusSkip,
			Duration:    2*time.Second + 250*time.Millisecond,
			Detail:      "line one|value\nline two",
		},
		{
			Transport:   "http",
			ImplantMode: "beacon",
			GRPCMethod:  "/sliverpb.Sliver/Cd",
			Scenario:    "change directory",
			Status:      StatusPass,
			Duration:    125 * time.Millisecond,
		},
	}
	for _, observation := range observations {
		if err := recorder.Add(observation); err != nil {
			t.Fatalf("Add() error = %v", err)
		}
	}

	dir := t.TempDir()
	paths, err := recorder.Write(dir)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	firstJSON := readTestFile(t, paths.JSON)
	firstMarkdown := readTestFile(t, paths.Markdown)

	if _, err := recorder.Write(dir); err != nil {
		t.Fatalf("second Write() error = %v", err)
	}
	if second := readTestFile(t, paths.JSON); !bytes.Equal(firstJSON, second) {
		t.Fatal("JSON report changed for identical input")
	}
	if second := readTestFile(t, paths.Markdown); !bytes.Equal(firstMarkdown, second) {
		t.Fatal("Markdown report changed for identical input")
	}

	report, err := LoadTargetReport(paths.JSON)
	if err != nil {
		t.Fatalf("LoadTargetReport() error = %v", err)
	}
	if len(report.Records) != 2 {
		t.Fatalf("got %d records, want 2", len(report.Records))
	}
	if got := report.Records[0].GRPCMethod; got != "/sliverpb.Sliver/Cd" {
		t.Fatalf("first sorted method = %q, want Cd", got)
	}
	if got := report.Records[1].Duration; got != observations[0].Duration {
		t.Fatalf("round-trip duration = %v, want %v", got, observations[0].Duration)
	}
	markdown := string(firstMarkdown)
	if !strings.Contains(markdown, `line one\|value<br>line two`) {
		t.Fatalf("Markdown detail was not escaped: %s", markdown)
	}
}

func TestRecorderRejectsDuplicateAndMalformedRecords(t *testing.T) {
	t.Parallel()

	recorder, err := NewRecorder(Target{OS: "windows", Arch: "arm64"})
	if err != nil {
		t.Fatalf("NewRecorder() error = %v", err)
	}
	valid := Observation{
		Transport:   "wg",
		ImplantMode: "beacon",
		GRPCMethod:  "/sliverpb.Sliver/Env",
		Scenario:    "read environment",
		Status:      StatusPass,
		Duration:    time.Second,
	}
	if err := recorder.Add(valid); err != nil {
		t.Fatalf("Add(valid) error = %v", err)
	}
	duplicate := valid
	duplicate.Status = StatusFail
	if err := recorder.Add(duplicate); err == nil || !strings.Contains(err.Error(), "duplicate record identity") {
		t.Fatalf("Add(duplicate) error = %v, want duplicate identity", err)
	}

	tests := []struct {
		name   string
		mutate func(*Observation)
	}{
		{name: "empty transport", mutate: func(value *Observation) { value.Transport = "" }},
		{name: "invalid mode token", mutate: func(value *Observation) { value.ImplantMode = "session/mode" }},
		{name: "blank method", mutate: func(value *Observation) { value.GRPCMethod = " " }},
		{name: "control in scenario", mutate: func(value *Observation) { value.Scenario = "bad\nscenario" }},
		{name: "unknown status", mutate: func(value *Observation) { value.Status = Status("unknown") }},
		{name: "negative duration", mutate: func(value *Observation) { value.Duration = -time.Second }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			observation := valid
			observation.Scenario += " " + test.name
			test.mutate(&observation)
			if err := recorder.Add(observation); err == nil {
				t.Fatal("Add() error = nil, want validation error")
			}
		})
	}
}

func TestLoadTargetReportRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	path := t.TempDir() + "/coverage-linux-amd64.json"
	data := []byte(`{
  "schema_version": 1,
  "kind": "sliver-e2e-target-coverage",
  "target": {"os": "linux", "arch": "amd64"},
  "records": [],
  "unexpected": true
}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := LoadTargetReport(path); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("LoadTargetReport() error = %v, want unknown field", err)
	}
}

func TestLoadTargetReportRejectsDuplicateJSONIdentityField(t *testing.T) {
	t.Parallel()

	path := t.TempDir() + "/coverage-linux-amd64.json"
	data := []byte(`{
  "schema_version": 1,
  "kind": "sliver-e2e-target-coverage",
  "target": {"os": "linux", "arch": "amd64"},
  "records": [{
    "target_os": "linux",
    "target_arch": "amd64",
    "transport": "mtls",
    "implant_mode": "session",
    "grpc_method": "/sliverpb.Sliver/Ls",
    "scenario": "first",
    "Scenario": "second",
    "status": "pass",
    "duration_ns": 0,
    "detail": ""
  }]
}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := LoadTargetReport(path); err == nil || !strings.Contains(err.Error(), `duplicate JSON field "scenario"`) {
		t.Fatalf("LoadTargetReport() error = %v, want duplicate scenario field", err)
	}
}

func TestLoadTargetReportRejectsNoncanonicalJSONIdentityField(t *testing.T) {
	t.Parallel()

	path := t.TempDir() + "/coverage-linux-amd64.json"
	data := []byte(`{
  "schema_version": 1,
  "kind": "sliver-e2e-target-coverage",
  "target": {"os": "linux", "arch": "amd64"},
  "records": [{
    "target_os": "linux",
    "target_arch": "amd64",
    "transport": "mtls",
    "implant_mode": "session",
    "grpc_method": "/sliverpb.Sliver/Ls",
    "Scenario": "listing",
    "status": "pass",
    "duration_ns": 0,
    "detail": ""
  }]
}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := LoadTargetReport(path); err == nil || !strings.Contains(err.Error(), `unknown field "Scenario"`) {
		t.Fatalf("LoadTargetReport() error = %v, want noncanonical Scenario field", err)
	}
}

func readTestFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return data
}
