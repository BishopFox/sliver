package completers

import (
	"github.com/jessevdk/go-flags"

	"github.com/maxlandon/readline"
)

// CompleteCommandArguments - Completes all values for arguments to a command.
// Arguments here are different from command options (--option).
// Many categories, from multiple sources in multiple contexts
func completeCommandArguments(cmd *flags.Command, arg string, lastWord string) (prefix string, completions []*readline.CompletionGroup) {

	// the prefix is the last word, by default
	prefix = lastWord

	// SEE completeOptionArguments FOR A WAY TO ADD COMPLETIONS TO SPECIFIC ARGUMENTS ------------------------------

	// found := argumentByName(cmd, arg)
	// var comp *readline.CompletionGroup // This group is used as a buffer, to add groups to final completions

	return
}
