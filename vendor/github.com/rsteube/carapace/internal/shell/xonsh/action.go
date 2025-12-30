package xonsh

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rsteube/carapace/internal/common"
)

var sanitizer = strings.NewReplacer( // TODO
	"\n", ``,
	"\t", ``,
	`'`, `\'`,
)

type richCompletion struct {
	Value       string
	Display     string
	Description string
	Style       string
}

// ActionRawValues formats values for xonsh.
func ActionRawValues(currentWord string, meta common.Meta, values common.RawValues) string {
	vals := make([]richCompletion, len(values))
	for index, val := range values {
		val.Value = sanitizer.Replace(val.Value)

		if strings.ContainsAny(val.Value, ` ()[]{}*$?\"|<>&;#`+"`") {
			if strings.Contains(val.Value, `\`) {
				val.Value = fmt.Sprintf("r'%v'", val.Value) // backslash needs raw string
			} else {
				val.Value = fmt.Sprintf("'%v'", val.Value)
			}
		}

		if !meta.Nospace.Matches(val.Value) {
			val.Value = val.Value + " "
		}

		vals[index] = richCompletion{
			Value:       val.Value,
			Display:     val.Display,
			Description: val.TrimmedDescription(),
			Style:       convertStyle("bg-default fg-default " + val.Style),
		}
	}
	m, _ := json.Marshal(vals)
	return string(m)
}
