package fish

import (
	"fmt"
	"github.com/rsteube/carapace/internal/common"
	"strings"
)

var sanitizer = strings.NewReplacer(
	"\n", ``,
	"\r", ``,
	"\t", ``,
)

// ActionRawValues formats values for fish.
func ActionRawValues(currentWord string, meta common.Meta, values common.RawValues) string {
	vals := make([]string, len(values))
	for index, val := range values {
		vals[index] = fmt.Sprintf("%v\t%v", sanitizer.Replace(val.Value), sanitizer.Replace(val.TrimmedDescription()))
	}
	return strings.Join(vals, "\n")
}
