package coverage

import (
	"fmt"
	"strings"
	"time"
)

func renderTargetMarkdown(report TargetReport) []byte {
	var output strings.Builder
	fmt.Fprintf(&output, "# Sliver comprehensive E2E coverage: %s/%s\n\n", markdown(report.Target.OS), markdown(report.Target.Arch))
	pass, fail, skip := 0, 0, 0
	for _, record := range report.Records {
		switch record.Status {
		case StatusPass:
			pass++
		case StatusFail:
			fail++
		case StatusSkip:
			skip++
		}
	}
	fmt.Fprintf(&output, "Pass: %d · Fail: %d · Skip: %d · Total: %d\n\n", pass, fail, skip, len(report.Records))
	output.WriteString("| Transport | Implant mode | gRPC method | Scenario | Status | Duration | Detail |\n")
	output.WriteString("|---|---|---|---|---:|---:|---|\n")
	for _, record := range report.Records {
		fmt.Fprintf(
			&output,
			"| %s | %s | %s | %s | %s | %s | %s |\n",
			markdown(record.Transport),
			markdown(record.ImplantMode),
			markdown(record.GRPCMethod),
			markdown(record.Scenario),
			strings.ToUpper(string(record.Status)),
			formatDuration(record.Duration),
			markdown(record.Detail),
		)
	}
	return []byte(output.String())
}

func renderGlobalMarkdown(report GlobalReport) []byte {
	var output strings.Builder
	output.WriteString("# Sliver comprehensive E2E coverage\n\n")
	output.WriteString("`NOT RUN` means the cross-product cell had no scenario record. An expected platform `SKIP` is generated from the catalog and is allowed; a recorded `SKIP` is a comprehensive-gate failure.\n\n")
	recordedSkips := len(report.RecordedSkipRecords())
	expectedSkips := report.Summary.Skip - recordedSkips
	output.WriteString("| Recorded results | Pass | Fail | Recorded skip | Expected platform skip | Not run | Total cross-product cells |\n")
	output.WriteString("|---:|---:|---:|---:|---:|---:|---:|\n")
	fmt.Fprintf(&output, "| %d | %d | %d | %d | %d | %d | %d |\n\n", report.Summary.Recorded, report.Summary.Pass, report.Summary.Fail, recordedSkips, expectedSkips, report.Summary.NotRun, report.Summary.TotalCells)
	renderRPCDispositionSummary(&output, report.RPCDispositions)

	for _, transport := range report.Transports {
		for _, mode := range report.ImplantModes {
			fmt.Fprintf(&output, "## %s / %s\n\n", markdown(transport), markdown(mode))
			output.WriteString("| gRPC method | Scenario |")
			for _, target := range report.Targets {
				fmt.Fprintf(&output, " %s/%s |", markdown(target.OS), markdown(target.Arch))
			}
			output.WriteByte('\n')
			output.WriteString("|---|---|")
			for range report.Targets {
				output.WriteString("---:|")
			}
			output.WriteByte('\n')
			for _, row := range report.Matrix {
				fmt.Fprintf(&output, "| %s | %s |", markdown(row.GRPCMethod), markdown(row.Scenario))
				for _, target := range report.Targets {
					cell := findCell(row.Cells, target, transport, mode)
					fmt.Fprintf(&output, " %s |", formatCell(cell))
				}
				output.WriteByte('\n')
			}
			output.WriteByte('\n')
		}
	}

	if renderExpectedSkips(&output, report) {
		output.WriteByte('\n')
	}

	nonPass := make([]Record, 0)
	for _, record := range report.Records {
		if record.Status != StatusPass {
			nonPass = append(nonPass, record)
		}
	}
	if len(nonPass) > 0 {
		output.WriteString("## Failed and recorded skipped results\n\n")
		output.WriteString("Every result in this section fails the comprehensive coverage gate.\n\n")
		output.WriteString("| Target | Transport | Mode | gRPC method | Scenario | Status | Detail |\n")
		output.WriteString("|---|---|---|---|---|---:|---|\n")
		for _, record := range nonPass {
			fmt.Fprintf(
				&output,
				"| %s/%s | %s | %s | %s | %s | %s | %s |\n",
				markdown(record.TargetOS), markdown(record.TargetArch),
				markdown(record.Transport), markdown(record.ImplantMode),
				markdown(record.GRPCMethod), markdown(record.Scenario),
				strings.ToUpper(string(record.Status)), markdown(record.Detail),
			)
		}
	}
	return []byte(output.String())
}

func renderRPCDispositionSummary(output *strings.Builder, dispositions []RPCDisposition) {
	counts := map[RPCDispositionClass]int{}
	for _, disposition := range dispositions {
		counts[disposition.Class]++
	}
	finiteCommands := counts[RPCCommandCovered] + counts[RPCCommandDeferred]

	output.WriteString("## SliverRPC disposition denominator\n\n")
	fmt.Fprintf(output, "The generated service contains %d RPC methods. The finite implant-command denominator is %d unique methods: %d covered by the executable scenario matrix and %d explicitly deferred. Lifecycle and tunnel/interactive RPCs are tracked separately because they do not fit the finite command matrix.\n\n",
		len(dispositions), finiteCommands, counts[RPCCommandCovered], counts[RPCCommandDeferred])
	output.WriteString("| Disposition | Unique gRPC methods |\n")
	output.WriteString("|---|---:|\n")
	fmt.Fprintf(output, "| Server-only control plane | %d |\n", counts[RPCServerOnly])
	fmt.Fprintf(output, "| Finite implant command — covered | %d |\n", counts[RPCCommandCovered])
	fmt.Fprintf(output, "| Finite implant command — deferred | %d |\n", counts[RPCCommandDeferred])
	fmt.Fprintf(output, "| Implant lifecycle | %d |\n", counts[RPCImplantLifecycle])
	fmt.Fprintf(output, "| Tunnel or interactive | %d |\n\n", counts[RPCTunnelInteractive])

	output.WriteString("### Finite implant command dispositions\n\n")
	output.WriteString("Every finite implant-facing command is listed below. A deferred method is outside the executable matrix and does not count as covered.\n\n")
	output.WriteString("| gRPC method | Disposition | Rationale |\n")
	output.WriteString("|---|---|---|\n")
	for _, disposition := range dispositions {
		status := ""
		switch disposition.Class {
		case RPCCommandCovered:
			status = "COVERED"
		case RPCCommandDeferred:
			status = "DEFERRED"
		default:
			continue
		}
		fmt.Fprintf(output, "| %s | %s | %s |\n", markdown(disposition.Method), status, markdown(disposition.Reason))
	}
	output.WriteByte('\n')
}

func renderExpectedSkips(output *strings.Builder, report GlobalReport) bool {
	wroteHeader := false
	for _, row := range report.Matrix {
		targets := make([]string, 0)
		reason := ""
		for _, target := range report.Targets {
			for _, cell := range row.Cells {
				if cell.TargetOS != target.OS || cell.TargetArch != target.Arch || cell.Status != string(StatusSkip) || cell.Recorded {
					continue
				}
				targets = append(targets, markdown(target.OS)+"/"+markdown(target.Arch))
				if reason == "" {
					reason = cell.Detail
				}
				break
			}
		}
		if len(targets) == 0 {
			continue
		}
		if !wroteHeader {
			output.WriteString("## Expected platform skips\n\n")
			output.WriteString("| gRPC method | Scenario | Unsupported targets | Reason |\n")
			output.WriteString("|---|---|---|---|\n")
			wroteHeader = true
		}
		fmt.Fprintf(output, "| %s | %s | %s | %s |\n", markdown(row.GRPCMethod), markdown(row.Scenario), strings.Join(targets, ", "), markdown(reason))
	}
	return wroteHeader
}

func findCell(cells []MatrixCell, target Target, transport, mode string) MatrixCell {
	for _, cell := range cells {
		if cell.TargetOS == target.OS && cell.TargetArch == target.Arch && cell.Transport == transport && cell.ImplantMode == mode {
			return cell
		}
	}
	return MatrixCell{Status: MatrixStatusNotRun}
}

func formatCell(cell MatrixCell) string {
	if cell.Status == MatrixStatusNotRun {
		return "NOT RUN"
	}
	if cell.Duration == 0 {
		return strings.ToUpper(cell.Status)
	}
	return fmt.Sprintf("%s (%s)", strings.ToUpper(cell.Status), formatDuration(cell.Duration))
}

func formatDuration(duration time.Duration) string {
	return duration.Round(time.Microsecond).String()
}

func markdown(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "|", "\\|")
	return strings.ReplaceAll(value, "\n", "<br>")
}
