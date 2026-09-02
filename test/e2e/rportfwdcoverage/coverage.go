// Package rportfwdcoverage records the fixed reverse-port-forward end-to-end
// scenarios independently of the comprehensive command coverage catalog.
// Reports are deterministic and intentionally omit timestamps.
package rportfwdcoverage

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
	"unicode/utf8"

	coverage "github.com/bishopfox/sliver/test/e2e/coverage"
)

const (
	SchemaVersion = 1

	TargetReportKind = "sliver-e2e-rportfwd-target-coverage"
	GlobalReportKind = "sliver-e2e-rportfwd-global-coverage"

	GlobalJSONFilename     = "rportfwd-coverage.json"
	GlobalMarkdownFilename = "rportfwd-coverage.md"

	TransportMTLS = "mtls"
	TransportWG   = "wg"
	TransportHTTP = "http"

	ScenarioInvalidDestination          = "invalid-destination"
	ScenarioInvalidSessionStopIsolation = "invalid-session-stop-isolation"
	ScenarioMetadataAuthority           = "metadata-authority"
	ScenarioBidirectionalEcho           = "bidirectional-echo"
	ScenarioSequentialConnections       = "sequential-connections"
	ScenarioConcurrentConnections       = "concurrent-connections"
	ScenarioEstablishedBeforeStop       = "established-before-stop"
	ScenarioStopClosesEstablished       = "stop-closes-established"
	ScenarioDisconnectClosesEstablished = "disconnect-closes-established"

	RequiredCombinations = 8 * 3 * 9
)

var fixedTargets = [...]coverage.Target{
	{OS: "darwin", Arch: "amd64"},
	{OS: "darwin", Arch: "arm64"},
	{OS: "linux", Arch: "386"},
	{OS: "linux", Arch: "amd64"},
	{OS: "linux", Arch: "arm64"},
	{OS: "windows", Arch: "386"},
	{OS: "windows", Arch: "amd64"},
	{OS: "windows", Arch: "arm64"},
}

var fixedTransports = [...]string{TransportMTLS, TransportWG, TransportHTTP}

var fixedScenarios = [...]string{
	ScenarioInvalidDestination,
	ScenarioInvalidSessionStopIsolation,
	ScenarioMetadataAuthority,
	ScenarioBidirectionalEcho,
	ScenarioSequentialConnections,
	ScenarioConcurrentConnections,
	ScenarioEstablishedBeforeStop,
	ScenarioStopClosesEstablished,
	ScenarioDisconnectClosesEstablished,
}

// Targets returns the complete platform matrix in canonical report order.
func Targets() []coverage.Target {
	return append([]coverage.Target(nil), fixedTargets[:]...)
}

// Transports returns the required transports in canonical report order.
func Transports() []string {
	return append([]string(nil), fixedTransports[:]...)
}

// Scenarios returns the stable required scenario identifiers in report order.
func Scenarios() []string {
	return append([]string(nil), fixedScenarios[:]...)
}

// Observation is one target-independent scenario result supplied to Recorder.
type Observation struct {
	Transport string
	Scenario  string
	Status    coverage.Status
	Duration  time.Duration
	Detail    string
}

// Record is one complete target/transport/scenario observation.
type Record struct {
	Target    coverage.Target `json:"target"`
	Transport string          `json:"transport"`
	Scenario  string          `json:"scenario"`
	Status    coverage.Status `json:"status"`
	Duration  time.Duration   `json:"duration_ns"`
	Detail    string          `json:"detail"`
}

// Identity is the unique key for one required scenario cell.
type Identity struct {
	Target    coverage.Target
	Transport string
	Scenario  string
}

func (record Record) Identity() Identity {
	return Identity{Target: record.Target, Transport: record.Transport, Scenario: record.Scenario}
}

func (identity Identity) String() string {
	return fmt.Sprintf("%s/%s %s %s", identity.Target.OS, identity.Target.Arch, identity.Transport, identity.Scenario)
}

// Validate checks that a record belongs to the fixed matrix. Unsupported cells
// are not represented, so every recorded result must explicitly pass or fail.
func (record Record) Validate() error {
	if !targetSupported(record.Target) {
		return fmt.Errorf("target %s/%s is outside the rportfwd matrix", record.Target.OS, record.Target.Arch)
	}
	if !contains(fixedTransports[:], record.Transport) {
		return fmt.Errorf("transport %q is outside the rportfwd matrix", record.Transport)
	}
	if !contains(fixedScenarios[:], record.Scenario) {
		return fmt.Errorf("scenario %q is outside the rportfwd matrix", record.Scenario)
	}
	if record.Status != coverage.StatusPass && record.Status != coverage.StatusFail {
		return fmt.Errorf("status %q is invalid: only %q and %q may be recorded", record.Status, coverage.StatusPass, coverage.StatusFail)
	}
	if record.Duration < 0 {
		return fmt.Errorf("duration must not be negative")
	}
	if !utf8.ValidString(record.Detail) {
		return fmt.Errorf("detail must be valid UTF-8")
	}
	if record.Status == coverage.StatusFail && strings.TrimSpace(record.Detail) == "" {
		return fmt.Errorf("failed record must include detail")
	}
	return nil
}

// ReportPaths contains deterministic JSON and Markdown output paths.
type ReportPaths struct {
	JSON     string
	Markdown string
}

// TargetReport is the on-disk representation for one OS/architecture cell.
// Partial and failed reports are valid artifacts; ValidateComplete applies the
// success gate separately.
type TargetReport struct {
	SchemaVersion int             `json:"schema_version"`
	Kind          string          `json:"kind"`
	Target        coverage.Target `json:"target"`
	Records       []Record        `json:"records"`
}

// Validate checks report structure and rejects duplicate identities.
func (report TargetReport) Validate() error {
	if report.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported schema version %d", report.SchemaVersion)
	}
	if report.Kind != TargetReportKind {
		return fmt.Errorf("invalid report kind %q", report.Kind)
	}
	if !targetSupported(report.Target) {
		return fmt.Errorf("target %s/%s is outside the rportfwd matrix", report.Target.OS, report.Target.Arch)
	}
	seen := make(map[Identity]struct{}, len(report.Records))
	for index, record := range report.Records {
		if err := record.Validate(); err != nil {
			return fmt.Errorf("record %d: %w", index, err)
		}
		if record.Target != report.Target {
			return fmt.Errorf("record %d target %s/%s does not match report target %s/%s", index, record.Target.OS, record.Target.Arch, report.Target.OS, report.Target.Arch)
		}
		identity := record.Identity()
		if _, ok := seen[identity]; ok {
			return fmt.Errorf("duplicate record identity: %s", identity)
		}
		seen[identity] = struct{}{}
	}
	return nil
}

// ValidateComplete requires the exact selected-transport by required-scenario
// cross product to be present and passing. Records for an unselected transport
// are rejected so a stale artifact cannot satisfy a narrower invocation.
func (report TargetReport) ValidateComplete(selectedTransports []string) error {
	if err := report.Validate(); err != nil {
		return err
	}
	selected, err := NormalizeTransports(selectedTransports)
	if err != nil {
		return err
	}
	wanted := make(map[string]struct{}, len(selected))
	for _, transport := range selected {
		wanted[transport] = struct{}{}
	}
	records := make(map[Identity]Record, len(report.Records))
	for _, record := range report.Records {
		if _, ok := wanted[record.Transport]; !ok {
			return fmt.Errorf("report contains unselected transport %q", record.Transport)
		}
		records[record.Identity()] = record
	}

	failed := 0
	missing := 0
	for _, transport := range selected {
		for _, scenario := range fixedScenarios {
			record, ok := records[Identity{Target: report.Target, Transport: transport, Scenario: scenario}]
			if !ok {
				missing++
				continue
			}
			if record.Status == coverage.StatusFail {
				failed++
			}
		}
	}
	if failed != 0 || missing != 0 {
		return fmt.Errorf(
			"rportfwd target coverage incomplete for %s/%s: %d FAIL, %d NOT RUN across %d required combinations",
			report.Target.OS,
			report.Target.Arch,
			failed,
			missing,
			len(selected)*len(fixedScenarios),
		)
	}
	return nil
}

// Recorder safely collects results for one target from concurrent steps.
type Recorder struct {
	mu      sync.Mutex
	target  coverage.Target
	records []Record
	seen    map[Identity]struct{}
}

func NewRecorder(target coverage.Target) (*Recorder, error) {
	if !targetSupported(target) {
		return nil, fmt.Errorf("target %s/%s is outside the rportfwd matrix", target.OS, target.Arch)
	}
	return &Recorder{target: target, seen: map[Identity]struct{}{}}, nil
}

// Add validates and records one observation without overwriting duplicates.
func (recorder *Recorder) Add(observation Observation) error {
	record := Record{
		Target:    recorder.target,
		Transport: observation.Transport,
		Scenario:  observation.Scenario,
		Status:    observation.Status,
		Duration:  observation.Duration,
		Detail:    observation.Detail,
	}
	if err := record.Validate(); err != nil {
		return err
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	identity := record.Identity()
	if _, ok := recorder.seen[identity]; ok {
		return fmt.Errorf("duplicate record identity: %s", identity)
	}
	recorder.seen[identity] = struct{}{}
	recorder.records = append(recorder.records, record)
	return nil
}

// Records returns a canonically sorted copy of all observations.
func (recorder *Recorder) Records() []Record {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	return sortedRecords(recorder.records)
}

// ValidateComplete applies the selected-transport completion gate to the
// recorder's current snapshot.
func (recorder *Recorder) ValidateComplete(selectedTransports []string) error {
	return recorder.report().ValidateComplete(selectedTransports)
}

// Write writes rportfwd-coverage-<os>-<arch>.json and .md beneath dir. It does
// not require completeness so a failing run can preserve partial diagnostics.
func (recorder *Recorder) Write(dir string) (ReportPaths, error) {
	report := recorder.report()
	return WriteTargetReports(dir, report.Target, report.Records)
}

func (recorder *Recorder) report() TargetReport {
	return TargetReport{
		SchemaVersion: SchemaVersion,
		Kind:          TargetReportKind,
		Target:        recorder.target,
		Records:       recorder.Records(),
	}
}

// TargetJSONFilename returns the exact per-target sentinel filename.
func TargetJSONFilename(target coverage.Target) string {
	return targetReportBase(target) + ".json"
}

// WriteTargetReports validates and writes deterministic target artifacts.
func WriteTargetReports(dir string, target coverage.Target, records []Record) (ReportPaths, error) {
	report := TargetReport{
		SchemaVersion: SchemaVersion,
		Kind:          TargetReportKind,
		Target:        target,
		Records:       sortedRecords(records),
	}
	if err := report.Validate(); err != nil {
		return ReportPaths{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ReportPaths{}, fmt.Errorf("create report directory: %w", err)
	}
	base := targetReportBase(target)
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

// LoadTargetReport strictly decodes and validates a per-target JSON artifact.
func LoadTargetReport(path string) (TargetReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return TargetReport{}, fmt.Errorf("read target report: %w", err)
	}
	if !utf8.Valid(data) {
		return TargetReport{}, fmt.Errorf("decode target report: JSON must be valid UTF-8")
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
	if err := report.Validate(); err != nil {
		return TargetReport{}, err
	}
	report.Records = sortedRecords(report.Records)
	return report, nil
}

// NormalizeTransports validates a nonempty selected transport set and returns
// it in canonical matrix order.
func NormalizeTransports(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("at least one transport is required")
	}
	wanted := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if !contains(fixedTransports[:], value) {
			return nil, fmt.Errorf("unknown rportfwd transport %q", value)
		}
		if _, duplicate := wanted[value]; duplicate {
			return nil, fmt.Errorf("duplicate rportfwd transport %q", value)
		}
		wanted[value] = struct{}{}
	}
	result := make([]string, 0, len(wanted))
	for _, transport := range fixedTransports {
		if _, ok := wanted[transport]; ok {
			result = append(result, transport)
		}
	}
	return result, nil
}

func targetSupported(target coverage.Target) bool {
	for _, candidate := range fixedTargets {
		if target == candidate {
			return true
		}
	}
	return false
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func targetReportBase(target coverage.Target) string {
	return "rportfwd-coverage-" + target.OS + "-" + target.Arch
}

func sortedRecords(records []Record) []Record {
	result := make([]Record, len(records))
	copy(result, records)
	sort.Slice(result, func(left, right int) bool {
		return compareIdentity(result[left].Identity(), result[right].Identity()) < 0
	})
	return result
}

func compareIdentity(left, right Identity) int {
	leftParts := [...]int{
		targetIndex(left.Target),
		stringIndex(fixedTransports[:], left.Transport),
		stringIndex(fixedScenarios[:], left.Scenario),
	}
	rightParts := [...]int{
		targetIndex(right.Target),
		stringIndex(fixedTransports[:], right.Transport),
		stringIndex(fixedScenarios[:], right.Scenario),
	}
	for index := range leftParts {
		if leftParts[index] < rightParts[index] {
			return -1
		}
		if leftParts[index] > rightParts[index] {
			return 1
		}
	}
	return 0
}

func targetIndex(target coverage.Target) int {
	for index, candidate := range fixedTargets {
		if target == candidate {
			return index
		}
	}
	return len(fixedTargets)
}

func stringIndex(values []string, value string) int {
	for index, candidate := range values {
		if value == candidate {
			return index
		}
	}
	return len(values)
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
	if err := validateTargetJSONKeys("target", root["target"]); err != nil {
		return err
	}
	if bytes.Equal(bytes.TrimSpace(root["records"]), []byte("null")) {
		return fmt.Errorf("records must be an array, not null")
	}
	var records []json.RawMessage
	if err := json.Unmarshal(root["records"], &records); err != nil {
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
			"target", "transport", "scenario", "status", "duration_ns", "detail",
		); err != nil {
			return err
		}
		if err := validateTargetJSONKeys(fmt.Sprintf("record %d target", index), record["target"]); err != nil {
			return err
		}
	}
	return nil
}

func validateTargetJSONKeys(context string, data json.RawMessage) error {
	var target map[string]json.RawMessage
	if err := json.Unmarshal(data, &target); err != nil {
		return fmt.Errorf("%s must be an object: %w", context, err)
	}
	return validateExactJSONKeys(context, target, "os", "arch")
}

func validateExactJSONKeys(context string, object map[string]json.RawMessage, required ...string) error {
	requiredKeys := make(map[string]struct{}, len(required))
	for _, key := range required {
		requiredKeys[key] = struct{}{}
	}
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, ok := requiredKeys[key]; !ok {
			return fmt.Errorf("unknown field %q in %s (JSON keys must use canonical case)", key, context)
		}
	}
	for _, key := range required {
		if _, ok := object[key]; !ok {
			return fmt.Errorf("missing field %q in %s", key, context)
		}
	}
	return nil
}
