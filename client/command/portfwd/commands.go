package portfwd

import (
	"github.com/bishopfox/sliver/client/command/completers"
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
	portfwdCmd := &cobra.Command{
		Use:         consts.PortfwdStr,
		Short:       "In-band TCP port forwarding",
		Long:        help.GetHelpFor([]string{consts.PortfwdStr}),
		GroupID:     consts.NetworkHelpGroup,
		Annotations: flags.RestrictTargets(consts.SessionCmdsFilter),
		Run: func(cmd *cobra.Command, args []string) {
			PortfwdCmd(cmd, con, args)
		},
	}
	flags.Bind("", true, portfwdCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	addCmd := &cobra.Command{
		Use:   consts.AddStr,
		Short: "Create a new port forwarding tunnel",
		Long:  help.GetHelpFor([]string{consts.PortfwdStr}),
		Run: func(cmd *cobra.Command, args []string) {
			PortfwdAddCmd(cmd, con, args)
		},
	}
	portfwdCmd.AddCommand(addCmd)
	flags.Bind("", false, addCmd, func(f *pflag.FlagSet) {
		f.StringP("remote", "r", "", "remote target host:port (e.g., 10.0.0.1:445)")
		f.StringP("bind", "b", "127.0.0.1:8080", "bind port forward to interface")
	})
	flags.BindFlagCompletions(addCmd, func(comp *carapace.ActionMap) {
		(*comp)["bind"] = completers.ClientInterfacesCompleter()
	})

	portfwdRmCmd := &cobra.Command{
		Use:   consts.RmStr,
		Short: "Remove a port forwarding tunnel",
		Long:  help.GetHelpFor([]string{consts.PortfwdStr}),
		Run: func(cmd *cobra.Command, args []string) {
			PortfwdRmCmd(cmd, con, args)
		},
	}
	portfwdCmd.AddCommand(portfwdRmCmd)
	flags.Bind("", false, portfwdRmCmd, func(f *pflag.FlagSet) {
		f.IntP("id", "i", 0, "id of portfwd to remove")
	})
	flags.BindFlagCompletions(portfwdRmCmd, func(comp *carapace.ActionMap) {
		(*comp)["id"] = PortfwdIDCompleter(con)
	})

	return []*cobra.Command{portfwdCmd}
}
