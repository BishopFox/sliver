package server

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
	"fmt"

	"github.com/maxlandon/gonsole"

	// "github.com/bishopfox/sliver/client/commands/sliver"
	"github.com/bishopfox/sliver/client/completion"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/help"
	"github.com/bishopfox/sliver/client/util"
)

const (
	// ANSI Colors
	normal    = "\033[0m"
	black     = "\033[30m"
	red       = "\033[31m"
	green     = "\033[32m"
	orange    = "\033[33m"
	blue      = "\033[34m"
	purple    = "\033[35m"
	cyan      = "\033[36m"
	gray      = "\033[37m"
	bold      = "\033[1m"
	clearln   = "\r\x1b[2K"
	upN       = "\033[%dA"
	downN     = "\033[%dB"
	underline = "\033[4m"
)

var (
	// Console Some commands might need to access the current context
	// in the course of the application execution.
	Console *gonsole.Console

	// Most commands just need access to a precise context.
	serverMenu *gonsole.Menu
)

// BindCommands - All commands bound in this function are NOT speaking with implants sessions,
// but can still be used when the user is interacting with Slivers. A precise context is passed so
// that we can selectively bind these commands to contexts from outside.
func BindCommands(cc *gonsole.Menu) {

	// Keep a reference to this context, command implementations might want to use it.
	serverMenu = cc

	// Default (local) Shell Exec ---------------------------------------------------------------

	// Unknown commands, in the server context, are automatically passed to the system.
	serverMenu.UnknownCommandHandler = util.Shell

	// Core Commands ----------------------------------------------------------------------------
	cc.AddCommand(constants.ExitStr, // Command name
		"Exit from the client/server console", // Short description (completions & hints)
		"",                                    // Long description
		constants.CoreServerGroup,             // The group of the command (completions, menu structuring)
		[]string{""},                          // A filter by which to hide/show the command.
		func() interface{} { return &Exit{} }) // The command generator, yielding instances.

	cc.AddCommand(constants.VersionStr,
		"Display version information",
		help.GetHelpFor(constants.VersionStr),
		constants.CoreServerGroup,
		[]string{""},
		func() interface{} { return &Version{} })

	cc.AddCommand(constants.LicensesStr,
		"Display project licenses (core & libraries)",
		"",
		constants.CoreServerGroup,
		[]string{""},
		func() interface{} { return &Licenses{} })

	updates := cc.AddCommand(constants.UpdateStr,
		"Check for newer Sliver console/server releases",
		help.GetHelpFor(constants.UpdateStr),
		constants.CoreServerGroup,
		[]string{""},
		func() interface{} { return &Updates{} })
	updates.AddOptionCompletionDynamic("Proxy", completion.NewURLCompleterProxyUpdate().CompleteURL) // Option completions
	updates.AddOptionCompletionDynamic("Save", Console.Completer.LocalPath)

	cc.AddCommand(constants.PlayersStr,
		"List operators and their status",
		help.GetHelpFor(constants.PlayersStr),
		constants.CoreServerGroup,
		[]string{""},
		func() interface{} { return &Operators{} })

	// Log management ---------------------------------------------------------------------------
	log := cc.AddCommand(constants.LogStr,
		"Manage log levels of one or more components",
		"",
		constants.CoreServerGroup,
		[]string{""},
		func() interface{} { return &Log{} })
	log.AddArgumentCompletion("Level", completion.LogLevels)
	log.AddArgumentCompletion("Components", completion.Loggers)

	// Jobs management --------------------------------------------------------------------------
	jobs := cc.AddCommand(constants.JobsStr,
		"Job management commands",
		help.GetHelpFor(constants.JobsStr),
		constants.CoreServerGroup,
		[]string{""},
		func() interface{} { return &Jobs{} })

	jobs.SubcommandsOptional = true

	kill := jobs.AddCommand(constants.JobsKillStr,
		"Kill one or more jobs given their ID",
		"", "", []string{""},
		func() interface{} { return &JobsKill{} })
	kill.AddArgumentCompletion("JobID", completion.JobIDs)

	jobs.AddCommand(constants.JobsKillAllStr,
		"Kill all active jobs on server",
		"", "", []string{""},
		func() interface{} { return &JobsKillAll{} })

	// Session Management ----------------------------------------------------------------------------
	interact := cc.AddCommand(constants.UseStr,
		"Interact with an implant",
		help.GetHelpFor(constants.UseStr),
		constants.SessionsGroup,
		[]string{""},
		func() interface{} { return &Interact{} })
	interact.AddArgumentCompletion("SessionID", completion.SessionIDs)

	sessions := cc.AddCommand(constants.SessionsStr,
		"Session management (all contexts)",
		help.GetHelpFor(constants.SessionsStr),
		constants.SessionsGroup,
		[]string{""},
		func() interface{} { return &Sessions{} })

	sessions.SubcommandsOptional = true

	sinteract := sessions.AddCommand(constants.InteractStr,
		"Interact with an implant",
		help.GetHelpFor(constants.UseStr),
		"",
		[]string{""},
		func() interface{} { return &Interact{} })
	sinteract.AddArgumentCompletion("SessionID", completion.SessionIDs)

	sessionKill := sessions.AddCommand(constants.KillStr,
		"Kill one or more implant sessions",
		"", "", []string{""},
		func() interface{} { return &SessionsKill{} })
	sessionKill.AddArgumentCompletion("SessionID", completion.SessionIDs)

	sessions.AddCommand(constants.JobsKillAllStr,
		"Kill all registered sessions",
		"", "", []string{""},
		func() interface{} { return &SessionsKillAll{} })

	sessions.AddCommand("clean",
		"Clean sessions marked Dead",
		"", "", []string{""},
		func() interface{} { return &SessionsClean{} })

	// Stage / Stager Generation -------------------------------------------------------------------------
	g := cc.AddCommand(constants.GenerateStr,
		"Configure and compile an implant (staged or stager)",
		"",
		constants.BuildsGroup,
		[]string{""},
		func() interface{} { return &Generate{} })

	s := g.AddCommand(constants.StageStr,
		"Configure and compile a Sliver (stage) implant",
		help.GetHelpFor(constants.GenerateStr),
		"", []string{""},
		func() interface{} { return &GenerateStage{} })
	s.AddOptionCompletion("Platform", completion.CompleteStagePlatforms)
	s.AddOptionCompletion("Format", completion.CompleteStageFormats)
	s.AddOptionCompletionDynamic("Save", Console.Completer.LocalPath)
	s.AddOptionCompletion("MTLS", completion.ServerInterfaceAddrs)
	s.AddOptionCompletion("HTTP", completion.ServerInterfaceAddrs)
	s.AddOptionCompletion("DNS", completion.ServerInterfaceAddrs)
	s.AddOptionCompletion("TCPPivot", completion.ActiveSessionIfaceAddrs)

	sg := g.AddCommand(constants.StagerStr,
		"Generate a stager shellcode payload using MSFVenom, (to file: --save, to stdout: --format",
		help.GetHelpFor(constants.StagerStr),
		"", []string{""},
		func() interface{} { return &GenerateStager{} })
	sg.AddOptionCompletion("Format", completion.CompleteMsfFormats)
	sg.AddOptionCompletion("Protocol", completion.CompleteMsfProtocols)
	sg.AddOptionCompletionDynamic("Save", Console.Completer.LocalPath)

	// Profiles Management / Generation ----------------------------------------------------------------
	p := cc.AddCommand(constants.NewProfileStr,
		"Configure and save a new (stage) implant profile",
		help.GetHelpFor(constants.NewProfileStr),
		constants.BuildsGroup,
		[]string{""},
		func() interface{} { return &NewProfile{} })
	p.AddOptionCompletion("Platform", completion.CompleteStagePlatforms)
	p.AddOptionCompletion("Format", completion.CompleteStageFormats)

	regenerate := cc.AddCommand(constants.RegenerateStr,
		"Recompile an implant by name, passed as argument (completed)",
		help.GetHelpFor(constants.RegenerateStr),
		constants.BuildsGroup,
		[]string{""},
		func() interface{} { return &Regenerate{} })
	regenerate.AddArgumentCompletion("ImplantName", completion.ImplantNames)

	pr := cc.AddCommand(constants.ProfilesStr,
		"List existing implant profiles",
		help.GetHelpFor(constants.ProfilesStr),
		constants.BuildsGroup,
		[]string{""},
		func() interface{} { return &Profiles{} })

	pr.SubcommandsOptional = true

	profileDelete := pr.AddCommand(constants.RmStr,
		"Delete one or more existing implant profiles",
		"", "", []string{""},
		func() interface{} { return &ProfileDelete{} })
	profileDelete.AddArgumentCompletion("Profile", completion.ImplantProfiles)

	cc.AddCommand(constants.ProfileGenerateStr,
		"Compile an implant based on a profile, passed as argument (completed)",
		help.GetHelpFor(constants.ProfileGenerateStr),
		constants.BuildsGroup,
		[]string{""},
		func() interface{} { return &ProfileGenerate{} })

	builds := cc.AddCommand(constants.ImplantBuildsStr,
		"List old implant builds",
		help.GetHelpFor(constants.ImplantBuildsStr),
		constants.BuildsGroup,
		[]string{""},
		func() interface{} { return &Builds{} })

	builds.SubcommandsOptional = true
	buildsRm := builds.AddCommand(constants.RmStr,
		"Remove one or more implant builds from the server database",
		help.GetHelpFor(fmt.Sprintf("%s.%s", constants.ImplantBuildsStr, constants.RmStr)),
		"",
		[]string{""},
		func() interface{} { return &RemoveBuild{} })
	buildsRm.AddArgumentCompletion("Names", completion.ImplantNames)

	cc.AddCommand(constants.ListCanariesStr,
		"List previously generated DNS canaries",
		help.GetHelpFor(constants.ListCanariesStr),
		constants.BuildsGroup,
		[]string{""},
		func() interface{} { return &Canaries{} })

	// Context-sensitive commands / alias -----------------------------------------------------------
	switch cc.Name {
	case constants.ServerMenu:
		cd := cc.AddCommand(constants.CdStr,
			"Change client working directory",
			"",
			constants.CoreServerGroup,
			[]string{""},
			func() interface{} { return &ChangeClientDirectory{} })
		// Comps
		cd.AddArgumentCompletionDynamic("Path", Console.Completer.LocalPath)

		// The info command is a session management one
		// in server context, but a core one in session context.
		// info := cc.AddCommand(constants.InfoStr,
		//         "Show session information",
		//         "",
		//         constants.SessionsGroup,
		//         []string{""},
		//         func() interface{} { return &sliver.Info{} })
		// info.AddArgumentCompletion("SessionID", completion.SessionIDs)

	case constants.SliverMenu:
		lcd := cc.AddCommand(constants.LcdStr,
			"Change the client working directory",
			"",
			constants.CoreServerGroup,
			[]string{""},
			func() interface{} { return &ChangeClientDirectory{} })
		lcd.AddArgumentCompletionDynamic("Path", Console.Completer.LocalPath)
	}
}
