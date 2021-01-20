package command

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

	---
	This file contains all of the code that binds a given string/flags/etc. to a
	command implementation function.

	Guidelines when adding a command:

		* Try to reuse the same short/long flags for the same paramenter,
		  e.g. "timeout" flags should always be -t and --timeout when possible.
		  Try to avoid creating flags that conflict with others even if you're
		  not using the flag, e.g. avoid using -t even if your command doesn't
		  have a --timeout.

		* Add a long-form help template to `client/help`

*/

import (
	"fmt"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/help"
	"github.com/bishopfox/sliver/client/licenses"
	"github.com/bishopfox/sliver/protobuf/rpcpb"

	"github.com/desertbit/grumble"
)

const (
	defaultMTLSLPort    = 8888
	defaultHTTPLPort    = 80
	defaultHTTPSLPort   = 443
	defaultDNSLPort     = 53
	defaultTCPPort      = 4444
	defaultTCPPivotPort = 9898

	defaultReconnect = 60
	defaultMaxErrors = 1000

	defaultTimeout = 60
)

// BindCommands - Bind commands to a App
func BindCommands(app *grumble.App, rpc rpcpb.SliverRPCClient) {

	app.SetPrintHelp(helpCmd) // Responsible for display long-form help templates, etc.

	app.AddCommand(&grumble.Command{
		Name:     consts.ShellStr,
		Help:     "Start an interactive shell",
		LongHelp: help.GetHelpFor(consts.ShellStr),
		Flags: func(f *grumble.Flags) {
			f.Bool("y", "no-pty", false, "disable use of pty on macos/linux")
			f.String("s", "shell-path", "", "path to shell interpreter")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			shell(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.ExecuteStr,
		Help:     "Execute a program on the remote system",
		LongHelp: help.GetHelpFor(consts.ExecuteStr),
		Flags: func(f *grumble.Flags) {
			f.Bool("s", "silent", false, "don't print the command output")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			execute(ctx, rpc)
			fmt.Println()
			return nil
		},
		AllowArgs: true,
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.MsfStr,
		Help:     "Execute an MSF payload in the current process",
		LongHelp: help.GetHelpFor(consts.MsfStr),
		Flags: func(f *grumble.Flags) {
			f.String("m", "payload", "meterpreter_reverse_https", "msf payload")
			f.String("o", "lhost", "", "listen host")
			f.Int("l", "lport", 4444, "listen port")
			f.String("e", "encoder", "", "msf encoder")
			f.Int("i", "iterations", 1, "iterations of the encoder")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			msf(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.MsfInjectStr,
		Help:     "Inject an MSF payload into a process",
		LongHelp: help.GetHelpFor(consts.MsfInjectStr),
		Flags: func(f *grumble.Flags) {
			f.Int("p", "pid", -1, "pid to inject into")
			f.String("m", "payload", "meterpreter_reverse_https", "msf payload")
			f.String("o", "lhost", "", "listen host")
			f.Int("l", "lport", 4444, "listen port")
			f.String("e", "encoder", "", "msf encoder")
			f.Int("i", "iterations", 1, "iterations of the encoder")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			msfInject(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.PsStr,
		Help:     "List remote processes",
		LongHelp: help.GetHelpFor(consts.PsStr),
		Flags: func(f *grumble.Flags) {
			f.Int("p", "pid", -1, "filter based on pid")
			f.String("e", "exe", "", "filter based on executable name")
			f.String("o", "owner", "", "filter based on owner")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			ps(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.DownloadStr,
		Help:     "Download a file",
		LongHelp: help.GetHelpFor(consts.DownloadStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			download(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.UploadStr,
		Help:     "Upload a file",
		LongHelp: help.GetHelpFor(consts.UploadStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			upload(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.ProcdumpStr,
		Help:     "Dump process memory",
		LongHelp: help.GetHelpFor(consts.ProcdumpStr),
		Flags: func(f *grumble.Flags) {
			f.Int("p", "pid", -1, "target pid")
			f.String("n", "name", "", "target process name")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			procdump(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.RunAsStr,
		Help:     "Run a new process in the context of the designated user (Windows Only)",
		LongHelp: help.GetHelpFor(consts.RunAsStr),
		Flags: func(f *grumble.Flags) {
			f.String("u", "username", "NT AUTHORITY\\SYSTEM", "user to impersonate")
			f.String("p", "process", "", "process to start")
			f.String("a", "args", "", "arguments for the process")
			f.Int("t", "timeout", 30, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			runAs(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.ImpersonateStr,
		Help:      "Impersonate a logged in user.",
		LongHelp:  help.GetHelpFor(consts.ImpersonateStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			impersonate(ctx, rpc)
			fmt.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", 30, "command timeout in seconds")
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.RevToSelfStr,
		Help:      "Revert to self: lose stolen Windows token",
		LongHelp:  help.GetHelpFor(consts.RevToSelfStr),
		AllowArgs: false,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			revToSelf(ctx, rpc)
			fmt.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", 30, "command timeout in seconds")
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.GetSystemStr,
		Help:     "Spawns a new sliver session as the NT AUTHORITY\\SYSTEM user (Windows Only)",
		LongHelp: help.GetHelpFor(consts.GetSystemStr),
		Flags: func(f *grumble.Flags) {
			f.String("p", "process", "spoolsv.exe", "SYSTEM process to inject into")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			getsystem(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.ExecuteAssemblyStr,
		Help:      "Loads and executes a .NET assembly in a child process (Windows Only)",
		LongHelp:  help.GetHelpFor(consts.ExecuteAssemblyStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			executeAssembly(ctx, rpc)
			fmt.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.String("p", "process", "notepad.exe", "hosting process to inject into")
			f.Bool("a", "amsi", false, "use AMSI bypass (disabled by default)")
			f.Bool("e", "etw", false, "patch EtwEventWrite function to avoid detection (disabled by default)")
			f.Bool("s", "save", false, "save output to file")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.ExecuteShellcodeStr,
		Help:      "Executes the given shellcode in the sliver process",
		LongHelp:  help.GetHelpFor(consts.ExecuteShellcodeStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			executeShellcode(ctx, rpc)
			fmt.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Bool("r", "rwx-pages", false, "Use RWX permissions for memory pages")
			f.Uint("p", "pid", 0, "Pid of process to inject into (0 means injection into ourselves)")
			f.String("n", "process", `c:\windows\system32\notepad.exe`, "Process to inject into when running in interactive mode")
			f.Bool("i", "interactive", false, "Inject into a new process and interact with it")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.SideloadStr,
		Help:     "Load and execute a shared object (shared library/DLL) in a remote process",
		LongHelp: help.GetHelpFor(consts.SideloadStr),
		Flags: func(f *grumble.Flags) {
			f.String("a", "args", "", "Arguments for the shared library function")
			f.String("e", "entry-point", "", "Entrypoint for the DLL (Windows only)")
			f.String("p", "process", `c:\windows\system32\notepad.exe`, "Path to process to host the shellcode")
			f.Bool("s", "save", false, "save output to file")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		AllowArgs: true,
		HelpGroup: consts.SliverHelpGroup,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			sideload(ctx, rpc)
			fmt.Println()
			return nil
		},
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.SpawnDllStr,
		Help:     "Load and execute a Reflective DLL in a remote process",
		LongHelp: help.GetHelpFor(consts.SpawnDllStr),
		Flags: func(f *grumble.Flags) {
			f.String("p", "process", `c:\windows\system32\notepad.exe`, "Path to process to host the shellcode")
			f.String("e", "export", "ReflectiveLoader", "Entrypoint of the Reflective DLL")
			f.Bool("s", "save", false, "save output to file")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		AllowArgs: true,
		HelpGroup: consts.SliverWinHelpGroup,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			spawnDll(ctx, rpc)
			fmt.Println()
			return nil
		},
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.MigrateStr,
		Help:      "Migrate into a remote process",
		LongHelp:  help.GetHelpFor(consts.MigrateStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			migrate(ctx, rpc)
			fmt.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})

	websitesCmd := &grumble.Command{
		Name:     consts.WebsitesStr,
		Help:     "Host static content (used with HTTP C2)",
		LongHelp: help.GetHelpFor(consts.WebsitesStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			websites(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	}
	websitesCmd.AddCommand(&grumble.Command{
		Name:     consts.RmStr,
		Help:     "Remove an entire website",
		LongHelp: help.GetHelpFor(fmt.Sprintf("%s.%s", consts.WebsitesStr, consts.RmStr)),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			removeWebsite(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	websitesCmd.AddCommand(&grumble.Command{
		Name:     consts.RmWebContentStr,
		Help:     "Remove content from a website",
		LongHelp: help.GetHelpFor(fmt.Sprintf("%s.%s", consts.WebsitesStr, consts.RmWebContentStr)),
		Flags: func(f *grumble.Flags) {
			// f.Bool("r", "recursive", false, "recursively add/rm content")
			// f.String("w", "website", "", "website name")
			// f.String("p", "web-path", "", "http path to host file at")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			removeWebsiteContent(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	websitesCmd.AddCommand(&grumble.Command{
		Name:     consts.AddWebContentStr,
		Help:     "Add content to a website",
		LongHelp: help.GetHelpFor(fmt.Sprintf("%s.%s", consts.WebsitesStr, consts.RmWebContentStr)),
		Flags: func(f *grumble.Flags) {
			// f.String("w", "website", "", "website name")
			// f.String("m", "content-type", "", "mime content-type (if blank use file ext.)")
			// f.String("p", "web-path", "/", "http path to host file at")
			// f.String("c", "content", "", "local file path/dir (must use --recursive for dir)")
			// f.Bool("r", "recursive", false, "recursively add/rm content")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			addWebsiteContent(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	websitesCmd.AddCommand(&grumble.Command{
		Name:     consts.WebContentTypeStr,
		Help:     "Update a path's content-type",
		LongHelp: help.GetHelpFor(fmt.Sprintf("%s.%s", consts.WebsitesStr, consts.WebContentTypeStr)),
		Flags: func(f *grumble.Flags) {
			// f.String("w", "website", "", "website name")
			// f.String("m", "content-type", "", "mime content-type (if blank use file ext.)")
			// f.String("p", "web-path", "/", "http path to host file at")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			updateWebsiteContent(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	app.AddCommand(websitesCmd)

	app.AddCommand(&grumble.Command{
		Name:      consts.TerminateStr,
		Help:      "Kill/terminate a process",
		LongHelp:  help.GetHelpFor(consts.TerminateStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			terminate(ctx, rpc)
			fmt.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Bool("f", "force", false, "disregard safety and kill the PID")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.ScreenshotStr,
		Help:      "Take a screenshot",
		LongHelp:  help.GetHelpFor(consts.ScreenshotStr),
		AllowArgs: false,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			screenshot(ctx, rpc)
			fmt.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.LoadExtensionStr,
		Help:      "Load a sliver extension",
		LongHelp:  help.GetHelpFor(consts.LoadExtensionStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			load(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.NamedPipeStr,
		Help:     "Start a named pipe pivot listener",
		LongHelp: help.GetHelpFor(consts.NamedPipeStr),
		Flags: func(f *grumble.Flags) {
			f.String("n", "name", "", "name of the named pipe")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			namedPipeListener(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.TCPListenerStr,
		Help:     "Start a TCP pivot listener",
		LongHelp: help.GetHelpFor(consts.TCPListenerStr),
		Flags: func(f *grumble.Flags) {
			f.String("s", "server", "0.0.0.0", "interface to bind server to")
			f.Int("l", "lport", defaultTCPPivotPort, "tcp listen port")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			tcpListener(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.PsExecStr,
		Help:     "Start a sliver service on a remote target",
		LongHelp: help.GetHelpFor(consts.PsExecStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.String("s", "service-name", "Sliver", "name that will be used to register the service")
			f.String("d", "service-description", "Sliver implant", "description of the service")
			f.String("p", "profile", "", "profile to use for service binary")
			f.String("b", "binpath", "c:\\windows\\temp", "directory to which the executable will be uploaded")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			psExec(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverWinHelpGroup,
		AllowArgs: true,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.BackdoorStr,
		Help:     "Infect a remote file with a sliver shellcode",
		LongHelp: help.GetHelpFor(consts.BackdoorStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.String("p", "profile", "", "profile to use for service binary")
		},
		AllowArgs: true,
		HelpGroup: consts.SliverWinHelpGroup,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			binject(ctx, rpc)
			fmt.Println()
			return nil
		},
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.MakeTokenStr,
		Help:     "Create a new Logon Session with the specified credentials",
		LongHelp: help.GetHelpFor(consts.MakeTokenStr),
		Flags: func(f *grumble.Flags) {
			f.String("u", "username", "", "username of the user to impersonate")
			f.String("p", "password", "", "password of the user to impersonate")
			f.String("d", "domain", "", "domain of the user to impersonate")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		AllowArgs: false,
		HelpGroup: consts.SliverWinHelpGroup,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			makeToken(ctx, rpc)
			fmt.Println()
			return nil
		},
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.SetStr,
		Help:     "Set agent option",
		LongHelp: help.GetHelpFor(consts.SetStr),
		Flags: func(f *grumble.Flags) {
			f.String("n", "name", "", "agent name to change to")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			setCmd(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.GetEnvStr,
		Help:      "List environment variables",
		LongHelp:  help.GetHelpFor(consts.GetEnvStr),
		AllowArgs: true,
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			getEnv(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.LicensesStr,
		Help:     "Open source licenses",
		LongHelp: help.GetHelpFor(consts.LicensesStr),
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			fmt.Println(licenses.All)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	route := &grumble.Command{
		Name:     consts.RouteStr,
		Help:     "Manage network routes",
		LongHelp: help.GetHelpFor(consts.RouteStr),
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			routes(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	}
	app.AddCommand(route)

	route.AddCommand(&grumble.Command{
		Name:     consts.RouteAddStr,
		Help:     "Add a network route",
		LongHelp: help.GetHelpFor(consts.RouteStr),
		Flags: func(f *grumble.Flags) {
			f.String("n", "network", "", "IP network in CIDR notation (ex: 192.168.1.1/24)")
			f.String("m", "netmask", "", "(Optional) Precise network mask (ex: 255.255.255.0)")
			f.Uint("s", "session-id", 0, "(Optional) Bind this route network to a precise implant, in case two routes might collide.")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			addRoute(ctx, rpc)
			fmt.Println()
			return nil
		},
	})

	route.AddCommand(&grumble.Command{
		Name:     consts.RouteRemoveStr,
		Help:     "Remove one or more network routes (with filters)",
		LongHelp: help.GetHelpFor(consts.RouteStr),
		Flags: func(f *grumble.Flags) {
			f.StringL("network", "", "IP or CIDR to filter")
			f.StringL("id", "", "Route ID")
			f.BoolL("close", false, "Close all connections forwarded through the route")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			removeRoute(ctx, rpc)
			fmt.Println()
			return nil
		},
	})

	route.AddCommand(&grumble.Command{
		Name:     consts.RoutePrintStr,
		Help:     "Print network routes (with filters)",
		LongHelp: help.GetHelpFor(consts.RouteStr),
		Flags: func(f *grumble.Flags) {
			f.String("n", "network", "", "IP or CIDR to filter")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			routes(ctx, rpc)
			fmt.Println()
			return nil
		},
	})

	portfwd := &grumble.Command{
		Name: consts.PortfwdStr,
		Help: "Manage port forwarders for sessions, or the active one (empty: print active port forwards)",
	}
	app.AddCommand(portfwd)

	portfwd.AddCommand(&grumble.Command{
		Name: consts.PortfwdPrintStr,
		Help: "Print active port forwarders, for all sessions, or the active one",
		Flags: func(f *grumble.Flags) {
			f.BoolL("tcp", false, "Show only TCP forwarders")
			f.BoolL("udp", false, "Show only UDP forwarders")
			f.BoolL("direct", false, "Show direct port forwarders only")
			f.BoolL("reverse", false, "Show reverse port forwarders only")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			printPortForwarders(ctx, rpc)
			fmt.Println()
			return nil
		},
	})

	portfwd.AddCommand(&grumble.Command{
		Name: consts.PortfwdCloseStr,
		Help: "Close one or more port forwarders, for the active session or all.",
		Flags: func(f *grumble.Flags) {
			f.StringL("id", "", "Port forwarder ID")
			f.BoolL("reverse", false, "Close all reverse port forwarders for the session")
			f.BoolL("direct", false, "Close all direct port forwarders for the session")
			f.BoolL("close-conns", false, "Close all connections currently handled by the forwarder (TCP only)")
			f.UintL("session-id", 0, "Close all forwarders for a given session")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			closePortForwarder(ctx, rpc)
			fmt.Println()
			return nil
		},
	})
}
