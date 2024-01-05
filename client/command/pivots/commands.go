package pivots

import (
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/generate"
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

	namedPipeCmd := &cobra.Command{
		Use:   consts.NamedPipeStr,
		Short: "Start a named pipe pivot listener",
		Long:  help.GetHelpFor([]string{consts.PivotsStr, consts.NamedPipeStr}),
		Run: func(cmd *cobra.Command, args []string) {
			StartNamedPipeListenerCmd(cmd, con, args)
		},
	}
	pivotsCmd.AddCommand(namedPipeCmd)
	flags.Bind("", false, namedPipeCmd, func(f *pflag.FlagSet) {
		f.StringP("bind", "b", "", "name of the named pipe to bind pivot listener")
		f.BoolP("allow-all", "a", false, "allow all users to connect")
	})

	tcpListenerCmd := &cobra.Command{
		Use:   consts.TCPListenerStr,
		Short: "Start a TCP pivot listener",
		Long:  help.GetHelpFor([]string{consts.PivotsStr, consts.TCPListenerStr}),
		Run: func(cmd *cobra.Command, args []string) {
			StartTCPListenerCmd(cmd, con, args)
		},
	}
	pivotsCmd.AddCommand(tcpListenerCmd)
	flags.Bind("", false, tcpListenerCmd, func(f *pflag.FlagSet) {
		f.StringP("bind", "b", "", "remote interface to bind pivot listener")
		f.Uint16P("lport", "l", generate.DefaultTCPPivotPort, "tcp pivot listener port")
	})

	pivotStopCmd := &cobra.Command{
		Use:   consts.StopStr,
		Short: "Stop a pivot listener",
		Long:  help.GetHelpFor([]string{consts.PivotsStr, consts.StopStr}),
		Run: func(cmd *cobra.Command, args []string) {
			StopPivotListenerCmd(cmd, con, args)
		},
	}
	pivotsCmd.AddCommand(pivotStopCmd)
	flags.Bind("", false, pivotStopCmd, func(f *pflag.FlagSet) {
		f.Uint32P("id", "i", 0, "id of the pivot listener to stop")
	})
	flags.BindFlagCompletions(pivotStopCmd, func(comp *carapace.ActionMap) {
		(*comp)["id"] = PivotIDCompleter(con)
	})

	pivotDetailsCmd := &cobra.Command{
		Use:   consts.DetailsStr,
		Short: "Get details of a pivot listener",
		Long:  help.GetHelpFor([]string{consts.PivotsStr, consts.StopStr}),
		Run: func(cmd *cobra.Command, args []string) {
			PivotDetailsCmd(cmd, con, args)
		},
	}
	pivotsCmd.AddCommand(pivotDetailsCmd)
	flags.Bind("", false, pivotDetailsCmd, func(f *pflag.FlagSet) {
		f.IntP("id", "i", 0, "id of the pivot listener to get details for")
	})
	flags.BindFlagCompletions(pivotDetailsCmd, func(comp *carapace.ActionMap) {
		(*comp)["id"] = PivotIDCompleter(con)
	})

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
