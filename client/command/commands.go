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

	---------------------------------------------------------------------
	This file contains all of the code that binds a given string/flags/etc. to a
	command implementation function.

	Guidelines when adding a command:

		* Try to reuse the same short/long flags for the same parameter,
		  e.g. "timeout" flags should always be -t and --timeout when possible.
		  Try to avoid creating flags that conflict with others even if you're
		  not using the flag, e.g. avoid using -t even if your command doesn't
		  have a --timeout.

		* Add a long-form help template to `client/help`

*/

import (
	"os"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/alias"
	"github.com/bishopfox/sliver/client/command/armory"
	"github.com/bishopfox/sliver/client/command/backdoor"
	"github.com/bishopfox/sliver/client/command/beacons"
	"github.com/bishopfox/sliver/client/command/builders"
	"github.com/bishopfox/sliver/client/command/completers"
	"github.com/bishopfox/sliver/client/command/cursed"
	"github.com/bishopfox/sliver/client/command/dllhijack"
	"github.com/bishopfox/sliver/client/command/environment"
	"github.com/bishopfox/sliver/client/command/exec"
	"github.com/bishopfox/sliver/client/command/extensions"
	"github.com/bishopfox/sliver/client/command/filesystem"
	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/command/hosts"
	"github.com/bishopfox/sliver/client/command/info"
	"github.com/bishopfox/sliver/client/command/jobs"
	"github.com/bishopfox/sliver/client/command/kill"
	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/command/monitor"
	"github.com/bishopfox/sliver/client/command/network"
	"github.com/bishopfox/sliver/client/command/operators"
	"github.com/bishopfox/sliver/client/command/pivots"
	"github.com/bishopfox/sliver/client/command/portfwd"
	operator "github.com/bishopfox/sliver/client/command/prelude-operator"
	"github.com/bishopfox/sliver/client/command/privilege"
	"github.com/bishopfox/sliver/client/command/processes"
	"github.com/bishopfox/sliver/client/command/reaction"
	"github.com/bishopfox/sliver/client/command/reconfig"
	"github.com/bishopfox/sliver/client/command/registry"
	"github.com/bishopfox/sliver/client/command/rportfwd"
	"github.com/bishopfox/sliver/client/command/screenshot"
	"github.com/bishopfox/sliver/client/command/sessions"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/command/shell"
	sgn "github.com/bishopfox/sliver/client/command/shikata-ga-nai"
	"github.com/bishopfox/sliver/client/command/socks"
	"github.com/bishopfox/sliver/client/command/tasks"
	"github.com/bishopfox/sliver/client/command/update"
	"github.com/bishopfox/sliver/client/command/use"
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

	// Load Reactions
	n, err := reaction.LoadReactions()
	if err != nil && !os.IsNotExist(err) {
		con.PrintErrorf("Failed to load reactions: %s\n", err)
	} else if n > 0 {
		con.PrintInfof("Loaded %d reaction(s) from disk\n", n)
	}

	// Load Aliases
	aliasManifests := assets.GetInstalledAliasManifests()
	n = 0
	for _, manifest := range aliasManifests {
		_, err = alias.LoadAlias(manifest, con)
		if err != nil {
			con.PrintErrorf("Failed to load alias: %s\n", err)
			continue
		}
		n++
	}
	if 0 < n {
		if n == 1 {
			con.PrintInfof("Loaded %d alias from disk\n", n)
		} else {
			con.PrintInfof("Loaded %d aliases from disk\n", n)
		}
	}

	// Load Extensions
	extensionManifests := assets.GetInstalledExtensionManifests()
	n = 0
	for _, manifest := range extensionManifests {
		ext, err := extensions.LoadExtensionManifest(manifest)
		// Absorb error in case there's no extensions manifest
		if err != nil {
			con.PrintErrorf("Failed to load extension: %s\n", err)
			continue
		}
		extensions.ExtensionRegisterCommand(ext, con)
		n++
	}
	if 0 < n {
		con.PrintInfof("Loaded %d extension(s) from disk\n", n)
	}
	con.App.SetPrintHelp(help.HelpCmd(con)) // Responsible for display long-form help templates, etc.

	// [ Aliases ] ---------------------------------------------

	aliasCmd := &grumble.Command{
		Name:     consts.AliasesStr,
		Help:     "List current aliases",
		LongHelp: help.GetHelpFor([]string{consts.AliasesStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			alias.AliasesCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	}
	con.App.AddCommand(aliasCmd)

	aliasCmd.AddCommand(&grumble.Command{
		Name:     consts.LoadStr,
		Help:     "Load a command alias",
		LongHelp: help.GetHelpFor([]string{consts.AliasesStr, consts.LoadStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			alias.AliasesLoadCmd(ctx, con)
			con.Println()
			return nil
		},
		Args: func(a *grumble.Args) {
			a.String("dir-path", "path to the alias directory")
		},
		Completer: func(prefix string, args []string) []string {
			return completers.LocalPathCompleter(prefix, args, con)
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	aliasCmd.AddCommand(&grumble.Command{
		Name:     consts.InstallStr,
		Help:     "Install a command alias",
		LongHelp: help.GetHelpFor([]string{consts.AliasesStr, consts.InstallStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			alias.AliasesInstallCmd(ctx, con)
			con.Println()
			return nil
		},
		Args: func(a *grumble.Args) {
			a.String("path", "path to the alias directory or tar.gz file")
		},
		Completer: func(prefix string, args []string) []string {
			return completers.LocalPathCompleter(prefix, args, con)
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	aliasCmd.AddCommand(&grumble.Command{
		Name:     consts.RmStr,
		Help:     "Remove an alias",
		LongHelp: help.GetHelpFor([]string{consts.RmStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			alias.AliasesRemoveCmd(ctx, con)
			con.Println()
			return nil
		},
		Args: func(a *grumble.Args) {
			a.String("name", "name of the alias to remove")
		},
		Completer: func(prefix string, args []string) []string {
			return alias.AliasCommandNameCompleter(prefix, args, con)
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	// [ Armory ] ---------------------------------------------

	armoryCmd := &grumble.Command{
		Name:     consts.ArmoryStr,
		Help:     "Automatically download and install extensions/aliases",
		LongHelp: help.GetHelpFor([]string{consts.ArmoryStr}),
		Flags: func(f *grumble.Flags) {
			f.Bool("I", "insecure", false, "skip tls certificate validation")
			f.String("p", "proxy", "", "specify a proxy url (e.g. http://localhost:8080)")
			f.Bool("c", "ignore-cache", false, "ignore metadata cache, force refresh")
			f.String("t", "timeout", "15m", "download timeout")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			armory.ArmoryCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	}
	con.App.AddCommand(armoryCmd)

	armoryCmd.AddCommand(&grumble.Command{
		Name:     consts.InstallStr,
		Help:     "Install an alias or extension",
		LongHelp: help.GetHelpFor([]string{consts.ArmoryStr, consts.InstallStr}),
		Flags: func(f *grumble.Flags) {
			f.Bool("I", "insecure", false, "skip tls certificate validation")
			f.String("p", "proxy", "", "specify a proxy url (e.g. http://localhost:8080)")
			f.Bool("c", "ignore-cache", false, "ignore metadata cache, force refresh")
			f.String("t", "timeout", "15m", "download timeout")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			armory.ArmoryInstallCmd(ctx, con)
			con.Println()
			return nil
		},
		Args: func(a *grumble.Args) {
			a.String("name", "name of the extension or alias to install")
		},
		Completer: func(prefix string, args []string) []string {
			return armory.AliasExtensionOrBundleCompleter(prefix, args, con)
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	armoryCmd.AddCommand(&grumble.Command{
		Name:     consts.UpdateStr,
		Help:     "Update installed an aliases and extensions",
		LongHelp: help.GetHelpFor([]string{consts.ArmoryStr, consts.UpdateStr}),
		Flags: func(f *grumble.Flags) {
			f.Bool("I", "insecure", false, "skip tls certificate validation")
			f.String("p", "proxy", "", "specify a proxy url (e.g. http://localhost:8080)")
			f.Bool("c", "ignore-cache", false, "ignore metadata cache, force refresh")
			f.String("t", "timeout", "15m", "download timeout")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			armory.ArmoryUpdateCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	armoryCmd.AddCommand(&grumble.Command{
		Name:     consts.SearchStr,
		Help:     "Search for aliases and extensions by name (regex)",
		LongHelp: help.GetHelpFor([]string{consts.ArmoryStr, consts.SearchStr}),
		Args: func(a *grumble.Args) {
			a.String("name", "a name regular expression")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			armory.ArmorySearchCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

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
			f.Bool("p", "persistent", false, "make persistent across restarts")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
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
			f.Bool("D", "disable-otp", false, "disable otp authentication")

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
			f.Bool("D", "disable-otp", false, "disable otp authentication")
			f.String("T", "long-poll-timeout", "1s", "server-side long poll timeout")
			f.String("J", "long-poll-jitter", "2s", "server-side long poll jitter")

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
			f.Bool("D", "disable-otp", false, "disable otp authentication")
			f.String("T", "long-poll-timeout", "1s", "server-side long poll timeout")
			f.String("J", "long-poll-jitter", "2s", "server-side long poll jitter")

			f.String("c", "cert", "", "PEM encoded certificate file")
			f.String("k", "key", "", "PEM encoded private key file")
			f.Bool("e", "lets-encrypt", false, "attempt to provision a let's encrypt certificate")
			f.Bool("E", "disable-randomized-jarm", false, "disable randomized jarm fingerprints")

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
			f.String("p", "profile", "", "implant profile name to link with the listener")
			f.String("u", "url", "", "URL to which the stager will call back to")
			f.String("c", "cert", "", "path to PEM encoded certificate file (HTTPS only)")
			f.String("k", "key", "", "path to PEM encoded private key file (HTTPS only)")
			f.Bool("e", "lets-encrypt", false, "attempt to provision a let's encrypt certificate (HTTPS only)")
			f.StringL("aes-encrypt-key", "", "encrypt stage with AES encryption key")
			f.StringL("aes-encrypt-iv", "", "encrypt stage with AES encryption iv")
			f.String("C", "compress", "none", "compress the stage before encrypting (zlib, gzip, deflate9, none)")
			f.Bool("P", "prepend-size", false, "prepend the size of the stage to the payload (to use with MSF stagers)")
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
		Name:     consts.OperatorsStr,
		Help:     "Manage operators",
		LongHelp: help.GetHelpFor([]string{consts.OperatorsStr}),
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

	// [ Reconfig ] ---------------------------------------------------------------

	con.App.AddCommand(&grumble.Command{
		Name:     consts.ReconfigStr,
		Help:     "Reconfigure the active beacon/session",
		LongHelp: help.GetHelpFor([]string{consts.ReconfigStr}),
		Flags: func(f *grumble.Flags) {
			f.String("r", "reconnect-interval", "", "reconnect interval for implant")
			f.String("i", "beacon-interval", "", "beacon callback interval")
			f.String("j", "beacon-jitter", "", "beacon callback jitter (random up to)")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			reconfig.ReconfigCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.RenameStr,
		Help:     "Rename the active beacon/session",
		LongHelp: help.GetHelpFor([]string{consts.RenameStr}),
		Flags: func(f *grumble.Flags) {
			f.String("n", "name", "", "change implant name to")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			reconfig.RenameCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	// [ Sessions ] --------------------------------------------------------------

	sessionsCmd := &grumble.Command{
		Name:     consts.SessionsStr,
		Help:     "Session management",
		LongHelp: help.GetHelpFor([]string{consts.SessionsStr}),
		Flags: func(f *grumble.Flags) {
			f.String("i", "interact", "", "interact with a session")
			f.String("k", "kill", "", "kill the designated session")
			f.Bool("K", "kill-all", false, "kill all the sessions")
			f.Bool("C", "clean", false, "clean out any sessions marked as [DEAD]")
			f.Bool("F", "force", false, "force session action without waiting for results")

			f.String("f", "filter", "", "filter sessions by substring")
			f.String("e", "filter-re", "", "filter sessions by regular expression")

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
		Help:     "Kill all stale/dead sessions",
		LongHelp: help.GetHelpFor([]string{consts.SessionsStr, consts.PruneStr}),
		Flags: func(f *grumble.Flags) {
			f.Bool("F", "force", false, "Force the killing of stale/dead sessions")

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
		Name:     consts.KillStr,
		Help:     "Kill a session",
		LongHelp: help.GetHelpFor([]string{consts.KillStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			kill.KillCmd(ctx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Bool("F", "force", false, "Force kill,  does not clean up")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	openSessionCmd := &grumble.Command{
		Name:     consts.InteractiveStr,
		Help:     "Task a beacon to open an interactive session (Beacon only)",
		LongHelp: help.GetHelpFor([]string{consts.InteractiveStr}),
		Flags: func(f *grumble.Flags) {
			f.String("m", "mtls", "", "mtls connection strings")
			f.String("g", "wg", "", "wg connection strings")
			f.String("b", "http", "", "http(s) connection strings")
			f.String("n", "dns", "", "dns connection strings")
			f.String("p", "named-pipe", "", "namedpipe connection strings")
			f.String("i", "tcp-pivot", "", "tcppivot connection strings")

			f.String("d", "delay", "0s", "delay opening the session (after checkin) for a given period of time")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			sessions.InteractiveCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	}
	con.App.AddCommand(openSessionCmd)

	// [ Close ] --------------------------------------------------------------
	closeSessionCmd := &grumble.Command{
		Name:     consts.CloseStr,
		Help:     "Close an interactive session without killing the remote process",
		LongHelp: help.GetHelpFor([]string{consts.CloseStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			sessions.CloseSessionCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	}
	con.App.AddCommand(closeSessionCmd)

	// [ Tasks ] --------------------------------------------------------------

	tasksCmd := &grumble.Command{
		Name:     consts.TasksStr,
		Help:     "Beacon task management",
		LongHelp: help.GetHelpFor([]string{consts.TasksStr}),
		Flags: func(f *grumble.Flags) {
			f.Bool("O", "overflow", false, "overflow terminal width (display truncated rows)")
			f.Int("S", "skip-pages", 0, "skip the first n page(s)")
			f.String("f", "filter", "", "filter based on task type (case-insensitive prefix matching)")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			tasks.TasksCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	}
	tasksCmd.AddCommand(&grumble.Command{
		Name:     consts.FetchStr,
		Help:     "Fetch the details of a beacon task",
		LongHelp: help.GetHelpFor([]string{consts.TasksStr, consts.FetchStr}),
		Flags: func(f *grumble.Flags) {
			f.Bool("O", "overflow", false, "overflow terminal width (display truncated rows)")
			f.Int("S", "skip-pages", 0, "skip the first n page(s)")
			f.String("f", "filter", "", "filter based on task type (case-insensitive prefix matching)")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("id", "beacon task ID", grumble.Default(""))
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			tasks.TasksFetchCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	tasksCmd.AddCommand(&grumble.Command{
		Name:     consts.CancelStr,
		Help:     "Cancel a pending beacon task",
		LongHelp: help.GetHelpFor([]string{consts.TasksStr, consts.CancelStr}),
		Flags: func(f *grumble.Flags) {
			f.Bool("O", "overflow", false, "overflow terminal width (display truncated rows)")
			f.Int("S", "skip-pages", 0, "skip the first n page(s)")
			f.String("f", "filter", "", "filter based on task type (case-insensitive prefix matching)")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("id", "beacon task ID", grumble.Default(""))
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			tasks.TasksCancelCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	con.App.AddCommand(tasksCmd)

	// [ Use ] --------------------------------------------------------------

	useCmd := &grumble.Command{
		Name:     consts.UseStr,
		Help:     "Switch the active session or beacon",
		LongHelp: help.GetHelpFor([]string{consts.UseStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("id", "beacon or session ID", grumble.Default(""))
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			use.UseCmd(ctx, con)
			con.Println()
			return nil
		},
		Completer: func(prefix string, args []string) []string {
			return use.BeaconAndSessionIDCompleter(prefix, args, con)
		},
		HelpGroup: consts.GenericHelpGroup,
	}
	useCmd.AddCommand(&grumble.Command{
		Name:     consts.SessionsStr,
		Help:     "Switch the active session",
		LongHelp: help.GetHelpFor([]string{consts.UseStr, consts.SessionsStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			use.UseSessionCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	useCmd.AddCommand(&grumble.Command{
		Name:     consts.BeaconsStr,
		Help:     "Switch the active beacon",
		LongHelp: help.GetHelpFor([]string{consts.UseStr, consts.BeaconsStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			use.UseBeaconCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	con.App.AddCommand(useCmd)

	// [ Settings ] --------------------------------------------------------------

	settingsCmd := &grumble.Command{
		Name:     consts.SettingsStr,
		Help:     "Manage client settings",
		LongHelp: help.GetHelpFor([]string{consts.SettingsStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			settings.SettingsCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	}
	settingsCmd.AddCommand(&grumble.Command{
		Name:     consts.SaveStr,
		Help:     "Save the current settings to disk",
		LongHelp: help.GetHelpFor([]string{consts.SettingsStr, consts.SaveStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			settings.SettingsSaveCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	settingsCmd.AddCommand(&grumble.Command{
		Name:     consts.TablesStr,
		Help:     "Modify tables setting (style)",
		LongHelp: help.GetHelpFor([]string{consts.SettingsStr, consts.TablesStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			settings.SettingsTablesCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	settingsCmd.AddCommand(&grumble.Command{
		Name:     "beacon-autoresults",
		Help:     "Automatically display beacon task results when completed",
		LongHelp: help.GetHelpFor([]string{consts.SettingsStr, "beacon-autoresults"}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			settings.SettingsBeaconsAutoResultCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	settingsCmd.AddCommand(&grumble.Command{
		Name:     "autoadult",
		Help:     "Automatically accept OPSEC warnings",
		LongHelp: help.GetHelpFor([]string{consts.SettingsStr, "autoadult"}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			settings.SettingsAutoAdultCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	settingsCmd.AddCommand(&grumble.Command{
		Name:     "always-overflow",
		Help:     "Disable table pagination",
		LongHelp: help.GetHelpFor([]string{consts.SettingsStr, "always-overflow"}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			settings.SettingsAlwaysOverflow(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	settingsCmd.AddCommand(&grumble.Command{
		Name:     "small-terminal",
		Help:     "Set the small terminal width",
		LongHelp: help.GetHelpFor([]string{consts.SettingsStr, "small-terminal"}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			settings.SettingsSmallTerm(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	con.App.AddCommand(settingsCmd)

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
		Completer: func(prefix string, args []string) []string {
			return use.BeaconAndSessionIDCompleter(prefix, args, con)
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

	// [ Shellcode Encoders ] --------------------------------------------------------------

	con.App.AddCommand(&grumble.Command{
		Name:     consts.ShikataGaNai,
		Help:     "Polymorphic binary shellcode encoder (ノ ゜Д゜)ノ ︵ 仕方がない",
		LongHelp: help.GetHelpFor([]string{consts.ShikataGaNai}),
		Args: func(a *grumble.Args) {
			a.String("shellcode", "binary shellcode file path")
		},
		Flags: func(f *grumble.Flags) {
			f.String("s", "save", "", "save output to local file")

			f.String("a", "arch", "amd64", "architecture of shellcode")
			f.Int("i", "iterations", 1, "number of iterations")
			f.String("b", "bad-chars", "", "hex encoded bad characters to avoid (e.g. 0001)")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			sgn.ShikataGaNaiCmd(ctx, con)
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
			f.Bool("o", "output", false, "capture command output")
			f.Bool("s", "save", false, "save output to a file")
			f.Bool("X", "loot", false, "save output as loot")
			f.Bool("S", "ignore-stderr", false, "don't print STDERR output")
			f.String("O", "stdout", "", "remote path to redirect STDOUT to")
			f.String("E", "stderr", "", "remote path to redirect STDERR to")
			f.String("n", "name", "", "name to assign loot (optional)")
			f.Uint("P", "ppid", 0, "parent process id (optional, Windows only)")

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
			f.Bool("i", "in-process", false, "Run in the current sliver process")
			f.String("r", "runtime", "", "Runtime to use for running the assembly (only supported when used with --in-process)")
			f.Bool("s", "save", false, "save output to file")
			f.Bool("X", "loot", false, "save output as loot")
			f.String("n", "name", "", "name to assign loot (optional)")
			f.Uint("P", "ppid", 0, "parent process id (optional)")
			f.String("A", "process-arguments", "", "arguments to pass to the hosting process")
			f.Bool("M", "amsi-bypass", false, "Bypass AMSI on Windows (only supported when used with --in-process)")
			f.Bool("E", "etw-bypass", false, "Bypass ETW on Windows (only supported when used with --in-process)")

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
			f.Bool("S", "shikata-ga-nai", false, "encode shellcode using shikata ga nai prior to execution")
			f.String("A", "architecture", "amd64", "architecture of the shellcode: 386, amd64 (used with --shikata-ga-nai flag)")
			f.Int("I", "iterations", 1, "number of encoding iterations (used with --shikata-ga-nai flag)")

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
			f.Bool("w", "unicode", false, "Command line is passed to unmanaged DLL function in UNICODE format. (default is ANSI)")
			f.Bool("s", "save", false, "save output to file")
			f.Bool("X", "loot", false, "save output as loot")
			f.String("n", "name", "", "name to assign loot (optional)")
			f.Bool("k", "keep-alive", false, "don't terminate host process once the execution completes")
			f.Uint("P", "ppid", 0, "parent process id (optional)")
			f.String("A", "process-arguments", "", "arguments to pass to the hosting process")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
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
			f.Bool("X", "loot", false, "save output as loot")
			f.String("n", "name", "", "name to assign loot (optional)")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Bool("k", "keep-alive", false, "don't terminate host process once the execution completes")
			f.Uint("P", "ppid", 0, "parent process id (optional)")
			f.String("A", "process-arguments", "", "arguments to pass to the hosting process")
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
			f.Bool("S", "disable-sgn", true, "disable shikata ga nai shellcode encoder")

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
			f.String("c", "custom-exe", "", "custom service executable to use instead of generating a new Sliver")
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
			f.String("c", "kerberos-config", "/etc/krb5.conf", "path to remote Kerberos config file")
			f.String("k", "kerberos-keytab", "", "path to Kerberos keytab file")
			f.String("r", "kerberos-realm", "", "Kerberos realm")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			exec.SSHCmd(ctx, con)
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
			f.String("O", "debug-file", "", "path to debug output")
			f.Bool("e", "evasion", false, "enable evasion features (e.g. overwrite user space hooks)")
			f.Bool("l", "skip-symbols", false, "skip symbol obfuscation")
			f.String("I", "template", "sliver", "implant code template")
			f.Bool("E", "external-builder", false, "use an external builder")
			f.Bool("G", "disable-sgn", false, "disable shikata ga nai shellcode encoder")

			f.String("c", "canary", "", "canary domain(s)")

			f.String("m", "mtls", "", "mtls connection strings")
			f.String("g", "wg", "", "wg connection strings")
			f.String("b", "http", "", "http(s) connection strings")
			f.String("n", "dns", "", "dns connection strings")
			f.String("p", "named-pipe", "", "named-pipe connection strings")
			f.String("i", "tcp-pivot", "", "tcp-pivot connection strings")

			f.Int("X", "key-exchange", generate.DefaultWGKeyExPort, "wg key-exchange port")
			f.Int("T", "tcp-comms", generate.DefaultWGNPort, "wg c2 comms port")

			f.Bool("R", "run-at-load", false, "run the implant entrypoint from DllMain/Constructor (shared library only)")

			f.String("Z", "strategy", "", "specify a connection strategy (r = random, rd = random domain, s = sequential)")
			f.Int("j", "reconnect", generate.DefaultReconnect, "attempt to reconnect every n second(s)")
			f.Int("P", "poll-timeout", generate.DefaultPollTimeout, "long poll request timeout")
			f.Int("k", "max-errors", generate.DefaultMaxErrors, "max number of connection errors")

			f.String("w", "limit-datetime", "", "limit execution to before datetime")
			f.Bool("x", "limit-domainjoined", false, "limit execution to domain joined machines")
			f.String("y", "limit-username", "", "limit execution to specified username")
			f.String("z", "limit-hostname", "", "limit execution to specified hostname")
			f.String("F", "limit-fileexists", "", "limit execution to hosts with this file in the filesystem")
			f.String("L", "limit-locale", "", "limit execution to hosts that match this locale")

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
		Name:     consts.BeaconStr,
		Help:     "Generate a beacon binary",
		LongHelp: help.GetHelpFor([]string{consts.GenerateStr, consts.BeaconStr}),
		Flags: func(f *grumble.Flags) {
			f.Int64("D", "days", 0, "beacon interval days")
			f.Int64("H", "hours", 0, "beacon interval hours")
			f.Int64("M", "minutes", 0, "beacon interval minutes")
			f.Int64("S", "seconds", 60, "beacon interval seconds")
			f.Int64("J", "jitter", 30, "beacon interval jitter in seconds")

			// Generate flags
			f.String("o", "os", "windows", "operating system")
			f.String("a", "arch", "amd64", "cpu architecture")
			f.String("N", "name", "", "agent name")
			f.Bool("d", "debug", false, "enable debug features")
			f.String("O", "debug-file", "", "path to debug output")
			f.Bool("e", "evasion", false, "enable evasion features  (e.g. overwrite user space hooks)")
			f.Bool("l", "skip-symbols", false, "skip symbol obfuscation")
			f.String("I", "template", "sliver", "implant code template")
			f.Bool("E", "external-builder", false, "use an external builder")
			f.Bool("G", "disable-sgn", false, "disable shikata ga nai shellcode encoder")

			f.String("c", "canary", "", "canary domain(s)")

			f.String("m", "mtls", "", "mtls connection strings")
			f.String("g", "wg", "", "wg connection strings")
			f.String("b", "http", "", "http(s) connection strings")
			f.String("n", "dns", "", "dns connection strings")
			f.String("p", "named-pipe", "", "named-pipe connection strings")
			f.String("i", "tcp-pivot", "", "tcp-pivot connection strings")

			f.Int("X", "key-exchange", generate.DefaultWGKeyExPort, "wg key-exchange port")
			f.Int("T", "tcp-comms", generate.DefaultWGNPort, "wg c2 comms port")

			f.Bool("R", "run-at-load", false, "run the implant entrypoint from DllMain/Constructor (shared library only)")

			f.String("Z", "strategy", "", "specify a connection strategy (r = random, rd = random domain, s = sequential)")
			f.Int("j", "reconnect", generate.DefaultReconnect, "attempt to reconnect every n second(s)")
			f.Int("P", "poll-timeout", generate.DefaultPollTimeout, "long poll request timeout")
			f.Int("k", "max-errors", generate.DefaultMaxErrors, "max number of connection errors")

			f.String("w", "limit-datetime", "", "limit execution to before datetime")
			f.Bool("x", "limit-domainjoined", false, "limit execution to domain joined machines")
			f.String("y", "limit-username", "", "limit execution to specified username")
			f.String("z", "limit-hostname", "", "limit execution to specified hostname")
			f.String("F", "limit-fileexists", "", "limit execution to hosts with this file in the filesystem")
			f.String("L", "limit-locale", "", "limit execution to hosts that match this locale")

			f.String("f", "format", "exe", "Specifies the output formats, valid values are: 'exe', 'shared' (for dynamic libraries), 'service' (see `psexec` for more info) and 'shellcode' (windows only)")
			f.String("s", "save", "", "directory/file to the binary to")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			generate.GenerateBeaconCmd(ctx, con)
			con.Println()
			return nil
		},
	})
	generateCmd.AddCommand(&grumble.Command{
		Name:     consts.StagerStr,
		Help:     "Generate a stager using Metasploit (requires local Metasploit installation)",
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
		Name:     consts.CompilerInfoStr,
		Help:     "Get information about the server's compiler",
		LongHelp: help.GetHelpFor([]string{consts.CompilerInfoStr}),
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
			f.String("s", "save", "", "directory/file to the binary to")
			f.Bool("G", "disable-sgn", false, "disable shikata ga nai shellcode encoder")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("name", "name of the profile", grumble.Default(""))
		},
		Completer: func(prefix string, args []string) []string {
			return generate.ProfileNameCompleter(prefix, args, con)
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			generate.ProfilesGenerateCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	profilesNewCmd := &grumble.Command{
		Name:     consts.NewStr,
		Help:     "Create a new implant profile (interactive session)",
		LongHelp: help.GetHelpFor([]string{consts.ProfilesStr, consts.NewStr}),
		Flags: func(f *grumble.Flags) {

			// Generate flags
			f.String("o", "os", "windows", "operating system")
			f.String("a", "arch", "amd64", "cpu architecture")

			f.Bool("d", "debug", false, "enable debug features")
			f.String("O", "debug-file", "", "path to debug output")
			f.Bool("e", "evasion", false, "enable evasion features")
			f.Bool("l", "skip-symbols", false, "skip symbol obfuscation")
			f.Bool("G", "disable-sgn", false, "disable shikata ga nai shellcode encoder")

			f.String("c", "canary", "", "canary domain(s)")

			f.String("N", "name", "", "implant name")
			f.String("m", "mtls", "", "mtls connection strings")
			f.String("g", "wg", "", "wg connection strings")
			f.String("b", "http", "", "http(s) connection strings")
			f.String("n", "dns", "", "dns connection strings")
			f.String("p", "named-pipe", "", "named-pipe connection strings")
			f.String("i", "tcp-pivot", "", "tcp-pivot connection strings")

			f.Int("X", "key-exchange", generate.DefaultWGKeyExPort, "wg key-exchange port")
			f.Int("T", "tcp-comms", generate.DefaultWGNPort, "wg c2 comms port")

			f.Bool("R", "run-at-load", false, "run the implant entrypoint from DllMain/Constructor (shared library only)")
			f.String("Z", "strategy", "", "specify a connection strategy (r = random, rd = random domain, s = sequential)")

			f.String("I", "template", "sliver", "implant code template")

			f.Int("j", "reconnect", generate.DefaultReconnect, "attempt to reconnect every n second(s)")
			f.Int("P", "poll-timeout", generate.DefaultPollTimeout, "long poll request timeout")
			f.Int("k", "max-errors", generate.DefaultMaxErrors, "max number of connection errors")

			f.String("w", "limit-datetime", "", "limit execution to before datetime")
			f.Bool("x", "limit-domainjoined", false, "limit execution to domain joined machines")
			f.String("y", "limit-username", "", "limit execution to specified username")
			f.String("z", "limit-hostname", "", "limit execution to specified hostname")
			f.String("F", "limit-fileexists", "", "limit execution to hosts with this file in the filesystem")
			f.String("L", "limit-locale", "", "limit execution to hosts that match this locale")

			f.String("f", "format", "exe", "Specifies the output formats, valid values are: 'exe', 'shared' (for dynamic libraries), 'service' (see `psexec` for more info) and 'shellcode' (windows only)")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("name", "name of the profile", grumble.Default(""))
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			generate.ProfilesNewCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	}
	profilesCmd.AddCommand(profilesNewCmd)

	// New Beacon Profile Command
	profilesNewCmd.AddCommand(&grumble.Command{
		Name:     consts.BeaconStr,
		Help:     "Create a new implant profile (beacon)",
		LongHelp: help.GetHelpFor([]string{consts.ProfilesStr, consts.NewStr, consts.BeaconStr}),
		Flags: func(f *grumble.Flags) {
			f.Int64("D", "days", 0, "beacon interval days")
			f.Int64("H", "hours", 0, "beacon interval hours")
			f.Int64("M", "minutes", 0, "beacon interval minutes")
			f.Int64("S", "seconds", 60, "beacon interval seconds")
			f.Int64("J", "jitter", 30, "beacon interval jitter in seconds")
			f.Bool("G", "disable-sgn", false, "disable shikata ga nai shellcode encoder")

			// Generate flags
			f.String("o", "os", "windows", "operating system")
			f.String("a", "arch", "amd64", "cpu architecture")

			f.Bool("d", "debug", false, "enable debug features")
			f.String("O", "debug-file", "", "path to debug output")
			f.Bool("e", "evasion", false, "enable evasion features")
			f.Bool("l", "skip-symbols", false, "skip symbol obfuscation")

			f.String("c", "canary", "", "canary domain(s)")

			f.String("N", "name", "", "implant name")
			f.String("m", "mtls", "", "mtls connection strings")
			f.String("g", "wg", "", "wg connection strings")
			f.String("b", "http", "", "http(s) connection strings")
			f.String("n", "dns", "", "dns connection strings")
			f.String("p", "named-pipe", "", "named-pipe connection strings")
			f.String("i", "tcp-pivot", "", "tcp-pivot connection strings")
			f.String("Z", "strategy", "", "specify a connection strategy (r = random, rd = random domain, s = sequential)")

			f.Int("X", "key-exchange", generate.DefaultWGKeyExPort, "wg key-exchange port")
			f.Int("T", "tcp-comms", generate.DefaultWGNPort, "wg c2 comms port")

			f.Bool("R", "run-at-load", false, "run the implant entrypoint from DllMain/Constructor (shared library only)")

			f.String("I", "template", "sliver", "implant code template")

			f.Int("j", "reconnect", generate.DefaultReconnect, "attempt to reconnect every n second(s)")
			f.Int("P", "poll-timeout", generate.DefaultPollTimeout, "long poll request timeout")
			f.Int("k", "max-errors", generate.DefaultMaxErrors, "max number of connection errors")

			f.String("w", "limit-datetime", "", "limit execution to before datetime")
			f.Bool("x", "limit-domainjoined", false, "limit execution to domain joined machines")
			f.String("y", "limit-username", "", "limit execution to specified username")
			f.String("z", "limit-hostname", "", "limit execution to specified hostname")
			f.String("F", "limit-fileexists", "", "limit execution to hosts with this file in the filesystem")
			f.String("L", "limit-locale", "", "limit execution to hosts that match this locale")

			f.String("f", "format", "exe", "Specifies the output formats, valid values are: 'exe', 'shared' (for dynamic libraries), 'service' (see `psexec` for more info) and 'shellcode' (windows only)")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("name", "name of the profile", grumble.Default(""))
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			generate.ProfilesNewBeaconCmd(ctx, con)
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
			a.String("name", "name of the profile", grumble.Default(""))
		},
		Completer: func(prefix string, args []string) []string {
			return generate.ProfileNameCompleter(prefix, args, con)
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
			f.String("o", "os", "", "filter builds by operating system")
			f.String("a", "arch", "", "filter builds by cpu architecture")
			f.String("f", "format", "", "filter builds by artifact format")
			f.Bool("s", "only-sessions", false, "filter interactive sessions")
			f.Bool("b", "only-beacons", false, "filter beacons")
			f.Bool("d", "no-debug", false, "filter builds by debug flag")

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
			a.String("name", "implant name", grumble.Default(""))
		},
		Completer: func(prefix string, args []string) []string {
			return generate.ImplantBuildNameCompleter(prefix, args, generate.ImplantBuildFilter{}, con)
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
		Name:     consts.CanariesStr,
		Help:     "List previously generated canaries",
		LongHelp: help.GetHelpFor([]string{consts.CanariesStr}),
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
		Name:     consts.MvStr,
		Help:     "Move or rename a file",
		LongHelp: help.GetHelpFor([]string{consts.MvStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("src", "path to source file")
			a.String("dst", "path to dest file")
		},
		Run: func(ctx *grumble.Context) error {
			err := filesystem.MvCmd(ctx, con)
			return err
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.LsStr,
		Help:     "List current directory",
		LongHelp: help.GetHelpFor([]string{consts.LsStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Bool("r", "reverse", false, "reverse sort order")
			f.Bool("m", "modified", false, "sort by modified time")
			f.Bool("s", "size", false, "sort by size")
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
			f.Bool("F", "force", false, "ignore safety and forcefully remove files")

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
			f.String("n", "name", "", "name to assign loot (optional)")
			f.String("T", "type", "", "force a specific loot type (file/cred) if looting (optional)")
			f.String("F", "file-type", "", "force a specific file type (binary/text) if looting (optional)")
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
			f.String("T", "type", "", "force a specific loot type (file/cred) if looting")
			f.String("F", "file-type", "", "force a specific file type (binary/text) if looting")
			f.String("n", "name", "", "name to assign the download if looting")
			f.Bool("r", "recurse", false, "recursively download all files in a directory")
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

			f.Bool("i", "ioc", false, "track uploaded file as an ioc")
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
			f.Bool("A", "all", false, "show all network adapters (default only shows IPv4)")

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
			f.Bool("n", "numeric", false, "display numeric addresses (disable hostname resolution)")
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
			f.Bool("O", "overflow", false, "overflow terminal width (display truncated rows)")
			f.Int("S", "skip-pages", 0, "skip the first n page(s)")
			f.Bool("T", "tree", false, "print process tree")

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
			f.String("s", "save", "", "save to file (will overwrite if exists)")
			f.Bool("X", "loot", false, "save output as loot")
			f.String("N", "loot-name", "", "name to assign when adding the memory dump to the loot store (optional)")

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
			f.Bool("F", "force", false, "disregard safety and kill the PID")

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
			f.String("u", "username", "", "user to impersonate")
			f.String("p", "process", "", "process to start")
			f.String("a", "args", "", "arguments for the process")
			f.String("d", "domain", "", "domain of the user")
			f.String("P", "password", "", "password of the user")
			f.Bool("s", "show-window", false, `
			Log on, but use the specified credentials on the network only. The new process uses the same token as the caller, but the system creates a new logon session within LSA, and the process uses the specified credentials as the default credentials.`)
			f.Bool("n", "net-only", false, "use ")
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
			f.String("T", "logon-type", "LOGON_NEW_CREDENTIALS", "logon type to use")
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

	con.App.AddCommand(&grumble.Command{
		Name:     consts.ChmodStr,
		Help:     "Change permissions on a file or directory",
		LongHelp: help.GetHelpFor([]string{consts.ChmodStr}),
		Flags: func(f *grumble.Flags) {
			f.Bool("r", "recursive", false, "recursively change permissions on files")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("path", "path to the file to remove")
			a.String("mode", "file permissions in octal, e.g. 0644")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			filesystem.ChmodCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.ChownStr,
		Help:     "Change owner on a file or directory",
		LongHelp: help.GetHelpFor([]string{consts.ChownStr}),
		Flags: func(f *grumble.Flags) {
			f.Bool("r", "recursive", false, "recursively change permissions on files")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("path", "path to the file to remove")
			a.String("uid", "User, e.g. root")
			a.String("gid", "Group, e.g. root")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			filesystem.ChownCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	con.App.AddCommand(&grumble.Command{
		Name:     consts.ChtimesStr,
		Help:     "Change access and modification times on a file (timestomp)",
		LongHelp: help.GetHelpFor([]string{consts.ChtimesStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.String("path", "path to the file to remove")
			a.String("atime", "Last accessed time in DateTime format, i.e. 2006-01-02 15:04:05")
			a.String("mtime", "Last modified time in DateTime format, i.e. 2006-01-02 15:04:05")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			filesystem.ChtimesCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
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
			f.String("s", "save", "", "save to file (will overwrite if exists)")
			f.Bool("X", "loot", false, "save output as loot")
			f.String("n", "name", "", "name to assign loot (optional)")

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

	// [ Beacons ] ---------------------------------------------

	beaconsCmd := &grumble.Command{
		Name:     consts.BeaconsStr,
		Help:     "Manage beacons",
		LongHelp: help.GetHelpFor([]string{consts.BeaconsStr}),
		Flags: func(f *grumble.Flags) {
			f.String("k", "kill", "", "kill a beacon")
			f.Bool("K", "kill-all", false, "kill all beacons")
			f.Bool("F", "force", false, "force killing of the beacon")
			f.String("f", "filter", "", "filter beacons by substring")
			f.String("e", "filter-re", "", "filter beacons by regular expression")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		HelpGroup: consts.GenericHelpGroup,
		Run: func(ctx *grumble.Context) error {
			con.Println()
			beacons.BeaconsCmd(ctx, con)
			con.Println()
			return nil
		},
	}
	beaconsCmd.AddCommand(&grumble.Command{
		Name:     consts.RmStr,
		Help:     "Remove a beacon",
		LongHelp: help.GetHelpFor([]string{consts.BeaconsStr, consts.RmStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		HelpGroup: consts.SliverWinHelpGroup,
		Run: func(ctx *grumble.Context) error {
			con.Println()
			beacons.BeaconsRmCmd(ctx, con)
			con.Println()
			return nil
		},
	})
	beaconsCmd.AddCommand(&grumble.Command{
		Name:     consts.WatchStr,
		Help:     "Watch your beacons",
		LongHelp: help.GetHelpFor([]string{consts.BeaconsStr, consts.WatchStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		HelpGroup: consts.SliverWinHelpGroup,
		Run: func(ctx *grumble.Context) error {
			con.Println()
			beacons.BeaconsWatchCmd(ctx, con)
			con.Println()
			return nil
		},
	})
	beaconsCmd.AddCommand(&grumble.Command{
		Name:     consts.PruneStr,
		Help:     "Prune stale beacons automatically",
		LongHelp: help.GetHelpFor([]string{consts.BeaconsStr, consts.PruneStr}),
		Flags: func(f *grumble.Flags) {
			f.String("d", "duration", "1h", "duration to prune beacons that have missed their last checkin")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		HelpGroup: consts.SliverWinHelpGroup,
		Run: func(ctx *grumble.Context) error {
			con.Println()
			beacons.BeaconsPruneCmd(ctx, con)
			con.Println()
			return nil
		},
	})
	con.App.AddCommand(beaconsCmd)

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
			f.String("H", "hive", "HKCU", "registry hive")
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
	registryCmd.AddCommand(&grumble.Command{
		Name:     consts.RegistryDeleteKeyStr,
		Help:     "Remove a registry key",
		LongHelp: help.GetHelpFor([]string{consts.RegistryDeleteKeyStr}),
		Args: func(a *grumble.Args) {
			a.String("registry-path", "registry path")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			registry.RegDeleteKeyCmd(ctx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.String("H", "hive", "HKCU", "registry hive")
			f.String("o", "hostname", "", "remote host to remove value from")
		},
	})
	registryCmd.AddCommand(&grumble.Command{
		Name:     consts.RegistryListSubStr,
		Help:     "List the sub keys under a registry key",
		LongHelp: help.GetHelpFor([]string{consts.RegistryListSubStr}),
		Args: func(a *grumble.Args) {
			a.String("registry-path", "registry path")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			registry.RegListSubKeysCmd(ctx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.String("H", "hive", "HKCU", "registry hive")
			f.String("o", "hostname", "", "remote host to write values to")
		},
	})

	registryCmd.AddCommand(&grumble.Command{
		Name:     consts.RegistryListValuesStr,
		Help:     "List the values for a registry key",
		LongHelp: help.GetHelpFor([]string{consts.RegistryListValuesStr}),
		Args: func(a *grumble.Args) {
			a.String("registry-path", "registry path")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			registry.RegListValuesCmd(ctx, con)
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

	// [ Reverse Port Forwarding ] --------------------------------------------------------------

	rportfwdCmd := &grumble.Command{
		Name:     consts.RportfwdStr,
		Help:     "reverse port forwardings",
		LongHelp: help.GetHelpFor([]string{consts.RportfwdStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			rportfwd.RportFwdListenersCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	}
	rportfwdCmd.AddCommand(&grumble.Command{
		Name:     consts.AddStr,
		Help:     "Add and start reverse port forwarding",
		LongHelp: help.GetHelpFor([]string{consts.RportfwdStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			rportfwd.StartRportFwdListenerCmd(ctx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.String("r", "remote", "", "remote address <ip>:<port> connection is forwarded to")
			f.String("b", "bind", "", "bind address <ip>:<port> implants listen on")
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})
	rportfwdCmd.AddCommand(&grumble.Command{
		Name:     consts.RmStr,
		Help:     "Stop and remove reverse port forwarding",
		LongHelp: help.GetHelpFor([]string{consts.RportfwdStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			rportfwd.StopRportFwdListenerCmd(ctx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Int("i", "id", 0, "id of portfwd to remove")
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})

	con.App.AddCommand(rportfwdCmd)

	// [ Pivots ] --------------------------------------------------------------

	pivotsCmd := &grumble.Command{
		Name:     consts.PivotsStr,
		Help:     "List pivots for active session",
		LongHelp: help.GetHelpFor([]string{consts.PivotsStr}),
		Run: func(ctx *grumble.Context) error {
			con.Println()
			pivots.PivotsCmd(ctx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		HelpGroup: consts.SliverHelpGroup,
	}
	con.App.AddCommand(pivotsCmd)

	pivotsCmd.AddCommand(&grumble.Command{
		Name:     consts.NamedPipeStr,
		Help:     "Start a named pipe pivot listener",
		LongHelp: help.GetHelpFor([]string{consts.PivotsStr, consts.NamedPipeStr}),
		Flags: func(f *grumble.Flags) {
			f.String("b", "bind", "", "name of the named pipe to bind pivot listener")
			f.Bool("a", "allow-all", false, "allow all users to connect")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			pivots.StartNamedPipeListenerCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	pivotsCmd.AddCommand(&grumble.Command{
		Name:     consts.TCPListenerStr,
		Help:     "Start a TCP pivot listener",
		LongHelp: help.GetHelpFor([]string{consts.PivotsStr, consts.TCPListenerStr}),
		Flags: func(f *grumble.Flags) {
			f.String("b", "bind", "", "remote interface to bind pivot listener")
			f.Int("l", "lport", generate.DefaultTCPPivotPort, "tcp pivot listener port")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			pivots.StartTCPListenerCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	pivotsCmd.AddCommand(&grumble.Command{
		Name:     consts.StopStr,
		Help:     "Stop a pivot listener",
		LongHelp: help.GetHelpFor([]string{consts.PivotsStr, consts.StopStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("i", "id", 0, "id of the pivot listener to stop")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			pivots.StopPivotListenerCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	pivotsCmd.AddCommand(&grumble.Command{
		Name:     consts.DetailsStr,
		Help:     "Get details of a pivot listener",
		LongHelp: help.GetHelpFor([]string{consts.PivotsStr, consts.StopStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("i", "id", 0, "id of the pivot listener to stop")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			pivots.PivotDetailsCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	pivotsCmd.AddCommand(&grumble.Command{
		Name:     "graph",
		Help:     "Get details of a pivot listener",
		LongHelp: help.GetHelpFor([]string{consts.PivotsStr, "graph"}),
		Flags: func(f *grumble.Flags) {
			f.Int("i", "id", 0, "id of the pivot listener to stop")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			pivots.PivotsGraphCmd(ctx, con)
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

	// [ Socks ] --------------------------------------------------------------

	socksCmd := &grumble.Command{
		Name:     consts.Socks5Str,
		Help:     "In-band SOCKS5 Proxy",
		LongHelp: help.GetHelpFor([]string{consts.Socks5Str}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "router timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			socks.SocksCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	}
	socksCmd.AddCommand(&grumble.Command{
		Name:     consts.StartStr,
		Help:     "Start an in-band SOCKS5 proxy",
		LongHelp: help.GetHelpFor([]string{consts.Socks5Str}),
		Flags: func(f *grumble.Flags) {
			f.String("H", "host", "127.0.0.1", "Bind a Socks5 Host")
			f.String("P", "port", "1081", "Bind a Socks5 Port")
			f.String("u", "user", "", "socks5 auth username (will generate random password)")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			socks.SocksStartCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})
	socksCmd.AddCommand(&grumble.Command{
		Name:     consts.StopStr,
		Help:     "Stop a SOCKS5 proxy",
		LongHelp: help.GetHelpFor([]string{consts.Socks5Str}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "router timeout in seconds")
			f.Uint64("i", "id", 0, "id of portfwd to remove")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			socks.SocksStopCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})
	con.App.AddCommand(socksCmd)

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
			a.String("path", "The file path on the remote host to the loot")
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
		Name:     consts.FetchStr,
		Help:     "Fetch a piece of loot from the server's loot store",
		LongHelp: help.GetHelpFor([]string{consts.LootStr, consts.FetchStr}),
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

	// [ Hosts ] --------------------------------------------------------------
	hostsCmd := &grumble.Command{
		Name:     consts.HostsStr,
		Help:     "Manage the database of hosts",
		LongHelp: help.GetHelpFor([]string{consts.HostsStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			hosts.HostsCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	}
	hostsCmd.AddCommand(&grumble.Command{
		Name:     consts.RmStr,
		Help:     "Remove a host from the database",
		LongHelp: help.GetHelpFor([]string{consts.HostsStr, consts.RmStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			hosts.HostsRmCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	iocCmd := &grumble.Command{
		Name:     consts.IOCStr,
		Help:     "Manage tracked IOCs on a given host",
		LongHelp: help.GetHelpFor([]string{consts.HostsStr, consts.IOCStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			hosts.HostsIOCCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	}
	iocCmd.AddCommand(&grumble.Command{
		Name:     consts.RmStr,
		Help:     "Delete IOCs from the database",
		LongHelp: help.GetHelpFor([]string{consts.HostsStr, consts.IOCStr, consts.RmStr}),
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			hosts.HostsIOCRmCmd(ctx, con)
			con.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})
	hostsCmd.AddCommand(iocCmd)
	con.App.AddCommand(hostsCmd)

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

	// [ Get Privs ] -----------------------------------------------------------------
	getprivsCmd := &grumble.Command{
		Name:      consts.GetPrivsStr,
		Help:      "Get current privileges (Windows only)",
		LongHelp:  help.GetHelpFor([]string{consts.GetPrivsStr}),
		HelpGroup: consts.SliverWinHelpGroup,
		Run: func(ctx *grumble.Context) error {
			con.Println()
			privilege.GetPrivsCmd(ctx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
	}
	con.App.AddCommand(getprivsCmd)

	// [ Extensions ] -----------------------------------------------------------------
	extensionCmd := &grumble.Command{
		Name:      consts.ExtensionsStr,
		Help:      "Manage extensions",
		LongHelp:  help.GetHelpFor([]string{consts.ExtensionsStr}),
		HelpGroup: consts.SliverHelpGroup,
		Run: func(ctx *grumble.Context) error {
			con.Println()
			extensions.ExtensionsCmd(ctx, con)
			con.Println()
			return nil
		},
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
	}

	extensionCmd.AddCommand(&grumble.Command{
		Name:      consts.ListStr,
		Help:      "List extensions loaded in the current session or beacon",
		LongHelp:  help.GetHelpFor([]string{consts.ExtensionsStr, consts.ListStr}),
		HelpGroup: consts.SliverHelpGroup,
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			extensions.ExtensionsListCmd(ctx, con)
			con.Println()
			return nil
		},
	})

	extensionCmd.AddCommand(&grumble.Command{
		Name:      consts.LoadStr,
		Help:      "Temporarily load an extension from a local directory",
		LongHelp:  help.GetHelpFor([]string{consts.ExtensionsStr, consts.LoadStr}),
		HelpGroup: consts.SliverHelpGroup,
		Run: func(ctx *grumble.Context) error {
			con.Println()
			extensions.ExtensionLoadCmd(ctx, con)
			con.Println()
			return nil
		},
		Args: func(a *grumble.Args) {
			a.String("dir-path", "path to the extension directory")
		},
		Completer: func(prefix string, args []string) []string {
			return completers.LocalPathCompleter(prefix, args, con)
		},
	})

	extensionCmd.AddCommand(&grumble.Command{
		Name:      consts.InstallStr,
		Help:      "Install an extension from a local directory or .tar.gz file",
		LongHelp:  help.GetHelpFor([]string{consts.ExtensionsStr, consts.InstallStr}),
		HelpGroup: consts.SliverHelpGroup,
		Run: func(ctx *grumble.Context) error {
			con.Println()
			extensions.ExtensionsInstallCmd(ctx, con)
			con.Println()
			return nil
		},
		Args: func(a *grumble.Args) {
			a.String("path", "path to the extension .tar.gz or directory")
		},
		Completer: func(prefix string, args []string) []string {
			return completers.LocalPathCompleter(prefix, args, con)
		},
	})

	extensionCmd.AddCommand(&grumble.Command{
		Name:      consts.RmStr,
		Help:      "Remove an installed extension",
		LongHelp:  help.GetHelpFor([]string{consts.ExtensionsStr, consts.RmStr}),
		HelpGroup: consts.SliverHelpGroup,
		Run: func(ctx *grumble.Context) error {
			con.Println()
			extensions.ExtensionsRemoveCmd(ctx, con)
			con.Println()
			return nil
		},
		Args: func(a *grumble.Args) {
			a.String("name", "the command name of the extension to remove")
		},
		Completer: func(prefix string, args []string) []string {
			return extensions.ExtensionsCommandNameCompleter(prefix, args, con)
		},
	})

	con.App.AddCommand(extensionCmd)

	// [ Prelude's Operator ] ------------------------------------------------------------
	operatorCmd := &grumble.Command{
		Name:      consts.PreludeOperatorStr,
		Help:      "Manage connection to Prelude's Operator",
		LongHelp:  help.GetHelpFor([]string{consts.PreludeOperatorStr}),
		HelpGroup: consts.GenericHelpGroup,
		Run: func(ctx *grumble.Context) error {
			con.Println()
			operator.OperatorCmd(ctx, con)
			con.Println()
			return nil
		},
	}
	operatorCmd.AddCommand(&grumble.Command{
		Name:      consts.ConnectStr,
		Help:      "Connect with Prelude's Operator",
		LongHelp:  help.GetHelpFor([]string{consts.PreludeOperatorStr, consts.ConnectStr}),
		HelpGroup: consts.GenericHelpGroup,
		Run: func(ctx *grumble.Context) error {
			con.Println()
			operator.ConnectCmd(ctx, con)
			con.Println()
			return nil
		},
		Args: func(a *grumble.Args) {
			a.String("connection-string", "connection string to the Operator Host (e.g. 127.0.0.1:1234)")
		},
		Flags: func(f *grumble.Flags) {
			f.Bool("s", "skip-existing", false, "Do not add existing sessions as Operator Agents")
			f.String("a", "aes-key", "abcdefghijklmnopqrstuvwxyz012345", "AES key for communication encryption")
			f.String("r", "range", "sliver", "Agents range")
		},
	})
	con.App.AddCommand(operatorCmd)

	// [ Curse Commands ] ------------------------------------------------------------

	cursedCmd := &grumble.Command{
		Name:      consts.Cursed,
		Help:      "Chrome/electron post-exploitation tool kit (∩｀-´)⊃━☆ﾟ.*･｡ﾟ",
		LongHelp:  help.GetHelpFor([]string{consts.Cursed}),
		HelpGroup: consts.GenericHelpGroup,
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			cursed.CursedCmd(ctx, con)
			con.Println()
			return nil
		},
	}
	cursedCmd.AddCommand(&grumble.Command{
		Name:      consts.RmStr,
		Help:      "Remove a Curse from a process",
		LongHelp:  help.GetHelpFor([]string{consts.Cursed, consts.CursedConsole}),
		HelpGroup: consts.GenericHelpGroup,
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
			f.Bool("k", "kill", false, "kill the process after removing the curse")
		},
		Args: func(a *grumble.Args) {
			a.Int("bind-port", "bind port of the Cursed process to stop")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			cursed.CursedRmCmd(ctx, con)
			con.Println()
			return nil
		},
	})
	cursedCmd.AddCommand(&grumble.Command{
		Name:      consts.CursedConsole,
		Help:      "Start a JavaScript console connected to a debug target",
		LongHelp:  help.GetHelpFor([]string{consts.Cursed, consts.CursedConsole}),
		HelpGroup: consts.GenericHelpGroup,
		Flags: func(f *grumble.Flags) {
			f.Int("r", "remote-debugging-port", 0, "remote debugging tcp port (0 = random)`")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			cursed.CursedConsoleCmd(ctx, con)
			con.Println()
			return nil
		},
	})
	cursedCmd.AddCommand(&grumble.Command{
		Name:      consts.CursedChrome,
		Help:      "Automatically inject a Cursed Chrome payload into a remote Chrome extension",
		LongHelp:  help.GetHelpFor([]string{consts.Cursed, consts.CursedChrome}),
		HelpGroup: consts.GenericHelpGroup,
		Flags: func(f *grumble.Flags) {
			f.Int("r", "remote-debugging-port", 0, "remote debugging tcp port (0 = random)")
			f.Bool("R", "restore", true, "restore the user's session after process termination")
			f.String("e", "exe", "", "chrome/chromium browser executable path (blank string = auto)")
			f.String("u", "user-data", "", "user data directory (blank string = auto)")
			f.String("p", "payload", "", "cursed chrome payload file path (.js)")
			f.Bool("k", "keep-alive", false, "keeps browser alive after last browser window closes")
			f.Bool("H", "headless", false, "start browser process in headless mode")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.StringList("args", "additional chrome cli arguments", grumble.Default([]string{}))
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			cursed.CursedChromeCmd(ctx, con)
			con.Println()
			return nil
		},
	})
	cursedCmd.AddCommand(&grumble.Command{
		Name:      consts.CursedEdge,
		Help:      "Automatically inject a Cursed Chrome payload into a remote Edge extension",
		LongHelp:  help.GetHelpFor([]string{consts.Cursed, consts.CursedEdge}),
		HelpGroup: consts.GenericHelpGroup,
		Flags: func(f *grumble.Flags) {
			f.Int("r", "remote-debugging-port", 0, "remote debugging tcp port (0 = random)")
			f.Bool("R", "restore", true, "restore the user's session after process termination")
			f.String("e", "exe", "", "edge browser executable path (blank string = auto)")
			f.String("u", "user-data", "", "user data directory (blank string = auto)")
			f.String("p", "payload", "", "cursed chrome payload file path (.js)")
			f.Bool("k", "keep-alive", false, "keeps browser alive after last browser window closes")
			f.Bool("H", "headless", false, "start browser process in headless mode")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.StringList("args", "additional edge cli arguments", grumble.Default([]string{}))
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			cursed.CursedEdgeCmd(ctx, con)
			con.Println()
			return nil
		},
	})
	cursedCmd.AddCommand(&grumble.Command{
		Name:      consts.CursedElectron,
		Help:      "Curse a remote Electron application",
		LongHelp:  help.GetHelpFor([]string{consts.Cursed, consts.CursedElectron}),
		HelpGroup: consts.GenericHelpGroup,
		Flags: func(f *grumble.Flags) {
			f.String("e", "exe", "", "remote electron executable absolute path")
			f.Int("r", "remote-debugging-port", 0, "remote debugging tcp port (0 = random)")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			a.StringList("args", "additional electron cli arguments", grumble.Default([]string{}))
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			cursed.CursedElectronCmd(ctx, con)
			con.Println()
			return nil
		},
	})
	cursedCmd.AddCommand(&grumble.Command{
		Name:      consts.CursedCookies,
		Help:      "Dump all cookies from cursed process",
		LongHelp:  help.GetHelpFor([]string{consts.Cursed, consts.CursedCookies}),
		HelpGroup: consts.GenericHelpGroup,
		Flags: func(f *grumble.Flags) {
			f.String("s", "save", "", "save to file")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			cursed.CursedCookiesCmd(ctx, con)
			con.Println()
			return nil
		},
	})
	cursedCmd.AddCommand(&grumble.Command{
		Name:      consts.ScreenshotStr,
		Help:      "Take a screenshot of a cursed process debug target",
		LongHelp:  help.GetHelpFor([]string{consts.Cursed, consts.ScreenshotStr}),
		HelpGroup: consts.GenericHelpGroup,
		Flags: func(f *grumble.Flags) {
			f.Int64("q", "quality", 100, "screenshot quality (1 - 100)")
			f.String("s", "save", "", "save to file")

			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			cursed.CursedScreenshotCmd(ctx, con)
			con.Println()
			return nil
		},
	})
	con.App.AddCommand(cursedCmd)

	// [ Builders ] ---------------------------------------------
	buildersCmd := &grumble.Command{
		Name:      consts.BuildersStr,
		Help:      "List external builders",
		LongHelp:  help.GetHelpFor([]string{consts.BuildersStr}),
		HelpGroup: consts.GenericHelpGroup,
		Flags: func(f *grumble.Flags) {
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			con.Println()
			builders.BuildersCmd(ctx, con)
			con.Println()
			return nil
		},
	}
	con.App.AddCommand(buildersCmd)
}
