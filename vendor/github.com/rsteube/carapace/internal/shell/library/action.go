package library

import (
	"strings"

	"github.com/rsteube/carapace/internal/common"
	"github.com/rsteube/carapace/pkg/style"
)

var sanitizer = strings.NewReplacer(
	"\n", ``,
	"\r", ``,
	"\t", ``,
)

var quoter = strings.NewReplacer(
	`\`, `\\`,
	` `, `\ `,
)

// ActionRawValues formats values for carapace if used as library.
func ActionRawValues(_ string, meta common.Meta, values common.RawValues) (common.RawValues, common.Meta) {
	sorted := make(common.RawValues, 0)

	values.EachTag(func(_ string, values common.RawValues) {
		for index, val := range values {
			val.Value = sanitizer.Replace(val.Value)
			val.Value = quoter.Replace(val.Value)
			if !meta.Nospace.Matches(val.Value) {
				val.Value += " "
			}
			if val.Style != "" {
				val.Style = style.SGR(val.Style)
			}
			values[index] = val
		}

		sorted = append(sorted, values...)
	})

	return sorted, meta
}
