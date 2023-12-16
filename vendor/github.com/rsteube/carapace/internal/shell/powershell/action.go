package powershell

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rsteube/carapace/internal/common"
	"github.com/rsteube/carapace/pkg/style"
	"github.com/rsteube/carapace/third_party/github.com/elves/elvish/pkg/ui"
)

var sanitizer = strings.NewReplacer( // TODO
	"\n", ``,
	"\t", ``,
)

type completionResult struct {
	CompletionText string
	ListItemText   string
	ToolTip        string
}

// CompletionResult doesn't like empty parameters, so just replace with space if needed.
func ensureNotEmpty(s string) string {
	if s == "" {
		return " "
	}
	return s
}

// ActionRawValues formats values for powershell.
func ActionRawValues(currentWord string, meta common.Meta, values common.RawValues) string {
	valueStyle := "default"
	if s := style.Carapace.Value; s != "" && ui.ParseStyling(s) != nil {
		valueStyle = s
	}

	descriptionStyle := "default"
	if s := style.Carapace.Description; s != "" && ui.ParseStyling(s) != nil {
		descriptionStyle = s
	}

	vals := make([]completionResult, 0, len(values))
	for _, val := range values {
		if val.Value != "" { // must not be empty - any empty `''` parameter in CompletionResult causes an error
			val.Value = sanitizer.Replace(val.Value)
			nospace := meta.Nospace.Matches(val.Value)

			if strings.ContainsAny(val.Value, ` {}()[]*$?\"|<>&(),;#`+"`") {
				val.Value = fmt.Sprintf("'%v'", val.Value)
			}

			if !nospace {
				val.Value = val.Value + " "
			}

			if val.Style == "" || ui.ParseStyling(val.Style) == nil {
				val.Style = valueStyle
			}

			listItemText := fmt.Sprintf("`e[21;22;23;24;25;29m`e[%vm%v`e[21;22;23;24;25;29;39;49m", sgr(val.Style), sanitizer.Replace(val.Display))
			if val.Description != "" {
				listItemText = listItemText + fmt.Sprintf("`e[%vm `e[%vm(%v)`e[21;22;23;24;25;29;39;49m", sgr(descriptionStyle+" bg-default"), sgr(descriptionStyle), sanitizer.Replace(val.TrimmedDescription()))
			}
			listItemText = listItemText + "`e[0m"

			vals = append(vals, completionResult{
				CompletionText: val.Value,
				ListItemText:   ensureNotEmpty(listItemText),
				ToolTip:        ensureNotEmpty(" "),
			})
		}
	}
	m, _ := json.Marshal(vals)
	return string(m)
}

func sgr(s string) string {
	if result := style.SGR(s); result != "" {
		return result
	}
	return "39;49"
}
