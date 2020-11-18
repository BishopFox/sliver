package completers

import (
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/maxlandon/readline"

	"github.com/bishopfox/sliver/client/commands"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/context"
)

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

// CompleteCommandArguments - Completes all values for arguments to a command. Arguments here are different from command options (--option).
// Many categories, from multiple sources in multiple contexts
func completeCommandArguments(cmd *flags.Command, arg string, lastWord string) (prefix string, completions []*readline.CompletionGroup) {

	found := commands.ArgumentByName(cmd, arg)
	var comp *readline.CompletionGroup // This group is used as a buffer, to add groups to final completions

	// Depends first on the current menu context.
	// Sometimes, because some commands/options may need completions that are usually part of another context (ex: I am in an implant context,
	// and I want to download a file to path ~/here/my/console/dir, I need local client completion).
	// Because, in addition, some commands/option arguments might not be explicit enough, we need to add special cases.
	switch context.Context.Menu {
	case context.Server:
		// For any argument with a path in it, we look for the current context' filesystem, and refine results based on a specific command.
		if strings.Contains(found.Name, "Path") || strings.Contains(found.Name, "LocalPath") || strings.Contains(found.Name, "OtherPath") {
			switch cmd.Name {
			case constants.CdStr:
				prefix, comp = completeLocalPath(lastWord)
				completions = append(completions, comp)
			case constants.LsStr, "cat":
				prefix, comp = completeLocalPathAndFiles(lastWord)
				completions = append(completions, comp)
			default:
				prefix, comp = completeLocalPathAndFiles(lastWord)
				completions = append(completions, comp)
			}

		}

		// case commands.GHOST_CONTEXT:
		//         switch found.Name {
		//         case "Path", "OtherPath", "RemotePath":
		//                 // Completion might differ slightly depending on the command
		//                 switch cmd.Name {
		//                 case constants.GhostCd, constants.GhostLs, constants.GhostMkdir:
		//                         return CompleteRemotePath(line, pos)
		//                 case constants.GhostCat, constants.GhostDownload, constants.GhostUpload, constants.GhostRm:
		//                         return CompleteRemotePathAndFiles(line, pos)
		//                 }
		//         case "LocalPath":
		//                 switch cmd.Name {
		//                 case constants.GhostUpload:
		//                         return completeLocalPathAndFiles(line, pos)
		//                 case constants.GhostDownload:
		//                         return CompleteLocalPath(line, pos)
		//                 }
		//         case "PID":
		//                 commands.Context.Shell.MaxTabCompleterRows = 10
		//                 return CompleteProcesses(line, pos)
		//         default: // If name is empty, return
	}
	return
}
