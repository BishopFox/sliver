package shellcodecoverage

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

const (
	// MatrixStatusNotApplicable marks an unsupported target/encoder cell.
	MatrixStatusNotApplicable = "n/a"
	// MatrixStatusNotRun marks a required cell with no recorded observation.
	MatrixStatusNotRun = "not_run"
)

// MatrixCell is one encoder result in a target/transport/mode/compression row.
type MatrixCell struct {
	Encoder      string        `json:"encoder"`
	Status       string        `json:"status"`
	Recorded     bool          `json:"recorded"`
	Duration     time.Duration `json:"duration_ns"`
	Detail       string        `json:"detail"`
	PayloadBytes int64         `json:"payload_bytes"`
}

// MatrixRow contains all encoder columns for one row identity.
type MatrixRow struct {
	Target      coverage.Target `json:"target"`
	Transport   string          `json:"transport"`
	ImplantMode string          `json:"implant_mode"`
	Compression string          `json:"compression"`
	Cells       []MatrixCell    `json:"cells"`
}

// Summary contains deterministic counts for the fixed matrix.
type Summary struct {
	Recorded      int `json:"recorded"`
	Pass          int `json:"pass"`
	Fail          int `json:"fail"`
	NotRun        int `json:"not_run"`
	NotApplicable int `json:"not_applicable"`
	Required      int `json:"required"`
	TotalCells    int `json:"total_cells"`
}

// GlobalReport is the deterministic aggregate JSON representation.
type GlobalReport struct {
	SchemaVersion int               `json:"schema_version"`
	Kind          string            `json:"kind"`
	Targets       []coverage.Target `json:"targets"`
	Transports    []string          `json:"transports"`
	ImplantModes  []string          `json:"implant_modes"`
	Compressions  []string          `json:"compressions"`
	Encoders      []string          `json:"encoders"`
	Summary       Summary           `json:"summary"`
	Records       []Record          `json:"records"`
	Matrix        []MatrixRow       `json:"matrix"`
}

// AggregateDirectory recursively loads shellcode-coverage-<os>-<arch>.json
// reports and builds the complete fixed matrix. Missing reports are allowed at
// aggregation time and appear as NOT RUN so reports can always be emitted.
// Malformed reports and record identities duplicated within or across files
// are rejected.
func AggregateDirectory(root string) (GlobalReport, error) {
	paths, err := targetReportPaths(root)
	if err != nil {
		return GlobalReport{}, err
	}

	records := make([]Record, 0)
	seen := map[Identity]string{}
	for _, path := range paths {
		report, err := LoadTargetReport(path)
		if err != nil {
			return GlobalReport{}, fmt.Errorf("load %s: %w", path, err)
		}
		expectedName := targetReportBase(report.Target) + ".json"
		if filepath.Base(path) != expectedName {
			return GlobalReport{}, fmt.Errorf("report %s has target %s/%s but must be named %s", path, report.Target.OS, report.Target.Arch, expectedName)
		}
		for _, record := range report.Records {
			identity := record.Identity()
			if previous, ok := seen[identity]; ok {
				return GlobalReport{}, fmt.Errorf("duplicate record identity %s in %s and %s", identity, previous, path)
			}
			seen[identity] = path
			records = append(records, record)
		}
	}

	return newGlobalReport(sortedRecords(records)), nil
}

// WriteGlobalReports writes shellcode-coverage.json and shellcode-coverage.md.
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
// its records and the fixed shellcode matrix.
func (report GlobalReport) Validate() error {
	if report.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported global schema version %d", report.SchemaVersion)
	}
	if report.Kind != GlobalReportKind {
		return fmt.Errorf("invalid global report kind %q", report.Kind)
	}
	if !reflect.DeepEqual(report.Targets, Targets()) {
		return fmt.Errorf("global report targets do not match the fixed shellcode matrix")
	}
	if !reflect.DeepEqual(report.Transports, Transports()) {
		return fmt.Errorf("global report transports do not match the fixed shellcode matrix")
	}
	if !reflect.DeepEqual(report.ImplantModes, ImplantModes()) {
		return fmt.Errorf("global report implant modes do not match the fixed shellcode matrix")
	}
	if !reflect.DeepEqual(report.Compressions, Compressions()) {
		return fmt.Errorf("global report compressions do not match the fixed shellcode matrix")
	}
	if !reflect.DeepEqual(report.Encoders, Encoders()) {
		return fmt.Errorf("global report encoders do not match the fixed shellcode matrix")
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
// Unsupported encoder cells are N/A and are intentionally excluded.
func (report GlobalReport) NotRunIdentities() []Identity {
	missing := make([]Identity, 0, report.Summary.NotRun)
	for _, row := range report.Matrix {
		for _, cell := range row.Cells {
			if cell.Status != MatrixStatusNotRun {
				continue
			}
			missing = append(missing, Identity{
				Target:      row.Target,
				Transport:   row.Transport,
				ImplantMode: row.ImplantMode,
				Compression: row.Compression,
				Encoder:     cell.Encoder,
			})
		}
	}
	return missing
}

// GateError returns nil only when all 192 required combinations explicitly
// pass. N/A cells never affect the gate.
func (report GlobalReport) GateError() error {
	if err := report.Validate(); err != nil {
		return fmt.Errorf("invalid shellcode coverage report: %w", err)
	}
	if report.Summary.Fail == 0 && report.Summary.NotRun == 0 {
		return nil
	}
	return fmt.Errorf(
		"shellcode coverage gate failed: %d FAIL, %d NOT RUN across %d required combinations",
		report.Summary.Fail,
		report.Summary.NotRun,
		report.Summary.Required,
	)
}

func targetReportPaths(root string) ([]string, error) {
	paths := []string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if strings.HasPrefix(name, "shellcode-coverage-") && strings.HasSuffix(name, ".json") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk shellcode report directory: %w", err)
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
		ImplantModes:  ImplantModes(),
		Compressions:  Compressions(),
		Encoders:      Encoders(),
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

	rows := make([]MatrixRow, 0, len(targets)*len(fixedTransports)*len(fixedImplantModes)*len(fixedCompressions))
	for _, target := range targets {
		for _, transport := range fixedTransports {
			for _, implantMode := range fixedImplantModes {
				for _, compression := range fixedCompressions {
					row := MatrixRow{
						Target:      target,
						Transport:   transport,
						ImplantMode: implantMode,
						Compression: compression,
						Cells:       make([]MatrixCell, 0, len(fixedEncoders)),
					}
					for _, encoder := range fixedEncoders {
						cell := MatrixCell{Encoder: encoder}
						if !EncoderSupported(target, encoder) {
							cell.Status = MatrixStatusNotApplicable
						} else if record, ok := recordsByIdentity[Identity{
							Target:      target,
							Transport:   transport,
							ImplantMode: implantMode,
							Compression: compression,
							Encoder:     encoder,
						}]; ok {
							cell.Status = string(record.Status)
							cell.Recorded = true
							cell.Duration = record.Duration
							cell.Detail = record.Detail
							cell.PayloadBytes = record.PayloadBytes
						} else {
							cell.Status = MatrixStatusNotRun
						}
						row.Cells = append(row.Cells, cell)
					}
					rows = append(rows, row)
				}
			}
		}
	}
	return rows
}

func summarize(matrix []MatrixRow) Summary {
	summary := Summary{}
	for _, row := range matrix {
		for _, cell := range row.Cells {
			summary.TotalCells++
			switch cell.Status {
			case string(coverage.StatusPass):
				summary.Recorded++
				summary.Pass++
				summary.Required++
			case string(coverage.StatusFail):
				summary.Recorded++
				summary.Fail++
				summary.Required++
			case MatrixStatusNotRun:
				summary.NotRun++
				summary.Required++
			case MatrixStatusNotApplicable:
				summary.NotApplicable++
			}
		}
	}
	return summary
}
