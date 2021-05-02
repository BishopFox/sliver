package gonsole

import (
	"strings"

	"github.com/jessevdk/go-flags"

	"github.com/maxlandon/readline"
)

// CompleteCommandArguments - Completes all values for arguments to a command.
// Arguments here are different from command options (--option).
// Many categories, from multiple sources in multiple menus
func (c *CommandCompleter) completeCommandArguments(gcmd *Command, cmd *flags.Command, arg string, lastWord string) (prefix string, completions []*readline.CompletionGroup) {

	// the prefix is the last word, by default
	prefix = lastWord
	found := argumentByName(cmd, arg)

	// Simple completers (no prefix)
	for argName, completer := range gcmd.argComps {
		if strings.Contains(found.Name, argName) {

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
	for argName, completer := range gcmd.argCompsDynamic {
		if strings.Contains(found.Name, argName) {
			pref, comps := completer(lastWord)
			prefix = pref
			completions = append(completions, comps...)
		}
	}

	return
}
