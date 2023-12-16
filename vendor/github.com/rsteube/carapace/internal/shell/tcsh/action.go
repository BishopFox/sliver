package tcsh

import (
	"fmt"
	"os"
	"strings"

	"github.com/rsteube/carapace/internal/common"
)

var sanitizer = strings.NewReplacer(
	"\n", ``,
	"\r", ``,
	"\t", ``,
)

var quoter = strings.NewReplacer(
	`&`, `\&`,
	`<`, `\<`,
	`>`, `\>`,
	"`", "\\`",
	`'`, `\'`,
	`"`, `\"`,
	`{`, ``, // TODO seems escaping is not working
	`}`, ``, // TODO seems escaping is not working
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
	`\`, `\\`,
)

func commonPrefix(a, b string) string {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	return a[0:i]
}

func commonDisplayPrefix(values ...common.RawValue) (prefix string) {
	for index, val := range values {
		if index == 0 {
			prefix = val.Display
		} else {
			prefix = commonPrefix(prefix, val.Display)
		}
	}
	return
}

func commonValuePrefix(values ...common.RawValue) (prefix string) {
	for index, val := range values {
		if index == 0 {
			prefix = val.Value
		} else {
			prefix = commonPrefix(prefix, val.Value)
		}
	}
	return
}

// ActionRawValues formats values for bash.
func ActionRawValues(currentWord string, meta common.Meta, values common.RawValues) string {
	lastSegment := currentWord // last segment of currentWord split by COMP_WORDBREAKS

	for _, r := range values {
		// TODO optimize
		if wordbreaks, ok := os.LookupEnv("COMP_WORDBREAKS"); ok {
			wordbreaks = strings.Replace(wordbreaks, " ", "", -1)
			if index := strings.LastIndexAny(currentWord, wordbreaks); index != -1 {
				r.Value = strings.TrimPrefix(r.Value, currentWord[:index+1])
				lastSegment = currentWord[index+1:]
			}
		}
	}

	if len(values) > 1 && commonDisplayPrefix(values...) != "" {
		// When all display values have the same prefix bash will insert is as partial completion (which skips prefixes/formatting).
		if valuePrefix := commonValuePrefix(values...); lastSegment != valuePrefix {
			// replace values with common value prefix (`\001` is removed in snippet and compopt nospace will be set)
			values = common.RawValuesFrom(commonValuePrefix(values...)) // TODO nospaceIndicator
			//values = common.RawValuesFrom(commonValuePrefix(values...) + nospaceIndicator)
		} else {
			// prevent insertion of partial display values by prefixing one with space
			values[0].Display = " " + values[0].Display
		}
	}

	vals := make([]string, len(values))
	for index, val := range values {
		if len(values) == 1 {
			vals[index] = quoter.Replace(sanitizer.Replace(val.Value))
		} else {
			if val.Description != "" {
				// TODO seems actual value needs to be used or it won't be shown if the prefix doesn't match
				vals[index] = fmt.Sprintf("%v_(%v)", quoter.Replace(sanitizer.Replace(val.Value)), quoter.Replace(strings.Replace(sanitizer.Replace(val.TrimmedDescription()), " ", "_", -1)))
			} else {
				vals[index] = quoter.Replace(sanitizer.Replace(val.Value))
			}
		}
	}
	return strings.Join(vals, "\n")
}
