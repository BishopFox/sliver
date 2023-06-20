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
	"github.com/reeflective/console"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command/alias"
	"github.com/bishopfox/sliver/client/command/armory"
	"github.com/bishopfox/sliver/client/command/beacons"
	"github.com/bishopfox/sliver/client/command/builders"
	"github.com/bishopfox/sliver/client/command/crack"
	"github.com/bishopfox/sliver/client/command/creds"
	"github.com/bishopfox/sliver/client/command/exit"
	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/command/hosts"
	"github.com/bishopfox/sliver/client/command/info"
	"github.com/bishopfox/sliver/client/command/jobs"
	"github.com/bishopfox/sliver/client/command/licenses"
	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/command/monitor"
	"github.com/bishopfox/sliver/client/command/operators"
	operator "github.com/bishopfox/sliver/client/command/prelude-operator"
	"github.com/bishopfox/sliver/client/command/reaction"
	"github.com/bishopfox/sliver/client/command/sessions"
	"github.com/bishopfox/sliver/client/command/settings"
	sgn "github.com/bishopfox/sliver/client/command/shikata-ga-nai"
	"github.com/bishopfox/sliver/client/command/update"
	"github.com/bishopfox/sliver/client/command/use"
	"github.com/bishopfox/sliver/client/command/websites"
	"github.com/bishopfox/sliver/client/command/wireguard"
	client "github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// ServerCommands returns all commands bound to the server menu, optionally
// accepting a function returning a list of additional (admin) commands.
func ServerCommands(con *client.SliverConsoleClient, serverCmds func() []*cobra.Command) console.Commands {
	serverCommands := func() *cobra.Command {
		server := &cobra.Command{
			Short: "Server commands",
			CompletionOptions: cobra.CompletionOptions{
				HiddenDefaultCmd: true,
			},
		}

		// [ Server-only ]
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
		bindCommands(consts.GenericHelpGroup, server, con,
			exit.Command,
			licenses.Commands,
			settings.Commands,
			alias.Commands,
			armory.Commands,
			update.Commands,
			operators.Commands,
			operator.Commands,
			creds.Commands,
			crack.Commands,
		)

		// C2 Network
		bindCommands(consts.NetworkHelpGroup, server, con,
			jobs.Commands,
			websites.Commands,
			wireguard.Commands,
		)

		// Payloads
		bindCommands(consts.PayloadsHelpGroup, server, con,
			sgn.Commands,
			generate.Commands,
			builders.Commands,
		)

		// Slivers
		bindCommands(consts.SliverHelpGroup, server, con,
			use.Commands,
			info.Commands,
			sessions.Commands,
			beacons.Commands,
			monitor.Commands,
			loot.Commands,
			hosts.Commands,
			reaction.Commands,
		)

		// [ Post-command declaration setup]-----------------------------------------

		// Everything below this line should preferably not be any command binding
		// (although you can do so without fear). If there are any final modifications
		// to make to the server menu command tree, it time to do them here.

		server.InitDefaultHelpCmd()
		server.SetHelpCommandGroupID(consts.GenericHelpGroup)

		return server
	}

	return serverCommands
}
