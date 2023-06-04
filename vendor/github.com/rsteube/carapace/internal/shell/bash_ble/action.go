package bash_ble

import (
	"fmt"
	"strings"

	"github.com/rsteube/carapace/internal/common"
)

// ActionRawValues formats values for bash_ble.
func ActionRawValues(currentWord string, meta common.Meta, values common.RawValues) string {
	vals := make([]string, len(values))
	for index, val := range values {
		suffix := " "
		if meta.Nospace.Matches(val.Value) {
			suffix = ""
		}
		vals[index] = fmt.Sprintf("%v\t%v\x1c%v\x1c%v\x1c%v", val.Value, val.Display, "", suffix, val.TrimmedDescription())
	}
	return strings.Join(vals, "\n")
}
