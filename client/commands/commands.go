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
	"context"
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/bishopfox/sliver/client/constants"
	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/help"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
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
	Server = flags.NewNamedParser("server", flags.HelpFlag)

	// Sliver - The parser used to process all commands directed at sliver implants.
	Sliver = flags.NewNamedParser("sliver", flags.HelpFlag)
)

// BindCommands - Binds all commands to their appropriate parsers, which have been instantiated already.
// The parser is also passed to the completers package function which registers additional completion
// choices to some commands, hereby also adding a control layer on input values.
func BindCommands(admin bool, completions func(parser *flags.Parser)) (err error) {

	switch cctx.Context.Menu {

	// Server commands
	case cctx.Server:
		Server = flags.NewNamedParser("server", flags.HelpFlag)

		// Stack up parsing options :
		// 1 - Add help options to all commands
		// 2 - Ignore unknown options (some commands needs args that are flags, ex: sideload)
		Server.Options = flags.IgnoreUnknown | flags.HelpFlag

		// If this client console is the server itself, bind admin commands.
		if admin {
			err = bindServerAdminCommands()
			if err != nil {
				return
			}
		}

		// Bind normal commands
		err = bindServerCommands()
		if err != nil {
			return
		}

		// Register additional completions
		completions(Server)

	// Session commands, with per-OS filtering
	case cctx.Sliver:
		Sliver = flags.NewNamedParser("sliver", flags.HelpFlag)

		// Stack up parsing options :
		// 1 - Add help options to all commands
		// 2 - Ignore unknown options (some commands needs args that are flags, ex: sideload)
		Sliver.Options = flags.IgnoreUnknown | flags.HelpFlag

		// Base commands apply to all sessions
		err = bindSliverCommands()
		if err != nil {
			return
		}

		// If session is Windows, bind Windows commands
		if cctx.Context.Sliver.OS == "windows" {
			err = bindWindowsCommands()
			if err != nil {
				return
			}
		}

		// Bind previously loaded extensions.
		for _, extensionBind := range LoadedExtensions {
			extensionBind()
		}

		// Register additional completions
		completions(Sliver)
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

// GetSession - Get session by session ID or name
func GetSession(arg string) *clientpb.Session {
	sessions, err := transport.RPC.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
		return nil
	}
	for _, session := range sessions.GetSessions() {
		if fmt.Sprintf("%d", session.ID) == arg {
			return session
		}
	}
	return nil
}

// ContextRequest - Forge a Request Protobuf metadata to be sent in a RPC request.
func ContextRequest(sess *clientpb.Session) (req *commonpb.Request) {
	req = &commonpb.Request{}
	if sess == nil {
		return req
	}
	req.SessionID = sess.ID

	// Get command timeout option flag
	req.Timeout = 60

	return
}

// This should be called for any dangerous (OPSEC-wise) functions
func isUserAnAdult() bool {
	confirm := false
	prompt := &survey.Confirm{Message: "This action is bad OPSEC, are you an adult?"}
	survey.AskOne(prompt, &confirm, nil)
	return confirm
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

	// core console
	// ----------------------------------------------------------------------------------------
	ex, err := Server.AddCommand(constants.ExitStr, // Command string
		"Exit from the client/server console", // Description (completions, help usage)
		"",                                    // Long description
		&Exit{})                               // Command implementation
	ex.Aliases = []string{"core"}

	v, err := Server.AddCommand(constants.VersionStr,
		"Display version information",
		help.GetHelpFor(constants.VersionStr),
		&Version{})
	v.Aliases = []string{"core"}

	li, err := Server.AddCommand(constants.LicensesStr,
		"Display project licenses (core & libraries)", "",
		&Licenses{})
	li.Aliases = []string{"core"}

	up, err := Server.AddCommand(constants.UpdateStr,
		"Check for newer Sliver console/server releases",
		help.GetHelpFor(constants.UpdateStr),
		&Updates{})
	up.Aliases = []string{"core"}

	op, err := Server.AddCommand(constants.PlayersStr,
		"List operators and their status",
		help.GetHelpFor(constants.PlayersStr),
		&Operators{})
	op.Aliases = []string{"core"}

	cd, err := Server.AddCommand(constants.CdStr,
		"Change client working directory",
		"",
		&ChangeClientDirectory{})
	cd.Aliases = []string{"core"}

	ls, err := Server.AddCommand(constants.LsStr,
		"List directory contents",
		"",
		&ListClientDirectories{})
	ls.Aliases = []string{"core"}

	// Jobs
	j, err := Server.AddCommand(constants.JobsStr,
		"Job management commands",
		help.GetHelpFor(constants.JobsStr),
		&Jobs{})
	j.Aliases = []string{"core"}
	j.SubcommandsOptional = true

	_, err = j.AddCommand(constants.JobsKillStr,
		"Kill one or more jobs given their ID",
		"",
		&JobsKill{})

	_, err = j.AddCommand(constants.JobsKillAllStr,
		"Kill all active jobs on server",
		"",
		&JobsKillAll{})

	// Log
	log, err := Server.AddCommand(constants.LogStr,
		"Manage log levels of one or more components",
		"",
		&Log{})
	log.Aliases = []string{"core"}

	// transports
	// ----------------------------------------------------------------------------------------
	m, err := Server.AddCommand(constants.MtlsStr,
		"Start an mTLS listener on the server, or on a routed session",
		help.GetHelpFor(constants.MtlsStr),
		&MTLSListener{})
	m.Aliases = []string{"transports"}

	d, err := Server.AddCommand(constants.DnsStr,
		"Start a DNS listener on the server",
		help.GetHelpFor(constants.DnsStr),
		&DNSListener{})
	d.Aliases = []string{"transports"}

	hs, err := Server.AddCommand(constants.HttpsStr,
		"Start an HTTP(S) listener on the server",
		help.GetHelpFor(constants.HttpsStr),
		&HTTPSListener{})
	hs.Aliases = []string{"transports"}

	h, err := Server.AddCommand(constants.HttpStr,
		"Start an HTTP listener on the server",
		help.GetHelpFor(constants.HttpStr),
		&HTTPListener{})
	h.Aliases = []string{"transports"}

	s, err := Server.AddCommand(constants.StageListenerStr,
		"Start a staging listener (TCP/HTTP/HTTPS), bound to a Sliver profile",
		help.GetHelpFor(constants.StageListenerStr), &StageListener{})
	s.Aliases = []string{"transports"}

	ws, err := Server.AddCommand(constants.WebsitesStr,
		"Manage websites (used with HTTP C2) (prints website name argument by default)", "",
		&Websites{})
	ws.Aliases = []string{"transports"}
	ws.SubcommandsOptional = true

	_, err = ws.AddCommand(constants.RmStr,
		"Remove an entire website", "",
		&WebsitesDelete{})
	_, err = ws.AddCommand(constants.AddWebContentStr,
		"Add content to a website", "",
		&WebsitesAddContent{})
	_, err = ws.AddCommand(constants.RmWebContentStr,
		"Remove content from a website", "",
		&WebsitesDeleteContent{})
	_, err = ws.AddCommand(constants.WebUpdateStr,
		"Update a website's content type", "",
		&WebsiteType{})

	// Implant generation
	// ----------------------------------------------------------------------------------------
	g, err := Server.AddCommand(constants.GenerateStr,
		"Configure and compile an implant (staged or stager)",
		help.GetHelpFor(constants.GenerateStr), &Generate{})
	g.Aliases = []string{"builds"}
	g.SubcommandsOptional = true

	_, err = g.AddCommand(constants.StagerStr,
		"Generate a stager shellcode payload using MSFVenom, (to file: --save, to stdout: --format",
		help.GetHelpFor(constants.StagerStr),
		&GenerateStager{})

	p, err := Server.AddCommand(constants.NewProfileStr,
		"Configure and save a new (stage) implant profile",
		help.GetHelpFor(constants.NewProfileStr),
		&NewProfile{})
	p.Aliases = []string{"builds"}

	r, err := Server.AddCommand(constants.RegenerateStr,
		"Recompile an implant by name, passed as argument (completed)",
		help.GetHelpFor(constants.RegenerateStr),
		&Regenerate{})
	r.Aliases = []string{"builds"}

	pr, err := Server.AddCommand(constants.ProfilesStr,
		"List existing implant profiles",
		help.GetHelpFor(constants.ProfilesStr), &Profiles{})
	pr.Aliases = []string{"builds"}
	pr.SubcommandsOptional = true

	_, err = pr.AddCommand(constants.ProfilesDeleteStr,
		"Delete one or more existing implant profiles", "",
		&ProfileDelete{})

	pg, err := Server.AddCommand(constants.ProfileGenerateStr,
		"Compile an implant based on a profile, passed as argument (completed)",
		help.GetHelpFor(constants.ProfileGenerateStr),
		&ProfileGenerate{})
	pg.Aliases = []string{"builds"}

	b, err := Server.AddCommand(constants.ImplantBuildsStr,
		"List old implant builds",
		help.GetHelpFor(constants.ImplantBuildsStr),
		&Builds{})
	b.Aliases = []string{"builds"}

	c, err := Server.AddCommand(constants.ListCanariesStr,
		"List previously generated DNS canaries",
		help.GetHelpFor(constants.ListCanariesStr),
		&Canaries{})
	c.Aliases = []string{"builds"}

	// Session management
	// ----------------------------------------------------------------------------------------
	i, err := Server.AddCommand(constants.UseStr,
		"Interact with an implant",
		help.GetHelpFor(constants.UseStr),
		&Interact{})
	i.Aliases = []string{"slivers"}

	se, err := Server.AddCommand(constants.SessionsStr,
		"Session management (all contexts)",
		help.GetHelpFor(constants.SessionsStr),
		&Sessions{})
	se.Aliases = []string{"slivers"}
	se.SubcommandsOptional = true

	_, err = se.AddCommand(constants.KillStr,
		"Kill one or more implant sessions", "",
		&SessionsKill{})
	_, err = se.AddCommand(constants.JobsKillAllStr,
		"Kill all registered sessions", "",
		&SessionsKillAll{})
	_, err = se.AddCommand("clean",
		"Clean sessions marked Dead", "",
		&SessionsClean{})

	info, err := Server.AddCommand(constants.InfoStr,
		"Show session information", "",
		&Info{})
	info.Aliases = []string{"slivers"}

	// Comm system
	// ----------------------------------------------------------------------------------------
	// Port forwarders
	pf, err := Server.AddCommand(constants.PortfwdStr,
		"Manage port forwarders for sessions, or the active one",
		"", &Portfwd{})
	pf.Aliases = []string{"comm"}
	pf.SubcommandsOptional = true

	_, err = pf.AddCommand(constants.PortfwdOpenStr,
		"Start a new port forwarder for the active session, or by specifying a session ID", "",
		&PortfwdOpen{})
	_, err = pf.AddCommand(constants.PortfwdCloseStr,
		"Close one or more port forwarders, for the active session or all, with filters", "",
		&PortfwdClose{})

	// Network Routes
	rt, err := Server.AddCommand(constants.RouteStr,
		"Manage network routes (prints them by default)", "",
		&Route{})
	rt.Aliases = []string{"comm"}
	rt.SubcommandsOptional = true

	_, err = rt.AddCommand(constants.RouteAddStr,
		"Add a network route (routes client proxies and C2 handlers)", "",
		&RouteAdd{})
	_, err = rt.AddCommand(constants.RouteRemoveStr,
		"Remove one or more network routes", "",
		&RouteRemove{})

	return
}

// All commands for controlling sliver implants are bound in this function.
func bindSliverCommands() (err error) {

	// Core
	// ----------------------------------------------------------------------------------------

	b, err := Sliver.AddCommand(constants.BackgroundStr,
		"Background the current session",
		help.GetHelpFor(constants.BackgroundStr),
		&Background{})
	b.Aliases = []string{"core"}

	k, err := Sliver.AddCommand(constants.KillStr,
		"Kill the current session",
		help.GetHelpFor(constants.KillStr),
		&Kill{})
	k.Aliases = []string{"core"}

	i, err := Sliver.AddCommand(constants.UseStr,
		"Interact with an implant",
		help.GetHelpFor(constants.UseStr),
		&Interact{})
	i.Aliases = []string{"core"}

	se, err := Sliver.AddCommand(constants.SessionsStr,
		"Session management (all contexts)",
		help.GetHelpFor(constants.SessionsStr),
		&Sessions{})
	se.Aliases = []string{"core"}
	se.SubcommandsOptional = true

	_, err = se.AddCommand(constants.KillStr,
		"Kill one or more implant sessions", "",
		&SessionsKill{})
	_, err = se.AddCommand(constants.JobsKillAllStr,
		"Kill all registered sessions", "",
		&SessionsKillAll{})
	_, err = se.AddCommand("clean",
		"Clean sessions marked Dead", "",
		&SessionsClean{})

	lcd, err := Sliver.AddCommand(constants.LcdStr,
		"Change the client working directory", "",
		&ChangeClientDirectory{})
	lcd.Aliases = []string{"core"}

	log, err := Sliver.AddCommand(constants.LogStr,
		"Manage log levels of one or more components", "",
		&Log{})
	log.Aliases = []string{"core"}

	set, err := Sliver.AddCommand(constants.SetStr,
		"Set a value for the current session", "",
		&Set{})
	set.Aliases = []string{"core"}

	env, err := Sliver.AddCommand(constants.GetEnvStr,
		"Get one or more host environment variables", "",
		&SessionEnv{})
	env.Aliases = []string{"core"}

	ping, err := Sliver.AddCommand(constants.PingStr,
		"Send round trip message to implant (does not use ICMP)", "",
		&Ping{})
	ping.Aliases = []string{"core"}

	sh, err := Sliver.AddCommand(constants.ShellStr,
		"Start an interactive shell on the session host (not opsec!)", "",
		&Shell{})
	sh.Aliases = []string{"core"}

	ext, err := Sliver.AddCommand(constants.LoadExtensionStr,
		"Load an extension through the current Sliver session", "",
		&LoadExtension{})
	ext.Aliases = []string{"core"}

	// Info
	// ----------------------------------------------------------------------------------------
	info, err := Sliver.AddCommand(constants.InfoStr,
		"Show session information", "",
		&Info{})
	info.Aliases = []string{"info"}
	uid, err := Sliver.AddCommand(constants.GetUIDStr,
		"Get session User ID", "",
		&UID{})
	uid.Aliases = []string{"info"}

	gid, err := Sliver.AddCommand(constants.GetGIDStr,
		"Get session User group ID", "",
		&GID{})
	gid.Aliases = []string{"info"}

	pid, err := Sliver.AddCommand(constants.GetPIDStr,
		"Get session Process ID", "",
		&PID{})
	pid.Aliases = []string{"info"}

	w, err := Sliver.AddCommand(constants.WhoamiStr,
		"Get session username", "",
		&Whoami{})
	w.Aliases = []string{"info"}

	sc, err := Sliver.AddCommand(constants.ScreenshotStr,
		"Take a screenshot", "",
		&Screenshot{})
	sc.Aliases = []string{"info"}

	// Filesystem
	// ----------------------------------------------------------------------------------------
	cd, err := Sliver.AddCommand(constants.CdStr,
		"Change session working directory", "",
		&ChangeDirectory{})
	cd.Aliases = []string{"filesystem"}

	ls, err := Sliver.AddCommand(constants.LsStr,
		"List session directory contents", "",
		&ListSessionDirectories{})
	ls.Aliases = []string{"filesystem"}

	rm, err := Sliver.AddCommand(constants.RmStr,
		"Remove directory/file contents from the session's host", "",
		&Rm{})
	rm.Aliases = []string{"filesystem"}

	mkd, err := Sliver.AddCommand(constants.MkdirStr,
		"Create one or more directories on the implant's host", "",
		&Mkdir{})
	mkd.Aliases = []string{"filesystem"}

	pwd, err := Sliver.AddCommand(constants.PwdStr,
		"Print the session current working directory", "",
		&Pwd{})
	pwd.Aliases = []string{"filesystem"}

	cat, err := Sliver.AddCommand(constants.CatStr,
		"Print one or more files to screen", "",
		&Cat{})
	cat.Aliases = []string{"filesystem"}

	dl, err := Sliver.AddCommand(constants.DownloadStr,
		"Download one or more files from the target to the client", "",
		&Download{})
	dl.Aliases = []string{"filesystem"}

	ul, err := Sliver.AddCommand(constants.UploadStr,
		"Upload one or more files from the client to the target filesystem", "",
		&Upload{})
	ul.Aliases = []string{"filesystem"}

	// Comm & Network
	// ----------------------------------------------------------------------------------------
	ifc, err := Sliver.AddCommand(constants.IfconfigStr,
		"Show session network interfaces", "",
		&Ifconfig{})
	ifc.Aliases = []string{"comm"}

	ns, err := Sliver.AddCommand(constants.NetstatStr,
		"Print network connection information", "",
		&Netstat{})
	ns.Aliases = []string{"comm"}

	pf, err := Sliver.AddCommand(constants.PortfwdStr,
		"Manage port forwarders for sessions, or the active one", "",
		&Portfwd{})
	pf.Aliases = []string{"comm"}
	pf.SubcommandsOptional = true

	_, err = pf.AddCommand(constants.PortfwdOpenStr,
		"Start a new port forwarder for the active session, or by specifying a session ID", "",
		&PortfwdOpen{})
	_, err = pf.AddCommand(constants.PortfwdCloseStr,
		"Close one or more port forwarders, for the active session or all, with filters", "",
		&PortfwdClose{})

	rt, err := Sliver.AddCommand(constants.RouteStr,
		"Manage network routes (prints them by default)", "",
		&Route{})
	rt.Aliases = []string{"comm"}
	rt.SubcommandsOptional = true

	// transports
	// ----------------------------------------------------------------------------------------
	tp, err := Sliver.AddCommand(constants.TCPListenerStr,
		"Start a TCP pivot listener (unencrypted!)", "",
		&TCPPivot{})
	tp.Aliases = []string{"transports"}

	// Proc
	// ----------------------------------------------------------------------------------------
	ps, err := Sliver.AddCommand(constants.PsStr,
		"List host processes", "",
		&PS{})
	ps.Aliases = []string{"process"}

	procDump, err := Sliver.AddCommand(constants.ProcdumpStr,
		"Dump process memory (process ID argument, or options)", "",
		&ProcDump{})
	procDump.Aliases = []string{"process"}

	term, err := Sliver.AddCommand(constants.TerminateStr,
		"Kill/terminate one or more running host processes", "",
		&Terminate{})
	term.Aliases = []string{"process"}

	// Execution
	// ----------------------------------------------------------------------------------------
	exec, err := Sliver.AddCommand(constants.ExecuteStr,
		"Execute a program on the remote system", "",
		&Execute{})
	exec.Aliases = []string{"execution"}

	msf, err := Sliver.AddCommand(constants.MsfStr,
		"Execute an MSF payload in the current process", "",
		&MSF{})
	msf.Aliases = []string{"execution"}

	msfi, err := Sliver.AddCommand(constants.MsfInjectStr,
		"Inject an MSF payload into a process (ID as argument)", "",
		&MSFInject{})
	msfi.Aliases = []string{"execution"}

	es, err := Sliver.AddCommand(constants.ExecuteShellcodeStr,
		"Executes the given shellcode in the sliver process", "",
		&ExecuteShellcode{})
	es.Aliases = []string{"execution"}

	sd, err := Sliver.AddCommand(constants.SideloadStr,
		"Load and execute a shared object (shared library/DLL) in a remote process", "",
		&Sideload{})
	sd.Aliases = []string{"execution"}

	return
}

// bindWindowsCommands - Commands available only to Windows sessions.
func bindWindowsCommands() (err error) {

	// transports
	// ----------------------------------------------------------------------------------------
	np, err := Sliver.AddCommand(constants.NamedPipeStr,
		"Start a named pipe pivot listener", "",
		&NamedPipePivot{})
	np.Aliases = []string{"transports"}

	// Proc
	// ----------------------------------------------------------------------------------------
	m, err := Sliver.AddCommand(constants.MigrateStr,
		"Migrate into a remote host process", "",
		&Migrate{})
	m.Aliases = []string{"proc"}

	// Priv
	// ----------------------------------------------------------------------------------------
	i, err := Sliver.AddCommand(constants.ImpersonateStr,
		"Impersonate a logged in user", "",
		&Impersonate{})
	i.Aliases = []string{"priv"}

	rs, err := Sliver.AddCommand(constants.RevToSelfStr,
		"Revert to self: lose stolen Windows token", "",
		&Rev2Self{})
	rs.Aliases = []string{"priv"}

	gs, err := Sliver.AddCommand(constants.GetSystemStr,
		"Spawns a new sliver session as the NT AUTHORITY\\SYSTEM user ", "",
		&GetSystem{})
	gs.Aliases = []string{"priv"}

	mt, err := Sliver.AddCommand(constants.MakeTokenStr,
		"Create a new Logon Session with the specified credentials", "",
		&MakeToken{})
	mt.Aliases = []string{"priv"}

	// Execution
	// ----------------------------------------------------------------------------------------
	ea, err := Sliver.AddCommand(constants.ExecuteAssemblyStr,
		"Loads and executes a .NET assembly in a child process", "",
		&ExecuteAssembly{})
	ea.Aliases = []string{"execution"}

	sd, err := Sliver.AddCommand(constants.SpawnDllStr,
		"Load and execute a Reflective DLL in a remote process", "",
		&SpawnDLL{})
	sd.Aliases = []string{"execution"}

	ra, err := Sliver.AddCommand(constants.RunAsStr,
		"Run a new process in the context of the designated user", "",
		&RunAs{})
	ra.Aliases = []string{"execution"}

	// Persistence
	// ----------------------------------------------------------------------------------------
	ss, err := Sliver.AddCommand(constants.PsExecStr,
		"Start a sliver service on the session target", "",
		&Service{})
	ss.Aliases = []string{"persistence"}

	bi, err := Sliver.AddCommand(constants.BackdoorStr,
		"Infect a remote file with a sliver shellcode", "",
		&Backdoor{})
	bi.Aliases = []string{"persistence"}

	return
}
