package gonsole

import (
	"strings"

	"github.com/jessevdk/go-flags"

	"github.com/maxlandon/readline"
)

// completeOptionArguments - Completes all values for arguments to a command. Arguments here are different from command options (--option).
// Many categories, from multiple sources in multiple menus
func (c *CommandCompleter) completeOptionArguments(gcmd *Command, cmd *flags.Command, opt *flags.Option, lastWord string) (prefix string, completions []*readline.CompletionGroup) {

	// By default the last word is the prefix
	prefix = lastWord

	// First of all: some options, no matter their menus and subject, have default values.
	// When we have such an option, we don't bother analyzing menu, we just build completions and return.
	if len(opt.Choices) > 0 {
		var comp = &readline.CompletionGroup{
			Name:        opt.ValueName, // Value names are specified in struct metadata fields
			DisplayType: readline.TabDisplayGrid,
		}
		for _, choice := range opt.Choices {
			if strings.HasPrefix(choice, lastWord) {
				comp.Suggestions = append(comp.Suggestions, choice)
			}
		}
		completions = append(completions, comp)
		return
	}

	// Simple completers (no prefix)
	for optName, completer := range gcmd.optComps {
		if strings.Contains(opt.Field().Name, optName) {
			comps := completer()

			for _, grp := range comps {
				var suggs []string
				var descs = map[string]string{}

				for _, sugg := range grp.Suggestions {
					if strings.HasPrefix(sugg, lastWord) {
						suggs = append(suggs, sugg)
						if desc, found := grp.Descriptions[sugg]; found {
							descs[sugg] = desc
						}
					}
				}
				grp.Suggestions = suggs
				grp.Descriptions = descs
			}
			completions = append(completions, comps...)
		}
	}

	// Dynamic prefix completers
	for optName, completer := range gcmd.optCompsDynamic {
		if strings.Contains(opt.Field().Name, optName) {
			pref, comps := completer(lastWord)
			prefix = pref
			completions = append(completions, comps...)
		}
	}

	return
}
