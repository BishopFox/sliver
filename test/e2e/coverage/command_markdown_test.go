package coverage

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCommandMarkdownAggregatesScenariosAndImplantModes(t *testing.T) {
	t.Parallel()

	linux := Target{OS: "linux", Arch: "amd64"}
	windows := Target{OS: "windows", Arch: "amd64"}
	allTargets := []Target{linux, windows}
	dimensions := Dimensions{
		Targets:      allTargets,
		Transports:   []string{TransportMTLS, TransportHTTP},
		ImplantModes: []string{ImplantModeSession, ImplantModeBeacon},
		Commands: []CommandExpectation{
			{Command: Command{GRPCMethod: "RecordedSkip", Scenario: "one"}, SupportedTargets: allTargets},
			{Command: Command{GRPCMethod: `Windows\Only`, Scenario: "one"}, SupportedTargets: []Target{windows}, UnsupportedReason: "Windows only"},
			{Command: Command{GRPCMethod: "Failure", Scenario: "one"}, SupportedTargets: allTargets},
			{Command: Command{GRPCMethod: `All|Pass`, Scenario: "second Linux-only scenario"}, SupportedTargets: []Target{linux}, UnsupportedReason: "Linux only"},
			{Command: Command{GRPCMethod: "Missing", Scenario: "one"}, SupportedTargets: allTargets},
			{Command: Command{GRPCMethod: `All|Pass`, Scenario: "first portable scenario"}, SupportedTargets: allTargets},
		},
	}

	recordsByTarget := map[Target][]Record{}
	for _, expectation := range dimensions.Commands {
		for _, target := range dimensions.Targets {
			if !expectation.supports(target) {
				continue
			}
			for _, transport := range dimensions.Transports {
				for _, mode := range dimensions.ImplantModes {
					record := Record{
						TargetOS:    target.OS,
						TargetArch:  target.Arch,
						Transport:   transport,
						ImplantMode: mode,
						GRPCMethod:  expectation.GRPCMethod,
						Scenario:    expectation.Scenario,
						Status:      StatusPass,
					}
					switch {
					case record.GRPCMethod == "Failure" && target == windows && transport == TransportHTTP && mode == ImplantModeBeacon:
						record.Status = StatusFail
					case record.GRPCMethod == "RecordedSkip" && target == linux && transport == TransportMTLS && mode == ImplantModeSession:
						record.Status = StatusSkip
					case record.GRPCMethod == "Missing" && target == windows && transport == TransportHTTP && mode == ImplantModeBeacon:
						continue
					}
					recordsByTarget[target] = append(recordsByTarget[target], record)
				}
			}
		}
	}

	root := t.TempDir()
	for _, target := range dimensions.Targets {
		writeTestTargetReport(t, filepath.Join(root, target.OS+"-"+target.Arch), target, recordsByTarget[target])
	}
	report, err := AggregateDirectory(root, dimensions)
	if err != nil {
		t.Fatalf("AggregateDirectory() error = %v", err)
	}
	paths, err := WriteGlobalReports(filepath.Join(root, "output"), report)
	if err != nil {
		t.Fatalf("WriteGlobalReports() error = %v", err)
	}
	if got, want := paths.CommandMarkdown, filepath.Join(root, "output", CommandMarkdownFilename); got != want {
		t.Fatalf("command Markdown path = %q, want %q", got, want)
	}

	want := strings.Join([]string{
		"| gRPC command | linux/amd64/mtls | linux/amd64/http | windows/amd64/mtls | windows/amd64/http |",
		"|---|:---:|:---:|:---:|:---:|",
		`| All\|Pass | ✅ | ✅ | ✅ | ✅ |`,
		"| Failure | ✅ | ✅ | ✅ | ❌ |",
		"| Missing | ✅ | ✅ | ✅ | ❌ |",
		"| RecordedSkip | ❌ | ✅ | ✅ | ✅ |",
		`| Windows\\Only | N/A | N/A | ✅ | ✅ |`,
		"",
	}, "\n")
	first := readTestFile(t, paths.CommandMarkdown)
	if got := string(first); got != want {
		t.Fatalf("command Markdown mismatch:\n--- got ---\n%s--- want ---\n%s", got, want)
	}
	if _, err := WriteGlobalReports(filepath.Join(root, "output"), report); err != nil {
		t.Fatalf("second WriteGlobalReports() error = %v", err)
	}
	if second := readTestFile(t, paths.CommandMarkdown); !bytes.Equal(first, second) {
		t.Fatal("command Markdown changed for identical input")
	}
}

func TestComprehensiveCommandMarkdownHasTwentyFourColumns(t *testing.T) {
	t.Parallel()

	report, err := AggregateDirectory(t.TempDir(), ComprehensiveDimensions())
	if err != nil {
		t.Fatalf("AggregateDirectory() error = %v", err)
	}
	lines := strings.Split(strings.TrimSuffix(string(renderCommandMarkdown(report)), "\n"), "\n")
	if got, want := len(lines), 43; got != want {
		t.Fatalf("Markdown lines = %d, want %d (header, delimiter, and 41 unique commands)", got, want)
	}

	wantHeader := []string{
		"gRPC command",
		"darwin/amd64/mtls", "darwin/amd64/wg", "darwin/amd64/http",
		"darwin/arm64/mtls", "darwin/arm64/wg", "darwin/arm64/http",
		"windows/386/mtls", "windows/386/wg", "windows/386/http",
		"windows/amd64/mtls", "windows/amd64/wg", "windows/amd64/http",
		"windows/arm64/mtls", "windows/arm64/wg", "windows/arm64/http",
		"linux/386/mtls", "linux/386/wg", "linux/386/http",
		"linux/amd64/mtls", "linux/amd64/wg", "linux/amd64/http",
		"linux/arm64/mtls", "linux/arm64/wg", "linux/arm64/http",
	}
	if got := markdownColumns(lines[0]); !reflect.DeepEqual(got, wantHeader) {
		t.Fatalf("header columns = %#v, want %#v", got, wantHeader)
	}

	rows := map[string][]string{}
	for _, line := range lines[2:] {
		columns := markdownColumns(line)
		if len(columns) != 25 {
			t.Fatalf("row has %d columns, want 25: %q", len(columns), line)
		}
		if _, exists := rows[columns[0]]; exists {
			t.Fatalf("duplicate command row %q", columns[0])
		}
		rows[columns[0]] = columns[1:]
	}
	if got, want := len(rows), 41; got != want {
		t.Fatalf("unique command rows = %d, want %d", got, want)
	}
	for _, status := range rows["Ping"] {
		if status != commandStatusFail {
			t.Fatalf("empty-input Ping status = %q, want %q", status, commandStatusFail)
		}
	}

	wantChmod := append(repeatedStatus(commandStatusUnsupported, 15), repeatedStatus(commandStatusFail, 9)...)
	if got := rows["Chmod"]; !reflect.DeepEqual(got, wantChmod) {
		t.Fatalf("Chmod statuses = %#v, want %#v", got, wantChmod)
	}
	wantCallExtension := append(repeatedStatus(commandStatusUnsupported, 6), repeatedStatus(commandStatusFail, 6)...)
	wantCallExtension = append(wantCallExtension, repeatedStatus(commandStatusUnsupported, 12)...)
	if got := rows["CallExtension"]; !reflect.DeepEqual(got, wantCallExtension) {
		t.Fatalf("CallExtension statuses = %#v, want %#v", got, wantCallExtension)
	}
}

func markdownColumns(line string) []string {
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return parts
}

func repeatedStatus(status string, count int) []string {
	values := make([]string, count)
	for index := range values {
		values[index] = status
	}
	return values
}
