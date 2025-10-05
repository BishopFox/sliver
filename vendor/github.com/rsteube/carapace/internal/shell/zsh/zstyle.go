package zsh

import (
	"fmt"
	"strings"

	"github.com/rsteube/carapace/internal/common"
	"github.com/rsteube/carapace/pkg/style"
	"github.com/rsteube/carapace/third_party/github.com/elves/elvish/pkg/ui"
)

type zstyles struct {
	rawValues common.RawValues
}

func (z zstyles) descriptionSGR() string {
	if s := style.Carapace.Description; s != "" && ui.ParseStyling(s) != nil {
		return style.SGR(s)
	}
	return style.SGR(style.Default)
}

func (z zstyles) valueSGR(val common.RawValue) string {
	if val.Style != "" && ui.ParseStyling(val.Style) != nil {
		return style.SGR(val.Style)
	}

	if ui.ParseStyling(style.Carapace.Value) != nil {
		return style.SGR(style.Carapace.Value)
	}
	return style.SGR(style.Default)

}

func (z zstyles) Format() string {
	replacer := strings.NewReplacer(
		"#", `\#`,
		"*", `\*`,
		"(", `\(`,
		")", `\)`,
		"[", `\[`,
		"]", `\]`,
		"|", `\|`,
		"~", `\~`,
	)

	formatted := make([]string, 0)
	if len(z.rawValues) < 500 { // disable styling for large amount of values (bad performance)
		for _, val := range z.rawValues {
			// match value with description
			formatted = append(formatted, fmt.Sprintf("=(#b)(%v)([ ]## -- *)=0=%v=%v", replacer.Replace(val.Display), z.valueSGR(val), z.descriptionSGR()))
			// only match value (also matches aliased completions that are placed on the same line if the space allows it)
			formatted = append(formatted, fmt.Sprintf("=(#b)(%v)=0=%v", replacer.Replace(val.Display), z.valueSGR(val)))
		}
	}
	formatted = append(formatted, fmt.Sprintf("=(#b)(%v)=0=%v", "-- *", z.descriptionSGR())) // match description for aliased completions
	return strings.Join(formatted, ":")
}
