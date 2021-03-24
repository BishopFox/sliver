package sliver

import (
	"github.com/jessevdk/go-flags"

	windowsCmds "github.com/bishopfox/sliver/client/commands/sliver/windows"
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

// BindCommands - Binds all commands for a given Sliver session.
// Also binds OS-specific commands, and extensions if any have been
// loaded in the current session.
func BindCommands(parser *flags.Parser) {

	// The context package checks, handles and reports any error arising from a struct
	// being registered as a command, and saves it in various group related things.
	// The following call is the contextual counterpart of RegisterServerCommand.
	var register = cctx.Commands.RegisterSliverCommand

	// 1 - OS-independent commands

	// Core Commands --------------------------------------------------------------------
	b, err := parser.AddCommand(constants.BackgroundStr,
		"Background the current session",
		help.GetHelpFor(constants.BackgroundStr),
		&Background{})
	register(err, b, constants.CoreSessionGroup)

	k, err := parser.AddCommand(constants.KillStr,
		"Kill the current session",
		help.GetHelpFor(constants.KillStr),
		&Kill{})
	register(err, k, constants.CoreSessionGroup)

	set, err := parser.AddCommand(constants.SetStr,
		"Set a value for the current session", "",
		&Set{})
	register(err, set, constants.CoreSessionGroup)

	env, err := parser.AddCommand(constants.GetEnvStr,
		"Get one or more host environment variables", "",
		&GetEnv{})
	register(err, env, constants.CoreSessionGroup)

	setenv, err := parser.AddCommand(constants.SetEnvStr,
		"Set an environment variable",
		help.GetHelpFor(constants.SetEnvStr),
		&SetEnv{})
	register(err, setenv, constants.CoreSessionGroup)

	ping, err := parser.AddCommand(constants.PingStr,
		"Send round trip message to implant (does not use ICMP)", "",
		&Ping{})
	register(err, ping, constants.CoreSessionGroup)

	sh, err := parser.AddCommand(constants.ShellStr,
		"Start an interactive shell on the session host (not opsec!)", "",
		&Shell{})
	register(err, sh, constants.CoreSessionGroup)

	// Info
	info, err := parser.AddCommand(constants.InfoStr,
		"Show session information", "",
		&Info{})
	register(err, info, constants.InfoGroup)

	uid, err := parser.AddCommand(constants.GetUIDStr,
		"Get session User ID", "",
		&UID{})
	register(err, uid, constants.InfoGroup)

	gid, err := parser.AddCommand(constants.GetGIDStr,
		"Get session User group ID", "",
		&GID{})
	register(err, gid, constants.InfoGroup)

	pid, err := parser.AddCommand(constants.GetPIDStr,
		"Get session Process ID", "",
		&PID{})
	register(err, pid, constants.InfoGroup)

	w, err := parser.AddCommand(constants.WhoamiStr,
		"Get session username", "",
		&Whoami{})
	register(err, w, constants.InfoGroup)

	sc, err := parser.AddCommand(constants.ScreenshotStr,
		"Take a screenshot", "",
		&Screenshot{})
	register(err, sc, constants.InfoGroup)

	ifc, err := parser.AddCommand(constants.IfconfigStr,
		"Show session network interfaces", "",
		&Ifconfig{})
	register(err, ifc, constants.InfoGroup)

	ns, err := parser.AddCommand(constants.NetstatStr,
		"Print network connection information", "",
		&Netstat{})
	register(err, ns, constants.InfoGroup)

	// Filesystem
	cd, err := parser.AddCommand(constants.CdStr,
		"Change session working directory", "",
		&ChangeDirectory{})
	register(err, cd, constants.FilesystemGroup)

	ls, err := parser.AddCommand(constants.LsStr,
		"List session directory contents", "",
		&ListSessionDirectories{})
	register(err, ls, constants.FilesystemGroup)

	rm, err := parser.AddCommand(constants.RmStr,
		"Remove directory/file contents from the session's host", "",
		&Rm{})
	register(err, rm, constants.FilesystemGroup)

	mkd, err := parser.AddCommand(constants.MkdirStr,
		"Create one or more directories on the implant's host", "",
		&Mkdir{})
	register(err, mkd, constants.MkdirStr)

	pwd, err := parser.AddCommand(constants.PwdStr,
		"Print the session current working directory", "",
		&Pwd{})
	register(err, pwd, constants.FilesystemGroup)

	cat, err := parser.AddCommand(constants.CatStr,
		"Print one or more files to screen", "",
		&Cat{})
	register(err, cat, constants.FilesystemGroup)

	dl, err := parser.AddCommand(constants.DownloadStr,
		"Download one or more files from the target to the client", "",
		&Download{})
	register(err, dl, constants.FilesystemGroup)

	ul, err := parser.AddCommand(constants.UploadStr,
		"Upload one or more files from the client to the target filesystem", "",
		&Upload{})
	register(err, ul, constants.FilesystemGroup)

	// Proc
	ps, err := parser.AddCommand(constants.PsStr,
		"List host processes", "",
		&PS{})
	register(err, ps, constants.ProcGroup)

	procDump, err := parser.AddCommand(constants.ProcdumpStr,
		"Dump process memory (process ID argument, or options)", "",
		&ProcDump{})
	register(err, procDump, constants.ProcGroup)

	term, err := parser.AddCommand(constants.TerminateStr,
		"Kill/terminate one or more running host processes", "",
		&Terminate{})
	register(err, term, constants.ProcGroup)

	// Execution
	exec, err := parser.AddCommand(constants.ExecuteStr,
		"Execute a program on the remote system", "",
		&Execute{})
	register(err, exec, constants.ExecuteGroup)

	msf, err := parser.AddCommand(constants.MsfStr,
		"Execute an MSF payload in the current process", "",
		&MSF{})
	register(err, msf, constants.ExecuteGroup)

	msfi, err := parser.AddCommand(constants.MsfInjectStr,
		"Inject an MSF payload into a process (ID as argument)", "",
		&MSFInject{})
	register(err, msfi, constants.ExecuteGroup)

	es, err := parser.AddCommand(constants.ExecuteShellcodeStr,
		"Executes the given shellcode in the sliver process", "",
		&ExecuteShellcode{})
	register(err, es, constants.ExecuteGroup)

	sd, err := parser.AddCommand(constants.SideloadStr,
		"Load and execute a shared object (shared library/DLL) in a remote process", "",
		&Sideload{})
	register(err, sd, constants.ExecuteGroup)

	// OS-specific commands second ---------------------------------------------------------
	if cctx.Context.Sliver.OS == "windows" {
		windowsCmds.BindCommands(parser)
	}

	// Extensions  -------------------------------------------------------------------------
	ext, err := parser.AddCommand(constants.LoadExtensionStr,
		"Load an extension through the current Sliver session", "",
		&LoadExtension{})
	register(err, ext, constants.ExtensionsGroup)

	// Get the extensions for the current session, if any
	if sessionExtensions, found := LoadedExtensions[cctx.Context.Sliver.ID]; found {
		for _, extensionBind := range sessionExtensions {
			extensionBind()
		}
	}

	return
}
