package wireguard

/*
   Sliver Implant Framework
   Copyright (C) 2019  Bishop Fox

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/command/completers"
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// Commands returns the `wg` command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	wgCmd := &cobra.Command{
		Use:     consts.WGStr,
		Short:   "WireGuard C2 commands (configs, listeners, etc.)",
		GroupID: consts.NetworkHelpGroup,
	}

	// Configurations
	wgConfigCmd := &cobra.Command{
		Use:   consts.WgConfigStr,
		Short: "Generate a new WireGuard client config",
		Long:  help.GetHelpFor([]string{consts.WgConfigStr}),
		Run: func(cmd *cobra.Command, args []string) {
			WGConfigCmd(cmd, con, args)
		},
	}
	wgCmd.AddCommand(wgConfigCmd)

	flags.Bind("wg-config", true, wgConfigCmd, func(f *pflag.FlagSet) {
		f.IntP("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	flags.Bind("wg-config", false, wgConfigCmd, func(f *pflag.FlagSet) {
		f.StringP("save", "s", "", "save configuration to file (.conf)")
	})
	completers.NewFlagCompsFor(wgConfigCmd, func(comp *carapace.ActionMap) {
		(*comp)["save"] = carapace.ActionFiles().Tag("directory/file to save config")
	})

	// Listeners
	wgListenCmd := &cobra.Command{
		Use:   consts.ListenStr,
		Short: "Start a WireGuard listener",
		Long:  help.GetHelpFor([]string{consts.WGStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ListenCmd(cmd, con, args)
		},
	}
	flags.Bind("WireGuard listener", false, wgCmd, func(f *pflag.FlagSet) {
		f.StringP("lhost", "L", "", "interface to bind server to")
		f.Uint32P("lport", "l", generate.DefaultWGLPort, "udp listen port")
		f.Uint32P("nport", "n", generate.DefaultWGNPort, "virtual tun interface listen port")
		f.Uint32P("key-port", "x", generate.DefaultWGKeyExPort, "virtual tun interface key exchange port")
		f.BoolP("persistent", "p", false, "make persistent across restarts")
	})
	wgCmd.AddCommand(wgListenCmd)

	return []*cobra.Command{wgCmd}
}

// SliverCommands returns all Wireguard commands that can be used on an active target.
func SliverCommands(con *console.SliverClient) []*cobra.Command {
	wgPortFwdCmd := &cobra.Command{
		Use:     consts.WgPortFwdStr,
		Short:   "List ports forwarded by the WireGuard tun interface",
		Long:    help.GetHelpFor([]string{consts.WgPortFwdStr}),
		GroupID: consts.NetworkHelpGroup,
		Annotations: flags.RestrictTargets(
			consts.WireguardCmdsFilter,
			consts.SessionCmdsFilter,
		),
		Run: func(cmd *cobra.Command, args []string) {
			WGPortFwdListCmd(cmd, con, args)
		},
	}
	flags.Bind("wg portforward", true, wgPortFwdCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	wgPortFwdAddCmd := &cobra.Command{
		Use:   consts.AddStr,
		Short: "Add a port forward from the WireGuard tun interface to a host on the target network",
		Long:  help.GetHelpFor([]string{consts.WgPortFwdStr, consts.AddStr}),
		Run: func(cmd *cobra.Command, args []string) {
			WGPortFwdAddCmd(cmd, con, args)
		},
	}
	flags.Bind("wg portforward", false, wgPortFwdAddCmd, func(f *pflag.FlagSet) {
		f.Int32P("bind", "b", 1080, "port to listen on the WireGuard tun interface")
		f.StringP("remote", "r", "", "remote target host:port (e.g., 10.0.0.1:445)")
	})
	wgPortFwdCmd.AddCommand(wgPortFwdAddCmd)

	wgPortFwdRmCmd := &cobra.Command{
		Use:   consts.RmStr,
		Short: "Remove a port forward from the WireGuard tun interface",
		Long:  help.GetHelpFor([]string{consts.WgPortFwdStr, consts.RmStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			WGPortFwdRmCmd(cmd, con, args)
		},
	}
	wgPortFwdCmd.AddCommand(wgPortFwdRmCmd)

	carapace.Gen(wgPortFwdRmCmd).PositionalCompletion(PortfwdIDCompleter(con).Usage("forwarder ID"))

	wgSocksCmd := &cobra.Command{
		Use:   consts.WgSocksStr,
		Short: "List socks servers listening on the WireGuard tun interface",
		Long:  help.GetHelpFor([]string{consts.WgSocksStr}),
		Run: func(cmd *cobra.Command, args []string) {
			WGSocksListCmd(cmd, con, args)
		},
		GroupID:     consts.NetworkHelpGroup,
		Annotations: flags.RestrictTargets(consts.WireguardCmdsFilter),
	}
	flags.Bind("wg socks", true, wgSocksCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	wgSocksStartCmd := &cobra.Command{
		Use:   consts.StartStr,
		Short: "Start a socks5 listener on the WireGuard tun interface",
		Long:  help.GetHelpFor([]string{consts.WgSocksStr, consts.StartStr}),
		Run: func(cmd *cobra.Command, args []string) {
			WGSocksStartCmd(cmd, con, args)
		},
	}
	wgSocksCmd.AddCommand(wgSocksStartCmd)
	flags.Bind("wg socks", false, wgSocksStartCmd, func(f *pflag.FlagSet) {
		f.Int32P("bind", "b", 3090, "port to listen on the WireGuard tun interface")
	})

	wgSocksStopCmd := &cobra.Command{
		Use:   consts.StopStr,
		Short: "Stop a socks5 listener on the WireGuard tun interface",
		Long:  help.GetHelpFor([]string{consts.WgSocksStr, consts.StopStr}),
		Run: func(cmd *cobra.Command, args []string) {
			WGSocksStopCmd(cmd, con, args)
		},
		Args: cobra.ExactArgs(1),
	}
	wgSocksCmd.AddCommand(wgSocksStopCmd)
	carapace.Gen(wgSocksStopCmd).PositionalCompletion(SocksIDCompleter(con).Usage("Socks server ID"))

	return []*cobra.Command{wgPortFwdCmd, wgSocksCmd}
}
