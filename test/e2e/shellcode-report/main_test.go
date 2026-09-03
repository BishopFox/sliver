package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	coverage "github.com/bishopfox/sliver/test/e2e/coverage"
	shellcodecoverage "github.com/bishopfox/sliver/test/e2e/shellcodecoverage"
)

func TestRunWritesPassingReports(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeCLIReports(t, filepath.Join(root, "input"), false)
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
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	for _, name := range []string{shellcodecoverage.GlobalJSONFilename, shellcodecoverage.GlobalMarkdownFilename} {
		path := filepath.Join(root, "output", name)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("output %s was not written: %v", name, err)
		}
		if !strings.Contains(stdout.String(), path) {
			t.Errorf("stdout = %q, want output path %q", stdout.String(), path)
		}
	}
}

func TestRunAlwaysWritesReportsBeforeMissingRecordGateFailure(t *testing.T) {
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
	if !strings.Contains(stderr.String(), "0 FAIL, 168 NOT RUN") {
		t.Fatalf("stderr = %q, want missing-record gate diagnostic", stderr.String())
	}
	assertCLIOutputsExist(t, filepath.Join(root, "output"))
}

func TestRunAlwaysWritesReportsBeforeExplicitFailureGateFailure(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeCLIReports(t, filepath.Join(root, "input"), true)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"-input", filepath.Join(root, "input"),
		"-output", filepath.Join(root, "output"),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("run() code = %d, want 1; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "1 FAIL, 0 NOT RUN") {
		t.Fatalf("stderr = %q, want explicit-failure gate diagnostic", stderr.String())
	}
	assertCLIOutputsExist(t, filepath.Join(root, "output"))
}

func TestRunRejectsMalformedInputBeforeWriting(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	input := filepath.Join(root, "input")
	if err := os.Mkdir(input, 0o755); err != nil {
		t.Fatalf("Mkdir(input) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(input, "shellcode-coverage-linux-amd64.json"), []byte("{not json\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	output := filepath.Join(root, "output")
	code := run([]string{"-input", input, "-output", output}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("run() code = %d, want 1; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "aggregate shellcode coverage reports") {
		t.Fatalf("stderr = %q, want aggregation diagnostic", stderr.String())
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("output directory exists after failed aggregation: %v", err)
	}
}

func TestRunRejectsUnexpectedArguments(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{"unexpected"}, &stdout, &stderr); code != 2 {
		t.Fatalf("run() code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "unexpected positional arguments") {
		t.Fatalf("stderr = %q, want argument diagnostic", stderr.String())
	}
}

func writeCLIReports(t *testing.T, root string, failFirst bool) {
	t.Helper()
	failed := false
	for _, target := range shellcodecoverage.Targets() {
		recorder, err := shellcodecoverage.NewRecorder(target)
		if err != nil {
			t.Fatalf("NewRecorder(%s/%s) error = %v", target.OS, target.Arch, err)
		}
		for _, transport := range shellcodecoverage.Transports() {
			for _, implantMode := range shellcodecoverage.ImplantModes() {
				for _, compression := range shellcodecoverage.Compressions() {
					for _, encoder := range shellcodecoverage.Encoders() {
						if !shellcodecoverage.EncoderSupported(target, encoder) {
							continue
						}
						requiredSamples := 1
						if encoder == shellcodecoverage.EncoderShikataGaNai {
							requiredSamples = 4
						}
						status := coverage.StatusPass
						detail := ""
						completedSamples := requiredSamples
						if failFirst && !failed {
							status = coverage.StatusFail
							detail = "callback timeout"
							completedSamples = 0
							failed = true
						}
						if err := recorder.Add(shellcodecoverage.Observation{
							Transport:        transport,
							ImplantMode:      implantMode,
							Compression:      compression,
							Encoder:          encoder,
							Status:           status,
							Duration:         time.Second,
							Detail:           detail,
							PayloadBytes:     2048,
							RequiredSamples:  requiredSamples,
							CompletedSamples: completedSamples,
						}); err != nil {
							t.Fatalf("Add(%s/%s) error = %v", target.OS, target.Arch, err)
						}
					}
				}
			}
		}
		if _, err := recorder.Write(filepath.Join(root, target.OS+"-"+target.Arch)); err != nil {
			t.Fatalf("Write(%s/%s) error = %v", target.OS, target.Arch, err)
		}
	}
}

func assertCLIOutputsExist(t *testing.T, output string) {
	t.Helper()
	for _, name := range []string{shellcodecoverage.GlobalJSONFilename, shellcodecoverage.GlobalMarkdownFilename} {
		if _, err := os.Stat(filepath.Join(output, name)); err != nil {
			t.Fatalf("output %s was not written: %v", name, err)
		}
	}
}
