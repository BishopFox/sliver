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
	"github.com/jessevdk/go-flags"

	"github.com/bishopfox/sliver/client/commands/sliver"
	"github.com/bishopfox/sliver/client/constants"
	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/help"
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

// BindCommands - The server commands are most of the time all bound the console, however, if in Sliver
// context, some of them are registered under slightly different names, such as cd => lcd and ls => lls.
//
// registerGroup - Function used to either print the error arising from registering a command (to be used for
// debugging) or to classify it into a group of commands (used for completion structure and context menus help)
// PLEASE PASS THIS FUNCTION TO ANY COMMAND YOU REGISTER HERE !!!
func BindCommands(parser *flags.Parser, registerGroup func(err error, cmd *flags.Command, group string)) (errs []error) {

	// 1 - Commands registered under the same name, under all contexts

	// Core Commands ----------------------------------------------------------------------------
	ex, err := parser.AddCommand(constants.ExitStr, // Command string
		"Exit from the client/server console", // Description (completions, help usage)
		"",                                    // Long description
		&Exit{})                               // Command implementation
	registerGroup(err, ex, constants.CoreServerGroup)

	hl, err := parser.AddCommand(constants.HelpStr,
		"Print commands help for the current menu", "",
		&Help{})
	registerGroup(err, hl, constants.CoreServerGroup)

	v, err := parser.AddCommand(constants.VersionStr,
		"Display version information",
		help.GetHelpFor(constants.VersionStr),
		&Version{})
	registerGroup(err, v, constants.CoreServerGroup)

	li, err := parser.AddCommand(constants.LicensesStr,
		"Display project licenses (core & libraries)", "",
		&Licenses{})
	registerGroup(err, li, constants.CoreServerGroup)

	up, err := parser.AddCommand(constants.UpdateStr,
		"Check for newer Sliver console/server releases",
		help.GetHelpFor(constants.UpdateStr),
		&Updates{})
	registerGroup(err, up, constants.CoreServerGroup)

	op, err := parser.AddCommand(constants.PlayersStr,
		"List operators and their status",
		help.GetHelpFor(constants.PlayersStr),
		&Operators{})
	registerGroup(err, op, constants.CoreServerGroup)

	// Console configuration management
	conf, err := parser.AddCommand(constants.ConfigStr,
		"Console configuration commands", "",
		&Config{})
	registerGroup(err, conf, constants.CoreServerGroup)

	if conf != nil {
		conf.SubcommandsOptional = true

		_, err = conf.AddCommand(constants.ConfigSaveStr,
			"Save the current console configuration, to be used by all user clients", "",
			&SaveConfig{})
		registerGroup(err, nil, constants.CoreServerGroup)

		_, err = conf.AddCommand(constants.ConfigPromptServerStr,
			"Set the server context right/left prompt (items/colors completed)", "",
			&PromptServer{})
		registerGroup(err, nil, constants.CoreServerGroup)

		_, err = conf.AddCommand(constants.ConfigPromptSliverStr,
			"Set the sliver context right/left prompt (items/colors completed)", "",
			&PromptSliver{})
		registerGroup(err, nil, constants.CoreServerGroup)

		_, err = conf.AddCommand(constants.ConfigHintsStr,
			"Show/hide console hints", "",
			&Hints{})
		registerGroup(err, nil, constants.CoreServerGroup)

		_, err = conf.AddCommand(constants.ConfigVimStr,
			"Set the console input mode to Vim editing mode", "",
			&Vim{})
		registerGroup(err, nil, constants.CoreServerGroup)

		_, err = conf.AddCommand(constants.ConfigEmacsStr,
			"Set the console input mode to Emacs editing mode", "",
			&Emacs{})
		registerGroup(err, nil, constants.CoreServerGroup)
	}

	// Log
	log, err := parser.AddCommand(constants.LogStr,
		"Manage log levels of one or more components",
		"",
		&Log{})
	registerGroup(err, log, constants.CoreServerGroup)

	// Jobs
	j, err := parser.AddCommand(constants.JobsStr,
		"Job management commands",
		help.GetHelpFor(constants.JobsStr),
		&Jobs{})
	registerGroup(err, j, constants.CoreServerGroup)

	if j != nil {
		j.SubcommandsOptional = true

		_, err = j.AddCommand(constants.JobsKillStr,
			"Kill one or more jobs given their ID",
			"",
			&JobsKill{})
		registerGroup(err, nil, constants.CoreServerGroup)

		_, err = j.AddCommand(constants.JobsKillAllStr,
			"Kill all active jobs on server",
			"",
			&JobsKillAll{})
		registerGroup(err, nil, constants.CoreServerGroup)
	}

	// Session Management ----------------------------------------------------------------------------
	i, err := parser.AddCommand(constants.UseStr,
		"Interact with an implant",
		help.GetHelpFor(constants.UseStr),
		&Interact{})
	registerGroup(err, i, constants.SessionsGroup)

	se, err := parser.AddCommand(constants.SessionsStr,
		"Session management (all contexts)",
		help.GetHelpFor(constants.SessionsStr),
		&Sessions{})
	registerGroup(err, se, constants.SessionsGroup)

	if se != nil {
		se.SubcommandsOptional = true

		_, err = se.AddCommand(constants.KillStr,
			"Kill one or more implant sessions", "",
			&SessionsKill{})
		registerGroup(err, nil, constants.SessionsGroup)

		_, err = se.AddCommand(constants.JobsKillAllStr,
			"Kill all registered sessions", "",
			&SessionsKillAll{})
		registerGroup(err, nil, constants.SessionsGroup)

		_, err = se.AddCommand("clean",
			"Clean sessions marked Dead", "",
			&SessionsClean{})
		registerGroup(err, nil, constants.SessionsGroup)
	}

	// Implant generation ----------------------------------------------------------------------------
	g, err := parser.AddCommand(constants.GenerateStr,
		"Configure and compile an implant (staged or stager)",
		help.GetHelpFor(constants.GenerateStr), &Generate{})
	registerGroup(err, g, constants.BuildsGroup)

	if g != nil {
		g.SubcommandsOptional = true

		_, err = g.AddCommand(constants.StagerStr,
			"Generate a stager shellcode payload using MSFVenom, (to file: --save, to stdout: --format",
			help.GetHelpFor(constants.StagerStr),
			&GenerateStager{})
		registerGroup(err, nil, constants.BuildsGroup)
	}

	p, err := parser.AddCommand(constants.NewProfileStr,
		"Configure and save a new (stage) implant profile",
		help.GetHelpFor(constants.NewProfileStr),
		&NewProfile{})
	registerGroup(err, p, constants.BuildsGroup)

	r, err := parser.AddCommand(constants.RegenerateStr,
		"Recompile an implant by name, passed as argument (completed)",
		help.GetHelpFor(constants.RegenerateStr),
		&Regenerate{})
	registerGroup(err, r, constants.BuildsGroup)

	pr, err := parser.AddCommand(constants.ProfilesStr,
		"List existing implant profiles",
		help.GetHelpFor(constants.ProfilesStr), &Profiles{})
	registerGroup(err, pr, constants.BuildsGroup)

	if pr != nil {
		pr.SubcommandsOptional = true

		_, err = pr.AddCommand(constants.ProfilesDeleteStr,
			"Delete one or more existing implant profiles", "",
			&ProfileDelete{})
		registerGroup(err, nil, constants.BuildsGroup)
	}

	pg, err := parser.AddCommand(constants.ProfileGenerateStr,
		"Compile an implant based on a profile, passed as argument (completed)",
		help.GetHelpFor(constants.ProfileGenerateStr),
		&ProfileGenerate{})
	registerGroup(err, pg, constants.BuildsGroup)

	b, err := parser.AddCommand(constants.ImplantBuildsStr,
		"List old implant builds",
		help.GetHelpFor(constants.ImplantBuildsStr),
		&Builds{})
	registerGroup(err, b, constants.BuildsGroup)

	c, err := parser.AddCommand(constants.ListCanariesStr,
		"List previously generated DNS canaries",
		help.GetHelpFor(constants.ListCanariesStr),
		&Canaries{})
	registerGroup(err, c, constants.BuildsGroup)

	// Port forwarders ----------------------------------------------------------------------------
	pf, err := parser.AddCommand(constants.PortfwdStr,
		"Manage port forwarders for sessions, or the active one",
		"", &Portfwd{})
	registerGroup(err, pf, constants.CommGroup)

	if pf != nil {
		pf.SubcommandsOptional = true

		_, err = pf.AddCommand(constants.PortfwdOpenStr,
			"Start a new port forwarder for the active session, or by specifying a session ID", "",
			&PortfwdOpen{})
		registerGroup(err, nil, constants.CommGroup)

		_, err = pf.AddCommand(constants.PortfwdCloseStr,
			"Close one or more port forwarders, for the active session or all, with filters", "",
			&PortfwdClose{})
		registerGroup(err, nil, constants.CommGroup)
	}

	// ----
	// 2 - Commands changing name when in different context

	switch cctx.Context.Menu {
	case cctx.Server:
		cd, err := parser.AddCommand(constants.CdStr,
			"Change client working directory",
			"",
			&ChangeClientDirectory{})
		registerGroup(err, cd, constants.CoreServerGroup)

		ls, err := parser.AddCommand(constants.LsStr,
			"List directory contents",
			"",
			&ListClientDirectories{})
		registerGroup(err, ls, constants.CoreServerGroup)

		// The info command is a session management one
		// in server context, but a core one in session context.
		info, err := parser.AddCommand(constants.InfoStr,
			"Show session information", "",
			&sliver.Info{})
		registerGroup(err, info, constants.SessionsGroup)

	case cctx.Sliver:
		lcd, err := parser.AddCommand(constants.LcdStr,
			"Change the client working directory", "",
			&ChangeClientDirectory{})
		registerGroup(err, lcd, constants.CoreServerGroup)
	}

	return
}
