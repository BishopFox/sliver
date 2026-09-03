// Package shellcodecoverage records and reports the fixed Sliver shellcode
// end-to-end test matrix. Reports omit timestamps and sort every collection so
// identical observations produce byte-for-byte identical output.
package shellcodecoverage

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
	// SchemaVersion is the on-disk report schema version.
	SchemaVersion = 2

	// TargetReportKind and GlobalReportKind identify shellcode coverage JSON.
	TargetReportKind = "sliver-e2e-shellcode-target-coverage"
	GlobalReportKind = "sliver-e2e-shellcode-global-coverage"

	// GlobalJSONFilename and GlobalMarkdownFilename are the aggregate outputs.
	GlobalJSONFilename     = "shellcode-coverage.json"
	GlobalMarkdownFilename = "shellcode-coverage.md"

	// RequiredSupportedCombinations is the fixed gate denominator.
	RequiredSupportedCombinations = 168
	// MinimumSGNSamples is the minimum randomized execution depth required for
	// every supported Shikata Ga Nai matrix cell.
	MinimumSGNSamples = 4

	TransportMTLS = "mtls"
	TransportWG   = "wg"
	TransportHTTP = "http"

	ImplantModeSession = "session"
	ImplantModeBeacon  = "beacon"

	CompressionNone  = "none"
	CompressionAPLib = "aplib"

	EncoderNone         = "none"
	EncoderShikataGaNai = "shikata_ga_nai"
	EncoderXOR          = "xor"
	EncoderXORDynamic   = "xor_dynamic"
)

var fixedTargets = [...]coverage.Target{
	{OS: "darwin", Arch: "arm64"},
	{OS: "linux", Arch: "amd64"},
	{OS: "linux", Arch: "arm64"},
	{OS: "windows", Arch: "amd64"},
}

var fixedTransports = [...]string{TransportMTLS, TransportWG, TransportHTTP}
var fixedImplantModes = [...]string{ImplantModeSession, ImplantModeBeacon}
var fixedCompressions = [...]string{CompressionNone, CompressionAPLib}
var fixedEncoders = [...]string{EncoderNone, EncoderShikataGaNai, EncoderXOR, EncoderXORDynamic}

// Targets returns the fixed shellcode-generation targets in report order.
func Targets() []coverage.Target {
	return append([]coverage.Target(nil), fixedTargets[:]...)
}

// Transports returns the fixed shellcode E2E transports in report order.
func Transports() []string {
	return append([]string(nil), fixedTransports[:]...)
}

// ImplantModes returns the fixed implant modes in report order.
func ImplantModes() []string {
	return append([]string(nil), fixedImplantModes[:]...)
}

// Compressions returns the fixed shellcode compression modes in report order.
func Compressions() []string {
	return append([]string(nil), fixedCompressions[:]...)
}

// Encoders returns the fixed shellcode encoder columns in report order.
func Encoders() []string {
	return append([]string(nil), fixedEncoders[:]...)
}

// EncoderSupported reports whether encoder is a required matrix cell for
// target. Unknown targets and encoders are unsupported.
func EncoderSupported(target coverage.Target, encoder string) bool {
	if !targetSupported(target) {
		return false
	}
	switch encoder {
	case EncoderNone:
		return true
	case EncoderShikataGaNai:
		return target.Arch == "amd64"
	case EncoderXOR, EncoderXORDynamic:
		return target.Arch == "amd64" || target.Arch == "arm64"
	default:
		return false
	}
}

// Observation is the target-independent result supplied to a Recorder.
type Observation struct {
	Transport        string
	ImplantMode      string
	Compression      string
	Encoder          string
	Status           coverage.Status
	Duration         time.Duration
	Detail           string
	PayloadBytes     int64
	RequiredSamples  int
	CompletedSamples int
}

// Record is one complete, supported shellcode matrix result.
type Record struct {
	Target           coverage.Target `json:"target"`
	Transport        string          `json:"transport"`
	ImplantMode      string          `json:"implant_mode"`
	Compression      string          `json:"compression"`
	Encoder          string          `json:"encoder"`
	Status           coverage.Status `json:"status"`
	Duration         time.Duration   `json:"duration_ns"`
	Detail           string          `json:"detail"`
	PayloadBytes     int64           `json:"payload_bytes"`
	RequiredSamples  int             `json:"required_samples"`
	CompletedSamples int             `json:"completed_samples"`
}

// Identity is the unique key for one supported matrix observation.
type Identity struct {
	Target      coverage.Target
	Transport   string
	ImplantMode string
	Compression string
	Encoder     string
}

// Identity returns the unique key for record.
func (record Record) Identity() Identity {
	return Identity{
		Target:      record.Target,
		Transport:   record.Transport,
		ImplantMode: record.ImplantMode,
		Compression: record.Compression,
		Encoder:     record.Encoder,
	}
}

// String returns a stable, human-readable matrix identity.
func (identity Identity) String() string {
	return fmt.Sprintf(
		"%s/%s %s %s %s %s",
		identity.Target.OS,
		identity.Target.Arch,
		identity.Transport,
		identity.ImplantMode,
		identity.Compression,
		identity.Encoder,
	)
}

// Validate checks that record belongs to the fixed supported matrix. Recorded
// skips are rejected: unsupported encoder cells are represented synthetically
// as N/A and supported cells must explicitly pass or fail.
func (record Record) Validate() error {
	if !targetSupported(record.Target) {
		return fmt.Errorf("target %s/%s is outside the shellcode matrix", record.Target.OS, record.Target.Arch)
	}
	if !contains(fixedTransports[:], record.Transport) {
		return fmt.Errorf("transport %q is outside the shellcode matrix", record.Transport)
	}
	if !contains(fixedImplantModes[:], record.ImplantMode) {
		return fmt.Errorf("implant mode %q is outside the shellcode matrix", record.ImplantMode)
	}
	if !contains(fixedCompressions[:], record.Compression) {
		return fmt.Errorf("compression %q is outside the shellcode matrix", record.Compression)
	}
	if !contains(fixedEncoders[:], record.Encoder) {
		return fmt.Errorf("encoder %q is outside the shellcode matrix", record.Encoder)
	}
	if !EncoderSupported(record.Target, record.Encoder) {
		return fmt.Errorf("encoder %q is not supported on %s/%s", record.Encoder, record.Target.OS, record.Target.Arch)
	}
	if record.Status != coverage.StatusPass && record.Status != coverage.StatusFail {
		return fmt.Errorf("status %q is invalid: only %q and %q may be recorded", record.Status, coverage.StatusPass, coverage.StatusFail)
	}
	if record.Duration < 0 {
		return fmt.Errorf("duration must not be negative")
	}
	if record.PayloadBytes < 0 {
		return fmt.Errorf("payload bytes must not be negative")
	}
	if record.RequiredSamples <= 0 {
		return fmt.Errorf("required samples must be positive")
	}
	if record.Encoder == EncoderShikataGaNai {
		if record.RequiredSamples < MinimumSGNSamples {
			return fmt.Errorf("SGN required samples must be at least %d", MinimumSGNSamples)
		}
	} else if record.RequiredSamples != 1 {
		return fmt.Errorf("encoder %q requires exactly one sample", record.Encoder)
	}
	if record.CompletedSamples < 0 {
		return fmt.Errorf("completed samples must not be negative")
	}
	if record.CompletedSamples > record.RequiredSamples {
		return fmt.Errorf("completed samples must not exceed required samples")
	}
	if record.Status == coverage.StatusPass && record.CompletedSamples != record.RequiredSamples {
		return fmt.Errorf("passing record must complete all required samples")
	}
	if record.Status == coverage.StatusFail && record.CompletedSamples == record.RequiredSamples {
		return fmt.Errorf("failed record must have an incomplete required sample set")
	}
	if !utf8.ValidString(record.Detail) {
		return fmt.Errorf("detail must be valid UTF-8")
	}
	return nil
}

// ReportPaths contains the deterministic JSON and Markdown output paths.
type ReportPaths struct {
	JSON     string
	Markdown string
}

// TargetReport is the on-disk representation for one shellcode target.
type TargetReport struct {
	SchemaVersion int             `json:"schema_version"`
	Kind          string          `json:"kind"`
	Target        coverage.Target `json:"target"`
	Records       []Record        `json:"records"`
}

// Validate checks a per-target report and rejects duplicate records.
func (report TargetReport) Validate() error {
	if report.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported schema version %d", report.SchemaVersion)
	}
	if report.Kind != TargetReportKind {
		return fmt.Errorf("invalid report kind %q", report.Kind)
	}
	if !targetSupported(report.Target) {
		return fmt.Errorf("target %s/%s is outside the shellcode matrix", report.Target.OS, report.Target.Arch)
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

// Recorder safely collects results for one target from concurrent subtests.
type Recorder struct {
	mu      sync.Mutex
	target  coverage.Target
	records []Record
	seen    map[Identity]struct{}
}

// NewRecorder returns a recorder for one fixed shellcode target.
func NewRecorder(target coverage.Target) (*Recorder, error) {
	if !targetSupported(target) {
		return nil, fmt.Errorf("target %s/%s is outside the shellcode matrix", target.OS, target.Arch)
	}
	return &Recorder{
		target: target,
		seen:   map[Identity]struct{}{},
	}, nil
}

// Add validates and records one observation. Duplicate identities are
// rejected instead of overwriting an earlier result.
func (recorder *Recorder) Add(observation Observation) error {
	record := Record{
		Target:           recorder.target,
		Transport:        observation.Transport,
		ImplantMode:      observation.ImplantMode,
		Compression:      observation.Compression,
		Encoder:          observation.Encoder,
		Status:           observation.Status,
		Duration:         observation.Duration,
		Detail:           observation.Detail,
		PayloadBytes:     observation.PayloadBytes,
		RequiredSamples:  observation.RequiredSamples,
		CompletedSamples: observation.CompletedSamples,
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

// Records returns a sorted snapshot of all observations.
func (recorder *Recorder) Records() []Record {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	return sortedRecords(recorder.records)
}

// Write writes shellcode-coverage-<os>-<arch>.json and .md beneath dir.
func (recorder *Recorder) Write(dir string) (ReportPaths, error) {
	return WriteTargetReports(dir, recorder.target, recorder.Records())
}

// WriteTargetReports validates and writes deterministic reports for target.
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

// LoadTargetReport strictly decodes and validates one per-target JSON report.
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
		if value == candidate {
			return true
		}
	}
	return false
}

func targetReportBase(target coverage.Target) string {
	return "shellcode-coverage-" + target.OS + "-" + target.Arch
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
		stringIndex(fixedImplantModes[:], left.ImplantMode),
		stringIndex(fixedCompressions[:], left.Compression),
		stringIndex(fixedEncoders[:], left.Encoder),
	}
	rightParts := [...]int{
		targetIndex(right.Target),
		stringIndex(fixedTransports[:], right.Transport),
		stringIndex(fixedImplantModes[:], right.ImplantMode),
		stringIndex(fixedCompressions[:], right.Compression),
		stringIndex(fixedEncoders[:], right.Encoder),
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
			"target", "transport", "implant_mode", "compression", "encoder",
			"status", "duration_ns", "detail", "payload_bytes", "required_samples", "completed_samples",
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
