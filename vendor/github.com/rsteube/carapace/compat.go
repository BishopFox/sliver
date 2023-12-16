package carapace

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func registerValidArgsFunction(cmd *cobra.Command) {
	if cmd.ValidArgsFunction == nil {
		cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			action := Action{}.Invoke(Context{Args: args, Value: toComplete}) // TODO just IvokedAction{} ok?
			if storage.hasPositional(cmd, len(args)) {
				action = storage.getPositional(cmd, len(args)).Invoke(Context{Args: args, Value: toComplete})
			}
			return cobraValuesFor(action), cobraDirectiveFor(action)
		}
	}
}

func registerFlagCompletion(cmd *cobra.Command) {
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if !storage.hasFlag(cmd, f.Name) {
			return // skip if not defined in carapace
		}
		if _, ok := cmd.GetFlagCompletionFunc(f.Name); ok {
			return // skip if already defined in cobra
		}

		err := cmd.RegisterFlagCompletionFunc(f.Name, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			a := storage.getFlag(cmd, f.Name)
			action := a.Invoke(Context{Args: args, Value: toComplete}) // TODO cmd might differ for persistentflags and either way args or cmd will be wrong
			return cobraValuesFor(action), cobraDirectiveFor(action)
		})
		if err != nil {
			LOG.Printf("failed to register flag completion func: %v", err.Error())
		}
	})
}

func cobraValuesFor(action InvokedAction) []string {
	result := make([]string, len(action.rawValues))
	for index, r := range action.rawValues {
		if r.Description != "" {
			result[index] = fmt.Sprintf("%v\t%v", r.Value, r.Description)
		} else {
			result[index] = r.Value
		}
	}
	return result
}

func cobraDirectiveFor(action InvokedAction) cobra.ShellCompDirective {
	directive := cobra.ShellCompDirectiveNoFileComp
	for _, val := range action.rawValues {
		if action.meta.Nospace.Matches(val.Value) {
			directive = directive | cobra.ShellCompDirectiveNoSpace
			break
		}
	}
	return directive
}

type compDirective cobra.ShellCompDirective

func (d compDirective) matches(cobraDirective cobra.ShellCompDirective) bool {
	return d&compDirective(cobraDirective) != 0
}

func (d compDirective) ToA(values ...string) Action {
	var action Action
	switch {
	case d.matches(cobra.ShellCompDirectiveError):
		return ActionMessage("an error occurred")
	case d.matches(cobra.ShellCompDirectiveFilterDirs):
		switch len(values) {
		case 0:
			action = ActionDirectories()
		default:
			action = ActionDirectories().Chdir(values[0])
		}
	case d.matches(cobra.ShellCompDirectiveFilterFileExt):
		extensions := make([]string, 0)
		for _, v := range values {
			extensions = append(extensions, "."+v)
		}
		return ActionFiles(extensions...)
	case len(values) == 0 && !d.matches(cobra.ShellCompDirectiveNoFileComp):
		action = ActionFiles()
	default:
		vals := make([]string, 0)
		for _, v := range values {
			if splitted := strings.SplitN(v, "\t", 2); len(splitted) == 2 {
				vals = append(vals, splitted[0], splitted[1])
			} else {
				vals = append(vals, splitted[0], "")
			}
		}
		action = ActionValuesDescribed(vals...)
	}

	if d.matches(cobra.ShellCompDirectiveNoSpace) {
		action = action.NoSpace()
	}

	return action
}
