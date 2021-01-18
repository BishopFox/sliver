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
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"

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
	// Server - All commands available in the main (server) menu are processed
	// by this Server parser. This is the basis of context separation for various
	// completions, hints, prompt system, etc.
	Server = flags.NewNamedParser("server", flags.IgnoreUnknown)

	// Sliver - The parser used to process all commands directed at sliver implants.
	Sliver = flags.NewNamedParser("sliver", flags.None)
)

// BindCommands - Binds all commands to their appropriate parsers, which have been instantiated already.
func BindCommands(admin bool) (err error) {

	Server = flags.NewNamedParser("server", flags.IgnoreUnknown)
	if admin {
		err = bindServerAdminCommands()
		if err != nil {
			return
		}
	}
	err = bindServerCommands()
	if err != nil {
		return
	}

	Sliver = flags.NewNamedParser("sliver", flags.None)
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
func OptionByName(cmd *flags.Command, option string) *flags.Option {

	if cmd == nil {
		return nil
	}
	// Get all (root) option groups.
	groups := cmd.Groups()

	// For each group, build completions
	for _, grp := range groups {
		// Add each option to completion group
		for _, opt := range grp.Options() {
			if opt.LongName == option {
				return opt
			}
		}
	}
	return nil
}

// bindServerAdminCommands - We bind commands only available to the server admin to the console command parser.
// Unfortunately we have to use, for each command, its Aliases field where we register its "namespace".
// There is a namespace field, however it messes up with the option printing/detection/parsing.
func bindServerAdminCommands() (err error) {

	np, err := Server.AddCommand(constants.NewPlayerStr, "Create a new player config file",
		help.GetHelpFor(constants.NewPlayerStr), &NewOperator{})
	np.Aliases = []string{"admin"}
	if err != nil {
		fmt.Println(util.Warn + err.Error())
		os.Exit(3)
	}

	kp, err := Server.AddCommand(constants.KickPlayerStr, "Kick a player from the server",
		help.GetHelpFor(constants.KickPlayerStr), &KickOperator{})
	kp.Aliases = []string{"admin"}
	if err != nil {
		fmt.Println(util.Warn + err.Error())
		os.Exit(3)
	}

	mm, err := Server.AddCommand(constants.MultiplayerModeStr, "Enable multiplayer mode on this server",
		help.GetHelpFor(constants.MultiplayerModeStr), &MultiplayerMode{})
	mm.Aliases = []string{"admin"}
	if err != nil {
		fmt.Println(util.Warn + err.Error())
		os.Exit(3)
	}

	return
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
		"Change client working directory", &ChangeClientDirectory{})
	cd.Aliases = []string{"core"}

	ls, err := Server.AddCommand(constants.LsStr, "List directory contents",
		"List directory contents", &ListClientDirectories{})
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
	// Option arguments mapping
	g.FindOptionByLongName("os").Choices = implantOS
	g.FindOptionByLongName("arch").Choices = implantArch
	g.FindOptionByLongName("format").Choices = implantFmt

	gs, err := g.AddCommand(constants.StagerStr, "Generate a stager payload using MSFVenom",
		help.GetHelpFor(constants.StagerStr), &GenerateStager{})
	g.FindOptionByLongName("os").Choices = implantOS
	g.FindOptionByLongName("arch").Choices = implantArch
	gs.FindOptionByLongName("format").Choices = msfTransformFormats

	p, err := Server.AddCommand(constants.NewProfileStr, "Configure and save a new (stage) implant profile",
		help.GetHelpFor(constants.NewProfileStr), &NewProfile{})
	p.Aliases = []string{"implants"}
	// Option arguments mapping
	p.FindOptionByLongName("os").Choices = implantOS
	p.FindOptionByLongName("arch").Choices = implantArch

	r, err := Server.AddCommand(constants.RegenerateStr, "Recompile an implant by name, passed as argument (completed)",
		help.GetHelpFor(constants.RegenerateStr), &Regenerate{})
	r.Aliases = []string{"implants"}

	pr, err := Server.AddCommand(constants.ProfilesStr, "List existing implant profiles",
		help.GetHelpFor(constants.ProfilesStr), &Profiles{})
	pr.Aliases = []string{"implants"}

	pg, err := Server.AddCommand(constants.ProfileGenerateStr, "Compile an implant based on a profile, passed as argument (completed)",
		help.GetHelpFor(constants.ProfileGenerateStr), &ProfileGenerate{})
	pg.Aliases = []string{"implants"}

	b, err := Server.AddCommand(constants.ImplantBuildsStr, "List old implant builds",
		help.GetHelpFor(constants.ImplantBuildsStr), &Builds{})
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

	// Filesystem
	cd, err := Sliver.AddCommand(constants.CdStr, "Change session working directory",
		"Change session working directory", &ChangeDirectory{})
	cd.Aliases = []string{"filesystem"}

	return
}
