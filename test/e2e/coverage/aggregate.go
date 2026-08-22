package coverage

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// MatrixStatusNotRun marks a cross-product cell for which no scenario record
// was found. It is intentionally not a valid Record status.
const MatrixStatusNotRun = "not_run"

// Dimensions declares the expected cross-product axes. A nonempty axis is
// strict: observed identities outside it are rejected. Empty axes are inferred
// from records for focused callers and tests.
type Dimensions struct {
	Targets      []Target
	Transports   []string
	ImplantModes []string
	Commands     []CommandExpectation
}

// ComprehensiveDimensions returns the requested Sliver E2E matrix. Darwin is
// the Go operating-system name for macOS and wg is Sliver's WireGuard transport
// name.
func ComprehensiveDimensions() Dimensions {
	return Dimensions{
		Targets:      comprehensiveTargets(),
		Transports:   []string{TransportMTLS, TransportWG, TransportHTTP},
		ImplantModes: []string{ImplantModeSession, ImplantModeBeacon},
		Commands:     ComprehensiveCatalog(),
	}
}

// Command identifies one command scenario represented by a matrix row.
type Command struct {
	GRPCMethod string `json:"grpc_method"`
	Scenario   string `json:"scenario"`
}

// MatrixCell is one target/transport/mode result in a global matrix row.
type MatrixCell struct {
	TargetOS    string        `json:"target_os"`
	TargetArch  string        `json:"target_arch"`
	Transport   string        `json:"transport"`
	ImplantMode string        `json:"implant_mode"`
	Status      string        `json:"status"`
	Duration    time.Duration `json:"duration_ns"`
	Detail      string        `json:"detail"`
	Recorded    bool          `json:"recorded"`
}

// MatrixRow contains the complete cross product for one gRPC method/scenario.
type MatrixRow struct {
	GRPCMethod string       `json:"grpc_method"`
	Scenario   string       `json:"scenario"`
	Cells      []MatrixCell `json:"cells"`
}

// Summary contains aggregate record and matrix counts.
type Summary struct {
	Recorded   int `json:"recorded"`
	Pass       int `json:"pass"`
	Fail       int `json:"fail"`
	Skip       int `json:"skip"`
	NotRun     int `json:"not_run"`
	TotalCells int `json:"total_cells"`
}

// GlobalReport is the deterministic aggregate JSON representation.
type GlobalReport struct {
	SchemaVersion   int              `json:"schema_version"`
	Kind            string           `json:"kind"`
	Targets         []Target         `json:"targets"`
	Transports      []string         `json:"transports"`
	ImplantModes    []string         `json:"implant_modes"`
	Commands        []Command        `json:"commands"`
	RPCDispositions []RPCDisposition `json:"rpc_dispositions"`
	Summary         Summary          `json:"summary"`
	Records         []Record         `json:"records"`
	Matrix          []MatrixRow      `json:"matrix"`
}

// FailedRecords returns a sorted copy of all failed scenario records.
func (r GlobalReport) FailedRecords() []Record {
	failed := make([]Record, 0, r.Summary.Fail)
	for _, record := range r.Records {
		if record.Status == StatusFail {
			failed = append(failed, record)
		}
	}
	return sortedRecords(failed)
}

// RecordedSkipRecords returns a sorted copy of all explicitly recorded skip
// results. Catalog validation rejects records for unsupported targets, so in a
// strict catalog aggregate every result returned here is a skipped scenario on
// a supported target/transport/mode cell. Synthetic platform skips are matrix
// cells with Recorded=false and are intentionally excluded.
func (r GlobalReport) RecordedSkipRecords() []Record {
	skipped := make([]Record, 0, r.Summary.Skip)
	for _, record := range r.Records {
		if record.Status == StatusSkip {
			skipped = append(skipped, record)
		}
	}
	return sortedRecords(skipped)
}

// NotRunIdentities returns every required cross-product identity without a
// recorded outcome. Expected platform-specific skips are not missing records.
func (r GlobalReport) NotRunIdentities() []Identity {
	missing := make([]Identity, 0, r.Summary.NotRun)
	for _, row := range r.Matrix {
		for _, cell := range row.Cells {
			if cell.Status != MatrixStatusNotRun {
				continue
			}
			missing = append(missing, Identity{
				TargetOS:    cell.TargetOS,
				TargetArch:  cell.TargetArch,
				Transport:   cell.Transport,
				ImplantMode: cell.ImplantMode,
				GRPCMethod:  row.GRPCMethod,
				Scenario:    row.Scenario,
			})
		}
	}
	return missing
}

// AggregateDirectory recursively finds per-target coverage-*.json reports and
// builds the global cross-product report. Malformed reports and identities
// duplicated within or across source files are rejected.
func AggregateDirectory(root string, expected Dimensions) (GlobalReport, error) {
	paths, err := targetReportPaths(root)
	if err != nil {
		return GlobalReport{}, err
	}
	if len(paths) == 0 && (len(expected.Targets) == 0 || len(expected.Transports) == 0 || len(expected.ImplantModes) == 0 || len(expected.Commands) == 0) {
		return GlobalReport{}, fmt.Errorf("no per-target coverage JSON reports found beneath %s", root)
	}

	records := make([]Record, 0)
	seen := map[Identity]string{}
	for _, path := range paths {
		report, err := LoadTargetReport(path)
		if err != nil {
			return GlobalReport{}, fmt.Errorf("load %s: %w", path, err)
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
	records = sortedRecords(records)

	dimensions, err := normalizedDimensions(expected, records)
	if err != nil {
		return GlobalReport{}, err
	}
	commands, expectations, err := normalizedCommands(expected.Commands, dimensions.Targets, records)
	if err != nil {
		return GlobalReport{}, err
	}
	matrix := buildMatrix(dimensions, commands, expectations, records)
	summary := summarize(matrix)
	return GlobalReport{
		SchemaVersion:   SchemaVersion,
		Kind:            GlobalReportKind,
		Targets:         dimensions.Targets,
		Transports:      dimensions.Transports,
		ImplantModes:    dimensions.ImplantModes,
		Commands:        commands,
		RPCDispositions: ComprehensiveRPCDispositions(),
		Summary:         summary,
		Records:         records,
		Matrix:          matrix,
	}, nil
}

// WriteGlobalReports writes coverage-summary.json and coverage-summary.md.
func WriteGlobalReports(dir string, report GlobalReport) (ReportPaths, error) {
	if err := report.validate(); err != nil {
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

func (r GlobalReport) validate() error {
	if r.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported global schema version %d", r.SchemaVersion)
	}
	if r.Kind != GlobalReportKind {
		return fmt.Errorf("invalid global report kind %q", r.Kind)
	}
	if !reflect.DeepEqual(r.RPCDispositions, ComprehensiveRPCDispositions()) {
		return fmt.Errorf("global report RPC dispositions do not match the comprehensive SliverRPC registry")
	}
	targets := map[Target]struct{}{}
	for _, target := range r.Targets {
		if err := target.Validate(); err != nil {
			return err
		}
		if _, ok := targets[target]; ok {
			return fmt.Errorf("duplicate global target %s/%s", target.OS, target.Arch)
		}
		targets[target] = struct{}{}
	}
	transports, err := validateGlobalTokens("transport", r.Transports)
	if err != nil {
		return err
	}
	modes, err := validateGlobalTokens("implant mode", r.ImplantModes)
	if err != nil {
		return err
	}

	commands := make(map[Command]struct{}, len(r.Commands))
	for index, command := range r.Commands {
		if err := validateText("global gRPC method", command.GRPCMethod); err != nil {
			return fmt.Errorf("global command %d: %w", index, err)
		}
		if err := validateText("global scenario", command.Scenario); err != nil {
			return fmt.Errorf("global command %d: %w", index, err)
		}
		if _, ok := commands[command]; ok {
			return fmt.Errorf("duplicate global command %s %q", command.GRPCMethod, command.Scenario)
		}
		if index > 0 && compareCommand(r.Commands[index-1], command) >= 0 {
			return fmt.Errorf("global commands are not in canonical order")
		}
		commands[command] = struct{}{}
	}

	records := make(map[Identity]Record, len(r.Records))
	for index, record := range r.Records {
		if err := record.Validate(); err != nil {
			return fmt.Errorf("global record %d: %w", index, err)
		}
		if _, ok := targets[Target{OS: record.TargetOS, Arch: record.TargetArch}]; !ok {
			return fmt.Errorf("global record %d has target outside matrix", index)
		}
		if _, ok := transports[record.Transport]; !ok {
			return fmt.Errorf("global record %d has transport outside matrix", index)
		}
		if _, ok := modes[record.ImplantMode]; !ok {
			return fmt.Errorf("global record %d has implant mode outside matrix", index)
		}
		if _, ok := commands[Command{GRPCMethod: record.GRPCMethod, Scenario: record.Scenario}]; !ok {
			return fmt.Errorf("global record %d has command outside matrix", index)
		}
		identity := record.Identity()
		if _, ok := records[identity]; ok {
			return fmt.Errorf("duplicate global record identity %s", identity)
		}
		if index > 0 && compareIdentity(r.Records[index-1].Identity(), identity) >= 0 {
			return fmt.Errorf("global records are not in canonical order")
		}
		records[identity] = record
	}

	if len(r.Matrix) != len(r.Commands) {
		return fmt.Errorf("global matrix has %d rows, want %d", len(r.Matrix), len(r.Commands))
	}
	cellCount := len(r.Targets) * len(r.Transports) * len(r.ImplantModes)
	matchedRecords := map[Identity]struct{}{}
	for rowIndex, row := range r.Matrix {
		command := r.Commands[rowIndex]
		if row.GRPCMethod != command.GRPCMethod || row.Scenario != command.Scenario {
			return fmt.Errorf("global matrix row %d does not match command catalog", rowIndex)
		}
		if len(row.Cells) != cellCount {
			return fmt.Errorf("global matrix row %d has %d cells, want %d", rowIndex, len(row.Cells), cellCount)
		}
		cellIndex := 0
		for _, target := range r.Targets {
			for _, transport := range r.Transports {
				for _, mode := range r.ImplantModes {
					cell := row.Cells[cellIndex]
					cellIndex++
					if cell.TargetOS != target.OS || cell.TargetArch != target.Arch || cell.Transport != transport || cell.ImplantMode != mode {
						return fmt.Errorf("global matrix row %d cell %d is out of canonical order", rowIndex, cellIndex-1)
					}
					if cell.Duration < 0 {
						return fmt.Errorf("global matrix row %d cell %d has negative duration", rowIndex, cellIndex-1)
					}
					if !utf8.ValidString(cell.Detail) {
						return fmt.Errorf("global matrix row %d cell %d detail is not valid UTF-8", rowIndex, cellIndex-1)
					}
					identity := Identity{
						TargetOS: target.OS, TargetArch: target.Arch,
						Transport: transport, ImplantMode: mode,
						GRPCMethod: command.GRPCMethod, Scenario: command.Scenario,
					}
					record, hasRecord := records[identity]
					if cell.Recorded {
						if !hasRecord {
							return fmt.Errorf("global matrix cell %s is recorded without a raw record", identity)
						}
						if cell.Status != string(record.Status) || cell.Duration != record.Duration || cell.Detail != record.Detail {
							return fmt.Errorf("global matrix cell %s does not match its raw record", identity)
						}
						matchedRecords[identity] = struct{}{}
						continue
					}
					if hasRecord {
						return fmt.Errorf("global matrix cell %s has an unreferenced raw record", identity)
					}
					switch cell.Status {
					case MatrixStatusNotRun:
						if cell.Duration != 0 || cell.Detail != "" {
							return fmt.Errorf("NOT RUN matrix cell %s contains result data", identity)
						}
					case string(StatusSkip):
						if cell.Duration != 0 || cell.Detail == "" {
							return fmt.Errorf("expected SKIP matrix cell %s is missing its reason", identity)
						}
					default:
						return fmt.Errorf("unrecorded matrix cell %s has invalid status %q", identity, cell.Status)
					}
				}
			}
		}
	}
	if len(matchedRecords) != len(records) {
		return fmt.Errorf("global matrix references %d of %d raw records", len(matchedRecords), len(records))
	}
	if summary := summarize(r.Matrix); summary != r.Summary {
		return fmt.Errorf("global summary does not match matrix: got %+v, want %+v", r.Summary, summary)
	}
	return nil
}

func validateGlobalTokens(name string, values []string) (map[string]struct{}, error) {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if err := validateToken("global "+name, value); err != nil {
			return nil, err
		}
		if _, ok := seen[value]; ok {
			return nil, fmt.Errorf("duplicate global %s %q", name, value)
		}
		seen[value] = struct{}{}
	}
	return seen, nil
}

func compareCommand(left, right Command) int {
	if comparison := strings.Compare(left.GRPCMethod, right.GRPCMethod); comparison != 0 {
		return comparison
	}
	return strings.Compare(left.Scenario, right.Scenario)
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
		if name == GlobalJSONFilename {
			return nil
		}
		if strings.HasPrefix(name, "coverage-") && strings.HasSuffix(name, ".json") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk report directory: %w", err)
	}
	sort.Strings(paths)
	return paths, nil
}

func normalizedDimensions(expected Dimensions, records []Record) (Dimensions, error) {
	result := Dimensions{}
	targetSeen := map[Target]struct{}{}
	for _, target := range expected.Targets {
		if err := target.Validate(); err != nil {
			return Dimensions{}, fmt.Errorf("expected target: %w", err)
		}
		if _, ok := targetSeen[target]; ok {
			return Dimensions{}, fmt.Errorf("duplicate expected target %s/%s", target.OS, target.Arch)
		}
		targetSeen[target] = struct{}{}
		result.Targets = append(result.Targets, target)
	}
	if len(expected.Targets) == 0 {
		for _, record := range records {
			targetSeen[Target{OS: record.TargetOS, Arch: record.TargetArch}] = struct{}{}
		}
		for target := range targetSeen {
			result.Targets = append(result.Targets, target)
		}
		sort.Slice(result.Targets, func(i, j int) bool {
			if result.Targets[i].OS != result.Targets[j].OS {
				return result.Targets[i].OS < result.Targets[j].OS
			}
			return result.Targets[i].Arch < result.Targets[j].Arch
		})
	} else {
		for _, record := range records {
			target := Target{OS: record.TargetOS, Arch: record.TargetArch}
			if _, ok := targetSeen[target]; !ok {
				return Dimensions{}, fmt.Errorf("unexpected target %s/%s", target.OS, target.Arch)
			}
		}
	}

	transportSeen := map[string]struct{}{}
	for _, transport := range expected.Transports {
		if err := validateToken("expected transport", transport); err != nil {
			return Dimensions{}, err
		}
		if _, ok := transportSeen[transport]; ok {
			return Dimensions{}, fmt.Errorf("duplicate expected transport %q", transport)
		}
		transportSeen[transport] = struct{}{}
		result.Transports = append(result.Transports, transport)
	}
	if len(expected.Transports) == 0 {
		for _, record := range records {
			transportSeen[record.Transport] = struct{}{}
		}
		result.Transports = sortedSet(transportSeen)
	} else {
		for _, record := range records {
			if _, ok := transportSeen[record.Transport]; !ok {
				return Dimensions{}, fmt.Errorf("unexpected transport %q", record.Transport)
			}
		}
	}

	modeSeen := map[string]struct{}{}
	for _, mode := range expected.ImplantModes {
		if err := validateToken("expected implant mode", mode); err != nil {
			return Dimensions{}, err
		}
		if _, ok := modeSeen[mode]; ok {
			return Dimensions{}, fmt.Errorf("duplicate expected implant mode %q", mode)
		}
		modeSeen[mode] = struct{}{}
		result.ImplantModes = append(result.ImplantModes, mode)
	}
	if len(expected.ImplantModes) == 0 {
		for _, record := range records {
			modeSeen[record.ImplantMode] = struct{}{}
		}
		result.ImplantModes = sortedSet(modeSeen)
	} else {
		for _, record := range records {
			if _, ok := modeSeen[record.ImplantMode]; !ok {
				return Dimensions{}, fmt.Errorf("unexpected implant mode %q", record.ImplantMode)
			}
		}
	}
	return result, nil
}

func sortedSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func commandsFromRecords(records []Record) []Command {
	seen := map[Command]struct{}{}
	for _, record := range records {
		seen[Command{GRPCMethod: record.GRPCMethod, Scenario: record.Scenario}] = struct{}{}
	}
	commands := make([]Command, 0, len(seen))
	for command := range seen {
		commands = append(commands, command)
	}
	sort.Slice(commands, func(i, j int) bool {
		if commands[i].GRPCMethod != commands[j].GRPCMethod {
			return commands[i].GRPCMethod < commands[j].GRPCMethod
		}
		return commands[i].Scenario < commands[j].Scenario
	})
	return commands
}

func normalizedCommands(expected []CommandExpectation, targets []Target, records []Record) ([]Command, map[Command]CommandExpectation, error) {
	if len(expected) == 0 {
		commands := commandsFromRecords(records)
		expectations := make(map[Command]CommandExpectation, len(commands))
		for _, command := range commands {
			expectations[command] = CommandExpectation{
				Command:          command,
				SupportedTargets: append([]Target(nil), targets...),
			}
		}
		return commands, expectations, nil
	}

	targetSet := make(map[Target]struct{}, len(targets))
	for _, target := range targets {
		targetSet[target] = struct{}{}
	}
	expectations := make(map[Command]CommandExpectation, len(expected))
	commands := make([]Command, 0, len(expected))
	for index, expectation := range expected {
		if err := validateText("expected gRPC method", expectation.GRPCMethod); err != nil {
			return nil, nil, fmt.Errorf("expected command %d: %w", index, err)
		}
		if err := validateText("expected scenario", expectation.Scenario); err != nil {
			return nil, nil, fmt.Errorf("expected command %d: %w", index, err)
		}
		command := expectation.Command
		if _, ok := expectations[command]; ok {
			return nil, nil, fmt.Errorf("duplicate expected command %s %q", command.GRPCMethod, command.Scenario)
		}
		if len(expectation.SupportedTargets) == 0 {
			return nil, nil, fmt.Errorf("expected command %s %q has no supported targets", command.GRPCMethod, command.Scenario)
		}
		supportedSeen := map[Target]struct{}{}
		for _, target := range expectation.SupportedTargets {
			if err := target.Validate(); err != nil {
				return nil, nil, fmt.Errorf("expected command %s %q: %w", command.GRPCMethod, command.Scenario, err)
			}
			if _, ok := supportedSeen[target]; ok {
				return nil, nil, fmt.Errorf("expected command %s %q has duplicate supported target %s/%s", command.GRPCMethod, command.Scenario, target.OS, target.Arch)
			}
			if _, ok := targetSet[target]; !ok {
				return nil, nil, fmt.Errorf("expected command %s %q supports target %s/%s outside the matrix", command.GRPCMethod, command.Scenario, target.OS, target.Arch)
			}
			supportedSeen[target] = struct{}{}
		}
		hasUnsupportedTargets := len(supportedSeen) != len(targetSet)
		if hasUnsupportedTargets {
			if err := validateText("unsupported reason", expectation.UnsupportedReason); err != nil {
				return nil, nil, fmt.Errorf("expected command %s %q: %w", command.GRPCMethod, command.Scenario, err)
			}
		} else if expectation.UnsupportedReason != "" {
			return nil, nil, fmt.Errorf("expected command %s %q has an unsupported reason but supports every target", command.GRPCMethod, command.Scenario)
		}
		expectation.SupportedTargets = append([]Target(nil), expectation.SupportedTargets...)
		expectations[command] = expectation
		commands = append(commands, command)
	}
	for _, record := range records {
		command := Command{GRPCMethod: record.GRPCMethod, Scenario: record.Scenario}
		expectation, ok := expectations[command]
		if !ok {
			return nil, nil, fmt.Errorf("unexpected command identity %s %q", command.GRPCMethod, command.Scenario)
		}
		target := Target{OS: record.TargetOS, Arch: record.TargetArch}
		if !expectation.supports(target) {
			return nil, nil, fmt.Errorf("recorded command %s %q on unsupported target %s/%s", command.GRPCMethod, command.Scenario, target.OS, target.Arch)
		}
	}
	sort.Slice(commands, func(i, j int) bool {
		if commands[i].GRPCMethod != commands[j].GRPCMethod {
			return commands[i].GRPCMethod < commands[j].GRPCMethod
		}
		return commands[i].Scenario < commands[j].Scenario
	})
	return commands, expectations, nil
}

func buildMatrix(dimensions Dimensions, commands []Command, expectations map[Command]CommandExpectation, records []Record) []MatrixRow {
	byIdentity := make(map[Identity]Record, len(records))
	for _, record := range records {
		byIdentity[record.Identity()] = record
	}
	rows := make([]MatrixRow, 0, len(commands))
	for _, command := range commands {
		row := MatrixRow{GRPCMethod: command.GRPCMethod, Scenario: command.Scenario}
		expectation := expectations[command]
		for _, target := range dimensions.Targets {
			for _, transport := range dimensions.Transports {
				for _, mode := range dimensions.ImplantModes {
					identity := Identity{
						TargetOS:    target.OS,
						TargetArch:  target.Arch,
						Transport:   transport,
						ImplantMode: mode,
						GRPCMethod:  command.GRPCMethod,
						Scenario:    command.Scenario,
					}
					cell := MatrixCell{
						TargetOS:    target.OS,
						TargetArch:  target.Arch,
						Transport:   transport,
						ImplantMode: mode,
						Status:      MatrixStatusNotRun,
					}
					if !expectation.supports(target) {
						cell.Status = string(StatusSkip)
						cell.Detail = expectation.UnsupportedReason
					}
					if record, ok := byIdentity[identity]; ok {
						cell.Status = string(record.Status)
						cell.Duration = record.Duration
						cell.Detail = record.Detail
						cell.Recorded = true
					}
					row.Cells = append(row.Cells, cell)
				}
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func summarize(matrix []MatrixRow) Summary {
	summary := Summary{}
	for _, row := range matrix {
		for _, cell := range row.Cells {
			summary.TotalCells++
			if cell.Recorded {
				summary.Recorded++
			}
			switch cell.Status {
			case string(StatusPass):
				summary.Pass++
			case string(StatusFail):
				summary.Fail++
			case string(StatusSkip):
				summary.Skip++
			case MatrixStatusNotRun:
				summary.NotRun++
			}
		}
	}
	return summary
}
