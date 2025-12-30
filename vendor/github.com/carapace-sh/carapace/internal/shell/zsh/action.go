package zsh

import (
	"fmt"
	"regexp"
	"strings"

	shlex "github.com/carapace-sh/carapace-shlex"
	"github.com/carapace-sh/carapace/internal/common"
	"github.com/carapace-sh/carapace/internal/env"
)

var sanitizer = strings.NewReplacer(
	"\n", ``,
	"\r", ``,
	"\t", ``,
)

var quotingReplacer = strings.NewReplacer(
	`'`, `'\''`,
)

var quotingEscapingReplacer = strings.NewReplacer(
	`\`, `\\`,
	`"`, `\"`,
	`$`, `\$`,
	"`", "\\`",
)

var defaultReplacer = strings.NewReplacer(
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

// additional replacement for use with `_describe` in shell script
var describeReplacer = strings.NewReplacer(
	`\`, `\\`,
	`:`, `\:`,
)

func quoteValue(s string) string {
	if strings.HasPrefix(s, "~/") || NamedDirectories.Matches(s) {
		return "~" + defaultReplacer.Replace(strings.TrimPrefix(s, "~")) // assume file path expansion
	}
	return defaultReplacer.Replace(s)
}

type state int

const (
	DEFAULT_STATE state = iota
	// Word starts with `"`.
	// Values need to end with `"` as well.
	// Weirdly regardless whether there are additional quotes within the word.
	QUOTING_ESCAPING_STATE
	// Word starts with `'`.
	// Values need to end with `'` as well.
	// Weirdly regardless whether there are additional quotes within the word.
	QUOTING_STATE
	// Word starts and ends with `"`.
	// Space suffix somehow ends up within the quotes.
	//    `"action"<TAB>`
	//    `"action "<CURSOR>`
	// Workaround for now is to force nospace.
	FULL_QUOTING_ESCAPING_STATE
	// Word starts and ends with `'`.
	// Space suffix somehow ends up within the quotes.
	//    `'action'<TAB>`
	//    `'action '<CURSOR>`
	// Workaround for now is to force nospace.
	FULL_QUOTING_STATE
)

// ActionRawValues formats values for zsh
func ActionRawValues(currentWord string, meta common.Meta, values common.RawValues) string {
	splitted, err := shlex.Split(env.Compline())
	state := DEFAULT_STATE
	if err == nil {
		rawValue := splitted.CurrentToken().RawValue
		// TODO use token state to determine actual state (might have mixture).
		switch {
		case regexp.MustCompile(`^'$|^'.*[^']$`).MatchString(rawValue):
			state = QUOTING_STATE
		case regexp.MustCompile(`^"$|^".*[^"]$`).MatchString(rawValue):
			state = QUOTING_ESCAPING_STATE
		case regexp.MustCompile(`^".*"$`).MatchString(rawValue):
			state = FULL_QUOTING_ESCAPING_STATE
		case regexp.MustCompile(`^'.*'$`).MatchString(rawValue):
			state = FULL_QUOTING_STATE
		}
	}

	tagGroup := make([]string, 0)
	values.EachTag(func(tag string, values common.RawValues) {
		vals := make([]string, len(values))
		displays := make([]string, len(values))
		for index, val := range values {
			value := sanitizer.Replace(val.Value)

			switch state {
			case QUOTING_ESCAPING_STATE:
				value = quotingEscapingReplacer.Replace(value)
				value = describeReplacer.Replace(value)
				value = value + `"`
			case QUOTING_STATE:
				value = quotingReplacer.Replace(value)
				value = describeReplacer.Replace(value)
				value = value + `'`
			case FULL_QUOTING_ESCAPING_STATE:
				value = quotingEscapingReplacer.Replace(value)
				value = describeReplacer.Replace(value)
			case FULL_QUOTING_STATE:
				value = quotingReplacer.Replace(value)
				value = describeReplacer.Replace(value)
			default:
				value = quoteValue(value)
				value = describeReplacer.Replace(value)
			}

			if !meta.Nospace.Matches(val.Value) {
				switch state {
				case FULL_QUOTING_ESCAPING_STATE, FULL_QUOTING_STATE: // nospace workaround
				default:
					value += " "
				}
			}

			display := sanitizer.Replace(val.Display)
			display = describeReplacer.Replace(display) // TODO check if this needs to be applied to description as well
			description := sanitizer.Replace(val.Description)

			vals[index] = value

			if strings.TrimSpace(description) == "" {
				displays[index] = display
			} else {
				displays[index] = fmt.Sprintf("%v:%v", display, description)
			}
		}
		tagGroup = append(tagGroup, strings.Join([]string{tag, strings.Join(displays, "\n"), strings.Join(vals, "\n")}, "\003"))
	})
	return fmt.Sprintf("%v\001%v\001%v\001", zstyles{values}.Format(), message{meta}.Format(), strings.Join(tagGroup, "\002")+"\002")
}
