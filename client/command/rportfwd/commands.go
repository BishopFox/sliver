package rportfwd

import (
	"github.com/rsteube/carapace"
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
	rportfwdCmd := &cobra.Command{
		Use:         consts.RportfwdStr,
		Short:       "reverse port forwardings",
		Long:        help.GetHelpFor([]string{consts.RportfwdStr}),
		GroupID:     consts.NetworkHelpGroup,
		Annotations: flags.RestrictTargets(consts.SessionCmdsFilter),
		Run: func(cmd *cobra.Command, args []string) {
			RportFwdListenersCmd(cmd, con, args)
		},
	}
	flags.Bind("", true, rportfwdCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	rportfwdAddCmd := &cobra.Command{
		Use:   consts.AddStr,
		Short: "Add and start reverse port forwarding",
		Long:  help.GetHelpFor([]string{consts.RportfwdStr}),
		Run: func(cmd *cobra.Command, args []string) {
			StartRportFwdListenerCmd(cmd, con, args)
		},
	}
	rportfwdCmd.AddCommand(rportfwdAddCmd)
	flags.Bind("", false, rportfwdAddCmd, func(f *pflag.FlagSet) {
		f.StringP("remote", "r", "", "remote address <ip>:<port> connection is forwarded to")
		f.StringP("bind", "b", "", "bind address <ip>:<port> for implants to listen on")
	})
	completers.NewFlagCompsFor(rportfwdAddCmd, func(comp *carapace.ActionMap) {
		(*comp)["remote"] = completers.ClientInterfacesCompleter()
	})

	rportfwdRmCmd := &cobra.Command{
		Use:   consts.RmStr,
		Short: "Stop and remove reverse port forwarding",
		Long:  help.GetHelpFor([]string{consts.RportfwdStr}),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			StopRportFwdListenerCmd(cmd, con, args)
		},
	}
	rmComps := completers.NewCompsFor(rportfwdRmCmd)
	rmComps.PositionalAnyCompletion(PortfwdIDCompleter(con).Usage("ID of portforwarder(s) to remove"))

	rportfwdCmd.AddCommand(rportfwdRmCmd)

	return []*cobra.Command{rportfwdCmd}
}
