package reaction

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
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/spf13/cobra"
)

// ReactionSaveCmd - Manage reactions to events.
func ReactionReloadCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	if _, err := os.Stat(GetReactionFilePath()); os.IsNotExist(err) {
		con.PrintErrorf("Missing reaction file %s\n", GetReactionFilePath())
		return
	}
	confirm := false
	prompt := &survey.Confirm{Message: "Reload reactions from disk?"}
	survey.AskOne(prompt, &confirm)
	if !confirm {
		return
	}

	n, err := LoadReactions()
	if err != nil {
		con.PrintErrorf("Failed to load reactions: %s\n", err)
	}
	con.Println()
	con.PrintInfof("Reloaded %d reactions from disk\n", n)
}
