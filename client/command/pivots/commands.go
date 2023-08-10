package pivots

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/command/completers"
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	pivotsCmd := &cobra.Command{
		Use:   consts.PivotsStr,
		Short: "List pivots for active session",
		Long:  help.GetHelpFor([]string{consts.PivotsStr}),
		Run: func(cmd *cobra.Command, args []string) {
			PivotsCmd(cmd, con, args)
		},
		GroupID: consts.SliverCoreHelpGroup,
	}
	flags.Bind("", true, pivotsCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	pivotStopCmd := &cobra.Command{
		Use:   consts.StopStr,
		Short: "Stop a pivot listener",
		Args:  cobra.ExactArgs(1),
		Long:  help.GetHelpFor([]string{consts.PivotsStr, consts.StopStr}),
		Run: func(cmd *cobra.Command, args []string) {
			StopPivotListenerCmd(cmd, con, args)
		},
	}
	pivotsCmd.AddCommand(pivotStopCmd)

	stopComs := completers.NewCompsFor(pivotStopCmd)
	stopComs.PositionalCompletion(PivotIDCompleter(con).Usage("id of the pivot listener to stop"))

	pivotDetailsCmd := &cobra.Command{
		Use:   consts.DetailsStr,
		Short: "Get details of a pivot listener",
		Long:  help.GetHelpFor([]string{consts.PivotsStr, consts.StopStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			PivotDetailsCmd(cmd, con, args)
		},
	}
	pivotsCmd.AddCommand(pivotDetailsCmd)

	detailsComps := completers.NewCompsFor(pivotDetailsCmd)
	detailsComps.PositionalCompletion(PivotIDCompleter(con).Usage("ID of the pivot listener to get details for"))

	graphCmd := &cobra.Command{
		Use:   consts.GraphStr,
		Short: "Get pivot listeners graph",
		Long:  help.GetHelpFor([]string{consts.PivotsStr, "graph"}),
		Run: func(cmd *cobra.Command, args []string) {
			PivotsGraphCmd(cmd, con, args)
		},
	}
	pivotsCmd.AddCommand(graphCmd)

	return []*cobra.Command{pivotsCmd}
}
