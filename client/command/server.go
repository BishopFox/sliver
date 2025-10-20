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
	"os"

	"github.com/gsmith257-cyber/better-sliver-package/client/command/alias"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/armory"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/beacons"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/builders"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/c2profiles"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/certificates"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/clean"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/crack"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/creds"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/exit"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/generate"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/hosts"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/info"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/jobs"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/licenses"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/loot"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/monitor"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/operators"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/reaction"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/sessions"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/settings"
	sgn "github.com/gsmith257-cyber/better-sliver-package/client/command/shikata-ga-nai"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/socks"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/taskmany"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/update"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/use"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/websites"
	"github.com/gsmith257-cyber/better-sliver-package/client/command/wireguard"
	client "github.com/gsmith257-cyber/better-sliver-package/client/console"
	consts "github.com/gsmith257-cyber/better-sliver-package/client/constants"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
)

// ServerCommands returns all commands bound to the server menu, optionally
// accepting a function returning a list of additional (admin) commands.
func ServerCommands(con *client.SliverClient, serverCmds func() []*cobra.Command) console.Commands {
	serverCommands := func() *cobra.Command {
		server := &cobra.Command{
			Short: "Server commands",
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
			server.AddGroup(&cobra.Group{ID: consts.MultiplayerHelpGroup, Title: consts.MultiplayerHelpGroup})
			server.AddCommand(serverCmds()...)
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
			operators.Commands,
			creds.Commands,
			crack.Commands,
			certificates.Commands,
			clean.Command,
		)

		// C2 Network
		bind(consts.NetworkHelpGroup,
			jobs.Commands,
			websites.Commands,
			wireguard.Commands,
			c2profiles.Commands,
			socks.RootCommands,
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

		// Only load reactions when the console is going to be started.
		if !con.IsCLI {
			n, err := reaction.LoadReactions()
			if err != nil && !os.IsNotExist(err) {
				con.PrintErrorf("Failed to load reactions: %s\n", err)
			} else if n > 0 {
				con.PrintInfof("Loaded %d reaction(s) from disk\n", n)
			}
		}

		server.InitDefaultHelpCmd()
		server.SetHelpCommandGroupID(consts.GenericHelpGroup)

		return server
	}

	return serverCommands
}
