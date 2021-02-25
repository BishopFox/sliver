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

	"github.com/bishopfox/sliver/client/readline"

	"github.com/bishopfox/sliver/client/commands"
	"github.com/bishopfox/sliver/client/constants"
	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// CompleteCommandArguments - Completes all values for arguments to a command.
// Arguments here are different from command options (--option).
// Many categories, from multiple sources in multiple contexts
func completeCommandArguments(cmd *flags.Command, arg string, lastWord string) (prefix string, completions []*readline.CompletionGroup) {

	// the prefix is the last word, by default
	prefix = lastWord

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

		// Paths
		if strings.Contains(found.Name, "Path") || strings.Contains(found.Name, "Save") {
			// For any argument with a path in it, we look for the current context'
			// filesystem, and refine results based on a specific command.
			switch cmd.Name {
			case constants.CdStr:
				prefix, comp = completeLocalPath(lastWord)
				completions = append(completions, comp)
			case constants.LsStr, "cat":
				prefix, comp = completeLocalPathAndFiles(lastWord)
				completions = append(completions, comp)

			case constants.WebContentTypeStr, constants.WebUpdateStr, constants.AddWebContentStr, constants.RmWebContentStr:
				// Make an exception for WebPath option in websites commands.
			default:
				prefix, comp = completeLocalPathAndFiles(lastWord)
				completions = append(completions, comp)
			}
		}

		// Jobs
		if strings.Contains(found.Name, "JobID") {
			completions = append(completions, jobIDs(lastWord))
		}

		// Prompt configuration
		if strings.Contains(found.Name, "Prompt") {
			switch cmd.Name {
			case constants.ConfigPromptServerStr:
				completions = append(completions, promptServerItems(lastWord)...)
			case constants.ConfigPromptSliverStr:
				completions = append(completions, promptSliverItems(lastWord)...)
			}
		}
		if strings.Contains(found.Name, "Display") {
			comp := &readline.CompletionGroup{
				Name:        "hint verbosity",
				Suggestions: hints,
				DisplayType: readline.TabDisplayGrid,
			}
			completions = append(completions, comp)
		}

		// URLs
		if strings.Contains(found.Name, "URL") {
			urlPrefix, comps := completeURL(lastWord, false, urlShemes)
			completions = append(completions, comps...)

			// We return this prefix because it is aware of paths lengths, etc...
			return urlPrefix, completions
		}

		// Help
		if strings.Contains(found.Name, "Component") {
			completions = append(completions, completeServerCommands(lastWord))
		}

	// When using a session, some paths are on the remote system, and some are the client console.
	case cctx.Sliver:
		if strings.Contains(found.Name, "RemotePath") || strings.Contains(found.Name, "OtherPath") || found.Name == "Path" {
			switch cmd.Name {
			case constants.CdStr, constants.MkdirStr:
				prefix, comp = completeRemotePath(lastWord)
				completions = append(completions, comp)
			case constants.LsStr, constants.RmStr, constants.CatStr, constants.DownloadStr, constants.UploadStr:
				prefix, comp = completeRemotePathAndFiles(lastWord)
				completions = append(completions, comp)
			case constants.LcdStr:
				prefix, comp = completeLocalPathAndFiles(lastWord)
				completions = append(completions, comp)
				// Load Extensions
			case constants.LoadExtensionStr:
				prefix, comp = completeLocalPathAndFiles(lastWord)
				completions = append(completions, comp)
			}
		}
		if strings.Contains(found.Name, "LocalPath") {
			switch cmd.Name {
			case constants.DownloadStr, constants.UploadStr:
				prefix, comp = completeLocalPathAndFiles(lastWord)
				completions = append(completions, comp)
			default:
				prefix, comp = completeLocalPathAndFiles(lastWord)
				completions = append(completions, comp)
			}
		}

		// Execute command
		if strings.Contains(found.Name, "Args") {
			prefix, comp = completeRemotePathAndFiles(lastWord)
			completions = append(completions, comp)
		}

		// Processes
		if strings.Contains(found.Name, "PID") {
			completions = append(completions, processes(lastWord))
		}

		// URL completer
		if strings.Contains(found.Name, "URL") {
			urlPrefix, comps := completeURL(lastWord, true, urlShemes)
			completions = append(completions, comps...)

			// We return this prefix because it is aware of paths lengths, etc...
			return urlPrefix, completions
		}

		// Help
		if strings.Contains(found.Name, "Component") {
			completions = append(completions, completeServerCommands(lastWord))
			completions = append(completions, completeSliverCommands(lastWord))
		}
	}

	// Completions that do not depend on context, and that should either be unique, or be appended to the comp list by default.

	// Sessions
	if strings.Contains(found.Name, "ImplantID") || strings.Contains(found.Name, "SessionID") {
		completions = append(completions, sessionIDs(lastWord))
	}

	// Logs
	if strings.Contains(found.Name, "Level") {
		completions = append(completions, completeLogLevels(lastWord))
	}
	if strings.Contains(found.Name, "Components") {
		completions = append(completions, completeLoggers(lastWord))
	}

	// Implant builds & profiles
	if strings.Contains(found.Name, "Profile") {
		completions = append(completions, implantProfiles(lastWord))
	}
	if strings.Contains(found.Name, "ImplantName") {
		completions = append(completions, implantNames(lastWord))
	}

	return
}

func completeServerCommands(lastWord string) (comp *readline.CompletionGroup) {
	comp = &readline.CompletionGroup{
		Name:        "server commands",
		DisplayType: readline.TabDisplayGrid,
		MaxLength:   10,
	}

	groups, cmds := cctx.Commands.GetServerGroups()
	for _, group := range groups {
		for _, cmd := range cmds[group] {
			if strings.HasPrefix(cmd.Name, lastWord) {
				comp.Suggestions = append(comp.Suggestions, cmd.Name)
			}
		}
	}
	return
}

func completeSliverCommands(lastWord string) (comp *readline.CompletionGroup) {
	comp = &readline.CompletionGroup{
		Name:        "session commands",
		DisplayType: readline.TabDisplayGrid,
		MaxLength:   10,
	}

	groups, cmds := cctx.Commands.GetSliverGroups()
	for _, group := range groups {
		for _, cmd := range cmds[group] {
			if strings.HasPrefix(cmd.Name, lastWord) {
				comp.Suggestions = append(comp.Suggestions, cmd.Name)
			}
		}
	}
	return
}

func jobIDs(lastWord string) (comp *readline.CompletionGroup) {
	comp = &readline.CompletionGroup{
		Name:         "jobs",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayList,
	}

	jobs, err := transport.RPC.GetJobs(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.RPCError+"%s", err)
		return
	}
	for _, job := range jobs.Active {
		jobID := strconv.Itoa(int(job.ID))
		if strings.HasPrefix(jobID, lastWord) {
			jobID := strconv.Itoa(int(job.ID))
			comp.Suggestions = append(comp.Suggestions, jobID)
			comp.Descriptions[jobID] = tui.DIM + job.Name + fmt.Sprintf(" (%s)", job.Description) + tui.RESET
		}
	}

	return
}

func completeLogLevels(lastWord string) (comp *readline.CompletionGroup) {
	comp = &readline.CompletionGroup{
		Name:         "levels",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayGrid,
	}
	for _, lvl := range logLevels {
		if strings.HasPrefix(lvl, lastWord) {
			comp.Suggestions = append(comp.Suggestions, lvl)
		}
	}

	return
}
func completeLoggers(lastWord string) (comp *readline.CompletionGroup) {
	comp = &readline.CompletionGroup{
		Name:         "loggers",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayGrid,
	}
	for _, logger := range loggers {
		if strings.HasPrefix(logger, lastWord) {
			comp.Suggestions = append(comp.Suggestions, logger)
		}
	}

	return
}

func implantProfiles(lastWord string) (comp *readline.CompletionGroup) {
	comp = &readline.CompletionGroup{
		Name:         "implant profiles",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayList,
		MaxLength:    20,
	}
	profiles := getSliverProfiles()
	for _, profile := range *profiles {
		if strings.HasPrefix(profile.Name, lastWord) {
			conf := profile.Config
			comp.Suggestions = append(comp.Suggestions, profile.Name)
			desc := fmt.Sprintf(" %s [%s/%s] -> %d C2s", conf.Format.String(), conf.GOOS, conf.GOARCH, len(conf.GetC2()))
			comp.Descriptions[profile.Name] = tui.DIM + desc
		}
	}

	return
}

func getSliverProfiles() *map[string]*clientpb.ImplantProfile {
	pbProfiles, err := transport.RPC.ImplantProfiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.Error+"Error %s", err)
		return nil
	}
	profiles := &map[string]*clientpb.ImplantProfile{}
	for _, profile := range pbProfiles.Profiles {
		(*profiles)[profile.Name] = profile
	}
	return profiles
}

func implantNames(lastWord string) (comp *readline.CompletionGroup) {
	comp = &readline.CompletionGroup{
		Name:         "implants",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayList,
		MaxLength:    20,
	}

	builds, err := transport.RPC.ImplantBuilds(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.RPCError+"%s\n", err)
		return
	}

	for name, implant := range builds.Configs {
		if strings.HasPrefix(implant.Name, lastWord) {
			comp.Suggestions = append(comp.Suggestions, name)
			desc := fmt.Sprintf(" %s [%s/%s] -> %d C2s", implant.Format.String(), implant.GOOS, implant.GOARCH, len(implant.GetC2()))
			comp.Descriptions[name] = tui.DIM + desc
		}
	}

	return
}

func sessionIDs(lastWord string) (comp *readline.CompletionGroup) {

	comp = &readline.CompletionGroup{
		Name:         "sessions",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayList,
	}

	sessions, err := transport.RPC.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.RPCError+"%s", err)
		return
	}
	for _, s := range sessions.Sessions {
		sessionID := strconv.Itoa(int(s.ID))
		if strings.HasPrefix(sessionID, lastWord) {
			comp.Suggestions = append(comp.Suggestions, sessionID)
			desc := fmt.Sprintf("[%s] - %s@%s - %s", s.Name, s.Username, s.Hostname, s.RemoteAddress)
			comp.Descriptions[sessionID] = tui.DIM + desc + tui.RESET
		}
	}
	return
}

func processes(lastWord string) (comp *readline.CompletionGroup) {
	comp = &readline.CompletionGroup{
		Name:         "host processes",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayList,
		MaxLength:    20,
	}

	session := cctx.Context.Sliver
	if session == nil {
		return
	}

	ps, err := transport.RPC.Ps(context.Background(), &sliverpb.PsReq{
		Request: cctx.Request(session.Session)})
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
		return
	}
	for _, proc := range ps.Processes {
		pid := strconv.Itoa(int(proc.Pid))
		if strings.HasPrefix(pid, lastWord) {
			comp.Suggestions = append(comp.Suggestions, pid)
			var color string
			if session != nil && proc.Pid == session.PID {
				color = tui.GREEN
			}
			desc := fmt.Sprintf("%s(%d - %s)  %s", color, proc.Ppid, proc.Owner, proc.Executable)
			comp.Descriptions[pid] = tui.DIM + desc + tui.RESET
		}
	}

	return
}
