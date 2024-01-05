package flags

import (
	"strings"

	"github.com/reeflective/console"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	DefaultTimeout = 60
)

// Bind is a convenience function to bind flags to a command, through newly created
// pflag.Flagset type. This function can be called any number of times for any command.
// desc        - An optional name for the flag set (can be empty, but might end up useful).
// persistent  - If true, the flags bound will apply to all subcommands of this command.
// cmd         - The pointer to the command the flags should be bound to.
// flags       - A function using this flag set as parameter, for you to register flags.
func Bind(desc string, persistent bool, cmd *cobra.Command, flags func(f *pflag.FlagSet)) {
	flagSet := pflag.NewFlagSet(desc, pflag.ContinueOnError)
	flags(flagSet)

	if persistent {
		cmd.PersistentFlags().AddFlagSet(flagSet)
	} else {
		cmd.Flags().AddFlagSet(flagSet)
	}
}

// RestrictTargets generates a cobra annotation map with a single console.CommandHiddenFilter key
// to a comma-separated list of filters to use in order to expose/hide commands based on requirements.
// Ex: cmd.Annotations = RestrictTargets("windows") will only show the command if the target is Windows.
// Ex: cmd.Annotations = RestrictTargets("windows", "beacon") show the command if target is a beacon on Windows.
func RestrictTargets(filters ...string) map[string]string {
	if len(filters) == 0 {
		return nil
	}

	if len(filters) == 1 {
		return map[string]string{
			console.CommandFilterKey: filters[0],
		}
	}

	filts := strings.Join(filters, ",")

	return map[string]string{
		console.CommandFilterKey: filts,
	}
}

// NewCompletions registers the command to the application completion engine and returns
// you a type through which you can register all sorts of completions for this command,
// from flag arguments, positional ones, per index or remaining, etc.
//
//	See https://rsteube.github.io/carapace/ for a complete documentation of carapace completions.
func NewCompletions(cmd *cobra.Command) *carapace.Carapace {
	return carapace.Gen(cmd)
}

// BindFlagCompletions is a convenience function for binding completers to flags requiring arguments.
// (It wraps a few steps to be used through the *carapace.Carapace type so you don't have to bother).
// cmd   - The target command/subcommand which flags to be completed.
// bind  - A function using a map "flag-name":carapace.Action for you to bind completions to the flag.
//
//	See https://rsteube.github.io/carapace/ for a complete documentation of carapace completions.
func BindFlagCompletions(cmd *cobra.Command, bind func(comp *carapace.ActionMap)) {
	comps := make(carapace.ActionMap)
	bind(&comps)

	carapace.Gen(cmd).FlagCompletion(comps)
}
