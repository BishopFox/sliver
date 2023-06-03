package ion

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rsteube/carapace/internal/common"
)

var sanitizer = strings.NewReplacer(
	"\n", ``,
	"\r", ``,
)

func sanitize(values []common.RawValue) []common.RawValue {
	for index, v := range values {
		(&values[index]).Value = sanitizer.Replace(v.Value)
		(&values[index]).Display = sanitizer.Replace(v.Display)
		(&values[index]).Description = sanitizer.Replace(v.Description)
	}
	return values
}

type suggestion struct {
	Value   string
	Display string
}

// ActionRawValues formats values for ion.
func ActionRawValues(currentWord string, meta common.Meta, values common.RawValues) string {
	vals := make([]suggestion, len(values))
	for index, val := range sanitize(values) {
		if !meta.Nospace.Matches(val.Value) {
			val.Value = val.Value + " "
		}

		if val.Description == "" {
			vals[index] = suggestion{Value: val.Value, Display: val.Display}
		} else {
			vals[index] = suggestion{Value: val.Value, Display: fmt.Sprintf(`%v (%v)`, val.Display, val.TrimmedDescription())}
		}
	}
	m, _ := json.Marshal(vals)
	return string(m)
}
