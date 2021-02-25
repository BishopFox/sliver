package sliver

import (
	"context"
	"fmt"

	"github.com/jessevdk/go-flags"
	"github.com/bishopfox/sliver/client/readline"

	windowsCmds "github.com/bishopfox/sliver/client/commands/windows"
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

// registerError - Format an error arising from binding a command. These errors generally
// happen because badly formatted metadata has been added in the command struct.
func registerError(command string, err error) (errs []error) {
	if err != nil {
		nerr := fmt.Errorf(util.CommandError+" %s%s%s - %s\n",
			readline.BOLD, command, readline.RESET, err.Error())
		errs = append(errs, nerr)
	}
	return errs
}

// BindCommands - Binds all commands for a given Sliver session.
// Also binds OS-specific commands, and extensions if any have been
// loaded in the current session.
func BindCommands(parser *flags.Parser, registerGroup func(err error, cmd *flags.Command, group string)) {

	// 1 - OS-independent commands

	// Core Commands --------------------------------------------------------------------
	b, err := parser.AddCommand(constants.BackgroundStr,
		"Background the current session",
		help.GetHelpFor(constants.BackgroundStr),
		&Background{})
	registerGroup(err, b, constants.CoreSessionGroup)

	k, err := parser.AddCommand(constants.KillStr,
		"Kill the current session",
		help.GetHelpFor(constants.KillStr),
		&Kill{})
	registerGroup(err, k, constants.CoreSessionGroup)

	set, err := parser.AddCommand(constants.SetStr,
		"Set a value for the current session", "",
		&Set{})
	registerGroup(err, set, constants.CoreSessionGroup)

	env, err := parser.AddCommand(constants.GetEnvStr,
		"Get one or more host environment variables", "",
		&SessionEnv{})
	registerGroup(err, env, constants.CoreSessionGroup)

	ping, err := parser.AddCommand(constants.PingStr,
		"Send round trip message to implant (does not use ICMP)", "",
		&Ping{})
	registerGroup(err, ping, constants.CoreSessionGroup)

	sh, err := parser.AddCommand(constants.ShellStr,
		"Start an interactive shell on the session host (not opsec!)", "",
		&Shell{})
	registerGroup(err, sh, constants.CoreSessionGroup)

	// Info
	info, err := parser.AddCommand(constants.InfoStr,
		"Show session information", "",
		&Info{})
	registerGroup(err, info, constants.InfoGroup)

	uid, err := parser.AddCommand(constants.GetUIDStr,
		"Get session User ID", "",
		&UID{})
	registerGroup(err, uid, constants.InfoGroup)

	gid, err := parser.AddCommand(constants.GetGIDStr,
		"Get session User group ID", "",
		&GID{})
	registerGroup(err, gid, constants.InfoGroup)

	pid, err := parser.AddCommand(constants.GetPIDStr,
		"Get session Process ID", "",
		&PID{})
	registerGroup(err, pid, constants.InfoGroup)

	w, err := parser.AddCommand(constants.WhoamiStr,
		"Get session username", "",
		&Whoami{})
	registerGroup(err, w, constants.InfoGroup)

	sc, err := parser.AddCommand(constants.ScreenshotStr,
		"Take a screenshot", "",
		&Screenshot{})
	registerGroup(err, sc, constants.InfoGroup)

	ifc, err := parser.AddCommand(constants.IfconfigStr,
		"Show session network interfaces", "",
		&Ifconfig{})
	registerGroup(err, ifc, constants.InfoGroup)

	ns, err := parser.AddCommand(constants.NetstatStr,
		"Print network connection information", "",
		&Netstat{})
	registerGroup(err, ns, constants.InfoGroup)

	// Filesystem
	cd, err := parser.AddCommand(constants.CdStr,
		"Change session working directory", "",
		&ChangeDirectory{})
	registerGroup(err, cd, constants.FilesystemGroup)

	ls, err := parser.AddCommand(constants.LsStr,
		"List session directory contents", "",
		&ListSessionDirectories{})
	registerGroup(err, ls, constants.FilesystemGroup)

	rm, err := parser.AddCommand(constants.RmStr,
		"Remove directory/file contents from the session's host", "",
		&Rm{})
	registerGroup(err, rm, constants.FilesystemGroup)

	mkd, err := parser.AddCommand(constants.MkdirStr,
		"Create one or more directories on the implant's host", "",
		&Mkdir{})
	registerGroup(err, mkd, constants.FilesystemGroup)

	pwd, err := parser.AddCommand(constants.PwdStr,
		"Print the session current working directory", "",
		&Pwd{})
	registerGroup(err, pwd, constants.FilesystemGroup)

	cat, err := parser.AddCommand(constants.CatStr,
		"Print one or more files to screen", "",
		&Cat{})
	registerGroup(err, cat, constants.FilesystemGroup)

	dl, err := parser.AddCommand(constants.DownloadStr,
		"Download one or more files from the target to the client", "",
		&Download{})
	registerGroup(err, dl, constants.FilesystemGroup)

	ul, err := parser.AddCommand(constants.UploadStr,
		"Upload one or more files from the client to the target filesystem", "",
		&Upload{})
	registerGroup(err, ul, constants.FilesystemGroup)

	// Proc
	ps, err := parser.AddCommand(constants.PsStr,
		"List host processes", "",
		&PS{})
	registerGroup(err, ps, constants.ProcGroup)

	procDump, err := parser.AddCommand(constants.ProcdumpStr,
		"Dump process memory (process ID argument, or options)", "",
		&ProcDump{})
	registerGroup(err, procDump, constants.ProcGroup)

	term, err := parser.AddCommand(constants.TerminateStr,
		"Kill/terminate one or more running host processes", "",
		&Terminate{})
	registerGroup(err, term, constants.ProcGroup)

	// Execution
	exec, err := parser.AddCommand(constants.ExecuteStr,
		"Execute a program on the remote system", "",
		&Execute{})
	registerGroup(err, exec, constants.ExecuteGroup)

	msf, err := parser.AddCommand(constants.MsfStr,
		"Execute an MSF payload in the current process", "",
		&MSF{})
	registerGroup(err, msf, constants.ExecuteGroup)

	msfi, err := parser.AddCommand(constants.MsfInjectStr,
		"Inject an MSF payload into a process (ID as argument)", "",
		&MSFInject{})
	registerGroup(err, msfi, constants.ExecuteGroup)

	es, err := parser.AddCommand(constants.ExecuteShellcodeStr,
		"Executes the given shellcode in the sliver process", "",
		&ExecuteShellcode{})
	registerGroup(err, es, constants.ExecuteGroup)

	sd, err := parser.AddCommand(constants.SideloadStr,
		"Load and execute a shared object (shared library/DLL) in a remote process", "",
		&Sideload{})
	registerGroup(err, sd, constants.ExecuteGroup)

	// OS-specific commands second ---------------------------------------------------------
	if cctx.Context.Sliver.OS == "windows" {
		windowsCmds.BindCommands(parser, registerGroup)
	}

	// Extensions  -------------------------------------------------------------------------
	ext, err := parser.AddCommand(constants.LoadExtensionStr,
		"Load an extension through the current Sliver session", "",
		&LoadExtension{})
	registerGroup(err, ext, constants.ExtensionsGroup)

	// Get the extensions for the current session, if any
	if sessionExtensions, found := LoadedExtensions[cctx.Context.Sliver.ID]; found {
		for _, extensionBind := range sessionExtensions {
			extensionBind()
		}
	}

	return
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
