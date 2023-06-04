package oil

import (
	"fmt"
	"strings"

	"github.com/rsteube/carapace/internal/common"
)

var sanitizer = strings.NewReplacer(
	"\n", ``,
	"\t", ``,
)

const nospaceIndicator = "\001"

// ActionRawValues formats values for oil.
func ActionRawValues(currentWord string, meta common.Meta, values common.RawValues) string {
	vals := make([]string, len(values))
	for index, val := range values {
		if meta.Nospace.Matches(val.Value) {
			val.Value = val.Value + nospaceIndicator
		}

		if len(values) == 1 {
			formattedVal := sanitizer.Replace(val.Value)
			vals[index] = formattedVal
		} else {
			if val.Description != "" {
				vals[index] = fmt.Sprintf("%v (%v)", val.Value, sanitizer.Replace(val.TrimmedDescription()))
			} else {
				vals[index] = val.Value
			}
		}
	}
	return strings.Join(vals, "\n")
}
