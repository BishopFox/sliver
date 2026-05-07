package ai

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

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
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/spf13/cobra"
)

// Commands returns the ai command.
func Commands(con *console.SliverClient) []*cobra.Command {
	return []*cobra.Command{newAICommand(consts.SliverCoreHelpGroup, con)}
}

// ServerCommands returns the ai command for the top-level client REPL.
func ServerCommands(con *console.SliverClient) []*cobra.Command {
	return []*cobra.Command{newAICommand(consts.GenericHelpGroup, con)}
}

func newAICommand(groupID string, con *console.SliverClient) *cobra.Command {
	return &cobra.Command{
		Use:     consts.AIStr,
		Short:   "Open the Sliver AI conversation TUI",
		Long:    help.GetHelpFor([]string{consts.AIStr}),
		Args:    cobra.NoArgs,
		GroupID: groupID,
		Run: func(cmd *cobra.Command, args []string) {
			AICmd(cmd, con, args)
		},
	}
}
