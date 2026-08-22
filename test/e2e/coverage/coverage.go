// Package coverage records and renders Sliver comprehensive end-to-end test
// coverage. Reports intentionally contain no timestamps so identical inputs
// produce byte-for-byte identical output.
package coverage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	// SchemaVersion is the on-disk report schema version.
	SchemaVersion = 1

	// TargetReportKind identifies a per-target report.
	TargetReportKind = "sliver-e2e-target-coverage"
	// GlobalReportKind identifies an aggregated report.
	GlobalReportKind = "sliver-e2e-global-coverage"

	// GlobalJSONFilename and GlobalMarkdownFilename are the aggregator outputs.
	GlobalJSONFilename     = "coverage-summary.json"
	GlobalMarkdownFilename = "coverage-summary.md"
)

// Status is the result of one end-to-end scenario.
type Status string

const (
	StatusPass Status = "pass"
	StatusFail Status = "fail"
	StatusSkip Status = "skip"
)

// Canonical transport and implant-mode names used by the comprehensive
// matrix. The fields remain strings so future transports and modes can still
// be represented without a schema change.
const (
	TransportMTLS = "mtls"
	TransportWG   = "wg"
	TransportHTTP = "http"

	ImplantModeSession = "session"
	ImplantModeBeacon  = "beacon"
)

// Valid reports whether s is a supported result status.
func (s Status) Valid() bool {
	switch s {
	case StatusPass, StatusFail, StatusSkip:
		return true
	default:
		return false
	}
}

// Target identifies the operating system and architecture under test.
type Target struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// Validate checks that the target is safe to use as both an identity and a
// report filename component.
func (t Target) Validate() error {
	if err := validateToken("target OS", t.OS); err != nil {
		return err
	}
	if err := validateToken("target architecture", t.Arch); err != nil {
		return err
	}
	return nil
}

// Observation is the target-independent portion of a scenario result supplied
// to a Recorder.
type Observation struct {
	Transport   string
	ImplantMode string
	GRPCMethod  string
	Scenario    string
	Status      Status
	Duration    time.Duration
	Detail      string
}

// Record is one complete scenario result. Duration is encoded as nanoseconds
// to preserve time.Duration exactly across aggregation.
type Record struct {
	TargetOS    string        `json:"target_os"`
	TargetArch  string        `json:"target_arch"`
	Transport   string        `json:"transport"`
	ImplantMode string        `json:"implant_mode"`
	GRPCMethod  string        `json:"grpc_method"`
	Scenario    string        `json:"scenario"`
	Status      Status        `json:"status"`
	Duration    time.Duration `json:"duration_ns"`
	Detail      string        `json:"detail"`
}

// Identity is the unique key for one scenario result.
type Identity struct {
	TargetOS    string
	TargetArch  string
	Transport   string
	ImplantMode string
	GRPCMethod  string
	Scenario    string
}

// Identity returns the unique key for r. Status, duration, and detail are not
// identity fields, so a scenario cannot be recorded twice with different
// outcomes.
func (r Record) Identity() Identity {
	return Identity{
		TargetOS:    r.TargetOS,
		TargetArch:  r.TargetArch,
		Transport:   r.Transport,
		ImplantMode: r.ImplantMode,
		GRPCMethod:  r.GRPCMethod,
		Scenario:    r.Scenario,
	}
}

func (i Identity) String() string {
	return fmt.Sprintf("%s/%s %s %s %s %q", i.TargetOS, i.TargetArch, i.Transport, i.ImplantMode, i.GRPCMethod, i.Scenario)
}

// Validate checks all required identity fields and result values.
func (r Record) Validate() error {
	if err := (Target{OS: r.TargetOS, Arch: r.TargetArch}).Validate(); err != nil {
		return err
	}
	if err := validateToken("transport", r.Transport); err != nil {
		return err
	}
	if err := validateToken("implant mode", r.ImplantMode); err != nil {
		return err
	}
	if err := validateText("gRPC method", r.GRPCMethod); err != nil {
		return err
	}
	if err := validateText("scenario", r.Scenario); err != nil {
		return err
	}
	if !r.Status.Valid() {
		return fmt.Errorf("invalid status %q", r.Status)
	}
	if r.Duration < 0 {
		return fmt.Errorf("duration must not be negative")
	}
	if !utf8.ValidString(r.Detail) {
		return fmt.Errorf("detail must be valid UTF-8")
	}
	return nil
}

// ReportPaths contains the JSON and Markdown files written by a report call.
type ReportPaths struct {
	JSON     string
	Markdown string
}

// TargetReport is the deterministic on-disk representation for one target.
type TargetReport struct {
	SchemaVersion int      `json:"schema_version"`
	Kind          string   `json:"kind"`
	Target        Target   `json:"target"`
	Records       []Record `json:"records"`
}

// Recorder safely collects results for one target from concurrent subtests.
type Recorder struct {
	mu      sync.Mutex
	target  Target
	records []Record
	seen    map[Identity]struct{}
}

// NewRecorder returns a recorder for target.
func NewRecorder(target Target) (*Recorder, error) {
	if err := target.Validate(); err != nil {
		return nil, err
	}
	return &Recorder{
		target: target,
		seen:   map[Identity]struct{}{},
	}, nil
}

// Add records one observation. Duplicate scenario identities are rejected at
// insertion time rather than silently overwriting an earlier outcome.
func (r *Recorder) Add(observation Observation) error {
	record := Record{
		TargetOS:    r.target.OS,
		TargetArch:  r.target.Arch,
		Transport:   observation.Transport,
		ImplantMode: observation.ImplantMode,
		GRPCMethod:  observation.GRPCMethod,
		Scenario:    observation.Scenario,
		Status:      observation.Status,
		Duration:    observation.Duration,
		Detail:      observation.Detail,
	}
	if err := record.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	identity := record.Identity()
	if _, ok := r.seen[identity]; ok {
		return fmt.Errorf("duplicate record identity: %s", identity)
	}
	r.seen[identity] = struct{}{}
	r.records = append(r.records, record)
	return nil
}

// Records returns a sorted snapshot of the recorder's results.
func (r *Recorder) Records() []Record {
	r.mu.Lock()
	defer r.mu.Unlock()
	return sortedRecords(r.records)
}

// Write writes coverage-<os>-<arch>.json and .md beneath dir.
func (r *Recorder) Write(dir string) (ReportPaths, error) {
	return WriteTargetReports(dir, r.target, r.Records())
}

// WriteTargetReports validates and writes deterministic JSON and Markdown for
// one target.
func WriteTargetReports(dir string, target Target, records []Record) (ReportPaths, error) {
	report := TargetReport{
		SchemaVersion: SchemaVersion,
		Kind:          TargetReportKind,
		Target:        target,
		Records:       sortedRecords(records),
	}
	if err := report.validate(); err != nil {
		return ReportPaths{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ReportPaths{}, fmt.Errorf("create report directory: %w", err)
	}

	base := "coverage-" + target.OS + "-" + target.Arch
	paths := ReportPaths{
		JSON:     filepath.Join(dir, base+".json"),
		Markdown: filepath.Join(dir, base+".md"),
	}
	if err := writeJSON(paths.JSON, report); err != nil {
		return ReportPaths{}, err
	}
	if err := os.WriteFile(paths.Markdown, renderTargetMarkdown(report), 0o644); err != nil {
		return ReportPaths{}, fmt.Errorf("write target Markdown report: %w", err)
	}
	return paths, nil
}

// LoadTargetReport strictly decodes and validates a per-target JSON report.
func LoadTargetReport(path string) (TargetReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return TargetReport{}, fmt.Errorf("read target report: %w", err)
	}
	if err := rejectDuplicateJSONFields(data); err != nil {
		return TargetReport{}, fmt.Errorf("decode target report: %w", err)
	}
	if err := validateTargetReportJSONKeys(data); err != nil {
		return TargetReport{}, fmt.Errorf("decode target report: %w", err)
	}
	var report TargetReport
	if err := decodeStrictJSON(data, &report); err != nil {
		return TargetReport{}, fmt.Errorf("decode target report: %w", err)
	}
	if err := report.validate(); err != nil {
		return TargetReport{}, err
	}
	report.Records = sortedRecords(report.Records)
	return report, nil
}

func (r TargetReport) validate() error {
	if r.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported schema version %d", r.SchemaVersion)
	}
	if r.Kind != TargetReportKind {
		return fmt.Errorf("invalid report kind %q", r.Kind)
	}
	if err := r.Target.Validate(); err != nil {
		return err
	}
	seen := make(map[Identity]struct{}, len(r.Records))
	for index, record := range r.Records {
		if err := record.Validate(); err != nil {
			return fmt.Errorf("record %d: %w", index, err)
		}
		if record.TargetOS != r.Target.OS || record.TargetArch != r.Target.Arch {
			return fmt.Errorf("record %d target %s/%s does not match report target %s/%s", index, record.TargetOS, record.TargetArch, r.Target.OS, r.Target.Arch)
		}
		identity := record.Identity()
		if _, ok := seen[identity]; ok {
			return fmt.Errorf("duplicate record identity: %s", identity)
		}
		seen[identity] = struct{}{}
	}
	return nil
}

func validateText(name, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", name)
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s must be valid UTF-8", name)
	}
	if value != strings.TrimSpace(value) {
		return fmt.Errorf("%s must not have leading or trailing whitespace", name)
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return fmt.Errorf("%s must not contain control characters", name)
		}
	}
	return nil
}

func validateToken(name, value string) error {
	if err := validateText(name, value); err != nil {
		return err
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '-' || character == '_' || character == '.' || character == '+' {
			continue
		}
		return fmt.Errorf("%s contains invalid character %q", name, character)
	}
	return nil
}

func sortedRecords(records []Record) []Record {
	result := make([]Record, len(records))
	copy(result, records)
	sort.Slice(result, func(i, j int) bool {
		return compareIdentity(result[i].Identity(), result[j].Identity()) < 0
	})
	return result
}

func compareIdentity(left, right Identity) int {
	leftParts := [...]string{left.TargetOS, left.TargetArch, left.Transport, left.ImplantMode, left.GRPCMethod, left.Scenario}
	rightParts := [...]string{right.TargetOS, right.TargetArch, right.Transport, right.ImplantMode, right.GRPCMethod, right.Scenario}
	for index := range leftParts {
		if comparison := strings.Compare(leftParts[index], rightParts[index]); comparison != 0 {
			return comparison
		}
	}
	return 0
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode JSON report: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write JSON report: %w", err)
	}
	return nil
}

func decodeStrictJSON(data []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON value")
		}
		return fmt.Errorf("invalid trailing JSON: %w", err)
	}
	return nil
}

func rejectDuplicateJSONFields(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := walkJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON value")
		}
		return fmt.Errorf("invalid trailing JSON: %w", err)
	}
	return nil
}

func walkJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]struct{}{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("JSON object key is not a string")
			}
			foldedKey := strings.ToLower(key)
			if _, ok := seen[foldedKey]; ok {
				return fmt.Errorf("duplicate JSON field %q", foldedKey)
			}
			seen[foldedKey] = struct{}{}
			if err := walkJSONValue(decoder); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := walkJSONValue(decoder); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	closing, err := decoder.Token()
	if err != nil {
		return err
	}
	closingDelimiter, ok := closing.(json.Delim)
	if !ok || (delimiter == '{' && closingDelimiter != '}') || (delimiter == '[' && closingDelimiter != ']') {
		return fmt.Errorf("mismatched JSON delimiter")
	}
	return nil
}

func validateTargetReportJSONKeys(data []byte) error {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return err
	}
	if err := validateExactJSONKeys("report", root, "schema_version", "kind", "target", "records"); err != nil {
		return err
	}
	if targetData, ok := root["target"]; ok {
		var target map[string]json.RawMessage
		if err := json.Unmarshal(targetData, &target); err != nil {
			return fmt.Errorf("target must be an object: %w", err)
		}
		if err := validateExactJSONKeys("target", target, "os", "arch"); err != nil {
			return err
		}
	}
	if recordsData, ok := root["records"]; ok && string(recordsData) != "null" {
		var records []json.RawMessage
		if err := json.Unmarshal(recordsData, &records); err != nil {
			return fmt.Errorf("records must be an array: %w", err)
		}
		for index, recordData := range records {
			var record map[string]json.RawMessage
			if err := json.Unmarshal(recordData, &record); err != nil {
				return fmt.Errorf("record %d must be an object: %w", index, err)
			}
			if err := validateExactJSONKeys(
				fmt.Sprintf("record %d", index),
				record,
				"target_os", "target_arch", "transport", "implant_mode",
				"grpc_method", "scenario", "status", "duration_ns", "detail",
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateExactJSONKeys(context string, object map[string]json.RawMessage, allowed ...string) error {
	allowedKeys := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		allowedKeys[key] = struct{}{}
	}
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, ok := allowedKeys[key]; !ok {
			return fmt.Errorf("unknown field %q in %s (JSON keys must use canonical case)", key, context)
		}
	}
	return nil
}
