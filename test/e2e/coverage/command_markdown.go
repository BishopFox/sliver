package coverage

import (
	"fmt"
	"sort"
	"strings"
)

const (
	commandStatusPass        = "✅"
	commandStatusFail        = "❌"
	commandStatusUnsupported = "N/A"
)

// renderCommandMarkdown collapses scenario and implant-mode results into one
// row per unique gRPC command and one column per target/transport pair.
func renderCommandMarkdown(report GlobalReport) []byte {
	var output strings.Builder
	output.WriteString("| gRPC command |")
	for _, target := range report.Targets {
		for _, transport := range report.Transports {
			fmt.Fprintf(&output, " %s/%s/%s |", markdown(target.OS), markdown(target.Arch), markdown(transport))
		}
	}
	output.WriteByte('\n')
	output.WriteString("|---|")
	for range report.Targets {
		for range report.Transports {
			output.WriteString(":---:|")
		}
	}
	output.WriteByte('\n')

	rowsByMethod := make(map[string][]MatrixRow)
	for _, row := range report.Matrix {
		rowsByMethod[row.GRPCMethod] = append(rowsByMethod[row.GRPCMethod], row)
	}
	methods := make([]string, 0, len(rowsByMethod))
	for method := range rowsByMethod {
		methods = append(methods, method)
	}
	sort.Strings(methods)

	for _, method := range methods {
		fmt.Fprintf(&output, "| %s |", markdown(method))
		for _, target := range report.Targets {
			for _, transport := range report.Transports {
				fmt.Fprintf(&output, " %s |", commandCellStatus(rowsByMethod[method], target, transport))
			}
		}
		output.WriteByte('\n')
	}
	return []byte(output.String())
}

func commandCellStatus(rows []MatrixRow, target Target, transport string) string {
	if !commandSupportsTarget(rows, target) {
		return commandStatusUnsupported
	}
	required := 0
	for _, row := range rows {
		for _, cell := range row.Cells {
			if cell.TargetOS != target.OS || cell.TargetArch != target.Arch || cell.Transport != transport {
				continue
			}
			if cell.Status == string(StatusSkip) && !cell.Recorded {
				continue
			}
			required++
			if cell.Status != string(StatusPass) || !cell.Recorded {
				return commandStatusFail
			}
		}
	}
	if required == 0 {
		return commandStatusFail
	}
	return commandStatusPass
}

func commandSupportsTarget(rows []MatrixRow, target Target) bool {
	for _, row := range rows {
		for _, cell := range row.Cells {
			if cell.TargetOS != target.OS || cell.TargetArch != target.Arch {
				continue
			}
			if cell.Status != string(StatusSkip) || cell.Recorded {
				return true
			}
		}
	}
	return false
}
