package nushell

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rsteube/carapace/internal/common"
)

type record struct {
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

var sanitizer = strings.NewReplacer(
	"\n", ``,
	"\r", ``,
)

var escaper = strings.NewReplacer(
	`\`, `\\`,
	`"`, `\"`,
)

func sanitize(values []common.RawValue) []common.RawValue {
	for index, v := range values {
		(&values[index]).Value = sanitizer.Replace(v.Value)
		(&values[index]).Display = sanitizer.Replace(v.Display)
		(&values[index]).Description = sanitizer.Replace(v.Description)
	}
	return values
}

// ActionRawValues formats values for nushell.
func ActionRawValues(currentWord string, meta common.Meta, values common.RawValues) string {
	vals := make([]record, len(values))
	for index, val := range sanitize(values) {
		nospace := meta.Nospace.Matches(val.Value)
		if strings.ContainsAny(val.Value, ` {}()[]<>$&"'|;#\`+"`") {
			switch {
			case strings.HasPrefix(val.Value, "~"):
				val.Value = fmt.Sprintf(`~"%v"`, escaper.Replace(val.Value[1:]))
			default:
				val.Value = fmt.Sprintf(`"%v"`, escaper.Replace(val.Value))
			}
		}

		if !nospace {
			val.Value = val.Value + " "
		}

		vals[index] = record{Value: val.Value, Description: val.TrimmedDescription()}
	}
	m, _ := json.Marshal(vals)
	return string(m)
}
