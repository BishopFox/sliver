package completers

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
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/evilsocket/islazy/tui"
	"github.com/jessevdk/go-flags"
	"github.com/maxlandon/readline"

	"github.com/bishopfox/sliver/client/commands"
	"github.com/bishopfox/sliver/client/constants"
	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// CompleteCommandArguments - Completes all values for arguments to a command.
// Arguments here are different from command options (--option).
// Many categories, from multiple sources in multiple contexts
func completeCommandArguments(cmd *flags.Command, arg string, lastWord string) (prefix string, completions []*readline.CompletionGroup) {

	found := commands.ArgumentByName(cmd, arg)
	var comp *readline.CompletionGroup // This group is used as a buffer, to add groups to final completions

	// Depends first on the current menu context.
	// Sometimes, because some commands/options may need completions that are usually
	// part of another context (ex: I am in an implant context, and I want to download
	// a file to path ~/here/my/console/dir, I need local client completion).
	// Because, in addition, some commands/option arguments might not be explicit enough,
	// we need to add special cases.
	switch cctx.Context.Menu {
	case cctx.Server:
		// For any argument with a path in it, we look for the current context'
		// filesystem, and refine results based on a specific command.
		if strings.Contains(found.Name, "Path") {
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

		// Jobs
		if strings.Contains(found.Name, "JobID") {
			jobs, err := transport.RPC.GetJobs(context.Background(), &commonpb.Empty{})
			if err != nil {
				fmt.Printf(util.RPCError+"%s", err)
				return
			}
			comp = &readline.CompletionGroup{
				Name:         "jobs",
				Descriptions: map[string]string{},
				DisplayType:  readline.TabDisplayList,
			}
			for _, job := range jobs.Active {
				jobID := strconv.Itoa(int(job.ID))
				comp.Suggestions = append(comp.Suggestions, jobID+" ")
				comp.Descriptions[jobID+" "] = tui.DIM + job.Description + tui.RESET
			}
			completions = append(completions, comp)
		}

	case cctx.Sliver:
		if strings.Contains(found.Name, "Path") {
			switch cmd.Name {
			case constants.CdStr, constants.LsStr, constants.MkdirStr:
				prefix, comp = completeRemotePath(lastWord)
				completions = append(completions, comp)
				// case constants.GhostCat, constants.GhostDownload, constants.GhostUpload, constants.GhostRm:
				//         return CompleteRemotePathAndFiles(line, pos)
			}
		}
		// case commands.GHOST_CONTEXT:
		// switch found.Name {
		// case "Path", "OtherPath", "RemotePath":
		// Completion might differ slightly depending on the command
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

	// Completions that do not depend on context, and that should either be unique, or be appended to the comp list by default.

	// Sessions
	if strings.Contains(found.Name, "ImplantID") || strings.Contains(found.Name, "SessionID") {
		sessions, err := transport.RPC.GetSessions(context.Background(), &commonpb.Empty{})
		if err != nil {
			fmt.Printf(util.RPCError+"%s", err)
			return
		}
		comp = &readline.CompletionGroup{
			Name:         "sessions",
			Descriptions: map[string]string{},
			DisplayType:  readline.TabDisplayList,
		}
		for _, s := range sessions.Sessions {
			sessionID := strconv.Itoa(int(s.ID))
			comp.Suggestions = append(comp.Suggestions, sessionID+" ")
			desc := fmt.Sprintf("[%s] - %s@%s - %s", s.Name, s.Username, s.Hostname, s.RemoteAddress)
			comp.Descriptions[sessionID+" "] = tui.DIM + desc + tui.RESET
		}
		completions = append(completions, comp)
	}

	return
}
