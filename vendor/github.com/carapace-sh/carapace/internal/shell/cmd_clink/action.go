package cmd_clink

import (
	"strings"

	"github.com/carapace-sh/carapace/internal/common"
)

var sanitizer = strings.NewReplacer(
	"\n", ``,
	"\r", ``,
	"\t", ``,
)

func ActionRawValues(currentWord string, meta common.Meta, values common.RawValues) string {
	vals := make([]string, len(values))
	for index, val := range values {
		appendChar := " "
		if meta.Nospace.Matches(val.Value) {
			appendChar = ""
		}
		vals[index] = strings.Join([]string{
			sanitizer.Replace(val.Value),
			sanitizer.Replace(val.Display),
			sanitizer.Replace(val.TrimmedDescription()),
			appendChar,
		}, "\t")
	}
	return strings.Join(vals, "\n")
}
