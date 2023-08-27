package command

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

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
	"github.com/spf13/cobra"

	"github.com/reeflective/console"

	"github.com/bishopfox/sliver/client/command/alias"
	"github.com/bishopfox/sliver/client/command/armory"
	"github.com/bishopfox/sliver/client/command/beacons"
	"github.com/bishopfox/sliver/client/command/builders"
	"github.com/bishopfox/sliver/client/command/crack"
	"github.com/bishopfox/sliver/client/command/creds"
	"github.com/bishopfox/sliver/client/command/dns"
	"github.com/bishopfox/sliver/client/command/exit"
	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/command/hosts"
	"github.com/bishopfox/sliver/client/command/http"
	"github.com/bishopfox/sliver/client/command/https"
	"github.com/bishopfox/sliver/client/command/info"
	"github.com/bishopfox/sliver/client/command/jobs"
	"github.com/bishopfox/sliver/client/command/licenses"
	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/command/monitor"
	"github.com/bishopfox/sliver/client/command/mtls"
	operator "github.com/bishopfox/sliver/client/command/prelude-operator"
	"github.com/bishopfox/sliver/client/command/reaction"
	"github.com/bishopfox/sliver/client/command/sessions"
	"github.com/bishopfox/sliver/client/command/settings"
	sgn "github.com/bishopfox/sliver/client/command/shikata-ga-nai"
	"github.com/bishopfox/sliver/client/command/taskmany"
	"github.com/bishopfox/sliver/client/command/tcp"
	"github.com/bishopfox/sliver/client/command/transports"
	"github.com/bishopfox/sliver/client/command/update"
	"github.com/bishopfox/sliver/client/command/use"
	"github.com/bishopfox/sliver/client/command/websites"
	"github.com/bishopfox/sliver/client/command/wireguard"
	client "github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// ***** Command Generators and Runner Binders ******

// ServerCommands returns all commands bound to the server menu, optionally
// accepting a function returning a list of additional (admin) commands.
func ServerCommands(con *client.SliverClient, serverCmds SliverBinder) console.Commands {
	serverCommands := func() *cobra.Command {
		server := &cobra.Command{
			Short:            "Server commands",
			TraverseChildren: true,
			SilenceUsage:     true,
			CompletionOptions: cobra.CompletionOptions{
				HiddenDefaultCmd: true,
			},
		}

		// Utility function to be used for binding new commands to
		// the sliver menu: call the function with the name of the
		// group under which this/these commands should be added,
		// and the group will be automatically created if needed.
		bind := makeBind(server, con)

		if serverCmds != nil {
			bind(consts.GenericHelpGroup,
				serverCmds,
			)
		}

		// [ Bind commands ] --------------------------------------------------------

		// Below are bounds all commands of the server menu, gathered by the group
		// under which they should be printed in help messages and/or completions.
		// You can either add a new bindCommands() call with a new group (which will
		// be automatically added to the command tree), or add your commands in one of
		// the present calls.

		// Core
		bind(consts.GenericHelpGroup,
			exit.Command,
			licenses.Commands,
			settings.Commands,
			alias.Commands,
			armory.Commands,
			update.Commands,
			operator.Commands,
			creds.Commands,
			crack.Commands,
		)

		// C2 Network
		bind(consts.NetworkHelpGroup,
			transports.Commands,
			jobs.Commands,
			mtls.Commands,
			dns.Commands,
			http.Commands,
			https.Commands,
			tcp.Commands,
			wireguard.Commands,
			websites.Commands,
		)

		// Payloads
		bind(consts.PayloadsHelpGroup,
			sgn.Commands,
			generate.Commands,
			builders.Commands,
		)

		// Slivers
		bind(consts.SliverHelpGroup,
			use.Commands,
			info.Commands,
			sessions.Commands,
			beacons.Commands,
			monitor.Commands,
			loot.Commands,
			hosts.Commands,
			reaction.Commands,
			taskmany.Command,
		)

		// [ Post-command declaration setup ]-----------------------------------------

		// Everything below this line should preferably not be any command binding
		// (although you can do so without fear). If there are any final modifications
		// to make to the server menu command tree, it time to do them here.

		server.InitDefaultHelpCmd()
		server.SetHelpCommandGroupID(consts.GenericHelpGroup)

		con.FilterCommands(server)

		return server
	}

	return serverCommands
}

// BindPreRun registers specific pre-execution runners for all
// leafs commands (and some nodes) of a given command/tree.
func BindPreRun(root *cobra.Command, runs ...CobraRunnerE) {
	for _, cmd := range root.Commands() {
		ePreE := cmd.PersistentPreRunE
		run, runE := cmd.Run, cmd.RunE

		// Don't modify commands in charge on their own tree.
		if ePreE != nil {
			continue
		}

		// Always go to find the leaf commands, irrespective
		// of what we do with this parent command.
		if cmd.HasSubCommands() {
			BindPreRun(cmd, runs...)
		}

		// If the command has no runners, there's nothing to bind:
		// If it has flags, any child command requiring them should
		// trigger the prerunners, which will connect to the server.
		if run == nil && runE == nil {
			continue
		}

		// Else we have runners, bind the pre-runs if possible.
		if cmd.PreRunE != nil {
			continue
		}

		// Compound all runners together.
		cRun := func(c *cobra.Command, args []string) error {
			for _, run := range runs {
				err := run(c, args)
				if err != nil {
					return err
				}
			}

			return nil
		}

		// Bind
		cmd.PreRunE = cRun
	}
}

// BindPostRun registers specific post-execution runners for all
// leafs commands (and some nodes) of a given command/tree.
func BindPostRun(root *cobra.Command, runs ...CobraRunnerE) {
	for _, cmd := range root.Commands() {
		ePostE := cmd.PersistentPostRunE
		run, runE := cmd.Run, cmd.RunE

		// Don't modify commands in charge on their own tree.
		if ePostE != nil {
			continue
		}

		// Always go to find the leaf commands, irrespective
		// of what we do with this parent command.
		if cmd.HasSubCommands() {
			BindPostRun(cmd, runs...)
		}

		// If the command has no runners, there's nothing to bind:
		// If it has flags, any child command requiring them should
		// trigger the prerunners, which will connect to the server.
		if run == nil && runE == nil {
			continue
		}

		// Else we have runners, bind the pre-runs if possible.
		if cmd.PostRunE != nil {
			continue
		}

		// Compound all runners together.
		cRun := func(c *cobra.Command, args []string) error {
			for _, run := range runs {
				err := run(c, args)
				if err != nil {
					return err
				}
			}

			return nil
		}

		// Bind
		cmd.PostRunE = cRun
	}
}
