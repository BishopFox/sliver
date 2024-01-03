package network

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	ifconfigCmd := &cobra.Command{
		Use:   consts.IfconfigStr,
		Short: "View network interface configurations",
		Long:  help.GetHelpFor([]string{consts.IfconfigStr}),
		Run: func(cmd *cobra.Command, args []string) {
			IfconfigCmd(cmd, con, args)
		},
		GroupID: consts.NetworkHelpGroup,
	}
	flags.Bind("", false, ifconfigCmd, func(f *pflag.FlagSet) {
		f.BoolP("all", "A", false, "show all network adapters (default only shows IPv4)")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	netstatCmd := &cobra.Command{
		Use:   consts.NetstatStr,
		Short: "Print network connection information",
		Long:  help.GetHelpFor([]string{consts.NetstatStr}),
		Run: func(cmd *cobra.Command, args []string) {
			NetstatCmd(cmd, con, args)
		},
		GroupID: consts.NetworkHelpGroup,
	}
	flags.Bind("", false, netstatCmd, func(f *pflag.FlagSet) {
		f.BoolP("tcp", "T", true, "display information about TCP sockets")
		f.BoolP("udp", "u", false, "display information about UDP sockets")
		f.BoolP("ip4", "4", true, "display information about IPv4 sockets")
		f.BoolP("ip6", "6", false, "display information about IPv6 sockets")
		f.BoolP("listen", "l", false, "display information about listening sockets")
		f.BoolP("numeric", "n", false, "display numeric addresses (disable hostname resolution)")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	return []*cobra.Command{ifconfigCmd, netstatCmd}
}
