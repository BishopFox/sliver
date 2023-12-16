package zsh

import (
	"fmt"
	"strings"

	"github.com/rsteube/carapace/internal/common"
)

var sanitizer = strings.NewReplacer(
	"\n", ``,
	"\r", ``,
	"\t", ``,
)

// TODO verify these are correct/complete (copied from bash)
var quoter = strings.NewReplacer(
	`\`, `\\`,
	`&`, `\&`,
	`<`, `\<`,
	`>`, `\>`,
	"`", "\\`",
	`'`, `\'`,
	`"`, `\"`,
	`{`, `\{`,
	`}`, `\}`,
	`$`, `\$`,
	`#`, `\#`,
	`|`, `\|`,
	`?`, `\?`,
	`(`, `\(`,
	`)`, `\)`,
	`;`, `\;`,
	` `, `\ `,
	`[`, `\[`,
	`]`, `\]`,
	`*`, `\*`,
	`~`, `\~`,
)

func quoteValue(s string) string {
	if strings.HasPrefix(s, "~/") || NamedDirectories.Matches(s) {
		return "~" + quoter.Replace(strings.TrimPrefix(s, "~")) // assume file path expansion
	}
	return quoter.Replace(s)
}

// ActionRawValues formats values for zsh
func ActionRawValues(currentWord string, meta common.Meta, values common.RawValues) string {

	tagGroup := make([]string, 0)
	values.EachTag(func(tag string, values common.RawValues) {
		vals := make([]string, len(values))
		displays := make([]string, len(values))
		for index, val := range values {
			val.Value = sanitizer.Replace(val.Value)
			val.Value = quoteValue(val.Value)
			val.Value = strings.ReplaceAll(val.Value, `\`, `\\`) // TODO find out why `_describe` needs another backslash
			val.Value = strings.ReplaceAll(val.Value, `:`, `\:`) // TODO find out why `_describe` needs another backslash
			if !meta.Nospace.Matches(val.Value) {
				val.Value = val.Value + " "
			}
			val.Display = sanitizer.Replace(val.Display)
			val.Display = strings.ReplaceAll(val.Display, `\`, `\\`) // TODO find out why `_describe` needs another backslash
			val.Display = strings.ReplaceAll(val.Display, `:`, `\:`) // TODO find out why `_describe` needs another backslash
			val.Description = sanitizer.Replace(val.Description)

			vals[index] = val.Value

			if strings.TrimSpace(val.Description) == "" {
				displays[index] = val.Display
			} else {
				displays[index] = fmt.Sprintf("%v:%v", val.Display, val.Description)
			}
		}
		tagGroup = append(tagGroup, strings.Join([]string{tag, strings.Join(displays, "\n"), strings.Join(vals, "\n")}, "\003"))
	})
	return fmt.Sprintf("%v\001%v\001%v\001", zstyles{values}.Format(), message{meta}.Format(), strings.Join(tagGroup, "\002")+"\002")
}
