package transports

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

	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// Commands returns the `transports` command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	transportsCmd := &cobra.Command{
		Use:     consts.TranportsStr,
		Short:   "Generic transport preparation & management",
		GroupID: consts.NetworkHelpGroup,
	}

	// Traffic encoders
	trafficEncodersCmd := &cobra.Command{
		Use:   consts.TrafficEncodersStr,
		Short: "Manage implant traffic encoders",
		Long:  help.GetHelpFor([]string{consts.GenerateStr, consts.TrafficEncodersStr}),
		Run: func(cmd *cobra.Command, args []string) {
			TrafficEncodersCmd(cmd, con, args)
		},
	}
	transportsCmd.AddCommand(trafficEncodersCmd)

	trafficEncodersAddCmd := &cobra.Command{
		Use:   consts.AddStr,
		Short: "Add a new traffic encoder to the server from the local file system",
		Long:  help.GetHelpFor([]string{consts.GenerateStr, consts.TrafficEncodersStr, consts.AddStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			TrafficEncodersAddCmd(cmd, con, args)
		},
	}
	flags.Bind("", false, trafficEncodersAddCmd, func(f *pflag.FlagSet) {
		f.BoolP("skip-tests", "s", false, "skip testing the traffic encoder (not recommended)")
	})
	carapace.Gen(trafficEncodersAddCmd).PositionalCompletion(carapace.ActionFiles("wasm").Tag("wasm files").Usage("local file path (expects .wasm)"))
	trafficEncodersCmd.AddCommand(trafficEncodersAddCmd)

	trafficEncodersRmCmd := &cobra.Command{
		Use:   consts.RmStr,
		Short: "Remove a traffic encoder from the server",
		Long:  help.GetHelpFor([]string{consts.GenerateStr, consts.TrafficEncodersStr, consts.RmStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			TrafficEncodersRemoveCmd(cmd, con, args)
		},
	}
	carapace.Gen(trafficEncodersRmCmd).PositionalCompletion(TrafficEncodersCompleter(con).Usage("traffic encoder to remove"))
	trafficEncodersCmd.AddCommand(trafficEncodersRmCmd)

	return []*cobra.Command{transportsCmd}
}

// Commands returns the `transports` command and its subcommands in the implant tree.
func SliverCommands(con *console.SliverClient) []*cobra.Command {
	transportsCmd := &cobra.Command{
		Use:     consts.TranportsStr,
		Short:   "Generic transport preparation & management",
		GroupID: consts.NetworkHelpGroup,
	}

	// Reconfig
	reconfigCmd := &cobra.Command{
		Use:         consts.ReconfigStr,
		Short:       "Reconfigure the active beacon/session",
		Long:        help.GetHelpFor([]string{consts.ReconfigStr}),
		Annotations: flags.RestrictTargets(consts.BeaconCmdsFilter),
		Run: func(cmd *cobra.Command, args []string) {
			ReconfigCmd(cmd, con, args)
		},
	}
	flags.Bind("reconfig", false, reconfigCmd, func(f *pflag.FlagSet) {
		f.StringP("reconnect-interval", "r", "", "reconnect interval for implant")
		f.StringP("beacon-interval", "i", "", "beacon callback interval")
		f.StringP("beacon-jitter", "j", "", "beacon callback jitter (random up to)")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	transportsCmd.AddCommand(reconfigCmd)

	return []*cobra.Command{transportsCmd}
}
