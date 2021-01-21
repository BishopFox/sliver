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
}
