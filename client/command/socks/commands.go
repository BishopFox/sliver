package socks

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
	socksCmd := &cobra.Command{
		Use:         consts.Socks5Str,
		Short:       "In-band SOCKS5 Proxy",
		Long:        help.GetHelpFor([]string{consts.Socks5Str}),
		GroupID:     consts.NetworkHelpGroup,
		Annotations: flags.RestrictTargets(consts.SessionCmdsFilter),
		Run: func(cmd *cobra.Command, args []string) {
			SocksCmd(cmd, con, args)
		},
	}
	flags.Bind("", true, socksCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	socksStartCmd := &cobra.Command{
		Use:   consts.StartStr,
		Short: "Start an in-band SOCKS5 proxy",
		Long:  help.GetHelpFor([]string{consts.Socks5Str}),
		Run: func(cmd *cobra.Command, args []string) {
			SocksStartCmd(cmd, con, args)
		},
	}
	socksCmd.AddCommand(socksStartCmd)
	flags.Bind("", false, socksStartCmd, func(f *pflag.FlagSet) {
		f.StringP("host", "H", "127.0.0.1", "Bind a Socks5 Host")
		f.StringP("port", "P", "1081", "Bind a Socks5 Port")
		f.StringP("user", "u", "", "socks5 auth username (will generate random password)")
	})
	flags.BindFlagCompletions(socksStartCmd, func(comp *carapace.ActionMap) {
		(*comp)["host"] = completers.ClientInterfacesCompleter()
	})

	socksStopCmd := &cobra.Command{
		Use:   consts.StopStr,
		Short: "Stop a SOCKS5 proxy",
		Long:  help.GetHelpFor([]string{consts.Socks5Str}),
		Run: func(cmd *cobra.Command, args []string) {
			SocksStopCmd(cmd, con, args)
		},
	}
	socksCmd.AddCommand(socksStopCmd)
	flags.Bind("", false, socksStopCmd, func(f *pflag.FlagSet) {
		f.Uint64P("id", "i", 0, "id of portfwd to remove")
	})
	flags.BindFlagCompletions(socksStopCmd, func(comp *carapace.ActionMap) {
		(*comp)["id"] = SocksIDCompleter(con)
	})

	return []*cobra.Command{socksCmd}
}
