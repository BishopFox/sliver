package use

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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

	"github.com/bishopfox/sliver/client/command/sessions"
	"github.com/bishopfox/sliver/client/console"
)

// UseSessionCmd - Change the active session
func UseSessionCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, err := sessions.SelectSession(false, con)
	if session != nil {
		con.ActiveTarget.Set(session, nil)
		con.PrintInfof("Active session %s (%s)\n", session.Name, session.ID)
	} else if err != nil {
		switch err {
		case sessions.ErrNoSessions:
			con.PrintErrorf("No sessions available\n")
		case sessions.ErrNoSelection:
			con.PrintErrorf("No session selected\n")
		default:
			con.PrintErrorf("%s\n", err)
		}
	}
}
