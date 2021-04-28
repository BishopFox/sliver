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
	defaultWGLPort      = 53
	defaultWGNPort      = 8888
	defaultWGKeyExPort  = 1337
	defaultHTTPLPort    = 80
	defaultHTTPSLPort   = 443
	defaultDNSLPort     = 53
	defaultTCPPivotPort = 9898

	defaultReconnect = 60
	defaultPoll      = 1
	defaultMaxErrors = 1000

	defaultTimeout = 60
)

// BindCommands - Bind commands to a App
func BindCommands(app *grumble.App, rpc rpcpb.SliverRPCClient) {

	app.SetPrintHelp(helpCmd) // Responsible for display long-form help templates, etc.

	app.AddCommand(&grumble.Command{
		Name:     consts.UpdateStr,
		Help:     "Check for updates",
		LongHelp: help.GetHelpFor(consts.UpdateStr),
		Flags: func(f *grumble.Flags) {
			f.Bool("P", "prereleases", false, "include pre-released (unstable) versions")
			f.String("p", "proxy", "", "specify a proxy url (e.g. http://localhost:8080)")
			f.String("s", "save", "", "save downloaded files to specific directory (default user home dir)")
			f.Bool("I", "insecure", false, "skip tls certificate validation")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			updates(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.VersionStr,
		Help:     "Display version information",
		LongHelp: help.GetHelpFor(consts.VersionStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			verboseVersions(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

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
			f.Bool("p", "persistent", false, "make persistent across restarts")
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
		Name:     consts.WGStr,
		Help:     "Start a WireGuard listener",
		LongHelp: help.GetHelpFor(consts.WGStr),
		Flags: func(f *grumble.Flags) {
			f.Int("l", "lport", defaultWGLPort, "udp listen port")
			f.Int("n", "nport", defaultWGNPort, "virtual tun interface listen port")
			f.Int("x", "key-port", defaultWGKeyExPort, "virtual tun inteface key exchange port")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Bool("p", "persistent", false, "make persistent across restarts")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			startWGListener(ctx, rpc)
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
			f.Int("l", "lport", defaultDNSLPort, "udp listen port")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Bool("p", "persistent", false, "make persistent across restarts")
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
			f.Bool("p", "persistent", false, "make persistent across restarts")
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
			f.Bool("p", "persistent", false, "make persistent across restarts")
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
			f.Bool("C", "clean", false, "Clean out any sessions marked as [DEAD]")

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
			f.Bool("T", "token", false, "execute command with current token (windows only)")
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

	generateCmd := &grumble.Command{
		Name:     consts.GenerateStr,
		Help:     "Generate a sliver binary",
		LongHelp: help.GetHelpFor(consts.GenerateStr),
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
			f.String("t", "http", "", "http(s) connection strings")
			f.String("n", "dns", "", "dns connection strings")
			f.String("p", "named-pipe", "", "named-pipe connection strings")
			f.String("i", "tcp-pivot", "", "tcp-pivot connection strings")

			f.Int("X", "key-exchange", defaultWGKeyExPort, "wg key-exchange port")
			f.Int("T", "tcp-comms", defaultWGNPort, "wg c2 comms port")

			f.Int("j", "reconnect", defaultReconnect, "attempt to reconnect every n second(s)")
			f.Int("p", "poll", defaultPoll, "attempt to poll every n second(s)")
			f.Int("k", "max-errors", defaultMaxErrors, "max number of connection errors")

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
			fmt.Println()
			generate(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	}
	generateCmd.AddCommand(&grumble.Command{
		Name:     consts.StagerStr,
		Help:     "Generate a sliver stager using MSF",
		LongHelp: help.GetHelpFor(consts.StagerStr),
		Flags: func(f *grumble.Flags) {
			f.String("o", "os", "windows", "operating system")
			f.String("a", "arch", "amd64", "cpu architecture")
			f.String("l", "lhost", "", "Listening host")
			f.Int("p", "lport", 8443, "Listening port")
			f.String("r", "protocol", "tcp", "Staging protocol (tcp/http/https)")
			f.String("f", "format", "raw", "Output format (msfvenom formats, see `help generate stager` for the list)")
			f.String("b", "badchars", "", "bytes to exclude from stage shellcode")
			f.String("s", "save", "", "directory to save the generated stager to")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			generateStager(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	app.AddCommand(generateCmd)

	app.AddCommand(&grumble.Command{
		Name:     consts.StageListenerStr,
		Help:     "Start a stager listener",
		LongHelp: help.GetHelpFor(consts.StageListenerStr),
		Flags: func(f *grumble.Flags) {
			f.String("p", "profile", "", "Implant profile to link with the listener")
			f.String("u", "url", "", "URL to which the stager will call back to")
			f.String("c", "cert", "", "path to PEM encoded certificate file (HTTPS only)")
			f.String("k", "key", "", "path to PEM encoded private key file (HTTPS only)")
			f.Bool("e", "lets-encrypt", false, "attempt to provision a let's encrypt certificate (HTTPS only)")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			stageListener(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.NewProfileStr,
		Help:     "Save a new implant profile",
		LongHelp: help.GetHelpFor(consts.NewProfileStr),
		Flags: func(f *grumble.Flags) {
			f.String("o", "os", "windows", "operating system")
			f.String("a", "arch", "amd64", "cpu architecture")
			f.Bool("d", "debug", false, "enable debug features")
			f.Bool("e", "evasion", false, "enable evasion features")
			f.Bool("s", "skip-symbols", false, "skip symbol obfuscation")

			f.String("m", "mtls", "", "mtls domain(s)")
			f.String("g", "wg", "", "wg domain(s)")
			f.String("t", "http", "", "http[s] domain(s)")
			f.String("n", "dns", "", "dns domain(s)")
			f.String("e", "named-pipe", "", "named-pipe connection strings")
			f.String("i", "tcp-pivot", "", "tcp-pivot connection strings")

			f.Int("X", "key-exchange", defaultWGKeyExPort, "wg key-exchange port")
			f.Int("T", "tcp-comms", defaultWGNPort, "wg c2 comms port")

			f.String("c", "canary", "", "canary domain(s)")

			f.Int("j", "reconnect", defaultReconnect, "attempt to reconnect every n second(s)")
			f.Int("k", "max-errors", defaultMaxErrors, "max number of connection errors")

			f.String("w", "limit-datetime", "", "limit execution to before datetime")
			f.Bool("x", "limit-domainjoined", false, "limit execution to domain joined machines")
			f.String("y", "limit-username", "", "limit execution to specified username")
			f.String("z", "limit-hostname", "", "limit execution to specified hostname")
			f.String("F", "limit-fileexists", "", "limit execution to hosts with this file in the filesystem")

			f.String("f", "format", "exe", "Specifies the output formats, valid values are: 'exe', 'shared' (for dynamic libraries), 'service' (see `psexec` for more info) and 'shellcode' (windows only)")

			f.String("p", "profile-name", "", "profile name")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			newProfile(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.RegenerateStr,
		Help:      "Regenerate an implant",
		LongHelp:  help.GetHelpFor(consts.RegenerateStr),
		AllowArgs: true,
		Flags: func(f *grumble.Flags) {
			f.String("s", "save", "", "directory/file to the binary to")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			regenerate(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	profilesCmd := &grumble.Command{
		Name:     consts.ProfilesStr,
		Help:     "List existing profiles",
		LongHelp: help.GetHelpFor(consts.ProfilesStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			profiles(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	}
	profilesCmd.AddCommand(&grumble.Command{
		Name:     consts.RmStr,
		Help:     "Remove a profile",
		LongHelp: help.GetHelpFor(fmt.Sprintf("%s.%s", consts.ProfilesStr, consts.RmStr)),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			rmProfile(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	app.AddCommand(profilesCmd)

	app.AddCommand(&grumble.Command{
		Name:     consts.ProfileGenerateStr,
		Help:     "Generate implant from a profile",
		LongHelp: help.GetHelpFor(consts.ProfileGenerateStr),
		Flags: func(f *grumble.Flags) {
			f.String("p", "name", "", "profile name")
			f.String("s", "save", "", "directory/file to the binary to")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			profileGenerate(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	implantBuildsCmd := &grumble.Command{
		Name:     consts.ImplantBuildsStr,
		Help:     "List implant builds",
		LongHelp: help.GetHelpFor(consts.ImplantBuildsStr),
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
	}
	implantBuildsCmd.AddCommand(&grumble.Command{
		Name:     consts.RmStr,
		Help:     "Remove implant build",
		LongHelp: help.GetHelpFor(fmt.Sprintf("%s.%s", consts.ImplantBuildsStr, consts.RmStr)),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			rmImplantBuild(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	app.AddCommand(implantBuildsCmd)

	app.AddCommand(&grumble.Command{
		Name:     consts.ListCanariesStr,
		Help:     "List previously generated canaries",
		LongHelp: help.GetHelpFor(consts.ListCanariesStr),
		Flags: func(f *grumble.Flags) {
			f.Bool("b", "burned", false, "show only triggered/burned canaries")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			canaries(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
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
			f.Bool("r", "recursive", false, "recursively remove files")
			f.Bool("f", "force", false, "ignore safety and forcefully remove files")

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

	app.AddCommand(&grumble.Command{
		Name:      consts.CatStr,
		Help:      "Dump file to stdout",
		LongHelp:  help.GetHelpFor(consts.CatStr),
		AllowArgs: true,
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Bool("c", "colorize-output", false, "colorize output")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			cat(ctx, rpc)
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
		Name:     consts.IfconfigStr,
		Help:     "View network interface configurations",
		LongHelp: help.GetHelpFor(consts.IfconfigStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			ifconfig(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.NetstatStr,
		Help:     "Print network connection information",
		LongHelp: help.GetHelpFor(consts.NetstatStr),
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			netstat(ctx, rpc)
			fmt.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Bool("t", "tcp", true, "display information about TCP sockets")
			f.Bool("u", "udp", false, "display information about UDP sockets")
			f.Bool("4", "ip4", true, "display information about IPv4 sockets")
			f.Bool("6", "ip6", false, "display information about IPv6 sockets")
			f.Bool("l", "listen", false, "display information about listening sockets")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
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
			f.String("m", "method", "", "Optional method (a method is required for a .NET DLL)")
			f.String("c", "class", "", "Optional class name (required for .NET DLL)")
			f.String("d", "app-domain", "", "AppDomain name to create for .NET assembly. Generated randomly if not set.")
			f.String("a", "arch", "x84", "Assembly target architecture: x86, x64, x84 (x86+x64)")
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
			f.Bool("k", "keep-alive", false, "don't terminate host process once the execution completes")
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
			f.Bool("k", "keep-alive", false, "don't terminate host process once the execution completes")
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
			f.Bool("r", "recursive", false, "recursively add/rm content")
			f.String("w", "website", "", "website name")
			f.String("p", "web-path", "", "http path to host file at")

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
			f.String("w", "website", "", "website name")
			f.String("m", "content-type", "", "mime content-type (if blank use file ext.)")
			f.String("p", "web-path", "/", "http path to host file at")
			f.String("c", "content", "", "local file path/dir (must use --recursive for dir)")
			f.Bool("r", "recursive", false, "recursively add/rm content")

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
			f.String("w", "website", "", "website name")
			f.String("m", "content-type", "", "mime content-type (if blank use file ext.)")
			f.String("p", "web-path", "/", "http path to host file at")

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
			f.Int("r", "reconnect", -1, "reconnect interval for agent")
			f.Int("p", "poll", -1, "poll interval for agent")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			updateSessionCmd(ctx, rpc)
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
		Name:      consts.SetEnvStr,
		Help:      "Set environment variables",
		LongHelp:  help.GetHelpFor(consts.SetEnvStr),
		AllowArgs: true,
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			setEnv(ctx, rpc)
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

	registryCmd := &grumble.Command{
		Name:     consts.RegistryStr,
		Help:     "Windows registry operations",
		LongHelp: help.GetHelpFor(consts.RegistryStr),
		Run: func(ctx *grumble.Context) error {
			return nil
		},
		HelpGroup: consts.SliverWinHelpGroup,
	}

	registryCmd.AddCommand(&grumble.Command{
		Name:     consts.RegistryReadStr,
		Help:     "Read values from the Windows registry",
		LongHelp: help.GetHelpFor(consts.RegistryReadStr),
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			registryReadCmd(ctx, rpc)
			fmt.Println()
			return nil
		},
		AllowArgs: true,
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.String("H", "hive", "HKCU", "egistry hive")
			f.String("o", "hostname", "", "remote host to read values from")
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})
	registryCmd.AddCommand(&grumble.Command{
		Name:      consts.RegistryWriteStr,
		Help:      "Write values to the Windows registry",
		LongHelp:  help.GetHelpFor(consts.RegistryWriteStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			registryWriteCmd(ctx, rpc)
			fmt.Println()
			return nil
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
		Name:      consts.RegistryCreateKeyStr,
		Help:      "Create a registry key",
		LongHelp:  help.GetHelpFor(consts.RegistryCreateKeyStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			regCreateKeyCmd(ctx, rpc)
			fmt.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.String("H", "hive", "HKCU", "registry hive")
			f.String("o", "hostname", "", "remote host to write values to")
		},
	})
	app.AddCommand(registryCmd)

	app.AddCommand(&grumble.Command{
		Name:     consts.PivotsListStr,
		Help:     "List pivots",
		LongHelp: help.GetHelpFor(consts.PivotsListStr),
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			listPivots(ctx, rpc)
			fmt.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.String("i", "id", "", "session id")
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	// [ WireGuard ] --------------------------------------------------------------

	app.AddCommand(&grumble.Command{
		Name:     consts.WgConfigStr,
		Help:     "Generate a new WireGuard client config",
		LongHelp: help.GetHelpFor(consts.WgConfigStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.String("s", "save", "", "save configuration to file (.conf)")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			getWGClientConfig(ctx, rpc)
			fmt.Println()
			return nil
		},
	})

	wgPortFwdCmd := &grumble.Command{
		Name:     consts.WgPortFwdStr,
		Help:     "List ports forwarded by the WireGuard tun interface",
		LongHelp: help.GetHelpFor(consts.WgPortFwdStr),
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			wgPortFwdListCmd(ctx, rpc)
			fmt.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
	}
	wgPortFwdCmd.AddCommand(&grumble.Command{
		Name:     "add",
		Help:     "Add a port forward from the WireGuard tun interface to a host on the target network",
		LongHelp: help.GetHelpFor(consts.WgPortFwdStr),
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			wgPortFwdAddCmd(ctx, rpc)
			fmt.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Int("b", "bind", 1080, "port to listen on the WireGuard tun interface")
			f.String("r", "remote", "", "remote target host:port (e.g., 10.0.0.1:445)")
		},
	})
	wgPortFwdCmd.AddCommand(&grumble.Command{
		Name:      "rm",
		Help:      "Remove a port forward from the WireGuard tun interface",
		LongHelp:  help.GetHelpFor(consts.WgPortFwdStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			wgPortFwdRmCmd(ctx, rpc)
			fmt.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
	})
	app.AddCommand(wgPortFwdCmd)

	wgSocksCmd := &grumble.Command{
		Name:     consts.WgSocksStr,
		Help:     "List socks servers listening on the WireGuard tun interface",
		LongHelp: help.GetHelpFor(consts.WgSocksStr),
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			wgSocksListCmd(ctx, rpc)
			fmt.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
	}

	wgSocksCmd.AddCommand(&grumble.Command{
		Name:     "start",
		Help:     "Start a socks5 listener on the WireGuard tun interface",
		LongHelp: help.GetHelpFor(consts.WgSocksStr),
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			wgSocksStartCmd(ctx, rpc)
			fmt.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Int("b", "bind", 3090, "port to listen on the WireGuard tun interface")
		},
	})

	wgSocksCmd.AddCommand(&grumble.Command{
		Name:      "rm",
		Help:      "Stop a socks5 listener on the WireGuard tun interface",
		LongHelp:  help.GetHelpFor(consts.WgSocksStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Print()
			wgSocksRmCmd(ctx, rpc)
			fmt.Print()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
	})
	app.AddCommand(wgSocksCmd)

	// [ Portfwd ] --------------------------------------------------------------
	portfwdCmd := &grumble.Command{
		Name:     consts.PortfwdStr,
		Help:     "In-band TCP port forwarding",
		LongHelp: help.GetHelpFor(consts.PortfwdStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			portfwd(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	}
	portfwdCmd.AddCommand(&grumble.Command{
		Name:     "add",
		Help:     "Create a new port forwarding tunnel",
		LongHelp: help.GetHelpFor(consts.PortfwdStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.String("r", "remote", "", "remote target host:port (e.g., 10.0.0.1:445)")
			f.String("b", "bind", "127.0.0.1:8080", "bind port forward to interface")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			portfwdAdd(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})
	portfwdCmd.AddCommand(&grumble.Command{
		Name:     "rm",
		Help:     "Remove a port forwarding tunnel",
		LongHelp: help.GetHelpFor(consts.PortfwdStr),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Int("i", "id", 0, "id of portfwd to remove")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			portfwdRm(ctx, rpc)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})
	app.AddCommand(portfwdCmd)
}
