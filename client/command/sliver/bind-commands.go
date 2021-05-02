package sliver

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
	"github.com/maxlandon/gonsole"

	windowsCmds "github.com/bishopfox/sliver/client/command/sliver/windows"
	"github.com/bishopfox/sliver/client/completion"
	"github.com/bishopfox/sliver/client/constants"
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
	// Console Some commands might need to access the current context
	// in the course of the application execution.
	Console *gonsole.Console

	// Most commands just need access to a precise context.
	sliverMenu *gonsole.Menu
)

// BindCommands - Register all commands available only when interacting with a Sliver session.
// This function will also, at the end, register Windows commands, in the same context but with
// filters so that only Windows-based hosts will have these Windows commands available.
func BindCommands(cc *gonsole.Menu) {

	// Keep a reference to this context, command implementations might want to use it.
	sliverMenu = cc

	// Core Commands --------------------------------------------------------------------
	cc.AddCommand(constants.BackgroundStr,
		"Background the current session",
		help.GetHelpFor(constants.BackgroundStr),
		constants.CoreSessionGroup,
		[]string{""},
		func() interface{} { return &Background{} })

	cc.AddCommand(constants.KillStr,
		"Kill the current session",
		help.GetHelpFor(constants.KillStr),
		constants.CoreSessionGroup,
		[]string{""},
		func() interface{} { return &Kill{} })

	cc.AddCommand(constants.SetStr,
		"Set a value for the current session", "",
		constants.CoreSessionGroup,
		[]string{""},
		func() interface{} { return &Set{} })

	cc.AddCommand(constants.GetEnvStr,
		"Get one or more host environment variables", "",
		constants.CoreSessionGroup,
		[]string{""},
		func() interface{} { return &GetEnv{} })

	cc.AddCommand(constants.SetEnvStr,
		"Set an environment variable",
		help.GetHelpFor(constants.SetEnvStr),
		constants.CoreSessionGroup,
		[]string{""},
		func() interface{} { return &SetEnv{} })

	cc.AddCommand(constants.PingStr,
		"Send round trip message to implant (does not use ICMP)", "",
		constants.CoreSessionGroup,
		[]string{""},
		func() interface{} { return &Ping{} })

	shell := cc.AddCommand(constants.ShellStr,
		"Start an interactive shell on the session host (not opsec!)", "",
		constants.CoreSessionGroup,
		[]string{""},
		func() interface{} { return &Shell{} })
	shell.AddOptionCompletionDynamic("Path", completion.CompleteRemotePathAndFiles)

	// Info ----------------------------------------------------------------------------
	cc.AddCommand(constants.InfoStr,
		"Show session information", "",
		constants.InfoGroup,
		[]string{""},
		func() interface{} { return &Info{} })

	cc.AddCommand(constants.GetUIDStr,
		"Get session User ID", "",
		constants.InfoGroup,
		[]string{""},
		func() interface{} { return &UID{} })

	cc.AddCommand(constants.GetGIDStr,
		"Get session User group ID", "",
		constants.InfoGroup,
		[]string{""},
		func() interface{} { return &GID{} })

	cc.AddCommand(constants.GetPIDStr,
		"Get session Process ID", "",
		constants.InfoGroup,
		[]string{""},
		func() interface{} { return &PID{} })

	cc.AddCommand(constants.WhoamiStr,
		"Get session username", "",
		constants.InfoGroup,
		[]string{""},
		func() interface{} { return &Whoami{} })

	cc.AddCommand(constants.ScreenshotStr,
		"Take a screenshot", "",
		constants.InfoGroup,
		[]string{""},
		func() interface{} { return &Screenshot{} })

	cc.AddCommand(constants.IfconfigStr,
		"Show session network interfaces", "",
		constants.InfoGroup,
		[]string{""},
		func() interface{} { return &Ifconfig{} })

	cc.AddCommand(constants.NetstatStr,
		"Print network connection information", "",
		constants.InfoGroup,
		[]string{""},
		func() interface{} { return &Netstat{} })

	// Filesystem --------------------------------------------------------------------------
	cd := cc.AddCommand(constants.CdStr,
		"Change session working directory", "",
		constants.FilesystemGroup,
		[]string{""},
		func() interface{} { return &ChangeDirectory{} })
	cd.AddArgumentCompletionDynamic("Path", completion.CompleteRemotePath)

	ls := cc.AddCommand(constants.LsStr,
		"List session directory contents", "",
		constants.FilesystemGroup,
		[]string{""},
		func() interface{} { return &ListSessionDirectories{} })
	ls.AddArgumentCompletionDynamic("Path", completion.CompleteRemotePathAndFiles)

	rm := cc.AddCommand(constants.RmStr,
		"Remove directory/file contents from the session's host", "",
		constants.FilesystemGroup,
		[]string{""},
		func() interface{} { return &Rm{} })
	rm.AddArgumentCompletionDynamic("Path", completion.CompleteRemotePathAndFiles)

	mkdir := cc.AddCommand(constants.MkdirStr,
		"Create one or more directories on the implant's host", "",
		constants.FilesystemGroup,
		[]string{""},
		func() interface{} { return &Mkdir{} })
	mkdir.AddArgumentCompletionDynamic("Path", completion.CompleteRemotePath)

	cc.AddCommand(constants.PwdStr,
		"Print the session current working directory", "",
		constants.FilesystemGroup,
		[]string{""},
		func() interface{} { return &Pwd{} })

	cat := cc.AddCommand(constants.CatStr,
		"Print one or more files to screen", "",
		constants.FilesystemGroup,
		[]string{""},
		func() interface{} { return &Cat{} })
	cat.AddArgumentCompletionDynamic("Path", completion.CompleteRemotePathAndFiles)

	download := cc.AddCommand(constants.DownloadStr,
		"Download one or more files from the target to the client", "",
		constants.FilesystemGroup,
		[]string{""},
		func() interface{} { return &Download{} })
	download.AddArgumentCompletionDynamic("LocalPath", Console.Completer.LocalPathAndFiles)
	download.AddArgumentCompletionDynamic("RemotePath", completion.CompleteRemotePathAndFiles)

	upload := cc.AddCommand(constants.UploadStr,
		"Upload one or more files from the client to the target filesystem", "",
		constants.FilesystemGroup,
		[]string{""},
		func() interface{} { return &Upload{} })
	upload.AddArgumentCompletionDynamic("RemotePath", completion.CompleteRemotePathAndFiles)
	upload.AddArgumentCompletionDynamic("LocalPath", Console.Completer.LocalPathAndFiles)

	// Proc -------------------------------------------------------------------------------
	cc.AddCommand(constants.PsStr,
		"List host processes", "",
		constants.ProcGroup,
		[]string{""},
		func() interface{} { return &PS{} })

	procDump := cc.AddCommand(constants.ProcdumpStr,
		"Dump process memory (process ID argument, or options)", "",
		constants.ProcGroup,
		[]string{""},
		func() interface{} { return &ProcDump{} })
	procDump.AddArgumentCompletion("PID", completion.SessionProcesses)
	procDump.AddOptionCompletion("Name", completion.SessionProcessNames)

	terminate := cc.AddCommand(constants.TerminateStr,
		"Kill/terminate one or more running host processes", "",
		constants.ProcGroup,
		[]string{""},
		func() interface{} { return &Terminate{} })
	terminate.AddArgumentCompletion("PID", completion.SessionProcesses)

	// Execution --------------------------------------------------------------------------
	exec := cc.AddCommand(constants.ExecuteStr,
		"Execute a program on the remote system", "",
		constants.ExecuteGroup,
		[]string{""},
		func() interface{} { return &Execute{} })
	exec.AddArgumentCompletionDynamic("Args", completion.CompleteRemotePathAndFiles)

	msf := cc.AddCommand(constants.MsfStr,
		"Execute an MSF payload in the current process", "",
		constants.ExecuteGroup,
		[]string{""},
		func() interface{} { return &MSF{} })
	msf.AddOptionCompletion("LHost", completion.ServerInterfaceAddrs)
	msf.AddOptionCompletion("Payload", completion.CompleteMsfVenomPayloads)
	msf.AddOptionCompletion("Encoder", completion.CompleteMsfEncoders)

	msfInject := cc.AddCommand(constants.MsfInjectStr,
		"Inject an MSF payload into a process (ID as argument)", "",
		constants.ExecuteGroup,
		[]string{""},
		func() interface{} { return &MSFInject{} })
	msfInject.AddArgumentCompletion("PID", completion.SessionProcesses)
	msf.AddOptionCompletion("LHost", completion.ServerInterfaceAddrs)
	msfInject.AddOptionCompletion("Payload", completion.CompleteMsfVenomPayloads)
	msfInject.AddOptionCompletion("Encoder", completion.CompleteMsfEncoders)

	execSh := cc.AddCommand(constants.ExecuteShellcodeStr,
		"Executes the given shellcode in the sliver process", "",
		constants.ExecuteGroup,
		[]string{""},
		func() interface{} { return &ExecuteShellcode{} })
	execSh.AddArgumentCompletionDynamic("LocalPath", Console.Completer.LocalPathAndFiles)
	execSh.AddOptionCompletionDynamic("RemotePath", completion.CompleteRemotePathAndFiles)
	execSh.AddOptionCompletion("PID", completion.SessionProcesses)

	sideload := cc.AddCommand(constants.SideloadStr,
		"Load and execute a shared object (shared library/DLL) in a remote process", "",
		constants.ExecuteGroup,
		[]string{""},
		func() interface{} { return &Sideload{} })
	sideload.AddArgumentCompletionDynamic("LocalPath", Console.Completer.LocalPathAndFiles)
	sideload.AddArgumentCompletionDynamic("Args", completion.CompleteRemotePathAndFiles)
	sideload.AddOptionCompletionDynamic("RemotePath", completion.CompleteRemotePathAndFiles)
	sideload.AddOptionCompletionDynamic("Save", Console.Completer.LocalPathAndFiles)

	// Extensions  -------------------------------------------------------------------------
	loadExtension := cc.AddCommand(constants.LoadExtensionStr,
		"Load an extension through the current Sliver session", "",
		constants.ExtensionsGroup,
		[]string{""},
		func() interface{} { return &LoadExtension{} })
	loadExtension.AddArgumentCompletionDynamic("Path", Console.Completer.LocalPathAndFiles)

	//  Network Tools ----------------------------------------------------------------------

	// WireGuard
	wgPortFwd := cc.AddCommand(constants.WgPortFwdStr,
		"Manage ports forwarded by the WireGuard tun interface. Prints them by default",
		help.GetHelpFor(constants.WgPortFwdStr),
		constants.NetworkToolsGroup,
		[]string{constants.WireGuardGroup},
		func() interface{} { return &WireGuardPortFwd{} })

	wgPortFwd.SubcommandsOptional = true

	wgPortfwdAdd := wgPortFwd.AddCommand("add",
		"Add a port forward from the WireGuard tun interface to a host on the target network",
		help.GetHelpFor(constants.WgPortFwdStr),
		"",
		[]string{""},
		func() interface{} { return &WireGuardPortFwdAdd{} })
	wgPortfwdAdd.AddOptionCompletion("Remote", completion.ServerInterfaceAddrs)

	wgPortfwdRm := wgPortFwd.AddCommand("rm",
		"Remove one or more port forwards from the WireGuard tun interface",
		help.GetHelpFor(constants.WgPortFwdStr),
		"",
		[]string{""},
		func() interface{} { return &WireGuardPortFwdAdd{} })
	wgPortfwdRm.AddArgumentCompletion("ID", completion.CompleteWireGuardPortfwds)

	wgSocks := cc.AddCommand(constants.WgSocksStr,
		"Manage Socks servers listening on the WireGuard tun interface. Lists them by default.",
		help.GetHelpFor(constants.WgSocksStr),
		constants.NetworkToolsGroup,
		[]string{constants.WireGuardGroup},
		func() interface{} { return &WireGuardSocks{} })

	wgSocks.SubcommandsOptional = true

	wgSocks.AddCommand("start",
		"Start a socks5 listener on the WireGuard tun interface",
		help.GetHelpFor(constants.WgSocksStr),
		"",
		[]string{""},
		func() interface{} { return &WireGuardSocksStart{} })

	wgSocksStop := wgSocks.AddCommand(constants.RmStr,
		"Stop one or more socks5 listeners on the WireGuard tun interface",
		help.GetHelpFor(constants.WgSocksStr),
		"",
		[]string{""},
		func() interface{} { return &WireGuardPortFwdAdd{} })
	wgSocksStop.AddArgumentCompletion("ID", completion.CompleteWireGuardSocksServers)

	// In-Band Port Forwards

	portfwd := cc.AddCommand(constants.PortfwdStr,
		"In-band TCP port forwarders management (add/rm only available in session menu)",
		help.GetHelpFor(constants.PortfwdStr),
		constants.NetworkToolsGroup,
		[]string{""},
		func() interface{} { return &Portfwd{} })

	portfwd.SubcommandsOptional = true

	portfwdAdd := portfwd.AddCommand("add",
		"Create a new port forwarding tunnel",
		help.GetHelpFor(constants.PortfwdStr),
		"",
		[]string{""},
		func() interface{} { return &PortfwdAdd{} })
	portfwdAdd.AddOptionCompletion("Bind", Console.Completer.ClientInterfaceAddrs)
	portfwdAdd.AddOptionCompletion("Remote", completion.ServerInterfaceAddrs)

	portfwdRm := portfwd.AddCommand(constants.RmStr,
		"Remove a port forwarding tunnel",
		help.GetHelpFor(constants.PortfwdStr),
		"",
		[]string{""},
		func() interface{} { return &PortfwdRm{} })
	portfwdRm.AddArgumentCompletion("ID", completion.CompleteInBandForwarders)

	// Windows -----------------------------------------------------------------------------
	windowsCmds.Console = Console
	windowsCmds.BindCommands(cc)
}
