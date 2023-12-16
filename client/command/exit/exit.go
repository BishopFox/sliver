package exit

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

	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
)

// Commands returns the `exit` command.
func Command(con *console.SliverClient) []*cobra.Command {
	return []*cobra.Command{{
		Use:         "exit",
		Short:       "Exit the program",
		Annotations: flags.RestrictTargets(constants.ConsoleCmdsFilter),
		Run: func(cmd *cobra.Command, args []string) {
			con.ExitConfirm()
		},
		GroupID: constants.GenericHelpGroup,
	}}
}
