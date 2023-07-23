package tasks

import (
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	tasksCmd := &cobra.Command{
		Use:   consts.TasksStr,
		Short: "Beacon task management",
		Long:  help.GetHelpFor([]string{consts.TasksStr}),
		Run: func(cmd *cobra.Command, args []string) {
			TasksCmd(cmd, con, args)
		},
		GroupID:     consts.SliverCoreHelpGroup,
		Annotations: flags.RestrictTargets(consts.BeaconCmdsFilter),
	}
	flags.Bind("tasks", true, tasksCmd, func(f *pflag.FlagSet) {
		f.IntP("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
		f.BoolP("overflow", "O", false, "overflow terminal width (display truncated rows)")
		f.IntP("skip-pages", "S", 0, "skip the first n page(s)")
		f.StringP("filter", "f", "", "filter based on task type (case-insensitive prefix matching)")
	})

	fetchCmd := &cobra.Command{
		Use:   consts.FetchStr,
		Short: "Fetch the details of a beacon task",
		Long:  help.GetHelpFor([]string{consts.TasksStr, consts.FetchStr}),
		Args:  cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			TasksFetchCmd(cmd, con, args)
		},
	}
	tasksCmd.AddCommand(fetchCmd)
	carapace.Gen(fetchCmd).PositionalCompletion(BeaconTaskIDCompleter(con).Usage("beacon task ID"))

	cancelCmd := &cobra.Command{
		Use:   consts.CancelStr,
		Short: "Cancel a pending beacon task",
		Long:  help.GetHelpFor([]string{consts.TasksStr, consts.CancelStr}),
		Args:  cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			TasksCancelCmd(cmd, con, args)
		},
	}
	tasksCmd.AddCommand(cancelCmd)
	carapace.Gen(cancelCmd).PositionalCompletion(BeaconPendingTasksCompleter(con).Usage("beacon task ID"))

	return []*cobra.Command{tasksCmd}
}
