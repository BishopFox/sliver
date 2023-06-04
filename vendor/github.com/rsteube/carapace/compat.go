package carapace

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func registerValidArgsFunction(cmd *cobra.Command) {
	if cmd.ValidArgsFunction == nil {
		cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			action := storage.getPositional(cmd, len(args)).Invoke(Context{Args: args, Value: toComplete})
			return cobraValuesFor(action), cobraDirectiveFor(action)
		}
	}
}

func registerFlagCompletion(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		err := cmd.RegisterFlagCompletionFunc(f.Name, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			a := storage.getFlag(cmd, f.Name)
			action := a.Invoke(Context{Args: args, Value: toComplete})
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
