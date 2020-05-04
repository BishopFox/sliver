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
	defaultMTLSLPort  = 8888
	defaultHTTPLPort  = 80
	defaultHTTPSLPort = 443
	defaultTCPPort    = 4444

	defaultReconnect = 60
	defaultMaxErrors = 1000

	defaultTimeout = 60
)

// BindCommands - Bind commands to a App
func BindCommands(app *grumble.App, rpc rpcpb.SliverRPCClient) {

	app.SetPrintHelp(helpCmd) // Responsible for display long-form help templates, etc.

	// [ Jobs ] -----------------------------------------------------------------
	app.AddCommand(&grumble.Command{
		Name:     consts.JobsStr,
		Help:     "Job control",
		LongHelp: help.GetHelpFor(consts.JobsStr),
		Flags: func(f *grumble.Flags) {
			f.Int("k", "kill", -1, "kill a background job")
			f.Bool("K", "kill-all", false, "kill all jobs")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			jobs(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.MtlsStr,
		Help:     "Start an mTLS listener",
		LongHelp: help.GetHelpFor(consts.MtlsStr),
		Flags: func(f *grumble.Flags) {
			f.String("s", "server", "", "interface to bind server to")
			f.Int("l", "lport", defaultMTLSLPort, "tcp listen port")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			startMTLSListener(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.DnsStr,
		Help:     "Start a DNS listener",
		LongHelp: help.GetHelpFor(consts.DnsStr),
		Flags: func(f *grumble.Flags) {
			f.String("d", "domains", "", "parent domain(s) to use for DNS c2")
			f.Bool("c", "no-canaries", false, "disable dns canary detection")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			startDNSListener(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.HttpStr,
		Help:     "Start an HTTP listener",
		LongHelp: help.GetHelpFor(consts.HttpStr),
		Flags: func(f *grumble.Flags) {
			f.String("d", "domain", "", "limit responses to specific domain")
			f.String("w", "website", "", "website name (see websites cmd)")
			f.Int("l", "lport", defaultHTTPLPort, "tcp listen port")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			startHTTPListener(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.HttpsStr,
		Help:     "Start an HTTPS listener",
		LongHelp: help.GetHelpFor(consts.HttpsStr),
		Flags: func(f *grumble.Flags) {
			f.String("d", "domain", "", "limit responses to specific domain")
			f.String("w", "website", "", "website name (see websites cmd)")
			f.Int("l", "lport", defaultHTTPSLPort, "tcp listen port")

			f.String("c", "cert", "", "PEM encoded certificate file")
			f.String("k", "key", "", "PEM encoded private key file")

			f.Bool("e", "lets-encrypt", false, "attempt to provision a let's encrypt certificate")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			startHTTPSListener(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.PlayersStr,
		Help:     "List operators",
		LongHelp: help.GetHelpFor(consts.PlayersStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			operatorsCmd(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.MultiplayerHelpGroup,
	})

	// [ Commands ] --------------------------------------------------------------

	app.AddCommand(&grumble.Command{
		Name:     consts.SessionsStr,
		Help:     "Session management",
		LongHelp: help.GetHelpFor(consts.SessionsStr),
		Flags: func(f *grumble.Flags) {
			f.String("i", "interact", "", "interact with a sliver")
			f.String("k", "kill", "", "Kill the designated session")
			f.Bool("K", "kill-all", false, "Kill all the sessions")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			sessions(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.BackgroundStr,
		Help:     "Background an active session",
		LongHelp: help.GetHelpFor(consts.BackgroundStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			background(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.KillStr,
		Help:      "Kill a session",
		LongHelp:  help.GetHelpFor(consts.KillStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			kill(ctx, rpc)
			fmt.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Bool("f", "force", false, "Force kill,  does not clean up")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.InfoStr,
		Help:     "Get info about session",
		LongHelp: help.GetHelpFor(consts.InfoStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			info(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.UseStr,
		Help:     "Switch the active session",
		LongHelp: help.GetHelpFor(consts.UseStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			use(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

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
			f.Bool("o", "output", false, "print the command output")
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
		Name:     consts.GenerateStr,
		Help:     "Generate a sliver binary",
		LongHelp: help.GetHelpFor(consts.GenerateStr),
		Flags: func(f *grumble.Flags) {
			f.String("o", "os", "windows", "operating system")
			f.String("a", "arch", "amd64", "cpu architecture")
			f.Bool("d", "debug", false, "enable debug features")
			f.Bool("b", "skip-symbols", false, "skip symbol obfuscation")

			f.String("c", "canary", "", "canary domain(s)")

			f.String("m", "mtls", "", "mtls connection strings")
			f.String("t", "http", "", "http(s) connection strings")
			f.String("n", "dns", "", "dns connection strings")

			f.Int("j", "reconnect", defaultReconnect, "attempt to reconnect every n second(s)")
			f.Int("k", "max-errors", defaultMaxErrors, "max number of connection errors")

			f.String("w", "limit-datetime", "", "limit execution to before datetime")
			f.Bool("x", "limit-domainjoined", false, "limit execution to domain joined machines")
			f.String("y", "limit-username", "", "limit execution to specified username")
			f.String("z", "limit-hostname", "", "limit execution to specified hostname")

			f.String("r", "format", "exe", "Specifies the output formats, valid values are: 'exe', 'shared' (for dynamic libraries) and 'shellcode' (windows only)")

			f.String("s", "save", "", "directory/file to the binary to")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			generate(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	// app.AddCommand(&grumble.Command{
	// 	Name:     consts.GenerateEggStr,
	// 	Help:     "Generate an egg shellcode (sliver stager)",
	// 	LongHelp: help.GetHelpFor(consts.GenerateEggStr),
	// 	Flags: func(f *grumble.Flags) {
	// 		f.String("o", "os", "windows", "operating system")
	// 		f.String("a", "arch", "amd64", "cpu architecture")
	// 		f.Bool("d", "debug", false, "enable debug features")

	// 		f.String("m", "mtls", "", "mtls connection strings")
	// 		f.String("t", "http", "", "http(s) connection strings")
	// 		f.String("n", "dns", "", "dns connection strings")

	// 		f.Int("j", "reconnect", 60, "attempt to reconnect every n second(s)")
	// 		f.Int("k", "max-errors", 1000, "max number of connection errors")

	// 		f.String("w", "limit-datetime", "", "limit execution to before datetime")
	// 		f.Bool("x", "limit-domainjoined", false, "limit execution to domain joined machines")
	// 		f.String("y", "limit-username", "", "limit execution to specified username")
	// 		f.String("z", "limit-hostname", "", "limit execution to specified hostname")

	// 		f.String("r", "format", "shellcode", "Fixed to 'shellcode' - do not change") // TODO: find a better way to handle this

	// 		f.String("s", "save", "", "directory to save the egg to")
	// 		f.String("c", "listener-url", "", "URL to fetch the stage from (tcp://SLIVER_SERVER:PORT or http(s)://SLIVER_SERVER:PORT")
	// 		f.String("v", "output-format", "raw", "Output format (msfvenom's style). All msfvenom's transform formats are supported")
	// 		f.String("x", "canary", "", "canary domain(s)")
	// 		f.Bool("s", "skip-symbols", false, "skip symbol obfuscation")
	// 	},
	// 	Run: func(ctx *grumble.Context) error {
	// 		fmt.Println()
	// 		generateEgg(ctx, rpc)
	// 		fmt.Println()
	// 		return nil
	// 	},
	// 	HelpGroup: consts.GenericHelpGroup,
	// })

	// app.AddCommand(&grumble.Command{
	// 	Name:     consts.NewProfileStr,
	// 	Help:     "Save a new sliver profile",
	// 	LongHelp: help.GetHelpFor(consts.NewProfileStr),
	// 	Flags: func(f *grumble.Flags) {
	// 		f.String("o", "os", "windows", "operating system")
	// 		f.String("a", "arch", "amd64", "cpu architecture")
	// 		f.Bool("d", "debug", false, "enable debug features")
	// 		f.Bool("s", "skip-symbols", false, "skip symbol obfuscation")

	// 		f.String("m", "mtls", "", "mtls domain(s)")
	// 		f.String("t", "http", "", "http[s] domain(s)")
	// 		f.String("n", "dns", "", "dns domain(s)")

	// 		f.String("c", "canary", "", "canary domain(s)")

	// 		f.Int("j", "reconnect", defaultReconnect, "attempt to reconnect every n second(s)")
	// 		f.Int("k", "max-errors", defaultMaxErrors, "max number of connection errors")

	// 		f.String("w", "limit-datetime", "", "limit execution to before datetime")
	// 		f.Bool("x", "limit-domainjoined", false, "limit execution to domain joined machines")
	// 		f.String("y", "limit-username", "", "limit execution to specified username")
	// 		f.String("z", "limit-hostname", "", "limit execution to specified hostname")

	// 		f.String("r", "format", "exe", "Specifies the output formats, valid values are: 'exe', 'shared' (for dynamic libraries) and 'shellcode' (windows only)")

	// 		f.String("p", "name", "", "profile name")
	// 	},
	// 	Run: func(ctx *grumble.Context) error {
	// 		fmt.Println()
	// 		newProfile(ctx, rpc)
	// 		fmt.Println()
	// 		return nil
	// 	},
	// 	HelpGroup: consts.GenericHelpGroup,
	// })

	// app.AddCommand(&grumble.Command{
	// 	Name:      consts.RegenerateStr,
	// 	Help:      "Regenerate target sliver",
	// 	LongHelp:  help.GetHelpFor(consts.RegenerateStr),
	// 	AllowArgs: true,
	// 	Flags: func(f *grumble.Flags) {
	// 		f.String("s", "save", "", "directory/file to the binary to")
	// 	},
	// 	Run: func(ctx *grumble.Context) error {
	// 		fmt.Println()
	// 		regenerate(ctx, rpc)
	// 		fmt.Println()
	// 		return nil
	// 	},
	// 	HelpGroup: consts.GenericHelpGroup,
	// })

	// app.AddCommand(&grumble.Command{
	// 	Name:     consts.ProfilesStr,
	// 	Help:     "List existing profiles",
	// 	LongHelp: help.GetHelpFor(consts.ProfilesStr),
	// 	Run: func(ctx *grumble.Context) error {
	// 		fmt.Println()
	// 		profiles(ctx, rpc)
	// 		fmt.Println()
	// 		return nil
	// 	},
	// 	HelpGroup: consts.GenericHelpGroup,
	// })

	// app.AddCommand(&grumble.Command{
	// 	Name:     consts.ProfileGenerateStr,
	// 	Help:     "Generate Sliver from a profile",
	// 	LongHelp: help.GetHelpFor(consts.ProfileGenerateStr),
	// 	Flags: func(f *grumble.Flags) {
	// 		f.String("p", "name", "", "profile name")
	// 		f.String("s", "save", "", "directory/file to the binary to")
	// 	},
	// 	AllowArgs: true,
	// 	Run: func(ctx *grumble.Context) error {
	// 		fmt.Println()
	// 		profileGenerate(ctx, rpc)
	// 		fmt.Println()
	// 		return nil
	// 	},
	// 	HelpGroup: consts.GenericHelpGroup,
	// })

	app.AddCommand(&grumble.Command{
		Name:     consts.ListSliverBuildsStr,
		Help:     "List old implant builds",
		LongHelp: help.GetHelpFor(consts.ListSliverBuildsStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			listImplantBuilds(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	// app.AddCommand(&grumble.Command{
	// 	Name:     consts.ListCanariesStr,
	// 	Help:     "List previously generated canaries",
	// 	LongHelp: help.GetHelpFor(consts.ListCanariesStr),
	// 	Flags: func(f *grumble.Flags) {
	// 		f.Bool("b", "burned", false, "show only triggered/burned canaries")
	// 	},
	// 	AllowArgs: true,
	// 	Run: func(ctx *grumble.Context) error {
	// 		fmt.Println()
	// 		canaries(ctx, rpc)
	// 		fmt.Println()
	// 		return nil
	// 	},
	// 	HelpGroup: consts.GenericHelpGroup,
	// })

	app.AddCommand(&grumble.Command{
		Name:     consts.MsfStr,
		Help:     "Execute an MSF payload in the current process",
		LongHelp: help.GetHelpFor(consts.MsfStr),
		Flags: func(f *grumble.Flags) {
			f.String("m", "payload", "meterpreter_reverse_https", "msf payload")
			f.String("h", "lhost", "", "listen host")
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
			f.String("h", "lhost", "", "listen host")
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
		Name:     consts.PingStr,
		Help:     "Send round trip message to implant (does not use ICMP)",
		LongHelp: help.GetHelpFor(consts.PingStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			ping(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.GetPIDStr,
		Help:     "Get session pid",
		LongHelp: help.GetHelpFor(consts.GetPIDStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			getPID(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.GetUIDStr,
		Help:     "Get session process UID",
		LongHelp: help.GetHelpFor(consts.GetUIDStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			getUID(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.GetGIDStr,
		Help:     "Get session process GID",
		LongHelp: help.GetHelpFor(consts.GetGIDStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			getGID(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.WhoamiStr,
		Help:     "Get session user execution context",
		LongHelp: help.GetHelpFor(consts.WhoamiStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			whoami(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.LsStr,
		Help:     "List current directory",
		LongHelp: help.GetHelpFor(consts.LsStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			ls(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.RmStr,
		Help:     "Remove a file or directory",
		LongHelp: help.GetHelpFor(consts.RmStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			rm(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.MkdirStr,
		Help:     "Make a directory",
		LongHelp: help.GetHelpFor(consts.MkdirStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			mkdir(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.CdStr,
		Help:     "Change directory",
		LongHelp: help.GetHelpFor(consts.CdStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			cd(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.PwdStr,
		Help:     "Print working directory",
		LongHelp: help.GetHelpFor(consts.PwdStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			pwd(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	// 	app.AddCommand(&grumble.Command{
	// 		Name:      consts.CatStr,
	// 		Help:      "Dump file to stdout",
	// 		LongHelp:  help.GetHelpFor(consts.CatStr),
	// 		AllowArgs: true,
	// 		Run: func(ctx *grumble.Context) error {
	// 			fmt.Println()
	// 			cat(ctx, rpc)
	// 			fmt.Println()
	// 			return nil
	// 		},
	// 		HelpGroup: consts.SliverHelpGroup,
	// 	})

	// 	app.AddCommand(&grumble.Command{
	// 		Name:     consts.DownloadStr,
	// 		Help:     "Download a file",
	// 		LongHelp: help.GetHelpFor(consts.DownloadStr),
	// 		Flags: func(f *grumble.Flags) {
	// 			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
	// 		},
	// 		AllowArgs: true,
	// 		Run: func(ctx *grumble.Context) error {
	// 			fmt.Println()
	// 			download(ctx, rpc)
	// 			fmt.Println()
	// 			return nil
	// 		},
	// 		HelpGroup: consts.SliverHelpGroup,
	// 	})

	// 	app.AddCommand(&grumble.Command{
	// 		Name:     consts.UploadStr,
	// 		Help:     "Upload a file",
	// 		LongHelp: help.GetHelpFor(consts.UploadStr),
	// 		Flags: func(f *grumble.Flags) {
	// 			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
	// 		},
	// 		AllowArgs: true,
	// 		Run: func(ctx *grumble.Context) error {
	// 			fmt.Println()
	// 			upload(ctx, rpc)
	// 			fmt.Println()
	// 			return nil
	// 		},
	// 		HelpGroup: consts.SliverHelpGroup,
	// 	})

	// 	app.AddCommand(&grumble.Command{
	// 		Name:     consts.IfconfigStr,
	// 		Help:     "View network interface configurations",
	// 		LongHelp: help.GetHelpFor(consts.IfconfigStr),
	// 		Run: func(ctx *grumble.Context) error {
	// 			fmt.Println()
	// 			ifconfig(ctx, rpc)
	// 			fmt.Println()
	// 			return nil
	// 		},
	// 		HelpGroup: consts.SliverHelpGroup,
	// 	})

	// 	app.AddCommand(&grumble.Command{
	// 		Name:     consts.NetstatStr,
	// 		Help:     "Print network connection information",
	// 		LongHelp: help.GetHelpFor(consts.NetstatStr),
	// 		Run: func(ctx *grumble.Context) error {
	// 			fmt.Println()
	// 			netstat(ctx, rpc)
	// 			fmt.Println()
	// 			return nil
	// 		},
	// 		Flags: func(f *grumble.Flags) {
	// 			f.Bool("t", "tcp", true, "display information about TCP sockets")
	// 			f.Bool("u", "udp", false, "display information about UDP sockets")
	// 			f.Bool("4", "ip4", true, "display information about IPv4 sockets")
	// 			f.Bool("6", "ip6", false, "display information about IPv6 sockets")
	// 			f.Bool("l", "listen", false, "display information about listening sockets")
	// 		},
	// 		HelpGroup: consts.SliverHelpGroup,
	// 	})

	// 	app.AddCommand(&grumble.Command{
	// 		Name:     consts.ProcdumpStr,
	// 		Help:     "Dump process memory",
	// 		LongHelp: help.GetHelpFor(consts.ProcdumpStr),
	// 		Flags: func(f *grumble.Flags) {
	// 			f.Int("p", "pid", -1, "target pid")
	// 			f.String("n", "name", "", "target process name")
	// 			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
	// 		},
	// 		Run: func(ctx *grumble.Context) error {
	// 			fmt.Println()
	// 			procdump(ctx, rpc)
	// 			fmt.Println()
	// 			return nil
	// 		},
	// 		HelpGroup: consts.SliverHelpGroup,
	// 	})

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
			f.String("p", "process", "notepad.exe", "Hosting process to inject into")
			f.Bool("a", "amsi", true, "Use AMSI bypass")
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

	// 	app.AddCommand(&grumble.Command{
	// 		Name:     consts.WebsitesStr,
	// 		Help:     "Host a static file on a website (used with HTTP C2)",
	// 		LongHelp: help.GetHelpFor(consts.WebsitesStr),
	// 		Flags: func(f *grumble.Flags) {
	// 			f.String("w", "website", "", "website name")
	// 			f.String("t", "content-type", "", "mime content-type (if blank use file ext.)")
	// 			f.String("p", "web-path", "/", "http path to host file at")
	// 			f.String("c", "content", "", "local file path/dir (must use --recursive for dir)")
	// 			f.Bool("r", "recursive", false, "recursively add content from dir, --web-path is prefixed")
	// 		},
	// 		AllowArgs: true,
	// 		Run: func(ctx *grumble.Context) error {
	// 			fmt.Println()
	// 			websites(ctx, rpc)
	// 			fmt.Println()
	// 			return nil
	// 		},
	// 		HelpGroup: consts.GenericHelpGroup,
	// 	})

	// 	app.AddCommand(&grumble.Command{
	// 		Name:      consts.TerminateStr,
	// 		Help:      "Kill a process",
	// 		LongHelp:  help.GetHelpFor(consts.TerminateStr),
	// 		AllowArgs: true,
	// 		Run: func(ctx *grumble.Context) error {
	// 			fmt.Println()
	// 			terminate(ctx, rpc)
	// 			fmt.Println()
	// 			return nil
	// 		},
	// 		HelpGroup: consts.GenericHelpGroup,
	// 	})

	// 	app.AddCommand(&grumble.Command{
	// 		Name:      consts.ScreenshotStr,
	// 		Help:      "Take a screenshot",
	// 		LongHelp:  help.GetHelpFor(consts.ScreenshotStr),
	// 		AllowArgs: false,
	// 		Run: func(ctx *grumble.Context) error {
	// 			fmt.Println()
	// 			screenshot(ctx, rpc)
	// 			fmt.Println()
	// 			return nil
	// 		},
	// 		HelpGroup: consts.SliverHelpGroup,
	// 	})

}
