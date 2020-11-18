package completers

import (
	"strings"

	"github.com/bishopfox/sliver/client/context"
	"github.com/jessevdk/go-flags"
	"github.com/maxlandon/readline"
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

// completeOptionArguments - Completes all values for arguments to a command. Arguments here are different from command options (--option).
// Many categories, from multiple sources in multiple contexts
func completeOptionArguments(cmd *flags.Command, opt *flags.Option, lastWord string) (prefix string, completions []*readline.CompletionGroup) {

	// opt := commands.OptionByName(cmd, arg)
	var comp *readline.CompletionGroup // This group is used as a buffer, to add groups to final completions

	// First of all: some options, no matter their contexts and subject, have default values.
	// When we have such an option, we don't bother analyzing context, we just build completions and return.
	if len(opt.Choices) > 0 {
		comp = &readline.CompletionGroup{
			Name:        opt.LongName,
			DisplayType: readline.TabDisplayGrid,
		}
		for i := range opt.Choices {
			comp.Suggestions = append(comp.Suggestions, opt.Choices[i]+" ")
		}
		completions = append(completions, comp)
		return
	}

	// Depends first on the current menu context.
	// We have a different problem here than for command arguments: options may pertain to different contexts, no matter the context
	// in which we are when using this option (module options, for instance). Therefore the filtering is a bit different, involved.
	//
	// We have 3 words, potentially different, with which we can filter:
	//
	// 1) '--option-name' is the string typed as input.
	// 2) 'OptionName' is the name of the struct/type for this option.
	// 3) 'ValueName' is the name of the value we expect.
	//
	switch context.Context.Menu {
	case context.Server:
		// Any arguments with a path name
		if strings.Contains(opt.Field().Name, "Path") {
			// Then we refine with value name, which might include words as 'binary', 'sliver', etc.
			switch opt.ValueName {
			case "local-path":
				prefix, comp = completeLocalPath(lastWord)
				completions = append(completions, comp)
			case "local-file":
				prefix, comp = completeLocalPathAndFiles(lastWord)
				completions = append(completions, comp)
			default:
				// We always have a default searching for files, locally
				prefix, comp = completeLocalPathAndFiles(lastWord)
				completions = append(completions, comp)
			}
		}

		// Often we "save" files that need paths
		if strings.Contains(opt.Field().Name, "Save") {
			// Then we refine with value name, which might include words as 'binary', 'sliver', etc.
			switch opt.ValueName {
			default:
				// We always have a default searching for files, locally
				prefix, comp = completeLocalPathAndFiles(lastWord)
				completions = append(completions, comp)
			}
		}

		// Certificate and key files
		if strings.Contains(opt.Field().Name, "Certificate") || strings.Contains(opt.Field().Name, "PrivateKey") {
			if strings.Contains(opt.ValueName, "path") {
				prefix, comp = completeLocalPath(lastWord)
				completions = append(completions, comp)
			} else {
				// We always have a default searching for files
				prefix, comp = completeLocalPathAndFiles(lastWord)
				completions = append(completions, comp)
			}
		}

		// Host addresses
		if strings.Contains(opt.Field().Name, "LHost") || strings.Contains(opt.Field().Name, "RHost") {
			// Locally, we add IPv4/IPv6 interfaces (check for option value-name for ip6)

			// If implants on, ask them their interfaces, and add a group for each implant.
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
