package rdp

import (
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Commands returns the RDP command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	rdpCmd := &cobra.Command{
		Use:   consts.RdpStr,
		Short: "Quick RDP - auto port-forward + launch RDP client",
		Long:  help.GetHelpFor([]string{consts.RdpStr}),
		Run: func(cmd *cobra.Command, args []string) {
			RdpConnectCmd(cmd, con, args)
		},
		GroupID:     consts.NetworkHelpGroup,
		Annotations: flags.RestrictTargets(consts.SessionCmdsFilter),
	}
	flags.Bind("", false, rdpCmd, func(f *pflag.FlagSet) {
		f.StringP("target", "t", "", "target host (default: session remote address)")
		f.StringP("remote-port", "r", "3389", "remote RDP port")
		f.StringP("bind-port", "b", "13389", "local bind port")
		f.StringP("username", "u", "", "RDP username (for auto-launch)")
		f.StringP("password", "p", "", "RDP password (for auto-launch)")
		f.StringP("domain", "d", "", "RDP domain (for auto-launch)")
		f.BoolP("no-launch", "n", false, "don't auto-launch RDP client")
		f.BoolP("enable", "e", false, "enable RDP on target via registry before connecting")
		f.Int64P("timeout", "T", 30, "grpc timeout in seconds")
	})

	return []*cobra.Command{rdpCmd}
}
