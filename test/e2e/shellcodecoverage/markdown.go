package shellcodecoverage

import (
	"fmt"
	"strings"
	"time"

	coverage "github.com/bishopfox/sliver/test/e2e/coverage"
)

func renderTargetMarkdown(report TargetReport) []byte {
	rows := buildMatrix([]coverage.Target{report.Target}, report.Records)
	summary := summarize(rows)

	var output strings.Builder
	fmt.Fprintf(&output, "# Sliver shellcode E2E coverage: %s/%s\n\n", markdown(report.Target.OS), markdown(report.Target.Arch))
	renderMatrixTable(&output, rows, false)
	renderLegend(&output)
	renderSummary(&output, summary)
	renderFailureDetails(&output, failedRecords(report.Records), notRunIdentities(rows))
	return []byte(output.String())
}

func renderGlobalMarkdown(report GlobalReport) []byte {
	var output strings.Builder
	output.WriteString("# Sliver shellcode E2E coverage\n\n")
	renderMatrixTable(&output, report.Matrix, true)
	renderLegend(&output)
	renderSummary(&output, report.Summary)
	renderFailureDetails(&output, report.FailedRecords(), report.NotRunIdentities())
	return []byte(output.String())
}

func renderMatrixTable(output *strings.Builder, rows []MatrixRow, includeTarget bool) {
	output.WriteByte('|')
	if includeTarget {
		output.WriteString(" Target |")
	}
	output.WriteString(" Transport | Mode | Compression |")
	for _, encoder := range fixedEncoders {
		fmt.Fprintf(output, " %s |", markdown(encoder))
	}
	output.WriteByte('\n')

	output.WriteByte('|')
	if includeTarget {
		output.WriteString("---|")
	}
	output.WriteString("---|---|---|")
	for range fixedEncoders {
		output.WriteString("---:|")
	}
	output.WriteByte('\n')

	for _, row := range rows {
		output.WriteByte('|')
		if includeTarget {
			fmt.Fprintf(output, " %s/%s |", markdown(row.Target.OS), markdown(row.Target.Arch))
		}
		fmt.Fprintf(
			output,
			" %s | %s | %s |",
			markdown(row.Transport),
			markdown(row.ImplantMode),
			markdown(row.Compression),
		)
		for _, cell := range row.Cells {
			fmt.Fprintf(output, " %s |", formatMatrixStatus(cell.Status))
		}
		output.WriteByte('\n')
	}
	output.WriteByte('\n')
}

func renderLegend(output *strings.Builder) {
	output.WriteString("Legend: ✅ PASS · ❌ FAIL · ➖ N/A (unsupported; never gates) · ⚪ NOT RUN (required and missing)\n\n")
}

func renderSummary(output *strings.Builder, summary Summary) {
	output.WriteString("## Summary\n\n")
	output.WriteString("| Required | Recorded | Pass | Fail | Not run | N/A | Total cells |\n")
	output.WriteString("|---:|---:|---:|---:|---:|---:|---:|\n")
	fmt.Fprintf(
		output,
		"| %d | %d | %d | %d | %d | %d | %d |\n\n",
		summary.Required,
		summary.Recorded,
		summary.Pass,
		summary.Fail,
		summary.NotRun,
		summary.NotApplicable,
		summary.TotalCells,
	)
}

func renderFailureDetails(output *strings.Builder, failed []Record, missing []Identity) {
	output.WriteString("## Failure details\n\n")
	if len(failed) == 0 && len(missing) == 0 {
		output.WriteString("None. All required combinations passed.\n")
		return
	}

	if len(failed) > 0 {
		output.WriteString("### Explicit failures\n\n")
		output.WriteString("| Target | Transport | Mode | Compression | Encoder | Samples | Payload bytes | Duration | Detail |\n")
		output.WriteString("|---|---|---|---|---|---:|---:|---:|---|\n")
		for _, record := range failed {
			fmt.Fprintf(
				output,
				"| %s/%s | %s | %s | %s | %s | %d/%d | %d | %s | %s |\n",
				markdown(record.Target.OS),
				markdown(record.Target.Arch),
				markdown(record.Transport),
				markdown(record.ImplantMode),
				markdown(record.Compression),
				markdown(record.Encoder),
				record.CompletedSamples,
				record.RequiredSamples,
				record.PayloadBytes,
				formatDuration(record.Duration),
				markdown(record.Detail),
			)
		}
		output.WriteByte('\n')
	}

	if len(missing) > 0 {
		output.WriteString("### Required combinations not run\n\n")
		output.WriteString("| Target | Transport | Mode | Compression | Encoder | Status |\n")
		output.WriteString("|---|---|---|---|---|---:|\n")
		for _, identity := range missing {
			fmt.Fprintf(
				output,
				"| %s/%s | %s | %s | %s | %s | ⚪ NOT RUN |\n",
				markdown(identity.Target.OS),
				markdown(identity.Target.Arch),
				markdown(identity.Transport),
				markdown(identity.ImplantMode),
				markdown(identity.Compression),
				markdown(identity.Encoder),
			)
		}
	}
}

func failedRecords(records []Record) []Record {
	failed := make([]Record, 0)
	for _, record := range records {
		if record.Status == coverage.StatusFail {
			failed = append(failed, record)
		}
	}
	return sortedRecords(failed)
}

func notRunIdentities(rows []MatrixRow) []Identity {
	missing := make([]Identity, 0)
	for _, row := range rows {
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

func formatMatrixStatus(status string) string {
	switch status {
	case string(coverage.StatusPass):
		return "✅ PASS"
	case string(coverage.StatusFail):
		return "❌ FAIL"
	case MatrixStatusNotApplicable:
		return "➖ N/A"
	case MatrixStatusNotRun:
		return "⚪ NOT RUN"
	default:
		return "❓ " + markdown(strings.ToUpper(status))
	}
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
