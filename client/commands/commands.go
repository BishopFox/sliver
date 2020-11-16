package commands

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

	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/context"
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

var (
	// Server - All commands available in the main (server) menu are processed
	// by this Server parser. This is the basis of context separation for various
	// completions, hints, prompt system, etc.
	Server = flags.NewNamedParser("server", flags.IgnoreUnknown)

	// Sliver - The parser used to process all commands directed at sliver implants.
	Sliver = flags.NewNamedParser("sliver", flags.None)
)

// BindCommands - Binds all commands to their appropriate parsers, which have been instantiated already.
func BindCommands() (err error) {
	err = bindServerCommands()
	if err != nil {
		return
	}
	err = bindSliverCommands()
	if err != nil {
		return
	}

	return
}

// ArgumentByName Get the name of a detected command's argument
func ArgumentByName(command *flags.Command, name string) *flags.Arg {
	args := command.Args()
	for _, arg := range args {
		if arg.Name == name {
			return arg
		}
	}

	// Maybe we can check for aliases, later...
	// Might sometimes push interesting things...

	return nil
}

// OptionByName - Returns an option for a command or a subcommand, identified by name
func OptionByName(ctx string, command, subCommand, option string) *flags.Option {

	var cmd *flags.Command

	switch ctx {
	case context.Server:
		cmd = Server.Find(command)
	case context.Sliver:
		cmd = Sliver.Find(command)
	}

	// Base command is found
	if cmd != nil {
		// If options are for a subcommand
		if subCommand != "" && len(cmd.Commands()) != 0 {
			sub := cmd.Find(subCommand)
			if sub != nil {
				for _, opt := range sub.Options() {
					if opt.LongName == option {
						return opt
					}
				}
				return nil
			}
			return nil
		}
		// If subcommand is not asked, return opt for base
		for _, opt := range cmd.Options() {
			if opt.LongName == option {
				return opt
			}
		}
	}
	return nil
}

// All commands concerning the server and/or the console itself are bound in this function.
// Unfortunately we have to use, for each command, its Aliases field where we register its "namespace".
// There is a namespace field, however it messes up with the option printing/detection/parsing.
func bindServerCommands() (err error) {

	// core console -------------------------
	ex, err := Server.AddCommand(constants.ExitStr, "Exit from the client/server console",
		"Exit from the client/server console", &Exit{})
	ex.Aliases = []string{"core"}

	v, err := Server.AddCommand(constants.VersionStr, "Display version information",
		help.GetHelpFor(constants.VersionStr), &Version{})
	v.Aliases = []string{"core"}

	up, err := Server.AddCommand(constants.UpdateStr, "Check for newer Sliver console/server releases",
		help.GetHelpFor(constants.UpdateStr), &Updates{})
	up.Aliases = []string{"core"}

	op, err := Server.AddCommand(constants.PlayersStr, "List operators and their status",
		help.GetHelpFor(constants.PlayersStr), &Operators{})
	op.Aliases = []string{"core"}

	cd, err := Server.AddCommand(constants.CdStr, "Change client working directory",
		"Change client working directory", &ChangeDirectory{})
	cd.Aliases = []string{"core"}

	ls, err := Server.AddCommand(constants.LsStr, "List directory contents",
		"List directory contents", &ListDirectories{})
	ls.Aliases = []string{"core"}

	// Jobs -------------------------
	j, err := Server.AddCommand(constants.JobsStr, "Job management commands",
		help.GetHelpFor(constants.JobsStr), &Jobs{})
	j.Aliases = []string{"core"}
	j.SubcommandsOptional = true

	_, err = j.AddCommand(constants.JobsKillStr, "Kill job given an ID",
		"", &JobsKill{})

	_, err = j.AddCommand(constants.JobsKillAllStr, "Kill all active jobs on server",
		"", &JobsKillAll{})

	// transports --------------------
	m, err := Server.AddCommand(constants.MtlsStr, "Start an mTLS listener on server",
		help.GetHelpFor(constants.MtlsStr), &MTLSListener{})
	m.Aliases = []string{"transports"}

	d, err := Server.AddCommand(constants.DnsStr, "Start a DNS listener",
		help.GetHelpFor(constants.DnsStr), &DNSListener{})
	d.Aliases = []string{"transports"}

	hs, err := Server.AddCommand(constants.HttpsStr, "Start an HTTP(S) listener",
		help.GetHelpFor(constants.HttpsStr), &HTTPSListener{})
	hs.Aliases = []string{"transports"}

	h, err := Server.AddCommand(constants.HttpStr, "Start an HTTP listener",
		help.GetHelpFor(constants.HttpStr), &HTTPListener{})
	h.Aliases = []string{"transports"}

	s, err := Server.AddCommand(constants.StagerStr, "Start a staging listener (TCP/HTTP/HTTPS)",
		help.GetHelpFor(constants.StagerStr), &StageListener{})
	s.Aliases = []string{"transports"}

	// Implant generation --------------
	g, err := Server.AddCommand(constants.GenerateStr, "Configure and compile an implant (staged or stager)",
		help.GetHelpFor(constants.GenerateStr), &Generate{})
	g.Aliases = []string{"implants"}
	g.SubcommandsOptional = true

	_, err = g.AddCommand(constants.StagerStr, "Generate a stager payload using MSFVenom",
		help.GetHelpFor(constants.StagerStr), &GenerateStager{})

	p, err := Server.AddCommand(constants.NewProfileStr, "Configure and save a new (stage) implant profile",
		help.GetHelpFor(constants.NewProfileStr), &NewProfile{})
	p.Aliases = []string{"implants"}

	r, err := Server.AddCommand(constants.RegenerateStr, "Recompile an implant by name, passed as argument (completed)",
		help.GetHelpFor(constants.RegenerateStr), &Regenerate{})
	r.Aliases = []string{"implants"}

	pr, err := Server.AddCommand(constants.ProfilesStr, "List existing implant profiles",
		help.GetHelpFor(constants.ProfilesStr), &Profiles{})
	pr.Aliases = []string{"implants"}

	pg, err := Server.AddCommand(constants.ProfileGenerateStr, "Compile an implant based on a profile, passed as argument (completed)",
		help.GetHelpFor(constants.ProfileGenerateStr), &ProfileGenerate{})
	pg.Aliases = []string{"implants"}

	b, err := Server.AddCommand(constants.ListSliverBuildsStr, "List old implant builds",
		help.GetHelpFor(constants.ListSliverBuildsStr), &Builds{})
	b.Aliases = []string{"implants"}

	c, err := Server.AddCommand(constants.ListCanariesStr, "List previously generated DNS canaries",
		help.GetHelpFor(constants.ListCanariesStr), &Canaries{})
	c.Aliases = []string{"implants"}

	// Session management ---------------
	i, err := Server.AddCommand(constants.InteractStr, "Interact with an implant",
		help.GetHelpFor(constants.InteractStr), &Interact{})
	i.Aliases = []string{"sessions"}

	return
}

// All commands for controlling sliver implants are bound in this function.
func bindSliverCommands() (err error) {

	// Session management
	i, err := Sliver.AddCommand(constants.InteractStr, "Interact with an implant",
		help.GetHelpFor(constants.InteractStr), &Interact{})
	i.Aliases = []string{"session"}

	b, err := Sliver.AddCommand(constants.BackgroundStr, "Background an active session",
		help.GetHelpFor(constants.BackgroundStr), &Background{})
	b.Aliases = []string{"session"}

	k, err := Sliver.AddCommand(constants.KillStr, "Kill this session",
		help.GetHelpFor(constants.KillStr), &Kill{})
	k.Aliases = []string{"session"}
	return
}
