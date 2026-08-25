package shellcodecoverage_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	coverage "github.com/bishopfox/sliver/test/e2e/coverage"
	shellcodecoverage "github.com/bishopfox/sliver/test/e2e/shellcodecoverage"
)

func TestFixedAxesAndEncoderSupport(t *testing.T) {
	t.Parallel()

	wantTargets := []coverage.Target{
		{OS: "darwin", Arch: "arm64"},
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
		{OS: "windows", Arch: "386"},
		{OS: "windows", Arch: "amd64"},
	}
	if got := shellcodecoverage.Targets(); !reflect.DeepEqual(got, wantTargets) {
		t.Fatalf("Targets() = %#v, want %#v", got, wantTargets)
	}
	if got, want := shellcodecoverage.Transports(), []string{
		shellcodecoverage.TransportMTLS,
		shellcodecoverage.TransportWG,
		shellcodecoverage.TransportHTTP,
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Transports() = %#v, want %#v", got, want)
	}
	if got, want := shellcodecoverage.ImplantModes(), []string{
		shellcodecoverage.ImplantModeSession,
		shellcodecoverage.ImplantModeBeacon,
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ImplantModes() = %#v, want %#v", got, want)
	}
	if got, want := shellcodecoverage.Compressions(), []string{
		shellcodecoverage.CompressionNone,
		shellcodecoverage.CompressionAPLib,
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Compressions() = %#v, want %#v", got, want)
	}
	if got, want := shellcodecoverage.Encoders(), []string{
		shellcodecoverage.EncoderNone,
		shellcodecoverage.EncoderShikataGaNai,
		shellcodecoverage.EncoderXOR,
		shellcodecoverage.EncoderXORDynamic,
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Encoders() = %#v, want %#v", got, want)
	}

	wantSupportedEncoders := map[coverage.Target][]string{
		{OS: "darwin", Arch: "arm64"}:  {shellcodecoverage.EncoderNone, shellcodecoverage.EncoderXOR, shellcodecoverage.EncoderXORDynamic},
		{OS: "linux", Arch: "amd64"}:   {shellcodecoverage.EncoderNone, shellcodecoverage.EncoderShikataGaNai, shellcodecoverage.EncoderXOR, shellcodecoverage.EncoderXORDynamic},
		{OS: "linux", Arch: "arm64"}:   {shellcodecoverage.EncoderNone, shellcodecoverage.EncoderXOR, shellcodecoverage.EncoderXORDynamic},
		{OS: "windows", Arch: "386"}:   {shellcodecoverage.EncoderNone, shellcodecoverage.EncoderShikataGaNai},
		{OS: "windows", Arch: "amd64"}: {shellcodecoverage.EncoderNone, shellcodecoverage.EncoderShikataGaNai, shellcodecoverage.EncoderXOR, shellcodecoverage.EncoderXORDynamic},
	}
	supportedEncoderCount := 0
	for _, target := range shellcodecoverage.Targets() {
		var got []string
		for _, encoder := range shellcodecoverage.Encoders() {
			if shellcodecoverage.EncoderSupported(target, encoder) {
				got = append(got, encoder)
				supportedEncoderCount++
			}
		}
		if !reflect.DeepEqual(got, wantSupportedEncoders[target]) {
			t.Errorf("supported encoders for %s/%s = %v, want %v", target.OS, target.Arch, got, wantSupportedEncoders[target])
		}
	}
	gotRequired := supportedEncoderCount * len(shellcodecoverage.Transports()) * len(shellcodecoverage.ImplantModes()) * len(shellcodecoverage.Compressions())
	if gotRequired != shellcodecoverage.RequiredSupportedCombinations {
		t.Fatalf("required supported combinations = %d, want %d", gotRequired, shellcodecoverage.RequiredSupportedCombinations)
	}
	if shellcodecoverage.EncoderSupported(coverage.Target{OS: "freebsd", Arch: "amd64"}, shellcodecoverage.EncoderNone) {
		t.Fatal("EncoderSupported() accepted an unknown target")
	}
	if shellcodecoverage.EncoderSupported(wantTargets[0], "unknown") {
		t.Fatal("EncoderSupported() accepted an unknown encoder")
	}

	mutable := shellcodecoverage.Targets()
	mutable[0].OS = "changed"
	if shellcodecoverage.Targets()[0].OS != "darwin" {
		t.Fatal("Targets() returned mutable package state")
	}
}

func TestRecorderValidationAndDeterministicTargetReports(t *testing.T) {
	t.Parallel()

	target := coverage.Target{OS: "linux", Arch: "amd64"}
	recorder, err := shellcodecoverage.NewRecorder(target)
	if err != nil {
		t.Fatalf("NewRecorder() error = %v", err)
	}
	observations := completeTargetObservations(target)
	for index := len(observations) - 1; index >= 0; index-- {
		if err := recorder.Add(observations[index]); err != nil {
			t.Fatalf("Add(%d) error = %v", index, err)
		}
	}
	if got, want := len(recorder.Records()), 48; got != want {
		t.Fatalf("len(Records()) = %d, want %d", got, want)
	}
	if err := recorder.Add(observations[0]); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate Add() error = %v, want duplicate diagnostic", err)
	}

	dirA := filepath.Join(t.TempDir(), "a")
	pathsA, err := recorder.Write(dirA)
	if err != nil {
		t.Fatalf("Write(a) error = %v", err)
	}
	if got, want := filepath.Base(pathsA.JSON), "shellcode-coverage-linux-amd64.json"; got != want {
		t.Fatalf("JSON filename = %q, want %q", got, want)
	}
	if got, want := filepath.Base(pathsA.Markdown), "shellcode-coverage-linux-amd64.md"; got != want {
		t.Fatalf("Markdown filename = %q, want %q", got, want)
	}

	recorderB, err := shellcodecoverage.NewRecorder(target)
	if err != nil {
		t.Fatalf("NewRecorder(b) error = %v", err)
	}
	for _, observation := range observations {
		if err := recorderB.Add(observation); err != nil {
			t.Fatalf("Add(b) error = %v", err)
		}
	}
	pathsB, err := recorderB.Write(filepath.Join(t.TempDir(), "b"))
	if err != nil {
		t.Fatalf("Write(b) error = %v", err)
	}
	assertFilesEqual(t, pathsA.JSON, pathsB.JSON)
	assertFilesEqual(t, pathsA.Markdown, pathsB.Markdown)

	loaded, err := shellcodecoverage.LoadTargetReport(pathsA.JSON)
	if err != nil {
		t.Fatalf("LoadTargetReport() error = %v", err)
	}
	if got, want := len(loaded.Records), 48; got != want {
		t.Fatalf("loaded records = %d, want %d", got, want)
	}
}

func TestRecorderRejectsMalformedAndOutOfMatrixObservations(t *testing.T) {
	t.Parallel()

	target := coverage.Target{OS: "darwin", Arch: "arm64"}
	valid := shellcodecoverage.Observation{
		Transport:    shellcodecoverage.TransportMTLS,
		ImplantMode:  shellcodecoverage.ImplantModeSession,
		Compression:  shellcodecoverage.CompressionNone,
		Encoder:      shellcodecoverage.EncoderNone,
		Status:       coverage.StatusPass,
		Duration:     time.Second,
		PayloadBytes: 1234,
	}
	tests := []struct {
		name   string
		mutate func(*shellcodecoverage.Observation)
		want   string
	}{
		{name: "skip", mutate: func(observation *shellcodecoverage.Observation) { observation.Status = coverage.StatusSkip }, want: "only \"pass\" and \"fail\""},
		{name: "unknown status", mutate: func(observation *shellcodecoverage.Observation) { observation.Status = coverage.Status("unknown") }, want: "status"},
		{name: "transport", mutate: func(observation *shellcodecoverage.Observation) { observation.Transport = "dns" }, want: "transport"},
		{name: "mode", mutate: func(observation *shellcodecoverage.Observation) { observation.ImplantMode = "interactive" }, want: "implant mode"},
		{name: "compression", mutate: func(observation *shellcodecoverage.Observation) { observation.Compression = "gzip" }, want: "compression"},
		{name: "encoder", mutate: func(observation *shellcodecoverage.Observation) { observation.Encoder = "rot13" }, want: "encoder"},
		{name: "unsupported encoder", mutate: func(observation *shellcodecoverage.Observation) {
			observation.Encoder = shellcodecoverage.EncoderShikataGaNai
		}, want: "not supported"},
		{name: "duration", mutate: func(observation *shellcodecoverage.Observation) { observation.Duration = -1 }, want: "duration"},
		{name: "payload bytes", mutate: func(observation *shellcodecoverage.Observation) { observation.PayloadBytes = -1 }, want: "payload bytes"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder, err := shellcodecoverage.NewRecorder(target)
			if err != nil {
				t.Fatalf("NewRecorder() error = %v", err)
			}
			observation := valid
			test.mutate(&observation)
			if err := recorder.Add(observation); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Add() error = %v, want diagnostic containing %q", err, test.want)
			}
		})
	}
	if _, err := shellcodecoverage.NewRecorder(coverage.Target{OS: "linux", Arch: "386"}); err == nil {
		t.Fatal("NewRecorder() accepted a target outside the fixed matrix")
	}
}

func TestAggregateFullPassAndNARendering(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeCompleteReports(t, filepath.Join(root, "input"), nil)
	report, err := shellcodecoverage.AggregateDirectory(filepath.Join(root, "input"))
	if err != nil {
		t.Fatalf("AggregateDirectory() error = %v", err)
	}
	wantSummary := shellcodecoverage.Summary{
		Recorded:      192,
		Pass:          192,
		Fail:          0,
		NotRun:        0,
		NotApplicable: 48,
		Required:      192,
		TotalCells:    240,
	}
	if report.Summary != wantSummary {
		t.Fatalf("Summary = %+v, want %+v", report.Summary, wantSummary)
	}
	if err := report.GateError(); err != nil {
		t.Fatalf("GateError() = %v, want nil", err)
	}
	if got := len(report.Matrix); got != 60 {
		t.Fatalf("matrix rows = %d, want 60", got)
	}

	paths, err := shellcodecoverage.WriteGlobalReports(filepath.Join(root, "output"), report)
	if err != nil {
		t.Fatalf("WriteGlobalReports() error = %v", err)
	}
	markdown := readFile(t, paths.Markdown)
	if !strings.HasPrefix(markdown, "# Sliver shellcode E2E coverage\n\n| Target | Transport | Mode | Compression | none | shikata_ga_nai | xor | xor_dynamic |") {
		t.Fatalf("global Markdown does not lead with the matrix table:\n%s", markdown)
	}
	for _, marker := range []string{"✅ PASS", "❌ FAIL", "➖ N/A", "⚪ NOT RUN", "| 192 | 192 | 192 | 0 | 0 | 48 | 240 |", "None. All required combinations passed."} {
		if !strings.Contains(markdown, marker) {
			t.Errorf("global Markdown omitted %q", marker)
		}
	}

	naCount := 0
	for _, row := range report.Matrix {
		for _, cell := range row.Cells {
			if cell.Status == shellcodecoverage.MatrixStatusNotApplicable {
				naCount++
				if cell.Recorded {
					t.Errorf("N/A cell was recorded: %+v %+v", row, cell)
				}
			}
		}
	}
	if naCount != 48 {
		t.Fatalf("N/A cells = %d, want 48", naCount)
	}
}

func TestAggregateExplicitFailureFailsGateAndRendersDetail(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	changed := false
	writeCompleteReports(t, filepath.Join(root, "input"), func(record *shellcodecoverage.Record) bool {
		if changed {
			return true
		}
		changed = true
		record.Status = coverage.StatusFail
		record.Detail = "runner exited | no callback\nserver remained healthy"
		return true
	})
	report, err := shellcodecoverage.AggregateDirectory(filepath.Join(root, "input"))
	if err != nil {
		t.Fatalf("AggregateDirectory() error = %v", err)
	}
	if got, want := report.Summary.Fail, 1; got != want {
		t.Fatalf("Fail = %d, want %d", got, want)
	}
	if got, want := report.Summary.Pass, 191; got != want {
		t.Fatalf("Pass = %d, want %d", got, want)
	}
	if err := report.GateError(); err == nil || !strings.Contains(err.Error(), "1 FAIL") {
		t.Fatalf("GateError() = %v, want one-failure diagnostic", err)
	}
	paths, err := shellcodecoverage.WriteGlobalReports(filepath.Join(root, "output"), report)
	if err != nil {
		t.Fatalf("WriteGlobalReports() error = %v", err)
	}
	markdown := readFile(t, paths.Markdown)
	for _, marker := range []string{"❌ FAIL", "### Explicit failures", "runner exited \\| no callback<br>server remained healthy"} {
		if !strings.Contains(markdown, marker) {
			t.Errorf("failure Markdown omitted %q", marker)
		}
	}
}

func TestAggregateMissingRecordIsNotRunAndFailsGate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	omitted := false
	writeCompleteReports(t, filepath.Join(root, "input"), func(_ *shellcodecoverage.Record) bool {
		if omitted {
			return true
		}
		omitted = true
		return false
	})
	report, err := shellcodecoverage.AggregateDirectory(filepath.Join(root, "input"))
	if err != nil {
		t.Fatalf("AggregateDirectory() error = %v", err)
	}
	if got, want := report.Summary.NotRun, 1; got != want {
		t.Fatalf("NotRun = %d, want %d", got, want)
	}
	if got, want := len(report.NotRunIdentities()), 1; got != want {
		t.Fatalf("len(NotRunIdentities()) = %d, want %d", got, want)
	}
	if err := report.GateError(); err == nil || !strings.Contains(err.Error(), "1 NOT RUN") {
		t.Fatalf("GateError() = %v, want one-NOT-RUN diagnostic", err)
	}
	paths, err := shellcodecoverage.WriteGlobalReports(filepath.Join(root, "output"), report)
	if err != nil {
		t.Fatalf("WriteGlobalReports() error = %v", err)
	}
	markdown := readFile(t, paths.Markdown)
	if !strings.Contains(markdown, "### Required combinations not run") || !strings.Contains(markdown, "⚪ NOT RUN") {
		t.Fatalf("missing-record Markdown omitted NOT RUN detail:\n%s", markdown)
	}
}

func TestAggregateRejectsDuplicateMalformedAndOutOfMatrixReports(t *testing.T) {
	t.Parallel()

	validRecord := shellcodecoverage.Record{
		Target:       coverage.Target{OS: "linux", Arch: "amd64"},
		Transport:    shellcodecoverage.TransportMTLS,
		ImplantMode:  shellcodecoverage.ImplantModeSession,
		Compression:  shellcodecoverage.CompressionNone,
		Encoder:      shellcodecoverage.EncoderNone,
		Status:       coverage.StatusPass,
		Duration:     time.Second,
		PayloadBytes: 4096,
	}
	tests := []struct {
		name  string
		setup func(*testing.T, string)
		want  string
	}{
		{
			name: "duplicate records in one report",
			setup: func(t *testing.T, root string) {
				writeRawTargetReport(t, root, shellcodecoverage.TargetReport{
					SchemaVersion: shellcodecoverage.SchemaVersion,
					Kind:          shellcodecoverage.TargetReportKind,
					Target:        validRecord.Target,
					Records:       []shellcodecoverage.Record{validRecord, validRecord},
				})
			},
			want: "duplicate record identity",
		},
		{
			name: "duplicate records across reports",
			setup: func(t *testing.T, root string) {
				for _, subdir := range []string{"one", "two"} {
					if _, err := shellcodecoverage.WriteTargetReports(filepath.Join(root, subdir), validRecord.Target, []shellcodecoverage.Record{validRecord}); err != nil {
						t.Fatalf("WriteTargetReports(%s) error = %v", subdir, err)
					}
				}
			},
			want: "duplicate record identity",
		},
		{
			name: "duplicate JSON field",
			setup: func(t *testing.T, root string) {
				data := marshalTargetReport(t, shellcodecoverage.TargetReport{
					SchemaVersion: shellcodecoverage.SchemaVersion,
					Kind:          shellcodecoverage.TargetReportKind,
					Target:        validRecord.Target,
					Records:       []shellcodecoverage.Record{validRecord},
				})
				data = bytes.Replace(data, []byte("\"kind\":"), []byte("\"kind\":\"duplicate\",\"kind\":"), 1)
				writeRawJSON(t, root, validRecord.Target, data)
			},
			want: "duplicate JSON field",
		},
		{
			name: "unknown JSON field",
			setup: func(t *testing.T, root string) {
				data := marshalTargetReport(t, shellcodecoverage.TargetReport{
					SchemaVersion: shellcodecoverage.SchemaVersion,
					Kind:          shellcodecoverage.TargetReportKind,
					Target:        validRecord.Target,
					Records:       []shellcodecoverage.Record{validRecord},
				})
				data = bytes.Replace(data, []byte("{"), []byte("{\"unexpected\":true,"), 1)
				writeRawJSON(t, root, validRecord.Target, data)
			},
			want: "unknown field",
		},
		{
			name: "recorded skip",
			setup: func(t *testing.T, root string) {
				record := validRecord
				record.Status = coverage.StatusSkip
				writeRawTargetReport(t, root, targetReport(record))
			},
			want: "only \"pass\" and \"fail\"",
		},
		{
			name: "unsupported target encoder",
			setup: func(t *testing.T, root string) {
				record := validRecord
				record.Target = coverage.Target{OS: "darwin", Arch: "arm64"}
				record.Encoder = shellcodecoverage.EncoderShikataGaNai
				writeRawTargetReport(t, root, targetReport(record))
			},
			want: "not supported",
		},
		{
			name: "transport outside matrix",
			setup: func(t *testing.T, root string) {
				record := validRecord
				record.Transport = "dns"
				writeRawTargetReport(t, root, targetReport(record))
			},
			want: "outside the shellcode matrix",
		},
		{
			name: "target filename mismatch",
			setup: func(t *testing.T, root string) {
				data := marshalTargetReport(t, targetReport(validRecord))
				path := filepath.Join(root, "shellcode-coverage-windows-amd64.json")
				if err := os.MkdirAll(root, 0o755); err != nil {
					t.Fatalf("MkdirAll() error = %v", err)
				}
				if err := os.WriteFile(path, data, 0o644); err != nil {
					t.Fatalf("WriteFile() error = %v", err)
				}
			},
			want: "must be named",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			test.setup(t, root)
			if _, err := shellcodecoverage.AggregateDirectory(root); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("AggregateDirectory() error = %v, want diagnostic containing %q", err, test.want)
			}
		})
	}
}

func TestGlobalReportsAreDeterministicAcrossInputOrder(t *testing.T) {
	t.Parallel()

	rootA := filepath.Join(t.TempDir(), "input-a")
	rootB := filepath.Join(t.TempDir(), "input-b")
	writeCompleteReports(t, rootA, nil)
	writeCompleteReportsReverse(t, rootB)
	reportA, err := shellcodecoverage.AggregateDirectory(rootA)
	if err != nil {
		t.Fatalf("AggregateDirectory(a) error = %v", err)
	}
	reportB, err := shellcodecoverage.AggregateDirectory(rootB)
	if err != nil {
		t.Fatalf("AggregateDirectory(b) error = %v", err)
	}
	if !reflect.DeepEqual(reportA, reportB) {
		t.Fatal("aggregate reports differ across input and insertion order")
	}
	pathsA, err := shellcodecoverage.WriteGlobalReports(filepath.Join(t.TempDir(), "output-a"), reportA)
	if err != nil {
		t.Fatalf("WriteGlobalReports(a) error = %v", err)
	}
	pathsB, err := shellcodecoverage.WriteGlobalReports(filepath.Join(t.TempDir(), "output-b"), reportB)
	if err != nil {
		t.Fatalf("WriteGlobalReports(b) error = %v", err)
	}
	assertFilesEqual(t, pathsA.JSON, pathsB.JSON)
	assertFilesEqual(t, pathsA.Markdown, pathsB.Markdown)
}

func completeTargetObservations(target coverage.Target) []shellcodecoverage.Observation {
	observations := make([]shellcodecoverage.Observation, 0)
	sequence := int64(1)
	for _, transport := range shellcodecoverage.Transports() {
		for _, implantMode := range shellcodecoverage.ImplantModes() {
			for _, compression := range shellcodecoverage.Compressions() {
				for _, encoder := range shellcodecoverage.Encoders() {
					if !shellcodecoverage.EncoderSupported(target, encoder) {
						continue
					}
					observations = append(observations, shellcodecoverage.Observation{
						Transport:    transport,
						ImplantMode:  implantMode,
						Compression:  compression,
						Encoder:      encoder,
						Status:       coverage.StatusPass,
						Duration:     time.Duration(sequence) * time.Millisecond,
						PayloadBytes: 1000 + sequence,
					})
					sequence++
				}
			}
		}
	}
	return observations
}

func writeCompleteReports(t *testing.T, root string, include func(*shellcodecoverage.Record) bool) {
	t.Helper()
	for _, target := range shellcodecoverage.Targets() {
		records := observationsToRecords(target, completeTargetObservations(target))
		filtered := make([]shellcodecoverage.Record, 0, len(records))
		for index := range records {
			if include == nil || include(&records[index]) {
				filtered = append(filtered, records[index])
			}
		}
		if _, err := shellcodecoverage.WriteTargetReports(filepath.Join(root, target.OS+"-"+target.Arch), target, filtered); err != nil {
			t.Fatalf("WriteTargetReports(%s/%s) error = %v", target.OS, target.Arch, err)
		}
	}
}

func writeCompleteReportsReverse(t *testing.T, root string) {
	t.Helper()
	targets := shellcodecoverage.Targets()
	for targetIndex := len(targets) - 1; targetIndex >= 0; targetIndex-- {
		target := targets[targetIndex]
		observations := completeTargetObservations(target)
		recorder, err := shellcodecoverage.NewRecorder(target)
		if err != nil {
			t.Fatalf("NewRecorder(%s/%s) error = %v", target.OS, target.Arch, err)
		}
		for index := len(observations) - 1; index >= 0; index-- {
			if err := recorder.Add(observations[index]); err != nil {
				t.Fatalf("Add(%s/%s) error = %v", target.OS, target.Arch, err)
			}
		}
		if _, err := recorder.Write(filepath.Join(root, target.Arch+"-"+target.OS)); err != nil {
			t.Fatalf("Write(%s/%s) error = %v", target.OS, target.Arch, err)
		}
	}
}

func observationsToRecords(target coverage.Target, observations []shellcodecoverage.Observation) []shellcodecoverage.Record {
	records := make([]shellcodecoverage.Record, 0, len(observations))
	for _, observation := range observations {
		records = append(records, shellcodecoverage.Record{
			Target:       target,
			Transport:    observation.Transport,
			ImplantMode:  observation.ImplantMode,
			Compression:  observation.Compression,
			Encoder:      observation.Encoder,
			Status:       observation.Status,
			Duration:     observation.Duration,
			Detail:       observation.Detail,
			PayloadBytes: observation.PayloadBytes,
		})
	}
	return records
}

func targetReport(record shellcodecoverage.Record) shellcodecoverage.TargetReport {
	return shellcodecoverage.TargetReport{
		SchemaVersion: shellcodecoverage.SchemaVersion,
		Kind:          shellcodecoverage.TargetReportKind,
		Target:        record.Target,
		Records:       []shellcodecoverage.Record{record},
	}
}

func writeRawTargetReport(t *testing.T, root string, report shellcodecoverage.TargetReport) {
	t.Helper()
	writeRawJSON(t, root, report.Target, marshalTargetReport(t, report))
}

func writeRawJSON(t *testing.T, root string, target coverage.Target, data []byte) {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	path := filepath.Join(root, "shellcode-coverage-"+target.OS+"-"+target.Arch+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func marshalTargetReport(t *testing.T, report shellcodecoverage.TargetReport) []byte {
	t.Helper()
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return append(data, '\n')
}

func assertFilesEqual(t *testing.T, left, right string) {
	t.Helper()
	leftData := []byte(readFile(t, left))
	rightData := []byte(readFile(t, right))
	if !bytes.Equal(leftData, rightData) {
		t.Fatalf("files differ:\n%s\n%s", left, right)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(data)
}
