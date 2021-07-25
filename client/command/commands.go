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
	"os"

	"github.com/bishopfox/sliver/client/command/backdoor"
	"github.com/bishopfox/sliver/client/command/dllhijack"
	"github.com/bishopfox/sliver/client/command/environment"
	"github.com/bishopfox/sliver/client/command/exec"
	"github.com/bishopfox/sliver/client/command/extensions"
	"github.com/bishopfox/sliver/client/command/filesystem"
	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/command/info"
	"github.com/bishopfox/sliver/client/command/jobs"
	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/command/monitor"
	"github.com/bishopfox/sliver/client/command/network"
	"github.com/bishopfox/sliver/client/command/operators"
	"github.com/bishopfox/sliver/client/command/pivots"
	"github.com/bishopfox/sliver/client/command/portfwd"
	"github.com/bishopfox/sliver/client/command/privilege"
	"github.com/bishopfox/sliver/client/command/processes"
	"github.com/bishopfox/sliver/client/command/reaction"
	"github.com/bishopfox/sliver/client/command/registry"
	"github.com/bishopfox/sliver/client/command/screenshot"
	"github.com/bishopfox/sliver/client/command/sessions"
	"github.com/bishopfox/sliver/client/command/shell"
	"github.com/bishopfox/sliver/client/command/update"
	"github.com/bishopfox/sliver/client/command/websites"
	"github.com/bishopfox/sliver/client/command/wireguard"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/licenses"
	"github.com/desertbit/grumble"
)

const (
	defaultTimeout = 60
)

// BindCommands - Bind commands to a App
func BindCommands(con *console.SliverConsoleClient) {

	n, err := reaction.LoadReactions()
	if err != nil && !os.IsNotExist(err) {
		con.PrintErrorf("Failed to load reactions: %s\n", err)
	} else if n > 0 {
		con.PrintInfof("Loaded %d reaction(s) from disk\n", n)
	}

	con.App.SetPrintHelp(help.HelpCmd(con)) // Responsible for display long-form help templates, etc.

	// [ Update ] --------------------------------------------------------------

	con.App.AddCommand(&grumble.Command{
		Name:     consts.UpdateStr,
		Help:     "Check for updates",
		LongHelp: help.GetHelpFor([]string{consts.UpdateStr}),
		Flags: func(f *grumble.Flags) {
			f.Bool("P", "prereleases", false, "include pre-released (unstable) versions")
			f.String("p", "proxy", "", "specify a proxy url (e.g. http://localhost:8080)")
			f.String("s", "save", "", "save downloaded files to specific directory (default user home dir)")
			f.Bool("I", "insecure", false, "skip tls certificate validation")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			update.UpdateCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.VersionStr,
		Help:     "Display version information",
		LongHelp: help.GetHelpFor([]string{consts.VersionStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			update.VerboseVersionsCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	// [ Jobs ] -----------------------------------------------------------------

	con.App.AddCommand(&grumble.Command{
		Name:     consts.JobsStr,
		Help:     "Job control",
		LongHelp: help.GetHelpFor([]string{consts.JobsStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("k", "kill", -1, "kill a background job")
			f.Bool("K", "kill-all", false, "kill all jobs")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			jobs.JobsCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.MtlsStr,
		Help:     "Start an mTLS listener",
		LongHelp: help.GetHelpFor([]string{consts.MtlsStr}),
		Flags: func(f *grumble.Flags) {
			f.String("L", "lhost", "", "interface to bind server to")
			f.Int("l", "lport", generate.DefaultMTLSLPort, "tcp listen port")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Bool("p", "persistent", false, "make persistent across restarts")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			jobs.MTLSListenerCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.WGStr,
		Help:     "Start a WireGuard listener",
		LongHelp: help.GetHelpFor([]string{consts.WGStr}),
		Flags: func(f *grumble.Flags) {
			f.String("L", "lhost", "", "interface to bind server to")
			f.Int("l", "lport", generate.DefaultWGLPort, "udp listen port")
			f.Int("n", "nport", generate.DefaultWGNPort, "virtual tun interface listen port")
			f.Int("x", "key-port", generate.DefaultWGKeyExPort, "virtual tun interface key exchange port")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Bool("p", "persistent", false, "make persistent across restarts")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			jobs.WGListenerCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.DnsStr,
		Help:     "Start a DNS listener",
		LongHelp: help.GetHelpFor([]string{consts.DnsStr}),
		Flags: func(f *grumble.Flags) {
			f.String("d", "domains", "", "parent domain(s) to use for DNS c2")
			f.Bool("c", "no-canaries", false, "disable dns canary detection")
			f.String("L", "lhost", "", "interface to bind server to")
			f.Int("l", "lport", generate.DefaultDNSLPort, "udp listen port")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Bool("p", "persistent", false, "make persistent across restarts")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			jobs.DNSListenerCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.HttpStr,
		Help:     "Start an HTTP listener",
		LongHelp: help.GetHelpFor([]string{consts.HttpStr}),
		Flags: func(f *grumble.Flags) {
			f.String("d", "domain", "", "limit responses to specific domain")
			f.String("w", "website", "", "website name (see websites cmd)")
			f.String("L", "lhost", "", "interface to bind server to")
			f.Int("l", "lport", generate.DefaultHTTPLPort, "tcp listen port")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Bool("p", "persistent", false, "make persistent across restarts")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			jobs.HTTPListenerCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.HttpsStr,
		Help:     "Start an HTTPS listener",
		LongHelp: help.GetHelpFor([]string{consts.HttpsStr}),
		Flags: func(f *grumble.Flags) {
			f.String("d", "domain", "", "limit responses to specific domain")
			f.String("w", "website", "", "website name (see websites cmd)")
			f.String("L", "lhost", "", "interface to bind server to")
			f.Int("l", "lport", generate.DefaultHTTPSLPort, "tcp listen port")

			f.String("c", "cert", "", "PEM encoded certificate file")
			f.String("k", "key", "", "PEM encoded private key file")

			f.Bool("e", "lets-encrypt", false, "attempt to provision a let's encrypt certificate")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Bool("p", "persistent", false, "make persistent across restarts")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			jobs.HTTPSListenerCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.StageListenerStr,
		Help:     "Start a stager listener",
		LongHelp: help.GetHelpFor([]string{consts.StageListenerStr}),
		Flags: func(f *grumble.Flags) {
			f.String("p", "profile", "", "Implant profile to link with the listener")
			f.String("u", "url", "", "URL to which the stager will call back to")
			f.String("c", "cert", "", "path to PEM encoded certificate file (HTTPS only)")
			f.String("k", "key", "", "path to PEM encoded private key file (HTTPS only)")
			f.Bool("e", "lets-encrypt", false, "attempt to provision a let's encrypt certificate (HTTPS only)")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			jobs.StageListenerCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	// [ Operators ] --------------------------------------------------------------

	con.App.AddCommand(&grumble.Command{
		Name:     consts.PlayersStr,
		Help:     "Manage operators",
		LongHelp: help.GetHelpFor([]string{consts.PlayersStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			operators.OperatorsCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.MultiplayerHelpGroup,
	})

	// [ Sessions ] --------------------------------------------------------------

	sessionsCmd := &grumble.Command{
		Name:     consts.SessionsStr,
		Help:     "Session management",
		LongHelp: help.GetHelpFor([]string{consts.SessionsStr}),
		Flags: func(f *grumble.Flags) {
			f.String("i", "interact", "", "interact with a sliver")
			f.String("k", "kill", "", "Kill the designated session")
			f.Bool("K", "kill-all", false, "Kill all the sessions")
			f.Bool("C", "clean", false, "Clean out any sessions marked as [DEAD]")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			sessions.SessionsCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	}
	sessionsCmd.AddCommand(&grumble.Command{
		Name:     consts.PruneStr,
		Help:     "Kill all stale sessions",
		LongHelp: help.GetHelpFor([]string{consts.SessionsStr, consts.PruneStr}),
		Flags: func(f *grumble.Flags) {

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			sessions.SessionsPruneCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})
	con.App.AddCommand(sessionsCmd)

	con.App.AddCommand(&grumble.Command{
		Name:     consts.ReconfigStr,
		Help:     "Reconfigure the active session",
		LongHelp: help.GetHelpFor([]string{consts.SessionsStr, consts.ReconfigStr}),
		Flags: func(f *grumble.Flags) {
			f.String("n", "name", "", "agent name to change to")
			f.Int("r", "reconnect", -1, "reconnect interval for agent")
			f.Int("p", "poll", -1, "poll interval for agent")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			sessions.SessionsReconfigCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.BackgroundStr,
		Help:     "Background an active session",
		LongHelp: help.GetHelpFor([]string{consts.BackgroundStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			sessions.BackgroundCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.UseStr,
		Help:     "Switch the active session",
		LongHelp: help.GetHelpFor([]string{consts.UseStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("session", "session ID or name", grumble.Default(""))
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			sessions.UseCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.KillStr,
		Help:     "Kill a session",
		LongHelp: help.GetHelpFor([]string{consts.KillStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			sessions.KillCmd(ctx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Bool("f", "force", false, "Force kill,  does not clean up")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	// [ Info ] --------------------------------------------------------------

	con.App.AddCommand(&grumble.Command{
		Name:     consts.InfoStr,
		Help:     "Get info about session",
		LongHelp: help.GetHelpFor([]string{consts.InfoStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("session", "session ID", grumble.Default(""))
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			info.InfoCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.PingStr,
		Help:     "Send round trip message to implant (does not use ICMP)",
		LongHelp: help.GetHelpFor([]string{consts.PingStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			info.PingCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.GetPIDStr,
		Help:     "Get session pid",
		LongHelp: help.GetHelpFor([]string{consts.GetPIDStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			info.PIDCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.GetUIDStr,
		Help:     "Get session process UID",
		LongHelp: help.GetHelpFor([]string{consts.GetUIDStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			info.UIDCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.GetGIDStr,
		Help:     "Get session process GID",
		LongHelp: help.GetHelpFor([]string{consts.GetGIDStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			info.GIDCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.WhoamiStr,
		Help:     "Get session user execution context",
		LongHelp: help.GetHelpFor([]string{consts.WhoamiStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			info.WhoamiCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	// [ Shell ] --------------------------------------------------------------

	con.App.AddCommand(&grumble.Command{
		Name:     consts.ShellStr,
		Help:     "Start an interactive shell",
		LongHelp: help.GetHelpFor([]string{consts.ShellStr}),
		Flags: func(f *grumble.Flags) {
			f.Bool("y", "no-pty", false, "disable use of pty on macos/linux")
			f.String("s", "shell-path", "", "path to shell interpreter")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			shell.ShellCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	// [ Exec ] --------------------------------------------------------------

	con.App.AddCommand(&grumble.Command{
		Name:     consts.ExecuteStr,
		Help:     "Execute a program on the remote system",
		LongHelp: help.GetHelpFor([]string{consts.ExecuteStr}),
		Flags: func(f *grumble.Flags) {
			f.Bool("T", "token", false, "execute command with current token (windows only)")
			f.Bool("s", "silent", false, "don't print the command output")
			f.Bool("X", "loot", false, "save output as loot")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("command", "command to execute")
			a.StringList("arguments", "arguments to the command")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			exec.ExecuteCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.ExecuteAssemblyStr,
		Help:     "Loads and executes a .NET assembly in a child process (Windows Only)",
		LongHelp: help.GetHelpFor([]string{consts.ExecuteAssemblyStr}),
		Args: func(a *grumble.Args) {
			a.String("filepath", "path the assembly file")
			a.StringList("arguments", "arguments to pass to the assembly entrypoint", grumble.Default([]string{}))
		},
		Flags: func(f *grumble.Flags) {
			f.String("p", "process", "notepad.exe", "hosting process to inject into")
			f.String("m", "method", "", "Optional method (a method is required for a .NET DLL)")
			f.String("c", "class", "", "Optional class name (required for .NET DLL)")
			f.String("d", "app-domain", "", "AppDomain name to create for .NET assembly. Generated randomly if not set.")
			f.String("a", "arch", "x84", "Assembly target architecture: x86, x64, x84 (x86+x64)")
			f.Bool("s", "save", false, "save output to file")
			f.Bool("X", "loot", false, "save output as loot")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			exec.ExecuteAssemblyCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.ExecuteShellcodeStr,
		Help:     "Executes the given shellcode in the sliver process",
		LongHelp: help.GetHelpFor([]string{consts.ExecuteShellcodeStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			exec.ExecuteShellcodeCmd(ctx, con)
			con.Println()
			return nil
		},
		Args: func(a *grumble.Args) {
			a.String("filepath", "path the shellcode file")
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

	con.App.AddCommand(&grumble.Command{
		Name:     consts.SideloadStr,
		Help:     "Load and execute a shared object (shared library/DLL) in a remote process",
		LongHelp: help.GetHelpFor([]string{consts.SideloadStr}),
		Flags: func(f *grumble.Flags) {
			f.String("e", "entry-point", "", "Entrypoint for the DLL (Windows only)")
			f.String("p", "process", `c:\windows\system32\notepad.exe`, "Path to process to host the shellcode")
			f.Bool("s", "save", false, "save output to file")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Bool("k", "keep-alive", false, "don't terminate host process once the execution completes")
		},
		Args: func(a *grumble.Args) {
			a.String("filepath", "path the shared library file")
			a.StringList("args", "arguments for the binary", grumble.Default([]string{}))
		},
		HelpGroup: consts.SliverHelpGroup,
		Run: func(ctx *grumble.Context) error {
			con.Println()
			exec.SideloadCmd(ctx, con)
			con.Println()
			return nil
		},
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.SpawnDllStr,
		Help:     "Load and execute a Reflective DLL in a remote process",
		LongHelp: help.GetHelpFor([]string{consts.SpawnDllStr}),
		Flags: func(f *grumble.Flags) {
			f.String("p", "process", `c:\windows\system32\notepad.exe`, "Path to process to host the shellcode")
			f.String("e", "export", "ReflectiveLoader", "Entrypoint of the Reflective DLL")
			f.Bool("s", "save", false, "save output to file")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Bool("k", "keep-alive", false, "don't terminate host process once the execution completes")
		},
		Args: func(a *grumble.Args) {
			a.String("filepath", "path the DLL file")
			a.StringList("arguments", "arguments to pass to the DLL entrypoint", grumble.Default([]string{}))
		},
		HelpGroup: consts.SliverWinHelpGroup,
		Run: func(ctx *grumble.Context) error {
			con.Println()
			exec.SpawnDllCmd(ctx, con)
			con.Println()
			return nil
		},
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.MigrateStr,
		Help:     "Migrate into a remote process",
		LongHelp: help.GetHelpFor([]string{consts.MigrateStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			exec.MigrateCmd(ctx, con)
			con.Println()
			return nil
		},
		Args: func(a *grumble.Args) {
			a.Uint("pid", "pid")
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.MsfStr,
		Help:     "Execute an MSF payload in the current process",
		LongHelp: help.GetHelpFor([]string{consts.MsfStr}),
		Flags: func(f *grumble.Flags) {
			f.String("m", "payload", "meterpreter_reverse_https", "msf payload")
			f.String("L", "lhost", "", "listen host")
			f.Int("l", "lport", 4444, "listen port")
			f.String("e", "encoder", "", "msf encoder")
			f.Int("i", "iterations", 1, "iterations of the encoder")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			exec.MsfCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.MsfInjectStr,
		Help:     "Inject an MSF payload into a process",
		LongHelp: help.GetHelpFor([]string{consts.MsfInjectStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("p", "pid", -1, "pid to inject into")
			f.String("m", "payload", "meterpreter_reverse_https", "msf payload")
			f.String("L", "lhost", "", "listen host")
			f.Int("l", "lport", 4444, "listen port")
			f.String("e", "encoder", "", "msf encoder")
			f.Int("i", "iterations", 1, "iterations of the encoder")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			exec.MsfInjectCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.PsExecStr,
		Help:     "Start a sliver service on a remote target",
		LongHelp: help.GetHelpFor([]string{consts.PsExecStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.String("s", "service-name", "Sliver", "name that will be used to register the service")
			f.String("d", "service-description", "Sliver implant", "description of the service")
			f.String("p", "profile", "", "profile to use for service binary")
			f.String("b", "binpath", "c:\\windows\\temp", "directory to which the executable will be uploaded")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			exec.PsExecCmd(ctx, con)
			con.Println()
			return nil
		},
		Args: func(a *grumble.Args) {
			a.String("hostname", "hostname")
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.SSHStr,
		Help:     "Run a SSH command on a remote host",
		LongHelp: help.GetHelpFor([]string{consts.SSHStr}),
		Args: func(a *grumble.Args) {
			a.String("hostname", "remote host to SSH to")
			a.StringList("command", "command line with arguments")
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Uint("p", "port", 22, "SSH port")
			f.String("i", "private-key", "", "path to private key file")
			f.String("P", "password", "", "SSH user password")
			f.String("l", "login", "", "username to use to connect")
			f.Bool("s", "skip-loot", false, "skip the prompt to use loot credentials")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			exec.SSHCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.PersistStr,
		Help:     "Persist a sliver across reboots",
		LongHelp: help.GetHelpFor([]string{consts.PersistStr}),
		Flags: func(f *grumble.Flags) {
			f.String("s", "sliver", "", "Sliver to persist")
			f.Bool("u", "unload", false, "Unload persistence")
			f.String("p", "path", "", "path to use for the implant")

			f.Int("i", "interval", 0, "interval in minutes for windows userland persistence")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			persist(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	// [ Generate ] --------------------------------------------------------------

	generateCmd := &grumble.Command{
		Name:     consts.GenerateStr,
		Help:     "Generate an implant binary",
		LongHelp: help.GetHelpFor([]string{consts.GenerateStr}),
		Flags: func(f *grumble.Flags) {
			f.String("o", "os", "windows", "operating system")
			f.String("a", "arch", "amd64", "cpu architecture")
			f.String("N", "name", "", "agent name")
			f.Bool("d", "debug", false, "enable debug features")
			f.Bool("e", "evasion", false, "enable evasion features")
			f.Bool("b", "skip-symbols", false, "skip symbol obfuscation")

			f.String("c", "canary", "", "canary domain(s)")

			f.String("m", "mtls", "", "mtls connection strings")
			f.String("g", "wg", "", "wg connection strings")
			f.String("H", "http", "", "http(s) connection strings")
			f.String("n", "dns", "", "dns connection strings")
			f.String("p", "named-pipe", "", "named-pipe connection strings")
			f.String("i", "tcp-pivot", "", "tcp-pivot connection strings")

			f.Int("X", "key-exchange", generate.DefaultWGKeyExPort, "wg key-exchange port")
			f.Int("T", "tcp-comms", generate.DefaultWGNPort, "wg c2 comms port")

			f.Int("j", "reconnect", generate.DefaultReconnect, "attempt to reconnect every n second(s)")
			f.Int("P", "poll", generate.DefaultPoll, "attempt to poll every n second(s)")
			f.Int("k", "max-errors", generate.DefaultMaxErrors, "max number of connection errors")

			f.String("w", "limit-datetime", "", "limit execution to before datetime")
			f.Bool("x", "limit-domainjoined", false, "limit execution to domain joined machines")
			f.String("y", "limit-username", "", "limit execution to specified username")
			f.String("z", "limit-hostname", "", "limit execution to specified hostname")
			f.String("F", "limit-fileexists", "", "limit execution to hosts with this file in the filesystem")

			f.String("f", "format", "exe", "Specifies the output formats, valid values are: 'exe', 'shared' (for dynamic libraries), 'service' (see `psexec` for more info) and 'shellcode' (windows only)")

			f.String("s", "save", "", "directory/file to the binary to")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			generate.GenerateCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	}
	generateCmd.AddCommand(&grumble.Command{
		Name:     consts.StagerStr,
		Help:     "Generate a implant stager using MSF",
		LongHelp: help.GetHelpFor([]string{consts.StagerStr}),
		Flags: func(f *grumble.Flags) {
			f.String("o", "os", "windows", "operating system")
			f.String("a", "arch", "amd64", "cpu architecture")
			f.String("L", "lhost", "", "Listening host")
			f.Int("l", "lport", 8443, "Listening port")
			f.String("r", "protocol", "tcp", "Staging protocol (tcp/http/https)")
			f.String("f", "format", "raw", "Output format (msfvenom formats, see `help generate stager` for the list)")
			f.String("b", "badchars", "", "bytes to exclude from stage shellcode")
			f.String("s", "save", "", "directory to save the generated stager to")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			generate.GenerateStagerCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	generateCmd.AddCommand(&grumble.Command{
		Name:     consts.CompilerStr,
		Help:     "Get information about the server's compiler",
		LongHelp: help.GetHelpFor([]string{consts.CompilerStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			generate.GenerateInfoCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	con.App.AddCommand(generateCmd)

	con.App.AddCommand(&grumble.Command{
		Name:     consts.RegenerateStr,
		Help:     "Regenerate an implant",
		LongHelp: help.GetHelpFor([]string{consts.RegenerateStr}),
		Args: func(a *grumble.Args) {
			a.String("implant-name", "name of the implant")
		},
		Flags: func(f *grumble.Flags) {
			f.String("s", "save", "", "directory/file to the binary to")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			generate.RegenerateCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	profilesCmd := &grumble.Command{
		Name:     consts.ProfilesStr,
		Help:     "List existing profiles",
		LongHelp: help.GetHelpFor([]string{consts.ProfilesStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			generate.ProfilesCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	}
	profilesCmd.AddCommand(&grumble.Command{
		Name:     consts.GenerateStr,
		Help:     "Generate implant from a profile",
		LongHelp: help.GetHelpFor([]string{consts.ProfilesStr, consts.GenerateStr}),
		Flags: func(f *grumble.Flags) {
			f.String("p", "name", "", "profile name")
			f.String("s", "save", "", "directory/file to the binary to")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			generate.ProfilesGenerateCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	profilesCmd.AddCommand(&grumble.Command{
		Name:     consts.NewStr,
		Help:     "Save a new implant profile",
		LongHelp: help.GetHelpFor([]string{consts.ProfilesStr, consts.NewStr}),
		Flags: func(f *grumble.Flags) {
			f.String("o", "os", "windows", "operating system")
			f.String("a", "arch", "amd64", "cpu architecture")
			f.Bool("d", "debug", false, "enable debug features")
			f.Bool("e", "evasion", false, "enable evasion features")
			f.Bool("s", "skip-symbols", false, "skip symbol obfuscation")

			f.String("m", "mtls", "", "mtls domain(s)")
			f.String("g", "wg", "", "wg domain(s)")
			f.String("H", "http", "", "http[s] domain(s)")
			f.String("n", "dns", "", "dns domain(s)")
			f.String("p", "named-pipe", "", "named-pipe connection strings")
			f.String("i", "tcp-pivot", "", "tcp-pivot connection strings")

			f.Int("X", "key-exchange", generate.DefaultWGKeyExPort, "wg key-exchange port")
			f.Int("T", "tcp-comms", generate.DefaultWGNPort, "wg c2 comms port")

			f.String("c", "canary", "", "canary domain(s)")

			f.Int("j", "reconnect", generate.DefaultReconnect, "attempt to reconnect every n second(s)")
			f.Int("k", "max-errors", generate.DefaultMaxErrors, "max number of connection errors")
			f.Int("P", "poll", generate.DefaultPoll, "attempt to poll every n second(s)")

			f.String("w", "limit-datetime", "", "limit execution to before datetime")
			f.Bool("x", "limit-domainjoined", false, "limit execution to domain joined machines")
			f.String("y", "limit-username", "", "limit execution to specified username")
			f.String("z", "limit-hostname", "", "limit execution to specified hostname")
			f.String("F", "limit-fileexists", "", "limit execution to hosts with this file in the filesystem")

			f.String("f", "format", "exe", "Specifies the output formats, valid values are: 'exe', 'shared' (for dynamic libraries), 'service' (see `psexec` for more info) and 'shellcode' (windows only)")

			f.String("N", "profile-name", "", "profile name")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			generate.ProfilesNewCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	profilesCmd.AddCommand(&grumble.Command{
		Name:     consts.RmStr,
		Help:     "Remove a profile",
		LongHelp: help.GetHelpFor([]string{consts.ProfilesStr, consts.RmStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("profile-name", "name of the profile")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			generate.ProfilesRmCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	con.App.AddCommand(profilesCmd)

	implantBuildsCmd := &grumble.Command{
		Name:     consts.ImplantBuildsStr,
		Help:     "List implant builds",
		LongHelp: help.GetHelpFor([]string{consts.ImplantBuildsStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			generate.ImplantsCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	}
	implantBuildsCmd.AddCommand(&grumble.Command{
		Name:     consts.RmStr,
		Help:     "Remove implant build",
		LongHelp: help.GetHelpFor([]string{consts.ImplantBuildsStr, consts.RmStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("implant-name", "implant name")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			generate.ImplantsRmCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	con.App.AddCommand(implantBuildsCmd)

	con.App.AddCommand(&grumble.Command{
		Name:     consts.ListCanariesStr,
		Help:     "List previously generated canaries",
		LongHelp: help.GetHelpFor([]string{consts.ListCanariesStr}),
		Flags: func(f *grumble.Flags) {
			f.Bool("b", "burned", false, "show only triggered/burned canaries")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			generate.CanariesCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	// [ Filesystem ] ---------------------------------------------

	con.App.AddCommand(&grumble.Command{
		Name:     consts.LsStr,
		Help:     "List current directory",
		LongHelp: help.GetHelpFor([]string{consts.LsStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("path", "path to enumerate", grumble.Default("."))
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			filesystem.LsCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.RmStr,
		Help:     "Remove a file or directory",
		LongHelp: help.GetHelpFor([]string{consts.RmStr}),
		Flags: func(f *grumble.Flags) {
			f.Bool("r", "recursive", false, "recursively remove files")
			f.Bool("f", "force", false, "ignore safety and forcefully remove files")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("path", "path to the file to remove")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			filesystem.RmCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.MkdirStr,
		Help:     "Make a directory",
		LongHelp: help.GetHelpFor([]string{consts.MkdirStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("path", "path to the directory to create")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			filesystem.MkdirCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.CdStr,
		Help:     "Change directory",
		LongHelp: help.GetHelpFor([]string{consts.CdStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("path", "path to the directory", grumble.Default("."))
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			filesystem.CdCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.PwdStr,
		Help:     "Print working directory",
		LongHelp: help.GetHelpFor([]string{consts.PwdStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			filesystem.PwdCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.CatStr,
		Help:     "Dump file to stdout",
		LongHelp: help.GetHelpFor([]string{consts.CatStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Bool("c", "colorize-output", false, "colorize output")
			f.Bool("x", "hex", false, "display as a hex dump")
			f.Bool("X", "loot", false, "save output as loot")
		},
		Args: func(a *grumble.Args) {
			a.String("path", "path to the file to print")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			filesystem.CatCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.DownloadStr,
		Help:     "Download a file",
		LongHelp: help.GetHelpFor([]string{consts.DownloadStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")

			f.Bool("X", "loot", false, "save output as loot")
		},
		Args: func(a *grumble.Args) {
			a.String("remote-path", "path to the file or directory to download")
			a.String("local-path", "local path where the downloaded file will be saved", grumble.Default("."))
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			filesystem.DownloadCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.UploadStr,
		Help:     "Upload a file",
		LongHelp: help.GetHelpFor([]string{consts.UploadStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("local-path", "local path to the file to upload")
			a.String("remote-path", "path to the file or directory to upload to", grumble.Default(""))
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			filesystem.UploadCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	// [ Network ] ---------------------------------------------

	con.App.AddCommand(&grumble.Command{
		Name:     consts.IfconfigStr,
		Help:     "View network interface configurations",
		LongHelp: help.GetHelpFor([]string{consts.IfconfigStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			network.IfconfigCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.NetstatStr,
		Help:     "Print network connection information",
		LongHelp: help.GetHelpFor([]string{consts.NetstatStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			network.NetstatCmd(ctx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Bool("T", "tcp", true, "display information about TCP sockets")
			f.Bool("u", "udp", false, "display information about UDP sockets")
			f.Bool("4", "ip4", true, "display information about IPv4 sockets")
			f.Bool("6", "ip6", false, "display information about IPv6 sockets")
			f.Bool("l", "listen", false, "display information about listening sockets")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	// [ Processes ] ---------------------------------------------

	con.App.AddCommand(&grumble.Command{
		Name:     consts.PsStr,
		Help:     "List remote processes",
		LongHelp: help.GetHelpFor([]string{consts.PsStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("p", "pid", -1, "filter based on pid")
			f.String("e", "exe", "", "filter based on executable name")
			f.String("o", "owner", "", "filter based on owner")
			f.Bool("c", "print-cmdline", false, "print command line arguments")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			processes.PsCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.ProcdumpStr,
		Help:     "Dump process memory",
		LongHelp: help.GetHelpFor([]string{consts.ProcdumpStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("p", "pid", -1, "target pid")
			f.String("n", "name", "", "target process name")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			processes.ProcdumpCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.TerminateStr,
		Help:     "Terminate a process on the remote system",
		LongHelp: help.GetHelpFor([]string{consts.TerminateStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			processes.TerminateCmd(ctx, con)
			con.Println()
			return nil
		},
		Args: func(a *grumble.Args) {
			a.Uint("pid", "pid")
		},
		Flags: func(f *grumble.Flags) {
			f.Bool("f", "force", false, "disregard safety and kill the PID")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	// [ Privileges ] ---------------------------------------------

	con.App.AddCommand(&grumble.Command{
		Name:     consts.RunAsStr,
		Help:     "Run a new process in the context of the designated user (Windows Only)",
		LongHelp: help.GetHelpFor([]string{consts.RunAsStr}),
		Flags: func(f *grumble.Flags) {
			f.String("u", "username", "NT AUTHORITY\\SYSTEM", "user to impersonate")
			f.String("p", "process", "", "process to start")
			f.String("a", "args", "", "arguments for the process")
			f.Int("t", "timeout", 30, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			privilege.RunAsCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.ImpersonateStr,
		Help:     "Impersonate a logged in user.",
		LongHelp: help.GetHelpFor([]string{consts.ImpersonateStr}),
		Args: func(a *grumble.Args) {
			a.String("username", "name of the user account to impersonate")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			privilege.ImpersonateCmd(ctx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", 30, "command timeout in seconds")
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.RevToSelfStr,
		Help:     "Revert to self: lose stolen Windows token",
		LongHelp: help.GetHelpFor([]string{consts.RevToSelfStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			privilege.RevToSelfCmd(ctx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", 30, "command timeout in seconds")
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.GetSystemStr,
		Help:     "Spawns a new sliver session as the NT AUTHORITY\\SYSTEM user (Windows Only)",
		LongHelp: help.GetHelpFor([]string{consts.GetSystemStr}),
		Flags: func(f *grumble.Flags) {
			f.String("p", "process", "spoolsv.exe", "SYSTEM process to inject into")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			privilege.GetSystemCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.MakeTokenStr,
		Help:     "Create a new Logon Session with the specified credentials",
		LongHelp: help.GetHelpFor([]string{consts.MakeTokenStr}),
		Flags: func(f *grumble.Flags) {
			f.String("u", "username", "", "username of the user to impersonate")
			f.String("p", "password", "", "password of the user to impersonate")
			f.String("d", "domain", "", "domain of the user to impersonate")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		HelpGroup: consts.SliverWinHelpGroup,
		Run: func(ctx *grumble.Context) error {
			con.Println()
			privilege.MakeTokenCmd(ctx, con)
			con.Println()
			return nil
		},
	})

	// [ Websites ] ---------------------------------------------

	websitesCmd := &grumble.Command{
		Name:     consts.WebsitesStr,
		Help:     "Host static content (used with HTTP C2)",
		LongHelp: help.GetHelpFor([]string{consts.WebsitesStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			websites.WebsitesCmd(ctx, con)
			con.Println()
			return nil
		},
		Args: func(a *grumble.Args) {
			a.String("name", "website name", grumble.Default(""))
		},
		HelpGroup: consts.GenericHelpGroup,
	}
	websitesCmd.AddCommand(&grumble.Command{
		Name:     consts.RmStr,
		Help:     "Remove an entire website and all of its contents",
		LongHelp: help.GetHelpFor([]string{consts.WebsitesStr, consts.RmStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			websites.WebsiteRmCmd(ctx, con)
			con.Println()
			return nil
		},
		Args: func(a *grumble.Args) {
			a.String("name", "website name", grumble.Default(""))
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	websitesCmd.AddCommand(&grumble.Command{
		Name:     consts.RmWebContentStr,
		Help:     "Remove specific content from a website",
		LongHelp: help.GetHelpFor([]string{consts.WebsitesStr, consts.RmWebContentStr}),
		Flags: func(f *grumble.Flags) {
			f.Bool("r", "recursive", false, "recursively add/rm content")
			f.String("w", "website", "", "website name")
			f.String("p", "web-path", "", "http path to host file at")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			websites.WebsitesRmContent(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	websitesCmd.AddCommand(&grumble.Command{
		Name:     consts.AddWebContentStr,
		Help:     "Add content to a website",
		LongHelp: help.GetHelpFor([]string{consts.WebsitesStr, consts.RmWebContentStr}),
		Flags: func(f *grumble.Flags) {
			f.String("w", "website", "", "website name")
			f.String("m", "content-type", "", "mime content-type (if blank use file ext.)")
			f.String("p", "web-path", "/", "http path to host file at")
			f.String("c", "content", "", "local file path/dir (must use --recursive for dir)")
			f.Bool("r", "recursive", false, "recursively add/rm content")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			websites.WebsitesAddContentCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	websitesCmd.AddCommand(&grumble.Command{
		Name:     consts.WebContentTypeStr,
		Help:     "Update a path's content-type",
		LongHelp: help.GetHelpFor([]string{consts.WebsitesStr, consts.WebContentTypeStr}),
		Flags: func(f *grumble.Flags) {
			f.String("w", "website", "", "website name")
			f.String("m", "content-type", "", "mime content-type (if blank use file ext.)")
			f.String("p", "web-path", "/", "http path to host file at")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			websites.WebsitesUpdateContentCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	con.App.AddCommand(websitesCmd)

	// [ Screenshot ] ---------------------------------------------

	con.App.AddCommand(&grumble.Command{
		Name:     consts.ScreenshotStr,
		Help:     "Take a screenshot",
		LongHelp: help.GetHelpFor([]string{consts.ScreenshotStr}),
		Flags: func(f *grumble.Flags) {
			f.Bool("X", "loot", false, "save output as loot")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			screenshot.ScreenshotCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	// [ Backdoor ] ---------------------------------------------

	con.App.AddCommand(&grumble.Command{
		Name:     consts.BackdoorStr,
		Help:     "Infect a remote file with a sliver shellcode",
		LongHelp: help.GetHelpFor([]string{consts.BackdoorStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.String("p", "profile", "", "profile to use for service binary")
		},
		Args: func(a *grumble.Args) {
			a.String("remote-file", "path to the file to backdoor")
		},
		HelpGroup: consts.SliverWinHelpGroup,
		Run: func(ctx *grumble.Context) error {
			con.Println()
			backdoor.BackdoorCmd(ctx, con)
			con.Println()
			return nil
		},
	})

	// [ Extensions ] ---------------------------------------------

	con.App.AddCommand(&grumble.Command{
		Name:     consts.LoadExtensionStr,
		Help:     "Load a sliver extension",
		LongHelp: help.GetHelpFor([]string{consts.LoadExtensionStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			extensions.LoadExtensionCmd(ctx, con)
			con.Println()
			return nil
		},
		Args: func(a *grumble.Args) {
			a.String("dir-path", "path to the extension directory")
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	// [ Environment ] ---------------------------------------------

	envCmd := &grumble.Command{
		Name:     consts.EnvStr,
		Help:     "List environment variables",
		LongHelp: help.GetHelpFor([]string{consts.EnvStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("name", "environment variable to fetch", grumble.Default(""))
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			environment.EnvGetCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	}
	envCmd.AddCommand(&grumble.Command{
		Name:     consts.SetStr,
		Help:     "Set environment variables",
		LongHelp: help.GetHelpFor([]string{consts.EnvStr, consts.SetStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("name", "environment variable name")
			a.String("value", "value to assign")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			environment.EnvSetCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	envCmd.AddCommand(&grumble.Command{
		Name:     consts.UnsetStr,
		Help:     "Clear environment variables",
		LongHelp: help.GetHelpFor([]string{consts.EnvStr, consts.UnsetStr}),
		Args: func(a *grumble.Args) {
			a.String("name", "environment variable name")
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			environment.EnvUnsetCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	con.App.AddCommand(envCmd)

	// [ Licenses ] ---------------------------------------------

	con.App.AddCommand(&grumble.Command{
		Name:     consts.LicensesStr,
		Help:     "Open source licenses",
		LongHelp: help.GetHelpFor([]string{consts.LicensesStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			con.Println(licenses.All)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	// [ Registry ] ---------------------------------------------

	registryCmd := &grumble.Command{
		Name:     consts.RegistryStr,
		Help:     "Windows registry operations",
		LongHelp: help.GetHelpFor([]string{consts.RegistryStr}),
		Run: func(ctx *grumble.Context) error {
			return nil
		},
		HelpGroup: consts.SliverWinHelpGroup,
	}
	registryCmd.AddCommand(&grumble.Command{
		Name:     consts.RegistryReadStr,
		Help:     "Read values from the Windows registry",
		LongHelp: help.GetHelpFor([]string{consts.RegistryReadStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			registry.RegReadCmd(ctx, con)
			con.Println()
			return nil
		},
		Args: func(a *grumble.Args) {
			a.String("registry-path", "registry path")
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.String("H", "hive", "HKCU", "egistry hive")
			f.String("o", "hostname", "", "remote host to read values from")
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})
	registryCmd.AddCommand(&grumble.Command{
		Name:     consts.RegistryWriteStr,
		Help:     "Write values to the Windows registry",
		LongHelp: help.GetHelpFor([]string{consts.RegistryWriteStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			registry.RegWriteCmd(ctx, con)
			con.Println()
			return nil
		},
		Args: func(a *grumble.Args) {
			a.String("registry-path", "registry path")
			a.String("value", "value to write")
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.String("H", "hive", "HKCU", "registry hive")
			f.String("o", "hostname", "", "remote host to write values to")
			f.String("T", "type", "string", "type of the value to write (string, dword, qword, binary). If binary, you must provide a path to a file with --path")
			f.String("p", "path", "", "path to the binary file to write")
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})
	registryCmd.AddCommand(&grumble.Command{
		Name:     consts.RegistryCreateKeyStr,
		Help:     "Create a registry key",
		LongHelp: help.GetHelpFor([]string{consts.RegistryCreateKeyStr}),
		Args: func(a *grumble.Args) {
			a.String("registry-path", "registry path")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			registry.RegCreateKeyCmd(ctx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.String("H", "hive", "HKCU", "registry hive")
			f.String("o", "hostname", "", "remote host to write values to")
		},
	})
	con.App.AddCommand(registryCmd)

	// [ Pivots ] --------------------------------------------------------------

	con.App.AddCommand(&grumble.Command{
		Name:     consts.PivotsListStr,
		Help:     "List pivots",
		LongHelp: help.GetHelpFor([]string{consts.PivotsListStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			pivots.PivotsCmd(ctx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.String("i", "id", "", "session id")
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.NamedPipeStr,
		Help:     "Start a named pipe pivot listener",
		LongHelp: help.GetHelpFor([]string{consts.NamedPipeStr}),
		Flags: func(f *grumble.Flags) {
			f.String("n", "name", "", "name of the named pipe")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			pivots.NamedPipeListenerCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.TCPListenerStr,
		Help:     "Start a TCP pivot listener",
		LongHelp: help.GetHelpFor([]string{consts.TCPListenerStr}),
		Flags: func(f *grumble.Flags) {
			f.String("s", "server", "0.0.0.0", "interface to bind server to")
			f.Int("l", "lport", generate.DefaultTCPPivotPort, "tcp listen port")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			pivots.TCPListenerCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	// [ WireGuard ] --------------------------------------------------------------

	con.App.AddCommand(&grumble.Command{
		Name:     consts.WgConfigStr,
		Help:     "Generate a new WireGuard client config",
		LongHelp: help.GetHelpFor([]string{consts.WgConfigStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.String("s", "save", "", "save configuration to file (.conf)")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			wireguard.WGConfigCmd(ctx, con)
			con.Println()
			return nil
		},
	})

	wgPortFwdCmd := &grumble.Command{
		Name:     consts.WgPortFwdStr,
		Help:     "List ports forwarded by the WireGuard tun interface",
		LongHelp: help.GetHelpFor([]string{consts.WgPortFwdStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			wireguard.WGPortFwdListCmd(ctx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
	}
	wgPortFwdCmd.AddCommand(&grumble.Command{
		Name:     consts.AddStr,
		Help:     "Add a port forward from the WireGuard tun interface to a host on the target network",
		LongHelp: help.GetHelpFor([]string{consts.WgPortFwdStr, consts.AddStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			wireguard.WGPortFwdAddCmd(ctx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Int("b", "bind", 1080, "port to listen on the WireGuard tun interface")
			f.String("r", "remote", "", "remote target host:port (e.g., 10.0.0.1:445)")
		},
	})
	wgPortFwdCmd.AddCommand(&grumble.Command{
		Name:     consts.RmStr,
		Help:     "Remove a port forward from the WireGuard tun interface",
		LongHelp: help.GetHelpFor([]string{consts.WgPortFwdStr, consts.RmStr}),
		Args: func(a *grumble.Args) {
			a.Int("id", "forwarder id")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			wireguard.WGPortFwdRmCmd(ctx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
	})
	con.App.AddCommand(wgPortFwdCmd)

	wgSocksCmd := &grumble.Command{
		Name:     consts.WgSocksStr,
		Help:     "List socks servers listening on the WireGuard tun interface",
		LongHelp: help.GetHelpFor([]string{consts.WgSocksStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			wireguard.WGSocksListCmd(ctx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
	}
	wgSocksCmd.AddCommand(&grumble.Command{
		Name:     consts.StartStr,
		Help:     "Start a socks5 listener on the WireGuard tun interface",
		LongHelp: help.GetHelpFor([]string{consts.WgSocksStr, consts.StartStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			wireguard.WGSocksStartCmd(ctx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Int("b", "bind", 3090, "port to listen on the WireGuard tun interface")
		},
	})
	wgSocksCmd.AddCommand(&grumble.Command{
		Name:     consts.StopStr,
		Help:     "Stop a socks5 listener on the WireGuard tun interface",
		LongHelp: help.GetHelpFor([]string{consts.WgSocksStr, consts.StopStr}),
		Args: func(a *grumble.Args) {
			a.Int("id", "forwarder id")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			wireguard.WGSocksStopCmd(ctx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
	})
	con.App.AddCommand(wgSocksCmd)

	// [ Portfwd ] --------------------------------------------------------------

	portfwdCmd := &grumble.Command{
		Name:     consts.PortfwdStr,
		Help:     "In-band TCP port forwarding",
		LongHelp: help.GetHelpFor([]string{consts.PortfwdStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			portfwd.PortfwdCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	}
	portfwdCmd.AddCommand(&grumble.Command{
		Name:     "add",
		Help:     "Create a new port forwarding tunnel",
		LongHelp: help.GetHelpFor([]string{consts.PortfwdStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.String("r", "remote", "", "remote target host:port (e.g., 10.0.0.1:445)")
			f.String("b", "bind", "127.0.0.1:8080", "bind port forward to interface")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			portfwd.PortfwdAddCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})
	portfwdCmd.AddCommand(&grumble.Command{
		Name:     "rm",
		Help:     "Remove a port forwarding tunnel",
		LongHelp: help.GetHelpFor([]string{consts.PortfwdStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Int("i", "id", 0, "id of portfwd to remove")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			portfwd.PortfwdRmCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})
	con.App.AddCommand(portfwdCmd)

	// [ Monitor ] --------------------------------------------------------------

	monitorCmd := &grumble.Command{
		Name: consts.MonitorStr,
		Help: "Monitor threat intel platforms for Sliver implants",
	}
	monitorCmd.AddCommand(&grumble.Command{
		Name: "start",
		Help: "Start the monitoring loops",
		Run: func(ctx *grumble.Context) error {
			con.Println()
			monitor.MonitorStartCmd(ctx, con)
			con.Println()
			return nil
		},
	})
	monitorCmd.AddCommand(&grumble.Command{
		Name: "stop",
		Help: "Stop the monitoring loops",
		Run: func(ctx *grumble.Context) error {
			con.Println()
			monitor.MonitorStopCmd(ctx, con)
			con.Println()
			return nil
		},
	})
	con.App.AddCommand(monitorCmd)

	// [ Loot ] --------------------------------------------------------------

	lootCmd := &grumble.Command{
		Name:     consts.LootStr,
		Help:     "Manage the server's loot store",
		LongHelp: help.GetHelpFor([]string{consts.LootStr}),
		Flags: func(f *grumble.Flags) {
			f.String("f", "filter", "", "filter based on loot type")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			loot.LootCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	}
	lootCmd.AddCommand(&grumble.Command{
		Name:     consts.LootLocalStr,
		Help:     "Add a local file to the server's loot store",
		LongHelp: help.GetHelpFor([]string{consts.LootStr, consts.LootLocalStr}),
		Args: func(a *grumble.Args) {
			a.String("path", "The local file path to the loot")
		},
		Flags: func(f *grumble.Flags) {
			f.String("n", "name", "", "name of this piece of loot")
			f.String("T", "type", "", "force a specific loot type (file/cred)")
			f.String("F", "file-type", "", "force a specific file type (binary/text)")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			loot.LootAddLocalCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	lootCmd.AddCommand(&grumble.Command{
		Name:     consts.LootRemoteStr,
		Help:     "Add a remote file from the current session to the server's loot store",
		LongHelp: help.GetHelpFor([]string{consts.LootStr, consts.LootRemoteStr}),
		Args: func(a *grumble.Args) {
			a.String("path", "The local file path to the loot")
		},
		Flags: func(f *grumble.Flags) {
			f.String("n", "name", "", "name of this piece of loot")
			f.String("T", "type", "", "force a specific loot type (file/cred)")
			f.String("F", "file-type", "", "force a specific file type (binary/text)")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			loot.LootAddRemoteCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	lootCmd.AddCommand(&grumble.Command{
		Name:     consts.LootCredsStr,
		Help:     "Add credentials to the server's loot store",
		LongHelp: help.GetHelpFor([]string{consts.LootStr, consts.LootCredsStr}),
		Flags: func(f *grumble.Flags) {
			f.String("n", "name", "", "name of this piece of loot")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			loot.LootAddCredentialCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	lootCmd.AddCommand(&grumble.Command{
		Name:     consts.RenameStr,
		Help:     "Re-name a piece of existing loot",
		LongHelp: help.GetHelpFor([]string{consts.LootStr, consts.RenameStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			loot.LootRenameCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	lootCmd.AddCommand(&grumble.Command{
		Name:     consts.LootFetchStr,
		Help:     "Fetch a piece of loot from the server's loot store",
		LongHelp: help.GetHelpFor([]string{consts.LootStr, consts.LootFetchStr}),
		Flags: func(f *grumble.Flags) {
			f.String("s", "save", "", "save loot to a local file")
			f.String("f", "filter", "", "filter based on loot type")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			loot.LootFetchCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	lootCmd.AddCommand(&grumble.Command{
		Name:     consts.RmStr,
		Help:     "Remove a piece of loot from the server's loot store",
		LongHelp: help.GetHelpFor([]string{consts.LootStr, consts.RmStr}),
		Flags: func(f *grumble.Flags) {
			f.String("f", "filter", "", "filter based on loot type")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			loot.LootRmCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	con.App.AddCommand(lootCmd)

	// [ Reactions ] -----------------------------------------------------------------

	reactionCmd := &grumble.Command{
		Name:     consts.ReactionStr,
		Help:     "Manage automatic reactions to events",
		LongHelp: help.GetHelpFor([]string{consts.ReactionStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			reaction.ReactionCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	}
	reactionCmd.AddCommand(&grumble.Command{
		Name:     consts.SetStr,
		Help:     "Set a reaction to an event",
		LongHelp: help.GetHelpFor([]string{consts.ReactionStr, consts.SetStr}),
		Flags: func(f *grumble.Flags) {
			f.String("e", "event", "", "specify the event type to react to")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			reaction.ReactionSetCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	reactionCmd.AddCommand(&grumble.Command{
		Name:     consts.UnsetStr,
		Help:     "Unset an existing reaction",
		LongHelp: help.GetHelpFor([]string{consts.ReactionStr, consts.UnsetStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("i", "id", 0, "the id of the reaction to remove")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			reaction.ReactionUnsetCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	reactionCmd.AddCommand(&grumble.Command{
		Name:     consts.SaveStr,
		Help:     "Save current reactions to disk",
		LongHelp: help.GetHelpFor([]string{consts.ReactionStr, consts.SaveStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			reaction.ReactionSaveCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	reactionCmd.AddCommand(&grumble.Command{
		Name:     consts.ReloadStr,
		Help:     "Reload reactions from disk, replaces the running configuration",
		LongHelp: help.GetHelpFor([]string{consts.ReactionStr, consts.ReloadStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			reaction.ReactionReloadCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	con.App.AddCommand(reactionCmd)

	// [ DLL Hijack ] -----------------------------------------------------------------

	dllhijackCmd := &grumble.Command{
		Name:      consts.DLLHijackStr,
		Help:      "Plant a DLL for a hijack scenario",
		LongHelp:  help.GetHelpFor([]string{consts.DLLHijackStr}),
		HelpGroup: consts.SliverWinHelpGroup,
		Run: func(ctx *grumble.Context) error {
			con.Println()
			dllhijack.DllHijackCmd(ctx, con)
			con.Println()
			return nil
		},
		Args: func(a *grumble.Args) {
			a.String("target-path", "Path to upload the DLL to on the remote system")
		},
		Flags: func(f *grumble.Flags) {
			f.String("r", "reference-path", "", "Path to the reference DLL on the remote system")
			f.String("R", "reference-file", "", "Path to the reference DLL on the local system")
			f.String("f", "file", "", "Local path to the DLL to plant for the hijack")
			f.String("p", "profile", "", "Profile name to use as a base DLL")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
	}
	con.App.AddCommand(dllhijackCmd)
}
