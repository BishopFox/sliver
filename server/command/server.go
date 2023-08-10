package command

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
	"github.com/spf13/cobra"

	"github.com/reeflective/team/server"
	"github.com/reeflective/team/server/commands"

	"github.com/bishopfox/sliver/client/command"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/server/command/assets"
	"github.com/bishopfox/sliver/server/command/builder"
	"github.com/bishopfox/sliver/server/command/certs"
	"github.com/bishopfox/sliver/server/command/version"
)

// TeamserverCommands is the equivalent of client/command.ServerCommands(), but for server-binary only ones.
func TeamserverCommands(team *server.Server, con *console.SliverClient) command.SliverBinder {
	return func(con *console.SliverClient) (cmds []*cobra.Command) {
		// Teamserver management
		teamclientCmds := commands.Generate(team, con.Teamclient)
		teamclientCmds.GroupID = constants.GenericHelpGroup
		cmds = append(cmds, teamclientCmds)

		// Sliver-specific
		cmds = append(cmds, version.Commands(con)...)
		cmds = append(cmds, assets.Commands()...)
		cmds = append(cmds, certs.Commands(con)...)

		// Commands requiring the server to be a remote teamclient.
		cmds = append(cmds, builder.Commands(con, team)...)

		return cmds
	}
}
