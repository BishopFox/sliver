package rportfwdcoverage

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	coverage "github.com/bishopfox/sliver/test/e2e/coverage"
)

// MatrixStatusNotRun marks a required cell for which no observation was
// recorded. It is deliberately not a valid Record status.
const MatrixStatusNotRun = "not_run"

// MatrixCell is one required scenario result in a target/transport row.
type MatrixCell struct {
	Scenario string        `json:"scenario"`
	Status   string        `json:"status"`
	Recorded bool          `json:"recorded"`
	Duration time.Duration `json:"duration_ns"`
	Detail   string        `json:"detail"`
}

// MatrixRow contains all required scenario cells for one target/transport.
type MatrixRow struct {
	Target    coverage.Target `json:"target"`
	Transport string          `json:"transport"`
	Cells     []MatrixCell    `json:"cells"`
}

// Summary contains deterministic counts for the fixed rportfwd matrix.
type Summary struct {
	Recorded   int `json:"recorded"`
	Pass       int `json:"pass"`
	Fail       int `json:"fail"`
	NotRun     int `json:"not_run"`
	Required   int `json:"required"`
	TotalCells int `json:"total_cells"`
}

// GlobalReport is the deterministic aggregate JSON representation.
type GlobalReport struct {
	SchemaVersion int               `json:"schema_version"`
	Kind          string            `json:"kind"`
	Targets       []coverage.Target `json:"targets"`
	Transports    []string          `json:"transports"`
	Scenarios     []string          `json:"scenarios"`
	Summary       Summary           `json:"summary"`
	Records       []Record          `json:"records"`
	Matrix        []MatrixRow       `json:"matrix"`
}

// AggregateDirectory recursively loads rportfwd-coverage-<os>-<arch>.json
// reports and builds the complete fixed matrix. Missing target artifacts and
// observations are retained as NOT RUN cells so aggregate diagnostics can
// always be emitted. Malformed, misnamed, duplicate-target, and
// duplicate-identity artifacts are rejected.
func AggregateDirectory(root string) (GlobalReport, error) {
	paths, err := targetReportPaths(root)
	if err != nil {
		return GlobalReport{}, err
	}

	records := make([]Record, 0)
	reportForTarget := make(map[coverage.Target]string, len(paths))
	identitySource := make(map[Identity]string)
	for _, path := range paths {
		report, err := LoadTargetReport(path)
		if err != nil {
			return GlobalReport{}, fmt.Errorf("load %s: %w", path, err)
		}
		expectedName := TargetJSONFilename(report.Target)
		if filepath.Base(path) != expectedName {
			return GlobalReport{}, fmt.Errorf(
				"report %s has target %s/%s but must be named %s",
				path,
				report.Target.OS,
				report.Target.Arch,
				expectedName,
			)
		}
		if previous, ok := reportForTarget[report.Target]; ok {
			return GlobalReport{}, fmt.Errorf(
				"duplicate target report for %s/%s in %s and %s",
				report.Target.OS,
				report.Target.Arch,
				previous,
				path,
			)
		}
		reportForTarget[report.Target] = path
		for _, record := range report.Records {
			identity := record.Identity()
			if previous, ok := identitySource[identity]; ok {
				return GlobalReport{}, fmt.Errorf(
					"duplicate record identity %s in %s and %s",
					identity,
					previous,
					path,
				)
			}
			identitySource[identity] = path
			records = append(records, record)
		}
	}

	return newGlobalReport(records), nil
}

// WriteGlobalReports writes rportfwd-coverage.json and
// rportfwd-coverage.md. Validation ensures callers cannot publish a summary or
// matrix that disagrees with the underlying records.
func WriteGlobalReports(dir string, report GlobalReport) (ReportPaths, error) {
	if err := report.Validate(); err != nil {
		return ReportPaths{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ReportPaths{}, fmt.Errorf("create report directory: %w", err)
	}
	paths := ReportPaths{
		JSON:     filepath.Join(dir, GlobalJSONFilename),
		Markdown: filepath.Join(dir, GlobalMarkdownFilename),
	}
	if err := writeJSON(paths.JSON, report); err != nil {
		return ReportPaths{}, err
	}
	if err := os.WriteFile(paths.Markdown, renderGlobalMarkdown(report), 0o644); err != nil {
		return ReportPaths{}, fmt.Errorf("write global Markdown report: %w", err)
	}
	return paths, nil
}

// Validate verifies that a global report is the canonical representation of
// its records and the fixed rportfwd matrix.
func (report GlobalReport) Validate() error {
	if report.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported global schema version %d", report.SchemaVersion)
	}
	if report.Kind != GlobalReportKind {
		return fmt.Errorf("invalid global report kind %q", report.Kind)
	}
	if !reflect.DeepEqual(report.Targets, Targets()) {
		return fmt.Errorf("global report targets do not match the fixed rportfwd matrix")
	}
	if !reflect.DeepEqual(report.Transports, Transports()) {
		return fmt.Errorf("global report transports do not match the fixed rportfwd matrix")
	}
	if !reflect.DeepEqual(report.Scenarios, Scenarios()) {
		return fmt.Errorf("global report scenarios do not match the fixed rportfwd matrix")
	}

	seen := make(map[Identity]struct{}, len(report.Records))
	for index, record := range report.Records {
		if err := record.Validate(); err != nil {
			return fmt.Errorf("global record %d: %w", index, err)
		}
		identity := record.Identity()
		if _, ok := seen[identity]; ok {
			return fmt.Errorf("duplicate global record identity: %s", identity)
		}
		if index > 0 && compareIdentity(report.Records[index-1].Identity(), identity) >= 0 {
			return fmt.Errorf("global records are not in canonical order at record %d", index)
		}
		seen[identity] = struct{}{}
	}

	canonical := newGlobalReport(report.Records)
	if report.Summary != canonical.Summary {
		return fmt.Errorf("global summary does not match records: got %+v, want %+v", report.Summary, canonical.Summary)
	}
	if !reflect.DeepEqual(report.Matrix, canonical.Matrix) {
		return fmt.Errorf("global matrix is not the canonical representation of its records")
	}
	return nil
}

// FailedRecords returns a sorted copy of every explicitly failed observation.
func (report GlobalReport) FailedRecords() []Record {
	failed := make([]Record, 0, report.Summary.Fail)
	for _, record := range report.Records {
		if record.Status == coverage.StatusFail {
			failed = append(failed, record)
		}
	}
	return sortedRecords(failed)
}

// NotRunIdentities returns every required identity without an observation.
func (report GlobalReport) NotRunIdentities() []Identity {
	missing := make([]Identity, 0, report.Summary.NotRun)
	for _, row := range report.Matrix {
		for _, cell := range row.Cells {
			if cell.Status != MatrixStatusNotRun {
				continue
			}
			missing = append(missing, Identity{
				Target:    row.Target,
				Transport: row.Transport,
				Scenario:  cell.Scenario,
			})
		}
	}
	return missing
}

// GateError returns nil only when all required combinations explicitly pass.
func (report GlobalReport) GateError() error {
	if err := report.Validate(); err != nil {
		return fmt.Errorf("invalid rportfwd coverage report: %w", err)
	}
	if report.Summary.Fail == 0 && report.Summary.NotRun == 0 {
		return nil
	}
	return fmt.Errorf(
		"rportfwd coverage gate failed: %d FAIL, %d NOT RUN across %d required combinations",
		report.Summary.Fail,
		report.Summary.NotRun,
		report.Summary.Required,
	)
}

func targetReportPaths(root string) ([]string, error) {
	paths := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if strings.HasPrefix(name, "rportfwd-coverage-") && strings.HasSuffix(name, ".json") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk rportfwd report directory: %w", err)
	}
	sort.Strings(paths)
	return paths, nil
}

func newGlobalReport(records []Record) GlobalReport {
	records = sortedRecords(records)
	matrix := buildMatrix(fixedTargets[:], records)
	return GlobalReport{
		SchemaVersion: SchemaVersion,
		Kind:          GlobalReportKind,
		Targets:       Targets(),
		Transports:    Transports(),
		Scenarios:     Scenarios(),
		Summary:       summarize(matrix),
		Records:       records,
		Matrix:        matrix,
	}
}

func buildMatrix(targets []coverage.Target, records []Record) []MatrixRow {
	recordsByIdentity := make(map[Identity]Record, len(records))
	for _, record := range records {
		recordsByIdentity[record.Identity()] = record
	}

	rows := make([]MatrixRow, 0, len(targets)*len(fixedTransports))
	for _, target := range targets {
		for _, transport := range fixedTransports {
			row := MatrixRow{
				Target:    target,
				Transport: transport,
				Cells:     make([]MatrixCell, 0, len(fixedScenarios)),
			}
			for _, scenario := range fixedScenarios {
				cell := MatrixCell{Scenario: scenario}
				identity := Identity{Target: target, Transport: transport, Scenario: scenario}
				if record, ok := recordsByIdentity[identity]; ok {
					cell.Status = string(record.Status)
					cell.Recorded = true
					cell.Duration = record.Duration
					cell.Detail = record.Detail
				} else {
					cell.Status = MatrixStatusNotRun
				}
				row.Cells = append(row.Cells, cell)
			}
			rows = append(rows, row)
		}
	}
	return rows
}

func summarize(matrix []MatrixRow) Summary {
	summary := Summary{}
	for _, row := range matrix {
		for _, cell := range row.Cells {
			summary.TotalCells++
			summary.Required++
			switch cell.Status {
			case string(coverage.StatusPass):
				summary.Recorded++
				summary.Pass++
			case string(coverage.StatusFail):
				summary.Recorded++
				summary.Fail++
			case MatrixStatusNotRun:
				summary.NotRun++
			}
		}
	}
	return summary
}
