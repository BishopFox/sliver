package command

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

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
	"os"

	"github.com/reeflective/console"
	"github.com/reeflective/console/commands/readline"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/command/alias"
	"github.com/bishopfox/sliver/client/command/armory"
	"github.com/bishopfox/sliver/client/command/beacons"
	"github.com/bishopfox/sliver/client/command/builders"
	"github.com/bishopfox/sliver/client/command/c2profiles"
	"github.com/bishopfox/sliver/client/command/crack"
	"github.com/bishopfox/sliver/client/command/creds"
	"github.com/bishopfox/sliver/client/command/exit"
	"github.com/bishopfox/sliver/client/command/extensions"
	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/command/hosts"
	"github.com/bishopfox/sliver/client/command/info"
	"github.com/bishopfox/sliver/client/command/jobs"
	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/command/monitor"
	"github.com/bishopfox/sliver/client/command/operators"
	operator "github.com/bishopfox/sliver/client/command/prelude-operator"
	"github.com/bishopfox/sliver/client/command/reaction"
	"github.com/bishopfox/sliver/client/command/sessions"
	"github.com/bishopfox/sliver/client/command/settings"
	sgn "github.com/bishopfox/sliver/client/command/shikata-ga-nai"
	"github.com/bishopfox/sliver/client/command/taskmany"
	"github.com/bishopfox/sliver/client/command/update"
	"github.com/bishopfox/sliver/client/command/use"
	"github.com/bishopfox/sliver/client/command/websites"
	"github.com/bishopfox/sliver/client/command/wireguard"
	client "github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/licenses"
)

// ServerCommands returns all commands bound to the server menu, optionally
// accepting a function returning a list of additional (admin) commands.
func ServerCommands(con *client.SliverConsoleClient, serverCmds func() []*cobra.Command) console.Commands {
	serverCommands := func() *cobra.Command {
		server := &cobra.Command{
			Short: "Server commands",
			CompletionOptions: cobra.CompletionOptions{
				HiddenDefaultCmd: true,
			},
		}

		// Load Reactions
		n, err := reaction.LoadReactions()
		if err != nil && !os.IsNotExist(err) {
			con.PrintErrorf("Failed to load reactions: %s\n", err)
		} else if n > 0 {
			con.PrintInfof("Loaded %d reaction(s) from disk\n", n)
		}

		// [ Groups ] ----------------------------------------------
		groups := []*cobra.Group{
			{ID: consts.GenericHelpGroup, Title: consts.GenericHelpGroup},
			{ID: consts.NetworkHelpGroup, Title: consts.NetworkHelpGroup},
			{ID: consts.PayloadsHelpGroup, Title: consts.PayloadsHelpGroup},
			{ID: consts.SliverHelpGroup, Title: consts.SliverHelpGroup},
		}
		server.AddGroup(groups...)

		// [ Exit ] ---------------------------------------------------------------
		exitCmd := &cobra.Command{
			Use:   "exit",
			Short: "Exit the program",
			Run: func(cmd *cobra.Command, args []string) {
				exit.ExitCmd(cmd, con, args)
			},
			GroupID: consts.GenericHelpGroup,
		}
		server.AddCommand(exitCmd)

		// [ Aliases ] ---------------------------------------------

		aliasCmd := &cobra.Command{
			Use:   consts.AliasesStr,
			Short: "List current aliases",
			Long:  help.GetHelpFor([]string{consts.AliasesStr}),
			Run: func(cmd *cobra.Command, args []string) {
				alias.AliasesCmd(cmd, con, args)
			},
			GroupID: consts.GenericHelpGroup,
		}
		server.AddCommand(aliasCmd)

		aliasLoadCmd := &cobra.Command{
			Use:   consts.LoadStr + " [ALIAS]",
			Short: "Load a command alias",
			Long:  help.GetHelpFor([]string{consts.AliasesStr, consts.LoadStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				alias.AliasesLoadCmd(cmd, con, args)
			},
		}
		carapace.Gen(aliasLoadCmd).PositionalCompletion(
			carapace.ActionDirectories().Tag("alias directory").Usage("path to the alias directory"))
		aliasCmd.AddCommand(aliasLoadCmd)

		aliasInstallCmd := &cobra.Command{
			Use:   consts.InstallStr + " [ALIAS]",
			Short: "Install a command alias",
			Long:  help.GetHelpFor([]string{consts.AliasesStr, consts.InstallStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				alias.AliasesInstallCmd(cmd, con, args)
			},
		}
		carapace.Gen(aliasInstallCmd).PositionalCompletion(carapace.ActionFiles().Tag("alias file"))
		aliasCmd.AddCommand(aliasInstallCmd)

		aliasRemove := &cobra.Command{
			Use:   consts.RmStr + " [ALIAS]",
			Short: "Remove an alias",
			Long:  help.GetHelpFor([]string{consts.RmStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				alias.AliasesRemoveCmd(cmd, con, args)
			},
		}
		carapace.Gen(aliasRemove).PositionalCompletion(alias.AliasCompleter())
		aliasCmd.AddCommand(aliasRemove)

		// [ Extensions ] ---------------------------------------------

		extCmd := &cobra.Command{
			Use:   consts.ExtensionsStr,
			Short: "List current exts",
			Long:  help.GetHelpFor([]string{consts.ExtensionsStr}),
			Run: func(cmd *cobra.Command, args []string) {
				extensions.ExtensionsCmd(cmd, con)
			},
			GroupID: consts.GenericHelpGroup,
		}
		server.AddCommand(extCmd)


		/*
			 parking 'load' for now - the difference between 'load' and 'install' is that 'load' should not move binaries and manifests to the client install dir.
			 Maybe we can revisit this if it's required - but the usecase I can think of is when developing extensions, and that will also occasionally require a manifest update
			// extLoadCmd := &cobra.Command{
			// 	Use:   consts.LoadStr + " [EXT]",
			// 	Short: "Load a command EXT",
			// 	Long:  help.GetHelpFor([]string{consts.ExtensionsStr, consts.LoadStr}),
			// 	Args:  cobra.ExactArgs(1),
			// 	Run: func(cmd *cobra.Command, args []string) {
			// 		extensions.ExtensionLoadCmd(cmd, con, args)
			// 	},
			// }
			// carapace.Gen(extLoadCmd).PositionalCompletion(
			// 	carapace.ActionDirectories().Tag("ext directory").Usage("path to the ext directory"))
			// extCmd.AddCommand(extLoadCmd)
		*/

		extInstallCmd := &cobra.Command{
			Use:   consts.InstallStr + " [filepath]",
			Short: "Install an extension from a local directory/file.",
			Long:  help.GetHelpFor([]string{consts.ExtensionsStr, consts.InstallStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				extensions.ExtensionsInstallCmd(cmd, con, args)
			},
		}
		carapace.Gen(extInstallCmd).PositionalCompletion(carapace.ActionFiles().Tag("ext file"))
		extCmd.AddCommand(extInstallCmd)

		extendo := &cobra.Command{
			Use:   consts.RmStr + " [Name]",
			Short: "Remove extension. Will remove all commands associated with the installed extension. Does not unload the extension from implants that have already loaded it, but removes the command from the client.",
			Long:  help.GetHelpFor([]string{consts.RmStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				extensions.ExtensionsRemoveCmd(cmd, con, args)
			},
		}
		carapace.Gen(extendo).PositionalCompletion(extensions.ManifestCompleter())
		extCmd.AddCommand(extendo)

		// [ Armory ] ---------------------------------------------

		armoryCmd := &cobra.Command{
			Use:   consts.ArmoryStr,
			Short: "Automatically download and install extensions/aliases",
			Long:  help.GetHelpFor([]string{consts.ArmoryStr}),
			Run: func(cmd *cobra.Command, args []string) {
				armory.ArmoryCmd(cmd, con, args)
			},
			GroupID: consts.GenericHelpGroup,
		}
		Flags("armory", true, armoryCmd, func(f *pflag.FlagSet) {
			f.BoolP("insecure", "I", false, "skip tls certificate validation")
			f.StringP("proxy", "p", "", "specify a proxy url (e.g. http://localhost:8080)")
			f.BoolP("ignore-cache", "c", false, "ignore metadata cache, force refresh")
			f.StringP("timeout", "t", "15m", "download timeout")
		})
		server.AddCommand(armoryCmd)

		armoryInstallCmd := &cobra.Command{
			Use:   consts.InstallStr,
			Short: "Install an alias or extension",
			Long:  help.GetHelpFor([]string{consts.ArmoryStr, consts.InstallStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				armory.ArmoryInstallCmd(cmd, con, args)
			},
		}
		carapace.Gen(armoryInstallCmd).PositionalCompletion(
			armory.AliasExtensionOrBundleCompleter().Usage("name of the extension or alias to install"))
		armoryCmd.AddCommand(armoryInstallCmd)

		armoryUpdateCmd := &cobra.Command{
			Use:   consts.UpdateStr,
			Short: "Update installed an aliases and extensions",
			Long:  help.GetHelpFor([]string{consts.ArmoryStr, consts.UpdateStr}),
			Run: func(cmd *cobra.Command, args []string) {
				armory.ArmoryUpdateCmd(cmd, con, args)
			},
		}
		armoryCmd.AddCommand(armoryUpdateCmd)

		armorySearchCmd := &cobra.Command{
			Use:   consts.SearchStr,
			Short: "Search for aliases and extensions by name (regex)",
			Long:  help.GetHelpFor([]string{consts.ArmoryStr, consts.SearchStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				armory.ArmorySearchCmd(cmd, con, args)
			},
		}
		carapace.Gen(armorySearchCmd).PositionalCompletion(carapace.ActionValues().Usage("a name regular expression"))
		armoryCmd.AddCommand(armorySearchCmd)

		// [ Update ] --------------------------------------------------------------

		updateCmd := &cobra.Command{
			Use:   consts.UpdateStr,
			Short: "Check for updates",
			Long:  help.GetHelpFor([]string{consts.UpdateStr}),
			Run: func(cmd *cobra.Command, args []string) {
				update.UpdateCmd(cmd, con, args)
			},
			GroupID: consts.GenericHelpGroup,
		}
		Flags("update", false, updateCmd, func(f *pflag.FlagSet) {
			f.BoolP("prereleases", "P", false, "include pre-released (unstable) versions")
			f.StringP("proxy", "p", "", "specify a proxy url (e.g. http://localhost:8080)")
			f.StringP("save", "s", "", "save downloaded files to specific directory (default user home dir)")
			f.BoolP("insecure", "I", false, "skip tls certificate validation")
			f.IntP("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		server.AddCommand(updateCmd)

		versionCmd := &cobra.Command{
			Use:   consts.VersionStr,
			Short: "Display version information",
			Long:  help.GetHelpFor([]string{consts.VersionStr}),
			Run: func(cmd *cobra.Command, args []string) {
				update.VerboseVersionsCmd(cmd, con, args)
			},
			GroupID: consts.GenericHelpGroup,
		}
		Flags("update", false, versionCmd, func(f *pflag.FlagSet) {
			f.IntP("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		server.AddCommand(versionCmd)

		// [ Jobs ] -----------------------------------------------------------------

		jobsCmd := &cobra.Command{
			Use:   consts.JobsStr,
			Short: "Job control",
			Long:  help.GetHelpFor([]string{consts.JobsStr}),
			Run: func(cmd *cobra.Command, args []string) {
				jobs.JobsCmd(cmd, con, args)
			},
			GroupID: consts.NetworkHelpGroup,
		}
		Flags("jobs", true, jobsCmd, func(f *pflag.FlagSet) {
			f.IntP("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		Flags("jobs", false, jobsCmd, func(f *pflag.FlagSet) {
			f.Int32P("kill", "k", -1, "kill a background job")
			f.BoolP("kill-all", "K", false, "kill all jobs")
			f.IntP("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		FlagComps(jobsCmd, func(comp *carapace.ActionMap) {
			(*comp)["kill"] = jobs.JobsIDCompleter(con)
		})
		server.AddCommand(jobsCmd)

		mtlsCmd := &cobra.Command{
			Use:   consts.MtlsStr,
			Short: "Start an mTLS listener",
			Long:  help.GetHelpFor([]string{consts.MtlsStr}),
			Run: func(cmd *cobra.Command, args []string) {
				jobs.MTLSListenerCmd(cmd, con, args)
			},
			GroupID: consts.NetworkHelpGroup,
		}
		Flags("mTLS listener", false, mtlsCmd, func(f *pflag.FlagSet) {
			f.StringP("lhost", "L", "", "interface to bind server to")
			f.Uint32P("lport", "l", generate.DefaultMTLSLPort, "tcp listen port")
		})
		server.AddCommand(mtlsCmd)

		wgCmd := &cobra.Command{
			Use:   consts.WGStr,
			Short: "Start a WireGuard listener",
			Long:  help.GetHelpFor([]string{consts.WGStr}),
			Run: func(cmd *cobra.Command, args []string) {
				jobs.WGListenerCmd(cmd, con, args)
			},
			GroupID: consts.NetworkHelpGroup,
		}
		Flags("WireGuard listener", false, wgCmd, func(f *pflag.FlagSet) {
			f.StringP("lhost", "L", "", "interface to bind server to")
			f.Uint32P("lport", "l", generate.DefaultWGLPort, "udp listen port")
			f.Uint32P("nport", "n", generate.DefaultWGNPort, "virtual tun interface listen port")
			f.Uint32P("key-port", "x", generate.DefaultWGKeyExPort, "virtual tun interface key exchange port")
		})
		server.AddCommand(wgCmd)

		dnsCmd := &cobra.Command{
			Use:   consts.DnsStr,
			Short: "Start a DNS listener",
			Long:  help.GetHelpFor([]string{consts.DnsStr}),
			Run: func(cmd *cobra.Command, args []string) {
				jobs.DNSListenerCmd(cmd, con, args)
			},
			GroupID: consts.NetworkHelpGroup,
		}
		Flags("DNS listener", false, dnsCmd, func(f *pflag.FlagSet) {
			f.StringP("domains", "d", "", "parent domain(s) to use for DNS c2")
			f.BoolP("no-canaries", "c", false, "disable dns canary detection")
			f.StringP("lhost", "L", "", "interface to bind server to")
			f.Uint32P("lport", "l", generate.DefaultDNSLPort, "udp listen port")
			f.BoolP("disable-otp", "D", false, "disable otp authentication")
		})
		server.AddCommand(dnsCmd)

		httpCmd := &cobra.Command{
			Use:   consts.HttpStr,
			Short: "Start an HTTP listener",
			Long:  help.GetHelpFor([]string{consts.HttpStr}),
			Run: func(cmd *cobra.Command, args []string) {
				jobs.HTTPListenerCmd(cmd, con, args)
			},
			GroupID: consts.NetworkHelpGroup,
		}
		Flags("HTTP listener", false, httpCmd, func(f *pflag.FlagSet) {
			f.StringP("domain", "d", "", "limit responses to specific domain")
			f.StringP("website", "w", "", "website name (see websites cmd)")
			f.StringP("lhost", "L", "", "interface to bind server to")
			f.Uint32P("lport", "l", generate.DefaultHTTPLPort, "tcp listen port")
			f.BoolP("disable-otp", "D", false, "disable otp authentication")
			f.StringP("long-poll-timeout", "T", "1s", "server-side long poll timeout")
			f.StringP("long-poll-jitter", "J", "2s", "server-side long poll jitter")
			f.BoolP("staging", "s", false, "enable staging")
		})
		server.AddCommand(httpCmd)

		httpsCmd := &cobra.Command{
			Use:   consts.HttpsStr,
			Short: "Start an HTTPS listener",
			Long:  help.GetHelpFor([]string{consts.HttpsStr}),
			Run: func(cmd *cobra.Command, args []string) {
				jobs.HTTPSListenerCmd(cmd, con, args)
			},
			GroupID: consts.NetworkHelpGroup,
		}
		Flags("HTTPS listener", false, httpsCmd, func(f *pflag.FlagSet) {
			f.StringP("domain", "d", "", "limit responses to specific domain")
			f.StringP("website", "w", "", "website name (see websites cmd)")
			f.StringP("lhost", "L", "", "interface to bind server to")
			f.Uint32P("lport", "l", generate.DefaultHTTPSLPort, "tcp listen port")
			f.BoolP("disable-otp", "D", false, "disable otp authentication")
			f.StringP("long-poll-timeout", "T", "1s", "server-side long poll timeout")
			f.StringP("long-poll-jitter", "J", "2s", "server-side long poll jitter")
			f.BoolP("enable-staging", "s", false, "enable staging")

			f.StringP("cert", "c", "", "PEM encoded certificate file")
			f.StringP("key", "k", "", "PEM encoded private key file")
			f.BoolP("lets-encrypt", "e", false, "attempt to provision a let's encrypt certificate")
			f.BoolP("disable-randomized-jarm", "E", false, "disable randomized jarm fingerprints")

		})
		server.AddCommand(httpsCmd)

		stageCmd := &cobra.Command{
			Use:   consts.StageListenerStr,
			Short: "Start a stager listener",
			Long:  help.GetHelpFor([]string{consts.StageListenerStr}),
			Run: func(cmd *cobra.Command, args []string) {
				jobs.StageListenerCmd(cmd, con, args)
			},
			GroupID: consts.NetworkHelpGroup,
		}
		Flags("stage listener", false, stageCmd, func(f *pflag.FlagSet) {
			f.StringP("profile", "p", "", "implant profile name to link with the listener")
			f.StringP("url", "u", "", "URL to which the stager will call back to")
			f.StringP("cert", "c", "", "path to PEM encoded certificate file (HTTPS only)")
			f.StringP("key", "k", "", "path to PEM encoded private key file (HTTPS only)")
			f.BoolP("lets-encrypt", "e", false, "attempt to provision a let's encrypt certificate (HTTPS only)")
			f.String("aes-encrypt-key", "", "encrypt stage with AES encryption key")
			f.String("aes-encrypt-iv", "", "encrypt stage with AES encryption iv")
			f.String("rc4-encrypt-key", "", "encrypt stage with RC4 encryption key")
			f.StringP("compress", "C", "none", "compress the stage before encrypting (zlib, gzip, deflate9, none)")
			f.BoolP("prepend-size", "P", false, "prepend the size of the stage to the payload (to use with MSF stagers)")
		})
		FlagComps(stageCmd, func(comp *carapace.ActionMap) {
			(*comp)["profile"] = generate.ProfileNameCompleter(con)
			(*comp)["cert"] = carapace.ActionFiles().Tag("certificate file")
			(*comp)["key"] = carapace.ActionFiles().Tag("key file")
			(*comp)["compress"] = carapace.ActionValues([]string{"zlib", "gzip", "deflate9", "none"}...).Tag("compression formats")
		})
		server.AddCommand(stageCmd)

		// [ Operators ] --------------------------------------------------------------

		operatorsCmd := &cobra.Command{
			Use:   consts.OperatorsStr,
			Short: "Manage operators",
			Long:  help.GetHelpFor([]string{consts.OperatorsStr}),
			Run: func(cmd *cobra.Command, args []string) {
				operators.OperatorsCmd(cmd, con, args)
			},
			GroupID: consts.GenericHelpGroup,
		}
		Flags("operators", false, operatorsCmd, func(f *pflag.FlagSet) {
			f.IntP("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		server.AddCommand(operatorsCmd)

		// Server-only commands.
		if serverCmds != nil {
			server.AddGroup(&cobra.Group{ID: consts.MultiplayerHelpGroup, Title: consts.MultiplayerHelpGroup})
			server.AddCommand(serverCmds()...)
		}

		// [ Sessions ] --------------------------------------------------------------

		sessionsCmd := &cobra.Command{
			Use:   consts.SessionsStr,
			Short: "Session management",
			Long:  help.GetHelpFor([]string{consts.SessionsStr}),
			Run: func(cmd *cobra.Command, args []string) {
				sessions.SessionsCmd(cmd, con, args)
			},
			GroupID: consts.SliverHelpGroup,
		}
		Flags("sessions", true, sessionsCmd, func(f *pflag.FlagSet) {
			f.IntP("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		Flags("sessions", false, sessionsCmd, func(f *pflag.FlagSet) {
			f.StringP("interact", "i", "", "interact with a session")
			f.StringP("kill", "k", "", "kill the designated session")
			f.BoolP("kill-all", "K", false, "kill all the sessions")
			f.BoolP("clean", "C", false, "clean out any sessions marked as [DEAD]")
			f.BoolP("force", "F", false, "force session action without waiting for results")

			f.StringP("filter", "f", "", "filter sessions by substring")
			f.StringP("filter-re", "e", "", "filter sessions by regular expression")
		})
		FlagComps(sessionsCmd, func(comp *carapace.ActionMap) {
			(*comp)["interact"] = use.BeaconAndSessionIDCompleter(con)
			(*comp)["kill"] = use.BeaconAndSessionIDCompleter(con)
		})
		server.AddCommand(sessionsCmd)

		sessionsPruneCmd := &cobra.Command{
			Use:   consts.PruneStr,
			Short: "Kill all stale/dead sessions",
			Long:  help.GetHelpFor([]string{consts.SessionsStr, consts.PruneStr}),
			Run: func(cmd *cobra.Command, args []string) {
				sessions.SessionsPruneCmd(cmd, con, args)
			},
		}
		Flags("prune", false, sessionsPruneCmd, func(f *pflag.FlagSet) {
			f.BoolP("force", "F", false, "Force the killing of stale/dead sessions")
		})
		sessionsCmd.AddCommand(sessionsPruneCmd)

		// [ Use ] --------------------------------------------------------------

		useCmd := &cobra.Command{
			Use:   consts.UseStr,
			Short: "Switch the active session or beacon",
			Long:  help.GetHelpFor([]string{consts.UseStr}),
			Run: func(cmd *cobra.Command, args []string) {
				use.UseCmd(cmd, con, args)
			},
			GroupID: consts.SliverHelpGroup,
		}
		Flags("use", true, useCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		carapace.Gen(useCmd).PositionalCompletion(use.BeaconAndSessionIDCompleter(con))
		server.AddCommand(useCmd)

		useSessionCmd := &cobra.Command{
			Use:   consts.SessionsStr,
			Short: "Switch the active session",
			Long:  help.GetHelpFor([]string{consts.UseStr, consts.SessionsStr}),
			Run: func(cmd *cobra.Command, args []string) {
				use.UseSessionCmd(cmd, con, args)
			},
		}
		carapace.Gen(useSessionCmd).PositionalCompletion(use.SessionIDCompleter(con))
		useCmd.AddCommand(useSessionCmd)

		useBeaconCmd := &cobra.Command{
			Use:   consts.BeaconsStr,
			Short: "Switch the active beacon",
			Long:  help.GetHelpFor([]string{consts.UseStr, consts.BeaconsStr}),
			Run: func(cmd *cobra.Command, args []string) {
				use.UseBeaconCmd(cmd, con, args)
			},
		}
		carapace.Gen(useBeaconCmd).PositionalCompletion(use.BeaconIDCompleter(con))
		useCmd.AddCommand(useBeaconCmd)

		// [ Settings ] --------------------------------------------------------------

		settingsCmd := &cobra.Command{
			Use:   consts.SettingsStr,
			Short: "Manage client settings",
			Long:  help.GetHelpFor([]string{consts.SettingsStr}),
			Run: func(cmd *cobra.Command, args []string) {
				settings.SettingsCmd(cmd, con, args)
			},
			GroupID: consts.GenericHelpGroup,
		}
		settingsCmd.AddCommand(&cobra.Command{
			Use:   consts.SaveStr,
			Short: "Save the current settings to disk",
			Long:  help.GetHelpFor([]string{consts.SettingsStr, consts.SaveStr}),
			Run: func(cmd *cobra.Command, args []string) {
				settings.SettingsSaveCmd(cmd, con, args)
			},
		})
		settingsCmd.AddCommand(&cobra.Command{
			Use:   consts.TablesStr,
			Short: "Modify tables setting (style)",
			Long:  help.GetHelpFor([]string{consts.SettingsStr, consts.TablesStr}),
			Run: func(cmd *cobra.Command, args []string) {
				settings.SettingsTablesCmd(cmd, con, args)
			},
		})
		settingsCmd.AddCommand(&cobra.Command{
			Use:   "beacon-autoresults",
			Short: "Automatically display beacon task results when completed",
			Long:  help.GetHelpFor([]string{consts.SettingsStr, "beacon-autoresults"}),
			Run: func(cmd *cobra.Command, args []string) {
				settings.SettingsBeaconsAutoResultCmd(cmd, con, args)
			},
		})
		settingsCmd.AddCommand(&cobra.Command{
			Use:   "autoadult",
			Short: "Automatically accept OPSEC warnings",
			Long:  help.GetHelpFor([]string{consts.SettingsStr, "autoadult"}),
			Run: func(cmd *cobra.Command, args []string) {
				settings.SettingsAutoAdultCmd(cmd, con, args)
			},
		})
		settingsCmd.AddCommand(&cobra.Command{
			Use:   "always-overflow",
			Short: "Disable table pagination",
			Long:  help.GetHelpFor([]string{consts.SettingsStr, "always-overflow"}),
			Run: func(cmd *cobra.Command, args []string) {
				settings.SettingsAlwaysOverflow(cmd, con, args)
			},
		})
		settingsCmd.AddCommand(&cobra.Command{
			Use:   "small-terminal",
			Short: "Set the small terminal width",
			Long:  help.GetHelpFor([]string{consts.SettingsStr, "small-terminal"}),
			Run: func(cmd *cobra.Command, args []string) {
				settings.SettingsSmallTerm(cmd, con, args)
			},
		})
		settingsCmd.AddCommand(&cobra.Command{
			Use:   "user-connect",
			Short: "Enable user connections/disconnections (can be very verbose when they use CLI)",
			Run: func(cmd *cobra.Command, args []string) {
				settings.SettingsUserConnect(cmd, con, args)
			},
		})
		settingsCmd.AddCommand(&cobra.Command{
			Use:   "console-logs",
			Short: "Log console output (toggle)",
			Long:  help.GetHelpFor([]string{consts.SettingsStr, "console-logs"}),
			Run: func(ctx *cobra.Command, args []string) {
				settings.SettingsConsoleLogs(ctx, con)
			},
		})
		server.AddCommand(settingsCmd)

		// [ Info ] --------------------------------------------------------------

		infoCmd := &cobra.Command{
			Use:   consts.InfoStr,
			Short: "Get info about session",
			Long:  help.GetHelpFor([]string{consts.InfoStr}),
			Run: func(cmd *cobra.Command, args []string) {
				info.InfoCmd(cmd, con, args)
			},
			GroupID: consts.SliverHelpGroup,
		}
		Flags("use", false, infoCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		carapace.Gen(infoCmd).PositionalCompletion(use.BeaconAndSessionIDCompleter(con))
		server.AddCommand(infoCmd)

		// [ Shellcode Encoders ] --------------------------------------------------------------

		shikataGaNaiCmd := &cobra.Command{
			Use:   consts.ShikataGaNai,
			Short: "Polymorphic binary shellcode encoder (ノ ゜Д゜)ノ ︵ 仕方がない",
			Long:  help.GetHelpFor([]string{consts.ShikataGaNai}),
			Run: func(cmd *cobra.Command, args []string) {
				sgn.ShikataGaNaiCmd(cmd, con, args)
			},
			Args:    cobra.ExactArgs(1),
			GroupID: consts.PayloadsHelpGroup,
		}
		server.AddCommand(shikataGaNaiCmd)
		Flags("shikata ga nai", false, shikataGaNaiCmd, func(f *pflag.FlagSet) {
			f.StringP("save", "s", "", "save output to local file")
			f.StringP("arch", "a", "amd64", "architecture of shellcode")
			f.IntP("iterations", "i", 1, "number of iterations")
			f.StringP("bad-chars", "b", "", "hex encoded bad characters to avoid (e.g. 0001)")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		FlagComps(shikataGaNaiCmd, func(comp *carapace.ActionMap) {
			(*comp)["arch"] = carapace.ActionValues("386", "amd64").Tag("shikata-ga-nai architectures")
			(*comp)["save"] = carapace.ActionFiles().Tag("directory/file to save shellcode")
		})
		carapace.Gen(shikataGaNaiCmd).PositionalCompletion(carapace.ActionFiles().Tag("shellcode file"))

		// [ Generate ] --------------------------------------------------------------

		generateCmd := &cobra.Command{
			Use:   consts.GenerateStr,
			Short: "Generate an implant binary",
			Long:  help.GetHelpFor([]string{consts.GenerateStr}),
			Run: func(cmd *cobra.Command, args []string) {
				generate.GenerateCmd(cmd, con, args)
			},
			GroupID: consts.PayloadsHelpGroup,
		}
		Flags("generate", true, generateCmd, func(f *pflag.FlagSet) {
			f.IntP("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		Flags("session", false, generateCmd, func(f *pflag.FlagSet) {
			f.StringP("os", "o", "windows", "operating system")
			f.StringP("arch", "a", "amd64", "cpu architecture")
			f.StringP("name", "N", "", "agent name")
			f.BoolP("debug", "d", false, "enable debug features")
			f.StringP("debug-file", "O", "", "path to debug output")
			f.BoolP("evasion", "e", false, "enable evasion features (e.g. overwrite user space hooks)")
			f.BoolP("skip-symbols", "l", false, "skip symbol obfuscation")
			f.StringP("template", "I", "sliver", "implant code template")
			f.BoolP("external-builder", "E", false, "use an external builder")
			f.BoolP("disable-sgn", "G", false, "disable shikata ga nai shellcode encoder")

			f.StringP("canary", "c", "", "canary domain(s)")

			f.StringP("mtls", "m", "", "mtls connection strings")
			f.StringP("wg", "g", "", "wg connection strings")
			f.StringP("http", "b", "", "http(s) connection strings")
			f.StringP("dns", "n", "", "dns connection strings")
			f.StringP("named-pipe", "p", "", "named-pipe connection strings")
			f.StringP("tcp-pivot", "i", "", "tcp-pivot connection strings")

			f.Uint32P("key-exchange", "X", generate.DefaultWGKeyExPort, "wg key-exchange port")
			f.Uint32P("tcp-comms", "T", generate.DefaultWGNPort, "wg c2 comms port")

			f.BoolP("run-at-load", "R", false, "run the implant entrypoint from DllMain/Constructor (shared library only)")
			f.BoolP("netgo", "q", false, "force the use of netgo")
			f.StringP("traffic-encoders", "A", "", "comma separated list of traffic encoders to enable")

			f.StringP("strategy", "Z", "", "specify a connection strategy (r = random, rd = random domain, s = sequential)")
			f.Int64P("reconnect", "j", generate.DefaultReconnect, "attempt to reconnect every n second(s)")
			f.Int64P("poll-timeout", "P", generate.DefaultPollTimeout, "long poll request timeout")
			f.Uint32P("max-errors", "k", generate.DefaultMaxErrors, "max number of connection errors")

			f.StringP("limit-datetime", "w", "", "limit execution to before datetime")
			f.BoolP("limit-domainjoined", "x", false, "limit execution to domain joined machines")
			f.StringP("limit-username", "y", "", "limit execution to specified username")
			f.StringP("limit-hostname", "z", "", "limit execution to specified hostname")
			f.StringP("limit-fileexists", "F", "", "limit execution to hosts with this file in the filesystem")
			f.StringP("limit-locale", "L", "", "limit execution to hosts that match this locale")

			f.StringP("format", "f", "exe", "Specifies the output formats, valid values are: 'exe', 'shared' (for dynamic libraries), 'service' (see: `psexec` for more info) and 'shellcode' (windows only)")
			f.StringP("save", "s", "", "directory/file to the binary to")
			f.StringP("c2profile", "C", constants.DefaultC2Profile, "HTTP C2 profile to use")
		})
		FlagComps(generateCmd, func(comp *carapace.ActionMap) {
			(*comp)["debug-file"] = carapace.ActionFiles()
			(*comp)["os"] = generate.OSCompleter(con)
			(*comp)["arch"] = generate.ArchCompleter(con)
			(*comp)["strategy"] = carapace.ActionValuesDescribed([]string{"r", "random", "rd", "random domain", "s", "sequential"}...).Tag("C2 strategy")
			(*comp)["format"] = generate.FormatCompleter()
			(*comp)["save"] = carapace.ActionFiles().Tag("directory/file to save implant")
			(*comp)["c2profile"] = generate.HTTPC2Completer(con)
		})
		server.AddCommand(generateCmd)

		generateBeaconCmd := &cobra.Command{
			Use:   consts.BeaconStr,
			Short: "Generate a beacon binary",
			Long:  help.GetHelpFor([]string{consts.GenerateStr, consts.BeaconStr}),
			Run: func(cmd *cobra.Command, args []string) {
				generate.GenerateBeaconCmd(cmd, con, args)
			},
		}
		Flags("beacon", false, generateBeaconCmd, func(f *pflag.FlagSet) {
			f.Int64P("days", "D", 0, "beacon interval days")
			f.Int64P("hours", "H", 0, "beacon interval hours")
			f.Int64P("minutes", "M", 0, "beacon interval minutes")
			f.Int64P("seconds", "S", 60, "beacon interval seconds")
			f.Int64P("jitter", "J", 30, "beacon interval jitter in seconds")

			// Generate flags
			f.StringP("os", "o", "windows", "operating system")
			f.StringP("arch", "a", "amd64", "cpu architecture")
			f.StringP("name", "N", "", "agent name")
			f.BoolP("debug", "d", false, "enable debug features")
			f.StringP("debug-file", "O", "", "path to debug output")
			f.BoolP("evasion", "e", false, "enable evasion features  (e.g. overwrite user space hooks)")
			f.BoolP("skip-symbols", "l", false, "skip symbol obfuscation")
			f.StringP("template", "I", "sliver", "implant code template")
			f.BoolP("external-builder", "E", false, "use an external builder")
			f.BoolP("disable-sgn", "G", false, "disable shikata ga nai shellcode encoder")

			f.StringP("canary", "c", "", "canary domain(s)")

			f.StringP("mtls", "m", "", "mtls connection strings")
			f.StringP("wg", "g", "", "wg connection strings")
			f.StringP("http", "b", "", "http(s) connection strings")
			f.StringP("dns", "n", "", "dns connection strings")
			f.StringP("named-pipe", "p", "", "named-pipe connection strings")
			f.StringP("tcp-pivot", "i", "", "tcp-pivot connection strings")

			f.Uint32P("key-exchange", "X", generate.DefaultWGKeyExPort, "wg key-exchange port")
			f.Uint32P("tcp-comms", "T", generate.DefaultWGNPort, "wg c2 comms port")

			f.BoolP("run-at-load", "R", false, "run the implant entrypoint from DllMain/Constructor (shared library only)")
			f.BoolP("netgo", "q", false, "force the use of netgo")
			f.StringP("traffic-encoders", "A", "", "comma separated list of traffic encoders to enable")

			f.StringP("strategy", "Z", "", "specify a connection strategy (r = random, rd = random domain, s = sequential)")
			f.Int64P("reconnect", "j", generate.DefaultReconnect, "attempt to reconnect every n second(s)")
			f.Int64P("poll-timeout", "P", generate.DefaultPollTimeout, "long poll request timeout")
			f.Uint32P("max-errors", "k", generate.DefaultMaxErrors, "max number of connection errors")

			f.StringP("limit-datetime", "w", "", "limit execution to before datetime")
			f.BoolP("limit-domainjoined", "x", false, "limit execution to domain joined machines")
			f.StringP("limit-username", "y", "", "limit execution to specified username")
			f.StringP("limit-hostname", "z", "", "limit execution to specified hostname")
			f.StringP("limit-fileexists", "F", "", "limit execution to hosts with this file in the filesystem")
			f.StringP("limit-locale", "L", "", "limit execution to hosts that match this locale")

			f.StringP("format", "f", "exe", "Specifies the output formats, valid values are: 'exe', 'shared' (for dynamic libraries), 'service' (see: `psexec` for more info) and 'shellcode' (windows only)")
			f.StringP("save", "s", "", "directory/file to the binary to")
			f.StringP("c2profile", "C", constants.DefaultC2Profile, "HTTP C2 profile to use")
		})
		FlagComps(generateBeaconCmd, func(comp *carapace.ActionMap) {
			(*comp)["debug-file"] = carapace.ActionFiles()
			(*comp)["os"] = generate.OSCompleter(con)
			(*comp)["arch"] = generate.ArchCompleter(con)
			(*comp)["strategy"] = carapace.ActionValuesDescribed([]string{"r", "random", "rd", "random domain", "s", "sequential"}...).Tag("C2 strategy")
			(*comp)["format"] = generate.FormatCompleter()
			(*comp)["save"] = carapace.ActionFiles().Tag("directory/file to save implant")
			(*comp)["c2profile"] = generate.HTTPC2Completer(con)
		})
		generateCmd.AddCommand(generateBeaconCmd)

		generateStagerCmd := &cobra.Command{
			Use:   consts.MsfStagerStr,
			Short: "Generate a stager using Metasploit (requires local Metasploit installation)",
			Long:  help.GetHelpFor([]string{consts.MsfStagerStr}),
			Run: func(cmd *cobra.Command, args []string) {
				generate.GenerateStagerCmd(cmd, con, args)
			},
		}
		Flags("stager", false, generateStagerCmd, func(f *pflag.FlagSet) {
			f.StringP("os", "o", "windows", "operating system")
			f.StringP("arch", "a", "amd64", "cpu architecture")
			f.StringP("lhost", "L", "", "Listening host")
			f.Uint32P("lport", "l", 8443, "Listening port")
			f.StringP("protocol", "r", "tcp", "Staging protocol (tcp/http/https)")
			f.StringP("format", "f", "raw", "Output format (msfvenom formats, see help generate msf-stager for the list)")
			f.StringP("badchars", "b", "", "bytes to exclude from stage shellcode")
			f.StringP("save", "s", "", "directory to save the generated stager to")
			f.StringP("advanced", "d", "", "Advanced options for the stager using URI query syntax (option1=value1&option2=value2...)")
		})
		generateCmd.AddCommand(generateStagerCmd)

		generateInfoCmd := &cobra.Command{
			Use:   consts.CompilerInfoStr,
			Short: "Get information about the server's compiler",
			Long:  help.GetHelpFor([]string{consts.CompilerInfoStr}),
			Run: func(cmd *cobra.Command, args []string) {
				generate.GenerateInfoCmd(cmd, con, args)
			},
		}
		generateCmd.AddCommand(generateInfoCmd)

		// Traffic Encoder SubCommands
		trafficEncodersCmd := &cobra.Command{
			Use:   consts.TrafficEncodersStr,
			Short: "Manage implant traffic encoders",
			Long:  help.GetHelpFor([]string{consts.GenerateStr, consts.TrafficEncodersStr}),
			Run: func(cmd *cobra.Command, args []string) {
				generate.TrafficEncodersCmd(cmd, con, args)
			},
		}
		generateCmd.AddCommand(trafficEncodersCmd)

		trafficEncodersAddCmd := &cobra.Command{
			Use:   consts.AddStr,
			Short: "Add a new traffic encoder to the server from the local file system",
			Long:  help.GetHelpFor([]string{consts.GenerateStr, consts.TrafficEncodersStr, consts.AddStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				generate.TrafficEncodersAddCmd(cmd, con, args)
			},
		}
		Flags("", false, trafficEncodersAddCmd, func(f *pflag.FlagSet) {
			f.BoolP("skip-tests", "s", false, "skip testing the traffic encoder (not recommended)")
		})
		carapace.Gen(trafficEncodersAddCmd).PositionalCompletion(carapace.ActionFiles("wasm").Tag("wasm files").Usage("local file path (expects .wasm)"))
		trafficEncodersCmd.AddCommand(trafficEncodersAddCmd)

		trafficEncodersRmCmd := &cobra.Command{
			Use:   consts.RmStr,
			Short: "Remove a traffic encoder from the server",
			Long:  help.GetHelpFor([]string{consts.GenerateStr, consts.TrafficEncodersStr, consts.RmStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				generate.TrafficEncodersRemoveCmd(cmd, con, args)
			},
		}
		carapace.Gen(trafficEncodersRmCmd).PositionalCompletion(generate.TrafficEncodersCompleter(con).Usage("traffic encoder to remove"))
		trafficEncodersCmd.AddCommand(trafficEncodersRmCmd)

		// [ Regenerate ] --------------------------------------------------------------

		regenerateCmd := &cobra.Command{
			Use:   consts.RegenerateStr,
			Short: "Regenerate an implant",
			Long:  help.GetHelpFor([]string{consts.RegenerateStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				generate.RegenerateCmd(cmd, con, args)
			},
			GroupID: consts.PayloadsHelpGroup,
		}
		Flags("regenerate", false, regenerateCmd, func(f *pflag.FlagSet) {
			f.StringP("save", "s", "", "directory/file to the binary to")
		})
		FlagComps(regenerateCmd, func(comp *carapace.ActionMap) {
			(*comp)["save"] = carapace.ActionFiles().Tag("directory/file to save implant")
		})
		carapace.Gen(regenerateCmd).PositionalCompletion(generate.ImplantBuildNameCompleter(con))
		server.AddCommand(regenerateCmd)

		// [ Profiles ] --------------------------------------------------------------

		profilesCmd := &cobra.Command{
			Use:   consts.ProfilesStr,
			Short: "List existing profiles",
			Long:  help.GetHelpFor([]string{consts.ProfilesStr}),
			Run: func(cmd *cobra.Command, args []string) {
				generate.ProfilesCmd(cmd, con, args)
			},
			GroupID: consts.PayloadsHelpGroup,
		}
		Flags("profiles", true, profilesCmd, func(f *pflag.FlagSet) {
			f.IntP("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		server.AddCommand(profilesCmd)

		profilesGenerateCmd := &cobra.Command{
			Use:   consts.GenerateStr,
			Short: "Generate implant from a profile",
			Long:  help.GetHelpFor([]string{consts.ProfilesStr, consts.GenerateStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				generate.ProfilesGenerateCmd(cmd, con, args)
			},
		}
		Flags("profiles", false, profilesGenerateCmd, func(f *pflag.FlagSet) {
			f.StringP("save", "s", "", "directory/file to the binary to")
			f.BoolP("disable-sgn", "G", false, "disable shikata ga nai shellcode encoder")
		})
		FlagComps(profilesGenerateCmd, func(comp *carapace.ActionMap) {
			(*comp)["save"] = carapace.ActionFiles().Tag("directory/file to save implant")
		})
		carapace.Gen(profilesGenerateCmd).PositionalCompletion(generate.ProfileNameCompleter(con))
		profilesCmd.AddCommand(profilesGenerateCmd)

		profilesNewCmd := &cobra.Command{
			Use:   consts.NewStr,
			Short: "Create a new implant profile (interactive session)",
			Long:  help.GetHelpFor([]string{consts.ProfilesStr, consts.NewStr}),
			Run: func(cmd *cobra.Command, args []string) {
				generate.ProfilesNewCmd(cmd, con, args)
			},
		}
		Flags("session", false, profilesNewCmd, func(f *pflag.FlagSet) {
			f.StringP("os", "o", "windows", "operating system")
			f.StringP("arch", "a", "amd64", "cpu architecture")

			f.BoolP("debug", "d", false, "enable debug features")
			f.StringP("debug-file", "O", "", "path to debug output")
			f.BoolP("evasion", "e", false, "enable evasion features (e.g. overwrite user space hooks)")
			f.BoolP("skip-symbols", "l", false, "skip symbol obfuscation")
			f.BoolP("disable-sgn", "G", false, "disable shikata ga nai shellcode encoder")

			f.StringP("canary", "c", "", "canary domain(s)")

			f.StringP("name", "N", "", "agent name")
			f.StringP("mtls", "m", "", "mtls connection strings")
			f.StringP("wg", "g", "", "wg connection strings")
			f.StringP("http", "b", "", "http(s) connection strings")
			f.StringP("dns", "n", "", "dns connection strings")
			f.StringP("named-pipe", "p", "", "named-pipe connection strings")
			f.StringP("tcp-pivot", "i", "", "tcp-pivot connection strings")

			f.Uint32P("key-exchange", "X", generate.DefaultWGKeyExPort, "wg key-exchange port")
			f.Uint32P("tcp-comms", "T", generate.DefaultWGNPort, "wg c2 comms port")

			f.BoolP("run-at-load", "R", false, "run the implant entrypoint from DllMain/Constructor (shared library only)")
			f.StringP("strategy", "Z", "", "specify a connection strategy (r = random, rd = random domain, s = sequential)")

			f.BoolP("netgo", "q", false, "force the use of netgo")
			f.StringP("traffic-encoders", "A", "", "comma separated list of traffic encoders to enable")

			f.StringP("template", "I", "sliver", "implant code template")

			f.Int64P("reconnect", "j", generate.DefaultReconnect, "attempt to reconnect every n second(s)")
			f.Int64P("poll-timeout", "P", generate.DefaultPollTimeout, "long poll request timeout")
			f.Uint32P("max-errors", "k", generate.DefaultMaxErrors, "max number of connection errors")

			f.StringP("limit-datetime", "w", "", "limit execution to before datetime")
			f.BoolP("limit-domainjoined", "x", false, "limit execution to domain joined machines")
			f.StringP("limit-username", "y", "", "limit execution to specified username")
			f.StringP("limit-hostname", "z", "", "limit execution to specified hostname")
			f.StringP("limit-fileexists", "F", "", "limit execution to hosts with this file in the filesystem")
			f.StringP("limit-locale", "L", "", "limit execution to hosts that match this locale")

			f.StringP("format", "f", "exe", "Specifies the output formats, valid values are: 'exe', 'shared' (for dynamic libraries), 'service' (see: `psexec` for more info) and 'shellcode' (windows only)")
			f.StringP("c2profile", "C", constants.DefaultC2Profile, "HTTP C2 profile to use")
		})
		FlagComps(profilesNewCmd, func(comp *carapace.ActionMap) {
			(*comp)["debug-file"] = carapace.ActionFiles()
			(*comp)["os"] = generate.OSCompleter(con)
			(*comp)["arch"] = generate.ArchCompleter(con)
			(*comp)["strategy"] = carapace.ActionValuesDescribed([]string{"r", "random", "rd", "random domain", "s", "sequential"}...).Tag("C2 strategy")
			(*comp)["format"] = generate.FormatCompleter()
			(*comp)["save"] = carapace.ActionFiles().Tag("directory/file to save implant")
			(*comp)["c2profile"] = generate.HTTPC2Completer(con)
		})
		carapace.Gen(profilesNewCmd).PositionalCompletion(carapace.ActionValues().Usage("name of the session profile (optional)"))
		profilesCmd.AddCommand(profilesNewCmd)

		// New Beacon Profile Command
		profilesNewBeaconCmd := &cobra.Command{
			Use:   consts.BeaconStr,
			Short: "Create a new implant profile (beacon)",
			Long:  help.GetHelpFor([]string{consts.ProfilesStr, consts.NewStr, consts.BeaconStr}),
			Run: func(cmd *cobra.Command, args []string) {
				generate.ProfilesNewBeaconCmd(cmd, con, args)
			},
		}
		Flags("beacon", false, profilesNewBeaconCmd, func(f *pflag.FlagSet) {
			f.Int64P("days", "D", 0, "beacon interval days")
			f.Int64P("hours", "H", 0, "beacon interval hours")
			f.Int64P("minutes", "M", 0, "beacon interval minutes")
			f.Int64P("seconds", "S", 60, "beacon interval seconds")
			f.Int64P("jitter", "J", 30, "beacon interval jitter in seconds")
			f.BoolP("disable-sgn", "G", false, "disable shikata ga nai shellcode encoder")

			// Generate flags
			f.StringP("os", "o", "windows", "operating system")
			f.StringP("arch", "a", "amd64", "cpu architecture")

			f.BoolP("debug", "d", false, "enable debug features")
			f.StringP("debug-file", "O", "", "path to debug output")
			f.BoolP("evasion", "e", false, "enable evasion features  (e.g. overwrite user space hooks)")
			f.BoolP("skip-symbols", "l", false, "skip symbol obfuscation")

			f.StringP("canary", "c", "", "canary domain(s)")

			f.StringP("name", "N", "", "agent name")
			f.StringP("mtls", "m", "", "mtls connection strings")
			f.StringP("wg", "g", "", "wg connection strings")
			f.StringP("http", "b", "", "http(s) connection strings")
			f.StringP("dns", "n", "", "dns connection strings")
			f.StringP("named-pipe", "p", "", "named-pipe connection strings")
			f.StringP("tcp-pivot", "i", "", "tcp-pivot connection strings")
			f.StringP("strategy", "Z", "", "specify a connection strategy (r = random, rd = random domain, s = sequential)")

			f.Uint32P("key-exchange", "X", generate.DefaultWGKeyExPort, "wg key-exchange port")
			f.Uint32P("tcp-comms", "T", generate.DefaultWGNPort, "wg c2 comms port")

			f.BoolP("run-at-load", "R", false, "run the implant entrypoint from DllMain/Constructor (shared library only)")
			f.BoolP("netgo", "q", false, "force the use of netgo")
			f.StringP("traffic-encoders", "A", "", "comma separated list of traffic encoders to enable")

			f.StringP("template", "I", "sliver", "implant code template")

			f.Int64P("reconnect", "j", generate.DefaultReconnect, "attempt to reconnect every n second(s)")
			f.Int64P("poll-timeout", "P", generate.DefaultPollTimeout, "long poll request timeout")
			f.Uint32P("max-errors", "k", generate.DefaultMaxErrors, "max number of connection errors")

			f.StringP("limit-datetime", "w", "", "limit execution to before datetime")
			f.BoolP("limit-domainjoined", "x", false, "limit execution to domain joined machines")
			f.StringP("limit-username", "y", "", "limit execution to specified username")
			f.StringP("limit-hostname", "z", "", "limit execution to specified hostname")
			f.StringP("limit-fileexists", "F", "", "limit execution to hosts with this file in the filesystem")
			f.StringP("limit-locale", "L", "", "limit execution to hosts that match this locale")

			f.StringP("format", "f", "exe", "Specifies the output formats, valid values are: 'exe', 'shared' (for dynamic libraries), 'service' (see: `psexec` for more info) and 'shellcode' (windows only)")
			f.StringP("c2profile", "C", constants.DefaultC2Profile, "HTTP C2 profile to use")
		})
		FlagComps(profilesNewBeaconCmd, func(comp *carapace.ActionMap) {
			(*comp)["debug-file"] = carapace.ActionFiles()
			(*comp)["os"] = generate.OSCompleter(con)
			(*comp)["arch"] = generate.ArchCompleter(con)
			(*comp)["strategy"] = carapace.ActionValuesDescribed([]string{"r", "random", "rd", "random domain", "s", "sequential"}...).Tag("C2 strategy")
			(*comp)["format"] = generate.FormatCompleter()
			(*comp)["save"] = carapace.ActionFiles().Tag("directory/file to save implant")
			(*comp)["c2profile"] = generate.HTTPC2Completer(con)
		})
		carapace.Gen(profilesNewBeaconCmd).PositionalCompletion(carapace.ActionValues().Usage("name of the beacon profile (optional)"))
		profilesNewCmd.AddCommand(profilesNewBeaconCmd)

		profilesRmCmd := &cobra.Command{
			Use:   consts.RmStr,
			Short: "Remove a profile",
			Long:  help.GetHelpFor([]string{consts.ProfilesStr, consts.RmStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				generate.ProfilesRmCmd(cmd, con, args)
			},
		}
		carapace.Gen(profilesRmCmd).PositionalCompletion(generate.ProfileNameCompleter(con))
		profilesCmd.AddCommand(profilesRmCmd)

		profilesStageCmd := &cobra.Command{
			Use:   consts.StageStr,
			Short: "Generate an encrypted and/or compressed implant",
			Long:  help.GetHelpFor([]string{consts.ProfilesStr, consts.StageStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				generate.ProfilesStageCmd(cmd, con, args)
			},
		}
		Flags("stage", false, profilesStageCmd, func(f *pflag.FlagSet) {
			f.StringP("name", "n", "", "implant name")
			f.String("aes-encrypt-key", "", "encrypt stage with AES encryption key")
			f.String("aes-encrypt-iv", "", "encrypt stage with AES encryption iv")
			f.String("rc4-encrypt-key", "", "encrypt stage with RC4 encryption key")
			f.StringP("compress", "C", "", "compress the stage before encrypting (zlib, gzip, deflate9, none)")
			f.BoolP("prepend-size", "P", false, "prepend the size of the stage to the payload (to use with MSF stagers)")
		})
		carapace.Gen(profilesStageCmd).PositionalCompletion(generate.ProfileNameCompleter(con))
		profilesCmd.AddCommand(profilesStageCmd)

		profilesInfoCmd := &cobra.Command{
			Use:   consts.InfoStr,
			Short: "Details about a profile",
			Long:  help.GetHelpFor([]string{consts.ProfilesStr, consts.RmStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				generate.PrintProfileInfo(args[0], con)
			},
		}
		carapace.Gen(profilesInfoCmd).PositionalCompletion(generate.ProfileNameCompleter(con))
		profilesCmd.AddCommand(profilesInfoCmd)

		implantBuildsCmd := &cobra.Command{
			Use:   consts.ImplantBuildsStr,
			Short: "List implant builds",
			Long:  help.GetHelpFor([]string{consts.ImplantBuildsStr}),
			Run: func(cmd *cobra.Command, args []string) {
				generate.ImplantsCmd(cmd, con, args)
			},
			GroupID: consts.PayloadsHelpGroup,
		}
		Flags("implants", true, implantBuildsCmd, func(f *pflag.FlagSet) {
			f.IntP("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		Flags("implants", false, implantBuildsCmd, func(f *pflag.FlagSet) {
			f.StringP("os", "o", "", "filter builds by operating system")
			f.StringP("arch", "a", "", "filter builds by cpu architecture")
			f.StringP("format", "f", "", "filter builds by artifact format")
			f.BoolP("only-sessions", "s", false, "filter interactive sessions")
			f.BoolP("only-beacons", "b", false, "filter beacons")
			f.BoolP("no-debug", "d", false, "filter builds by debug flag")
		})
		FlagComps(profilesNewBeaconCmd, func(comp *carapace.ActionMap) {
			(*comp)["os"] = generate.OSCompleter(con)
			(*comp)["arch"] = generate.ArchCompleter(con)
			(*comp)["format"] = generate.FormatCompleter()
		})
		server.AddCommand(implantBuildsCmd)

		implantsRmCmd := &cobra.Command{
			Use:   consts.RmStr,
			Short: "Remove implant build",
			Long:  help.GetHelpFor([]string{consts.ImplantBuildsStr, consts.RmStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				generate.ImplantsRmCmd(cmd, con, args)
			},
		}
		carapace.Gen(implantsRmCmd).PositionalCompletion(generate.ImplantBuildNameCompleter(con))
		implantBuildsCmd.AddCommand(implantsRmCmd)

		implantStageCmd := &cobra.Command{
			Use:   consts.StageStr,
			Short: "Serve a previously generated implant",
			Long:  help.GetHelpFor([]string{consts.ImplantBuildsStr, consts.StageStr}),
			Run: func(cmd *cobra.Command, args []string) {
				generate.ImplantsStageCmd(cmd, con, args)
			},
		}
		implantBuildsCmd.AddCommand(implantStageCmd)

		canariesCmd := &cobra.Command{
			Use:   consts.CanariesStr,
			Short: "List previously generated canaries",
			Long:  help.GetHelpFor([]string{consts.CanariesStr}),
			Run: func(cmd *cobra.Command, args []string) {
				generate.CanariesCmd(cmd, con, args)
			},
			GroupID: consts.PayloadsHelpGroup,
		}
		Flags("canaries", false, canariesCmd, func(f *pflag.FlagSet) {
			f.BoolP("burned", "b", false, "show only triggered/burned canaries")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		// [ Websites ] ---------------------------------------------

		websitesCmd := &cobra.Command{
			Use:   consts.WebsitesStr,
			Short: "Host static content (used with HTTP C2)",
			Long:  help.GetHelpFor([]string{consts.WebsitesStr}),
			Run: func(cmd *cobra.Command, args []string) {
				websites.WebsitesCmd(cmd, con, args)
			},
			GroupID: consts.NetworkHelpGroup,
		}
		server.AddCommand(websitesCmd)
		Flags("websites", true, websitesCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		carapace.Gen(websitesCmd).PositionalCompletion(websites.WebsiteNameCompleter(con))

		websitesRmCmd := &cobra.Command{
			Use:   consts.RmStr,
			Short: "Remove an entire website and all of its contents",
			Long:  help.GetHelpFor([]string{consts.WebsitesStr, consts.RmStr}),
			Run: func(cmd *cobra.Command, args []string) {
				websites.WebsiteRmCmd(cmd, con, args)
			},
		}
		carapace.Gen(websitesRmCmd).PositionalCompletion(websites.WebsiteNameCompleter(con))
		websitesCmd.AddCommand(websitesRmCmd)

		websitesRmWebContentCmd := &cobra.Command{
			Use:   consts.RmWebContentStr,
			Short: "Remove specific content from a website",
			Long:  help.GetHelpFor([]string{consts.WebsitesStr, consts.RmWebContentStr}),
			Run: func(cmd *cobra.Command, args []string) {
				websites.WebsitesRmContent(cmd, con, args)
			},
		}
		Flags("websites", false, websitesRmWebContentCmd, func(f *pflag.FlagSet) {
			f.BoolP("recursive", "r", false, "recursively add/rm content")
			f.StringP("website", "w", "", "website name")
			f.StringP("web-path", "p", "", "http path to host file at")
		})
		websitesCmd.AddCommand(websitesRmWebContentCmd)
		FlagComps(websitesRmWebContentCmd, func(comp *carapace.ActionMap) {
			(*comp)["website"] = websites.WebsiteNameCompleter(con)
		})

		websitesContentCmd := &cobra.Command{
			Use:   consts.AddWebContentStr,
			Short: "Add content to a website",
			Long:  help.GetHelpFor([]string{consts.WebsitesStr, consts.RmWebContentStr}),
			Run: func(cmd *cobra.Command, args []string) {
				websites.WebsitesAddContentCmd(cmd, con, args)
			},
		}
		Flags("websites", false, websitesContentCmd, func(f *pflag.FlagSet) {
			f.StringP("website", "w", "", "website name")
			f.StringP("content-type", "m", "", "mime content-type (if blank use file ext.)")
			f.StringP("web-path", "p", "/", "http path to host file at")
			f.StringP("content", "c", "", "local file path/dir (must use --recursive for dir)")
			f.BoolP("recursive", "r", false, "recursively add/rm content")
		})
		FlagComps(websitesContentCmd, func(comp *carapace.ActionMap) {
			(*comp)["content"] = carapace.ActionFiles().Tag("content directory/files")
			(*comp)["website"] = websites.WebsiteNameCompleter(con)
		})
		websitesCmd.AddCommand(websitesContentCmd)

		websitesContentTypeCmd := &cobra.Command{
			Use:   consts.WebContentTypeStr,
			Short: "Update a path's content-type",
			Long:  help.GetHelpFor([]string{consts.WebsitesStr, consts.WebContentTypeStr}),
			Run: func(cmd *cobra.Command, args []string) {
				websites.WebsitesUpdateContentCmd(cmd, con, args)
			},
		}
		Flags("websites", false, websitesContentTypeCmd, func(f *pflag.FlagSet) {
			f.StringP("website", "w", "", "website name")
			f.StringP("content-type", "m", "", "mime content-type (if blank use file ext.)")
			f.StringP("web-path", "p", "/", "http path to host file at")
		})
		websitesCmd.AddCommand(websitesContentTypeCmd)
		FlagComps(websitesContentTypeCmd, func(comp *carapace.ActionMap) {
			(*comp)["website"] = websites.WebsiteNameCompleter(con)
		})

		// [ Beacons ] ---------------------------------------------

		beaconsCmd := &cobra.Command{
			Use:     consts.BeaconsStr,
			Short:   "Manage beacons",
			Long:    help.GetHelpFor([]string{consts.BeaconsStr}),
			GroupID: consts.SliverHelpGroup,
			Run: func(cmd *cobra.Command, args []string) {
				beacons.BeaconsCmd(cmd, con, args)
			},
		}
		Flags("beacons", true, beaconsCmd, func(f *pflag.FlagSet) {
			f.IntP("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		Flags("beacons", false, beaconsCmd, func(f *pflag.FlagSet) {
			f.StringP("kill", "k", "", "kill the designated beacon")
			f.BoolP("kill-all", "K", false, "kill all beacons")
			f.BoolP("force", "F", false, "force killing the beacon")

			f.StringP("filter", "f", "", "filter beacons by substring")
			f.StringP("filter-re", "e", "", "filter beacons by regular expression")
		})
		FlagComps(beaconsCmd, func(comp *carapace.ActionMap) {
			(*comp)["kill"] = use.BeaconIDCompleter(con)
		})
		beaconsRmCmd := &cobra.Command{
			Use:   consts.RmStr,
			Short: "Remove a beacon",
			Long:  help.GetHelpFor([]string{consts.BeaconsStr, consts.RmStr}),
			Run: func(cmd *cobra.Command, args []string) {
				beacons.BeaconsRmCmd(cmd, con, args)
			},
		}
		carapace.Gen(beaconsRmCmd).PositionalCompletion(use.BeaconIDCompleter(con))
		beaconsCmd.AddCommand(beaconsRmCmd)

		beaconsWatchCmd := &cobra.Command{
			Use:   consts.WatchStr,
			Short: "Watch your beacons",
			Long:  help.GetHelpFor([]string{consts.BeaconsStr, consts.WatchStr}),
			Run: func(cmd *cobra.Command, args []string) {
				beacons.BeaconsWatchCmd(cmd, con, args)
			},
		}
		beaconsCmd.AddCommand(beaconsWatchCmd)

		beaconsPruneCmd := &cobra.Command{
			Use:   consts.PruneStr,
			Short: "Prune stale beacons automatically",
			Long:  help.GetHelpFor([]string{consts.BeaconsStr, consts.PruneStr}),
			Run: func(cmd *cobra.Command, args []string) {
				beacons.BeaconsPruneCmd(cmd, con, args)
			},
		}
		Flags("beacons", false, beaconsPruneCmd, func(f *pflag.FlagSet) {
			f.StringP("duration", "d", "1h", "duration to prune beacons that have missed their last checkin")
		})
		beaconsCmd.AddCommand(beaconsPruneCmd)
		server.AddCommand(beaconsCmd)

		// [ Licenses ] ---------------------------------------------

		server.AddCommand(&cobra.Command{
			Use:   consts.LicensesStr,
			Short: "Open source licenses",
			Long:  help.GetHelpFor([]string{consts.LicensesStr}),
			Run: func(cmd *cobra.Command, args []string) {
				con.Println(licenses.All)
			},
			GroupID: consts.GenericHelpGroup,
		})

		// [ WireGuard ] --------------------------------------------------------------

		wgConfigCmd := &cobra.Command{
			Use:   consts.WgConfigStr,
			Short: "Generate a new WireGuard client config",
			Long:  help.GetHelpFor([]string{consts.WgConfigStr}),
			Run: func(cmd *cobra.Command, args []string) {
				wireguard.WGConfigCmd(cmd, con, args)
			},
			GroupID: consts.NetworkHelpGroup,
		}
		server.AddCommand(wgConfigCmd)

		Flags("wg-config", true, wgConfigCmd, func(f *pflag.FlagSet) {
			f.IntP("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		Flags("wg-config", false, wgConfigCmd, func(f *pflag.FlagSet) {
			f.StringP("save", "s", "", "save configuration to file (.conf)")
		})
		FlagComps(wgConfigCmd, func(comp *carapace.ActionMap) {
			(*comp)["save"] = carapace.ActionFiles().Tag("directory/file to save config")
		})

		// [ Monitor ] --------------------------------------------------------------

		monitorCmd := &cobra.Command{
			Use:     consts.MonitorStr,
			Short:   "Monitor threat intel platforms for Sliver implants",
			GroupID: consts.SliverHelpGroup,
		}

		configCmd := &cobra.Command{
			Use:   consts.MonitorConfigStr,
			Short: "Configure monitor API keys",
			Run: func(cmd *cobra.Command, args []string) {
				monitor.MonitorConfigCmd(cmd, con, args)
			},
		}
		monitorCmd.AddCommand(&cobra.Command{
			Use:   "start",
			Short: "Start the monitoring loops",
			Run: func(cmd *cobra.Command, args []string) {
				monitor.MonitorStartCmd(cmd, con, args)
			},
		})
		monitorCmd.AddCommand(&cobra.Command{
			Use:   "stop",
			Short: "Stop the monitoring loops",
			Run: func(cmd *cobra.Command, args []string) {
				monitor.MonitorStopCmd(cmd, con, args)
			},
		})
		configCmd.AddCommand(&cobra.Command{
			Use:   "add",
			Short: "Add API key configuration",
			Run: func(cmd *cobra.Command, args []string) {
				monitor.MonitorAddConfigCmd(cmd, con, args)
			},
		})

		configCmd.AddCommand(&cobra.Command{
			Use:   "del",
			Short: "Remove API key configuration",
			Run: func(cmd *cobra.Command, args []string) {
				monitor.MonitorDelConfigCmd(cmd, con, args)
			},
		})

		monitorCmd.AddCommand(configCmd)
		server.AddCommand(monitorCmd)

		// [ Loot ] --------------------------------------------------------------

		lootCmd := &cobra.Command{
			Use:   consts.LootStr,
			Short: "Manage the server's loot store",
			Long:  help.GetHelpFor([]string{consts.LootStr}),
			Run: func(cmd *cobra.Command, args []string) {
				loot.LootCmd(cmd, con, args)
			},
			GroupID: consts.SliverHelpGroup,
		}
		Flags("loot", true, lootCmd, func(f *pflag.FlagSet) {
			f.IntP("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		Flags("loot", false, lootCmd, func(f *pflag.FlagSet) {
			f.StringP("filter", "f", "", "filter based on loot type")
		})

		lootAddCmd := &cobra.Command{
			Use:   consts.LootLocalStr,
			Short: "Add a local file to the server's loot store",
			Long:  help.GetHelpFor([]string{consts.LootStr, consts.LootLocalStr}),
			Run: func(cmd *cobra.Command, args []string) {
				loot.LootAddLocalCmd(cmd, con, args)
			},
			Args: cobra.ExactArgs(1),
		}
		lootCmd.AddCommand(lootAddCmd)
		Flags("loot", false, lootAddCmd, func(f *pflag.FlagSet) {
			f.StringP("name", "n", "", "name of this piece of loot")
			f.StringP("type", "T", "", "force a specific loot type (file/cred)")
			f.StringP("file-type", "F", "", "force a specific file type (binary/text)")
		})
		FlagComps(lootAddCmd, func(comp *carapace.ActionMap) {
			(*comp)["type"] = carapace.ActionValues("file", "cred").Tag("loot type")
			(*comp)["file-type"] = carapace.ActionValues("binary", "text").Tag("loot file type")
		})
		carapace.Gen(lootAddCmd).PositionalCompletion(
			carapace.ActionFiles().Tag("local loot file").Usage("The local file path to the loot"))

		lootRemoteCmd := &cobra.Command{
			Use:   consts.LootRemoteStr,
			Short: "Add a remote file from the current session to the server's loot store",
			Long:  help.GetHelpFor([]string{consts.LootStr, consts.LootRemoteStr}),
			Run: func(cmd *cobra.Command, args []string) {
				loot.LootAddRemoteCmd(cmd, con, args)
			},
			Args: cobra.ExactArgs(1),
		}
		lootCmd.AddCommand(lootRemoteCmd)
		Flags("loot", false, lootRemoteCmd, func(f *pflag.FlagSet) {
			f.StringP("name", "n", "", "name of this piece of loot")
			f.StringP("type", "T", "", "force a specific loot type (file/cred)")
			f.StringP("file-type", "F", "", "force a specific file type (binary/text)")
		})
		FlagComps(lootRemoteCmd, func(comp *carapace.ActionMap) {
			(*comp)["type"] = carapace.ActionValues("file", "cred").Tag("loot type")
			(*comp)["file-type"] = carapace.ActionValues("binary", "text").Tag("loot file type")
		})
		carapace.Gen(lootRemoteCmd).PositionalCompletion(carapace.ActionValues().Usage("The file path on the remote host to the loot"))

		lootRenameCmd := &cobra.Command{
			Use:   consts.RenameStr,
			Short: "Re-name a piece of existing loot",
			Long:  help.GetHelpFor([]string{consts.LootStr, consts.RenameStr}),
			Run: func(cmd *cobra.Command, args []string) {
				loot.LootRenameCmd(cmd, con, args)
			},
		}
		lootCmd.AddCommand(lootRenameCmd)

		lootFetchCmd := &cobra.Command{
			Use:   consts.FetchStr,
			Short: "Fetch a piece of loot from the server's loot store",
			Long:  help.GetHelpFor([]string{consts.LootStr, consts.FetchStr}),
			Run: func(cmd *cobra.Command, args []string) {
				loot.LootFetchCmd(cmd, con, args)
			},
		}
		lootCmd.AddCommand(lootFetchCmd)
		Flags("loot", false, lootFetchCmd, func(f *pflag.FlagSet) {
			f.StringP("save", "s", "", "save loot to a local file")
			f.StringP("filter", "f", "", "filter based on loot type")
		})
		FlagComps(lootFetchCmd, func(comp *carapace.ActionMap) {
			(*comp)["save"] = carapace.ActionFiles().Tag("directory/file to save loot")
		})

		lootRmCmd := &cobra.Command{
			Use:   consts.RmStr,
			Short: "Remove a piece of loot from the server's loot store",
			Long:  help.GetHelpFor([]string{consts.LootStr, consts.RmStr}),
			Run: func(cmd *cobra.Command, args []string) {
				loot.LootRmCmd(cmd, con, args)
			},
		}
		lootCmd.AddCommand(lootRmCmd)
		Flags("loot", false, lootRmCmd, func(f *pflag.FlagSet) {
			f.StringP("filter", "f", "", "filter based on loot type")
		})

		server.AddCommand(lootCmd)

		// [ Credentials ] ------------------------------------------------------------
		credsCmd := &cobra.Command{
			Use:     consts.CredsStr,
			Short:   "Manage the database of credentials",
			Long:    help.GetHelpFor([]string{consts.CredsStr}),
			GroupID: consts.GenericHelpGroup,
			Run: func(cmd *cobra.Command, args []string) {
				creds.CredsCmd(cmd, con, args)
			},
		}
		Flags("creds", true, credsCmd, func(f *pflag.FlagSet) {
			f.IntP("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		server.AddCommand(credsCmd)

		credsAddCmd := &cobra.Command{
			Use:   consts.AddStr,
			Short: "Add a credential to the database",
			Long:  help.GetHelpFor([]string{consts.CredsStr, consts.AddStr}),
			Run: func(cmd *cobra.Command, args []string) {
				creds.CredsAddCmd(cmd, con, args)
			},
		}
		Flags("", false, credsAddCmd, func(f *pflag.FlagSet) {
			f.StringP("collection", "c", "", "name of collection")
			f.StringP("username", "u", "", "username for the credential")
			f.StringP("plaintext", "p", "", "plaintext for the credential")
			f.StringP("hash", "P", "", "hash of the credential")
			f.StringP("hash-type", "H", "", "hash type of the credential")
		})
		FlagComps(credsAddCmd, func(comp *carapace.ActionMap) {
			(*comp)["hash-type"] = creds.CredsHashTypeCompleter(con)
		})
		credsCmd.AddCommand(credsAddCmd)

		credsAddFileCmd := &cobra.Command{
			Use:   consts.FileStr,
			Short: "Add a credential to the database",
			Long:  help.GetHelpFor([]string{consts.CredsStr, consts.AddStr, consts.FileStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				creds.CredsAddHashFileCmd(cmd, con, args)
			},
		}
		Flags("", false, credsAddFileCmd, func(f *pflag.FlagSet) {
			f.StringP("collection", "c", "", "name of collection")
			f.StringP("file-format", "F", creds.HashNewlineFormat, "file format of the credential file")
			f.StringP("hash-type", "H", "", "hash type of the credential")
		})
		FlagComps(credsAddFileCmd, func(comp *carapace.ActionMap) {
			(*comp)["collection"] = creds.CredsCollectionCompleter(con)
			(*comp)["file-format"] = creds.CredsHashFileFormatCompleter(con)
			(*comp)["hash-type"] = creds.CredsHashTypeCompleter(con)
		})
		carapace.Gen(credsAddFileCmd).PositionalCompletion(carapace.ActionFiles().Tag("credential file"))
		credsAddCmd.AddCommand(credsAddFileCmd)

		credsRmCmd := &cobra.Command{
			Use:   consts.RmStr,
			Short: "Remove a credential to the database",
			Long:  help.GetHelpFor([]string{consts.CredsStr, consts.RmStr}),
			Run: func(cmd *cobra.Command, args []string) {
				creds.CredsRmCmd(cmd, con, args)
			},
		}
		carapace.Gen(credsRmCmd).PositionalCompletion(creds.CredsCredentialIDCompleter(con).Usage("id of credential to remove (leave empty to select)"))
		credsCmd.AddCommand(credsRmCmd)

		// [ Hosts ] ---------------------------------------------------------------------

		hostsCmd := &cobra.Command{
			Use:   consts.HostsStr,
			Short: "Manage the database of hosts",
			Long:  help.GetHelpFor([]string{consts.HostsStr}),
			Run: func(cmd *cobra.Command, args []string) {
				hosts.HostsCmd(cmd, con, args)
			},
			GroupID: consts.SliverHelpGroup,
		}
		server.AddCommand(hostsCmd)
		Flags("hosts", true, hostsCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		hostsRmCmd := &cobra.Command{
			Use:   consts.RmStr,
			Short: "Remove a host from the database",
			Long:  help.GetHelpFor([]string{consts.HostsStr, consts.RmStr}),
			Run: func(cmd *cobra.Command, args []string) {
				hosts.HostsRmCmd(cmd, con, args)
			},
		}
		hostsCmd.AddCommand(hostsRmCmd)

		hostsIOCCmd := &cobra.Command{
			Use:   consts.IOCStr,
			Short: "Manage tracked IOCs on a given host",
			Long:  help.GetHelpFor([]string{consts.HostsStr, consts.IOCStr}),
			Run: func(cmd *cobra.Command, args []string) {
				hosts.HostsIOCCmd(cmd, con, args)
			},
		}
		hostsCmd.AddCommand(hostsIOCCmd)

		hostsIOCRmCmd := &cobra.Command{
			Use:   consts.RmStr,
			Short: "Delete IOCs from the database",
			Long:  help.GetHelpFor([]string{consts.HostsStr, consts.IOCStr, consts.RmStr}),
			Run: func(cmd *cobra.Command, args []string) {
				hosts.HostsIOCRmCmd(cmd, con, args)
			},
		}
		hostsIOCCmd.AddCommand(hostsIOCRmCmd)

		// [ Reactions ] -----------------------------------------------------------------

		reactionCmd := &cobra.Command{
			Use:   consts.ReactionStr,
			Short: "Manage automatic reactions to events",
			Long:  help.GetHelpFor([]string{consts.ReactionStr}),
			Run: func(cmd *cobra.Command, args []string) {
				reaction.ReactionCmd(cmd, con, args)
			},
			GroupID: consts.SliverHelpGroup,
		}
		server.AddCommand(reactionCmd)

		reactionSetCmd := &cobra.Command{
			Use:   consts.SetStr,
			Short: "Set a reaction to an event",
			Long:  help.GetHelpFor([]string{consts.ReactionStr, consts.SetStr}),
			Run: func(cmd *cobra.Command, args []string) {
				reaction.ReactionSetCmd(cmd, con, args)
			},
		}
		reactionCmd.AddCommand(reactionSetCmd)
		Flags("reactions", false, reactionSetCmd, func(f *pflag.FlagSet) {
			f.StringP("event", "e", "", "specify the event type to react to")
		})

		FlagComps(reactionSetCmd, func(comp *carapace.ActionMap) {
			(*comp)["event"] = carapace.ActionValues(
				consts.SessionOpenedEvent,
				consts.SessionClosedEvent,
				consts.SessionUpdateEvent,
				consts.BeaconRegisteredEvent,
				consts.CanaryEvent,
				consts.WatchtowerEvent,
			)
		})

		reactionUnsetCmd := &cobra.Command{
			Use:   consts.UnsetStr,
			Short: "Unset an existing reaction",
			Long:  help.GetHelpFor([]string{consts.ReactionStr, consts.UnsetStr}),
			Run: func(cmd *cobra.Command, args []string) {
				reaction.ReactionUnsetCmd(cmd, con, args)
			},
		}
		reactionCmd.AddCommand(reactionUnsetCmd)
		Flags("reactions", false, reactionUnsetCmd, func(f *pflag.FlagSet) {
			f.IntP("id", "i", 0, "the id of the reaction to remove")
		})
		FlagComps(reactionUnsetCmd, func(comp *carapace.ActionMap) {
			(*comp)["id"] = reaction.ReactionIDCompleter(con)
		})

		reactionSaveCmd := &cobra.Command{
			Use:   consts.SaveStr,
			Short: "Save current reactions to disk",
			Long:  help.GetHelpFor([]string{consts.ReactionStr, consts.SaveStr}),
			Run: func(cmd *cobra.Command, args []string) {
				reaction.ReactionSaveCmd(cmd, con, args)
			},
		}
		reactionCmd.AddCommand(reactionSaveCmd)

		reactionReloadCmd := &cobra.Command{
			Use:   consts.ReloadStr,
			Short: "Reload reactions from disk, replaces the running configuration",
			Long:  help.GetHelpFor([]string{consts.ReactionStr, consts.ReloadStr}),
			Run: func(cmd *cobra.Command, args []string) {
				reaction.ReactionReloadCmd(cmd, con, args)
			},
		}
		reactionCmd.AddCommand(reactionReloadCmd)

		// [ Prelude's Operator ] ------------------------------------------------------------
		operatorCmd := &cobra.Command{
			Use:     consts.PreludeOperatorStr,
			Short:   "Manage connection to Prelude's Operator",
			Long:    help.GetHelpFor([]string{consts.PreludeOperatorStr}),
			GroupID: consts.GenericHelpGroup,
			Run: func(cmd *cobra.Command, args []string) {
				operator.OperatorCmd(cmd, con, args)
			},
		}
		server.AddCommand(operatorCmd)

		operatorConnectCmd := &cobra.Command{
			Use:   consts.ConnectStr,
			Short: "Connect with Prelude's Operator",
			Long:  help.GetHelpFor([]string{consts.PreludeOperatorStr, consts.ConnectStr}),
			Run: func(cmd *cobra.Command, args []string) {
				operator.ConnectCmd(cmd, con, args)
			},
			Args: cobra.ExactArgs(1),
		}
		operatorCmd.AddCommand(operatorConnectCmd)
		Flags("operator", false, operatorConnectCmd, func(f *pflag.FlagSet) {
			f.BoolP("skip-existing", "s", false, "Do not add existing sessions as Operator Agents")
			f.StringP("aes-key", "a", "abcdefghijklmnopqrstuvwxyz012345", "AES key for communication encryption")
			f.StringP("range", "r", "sliver", "Agents range")
		})
		carapace.Gen(operatorConnectCmd).PositionalCompletion(
			carapace.ActionValues().Usage("connection string to the Operator Host (e.g. 127.0.0.1:1234)"))

		// [ Builders ] ---------------------------------------------

		buildersCmd := &cobra.Command{
			Use:   consts.BuildersStr,
			Short: "List external builders",
			Long:  help.GetHelpFor([]string{consts.BuildersStr}),
			Run: func(cmd *cobra.Command, args []string) {
				builders.BuildersCmd(cmd, con, args)
			},
			GroupID: consts.PayloadsHelpGroup,
		}
		server.AddCommand(buildersCmd)
		Flags("builders", false, buildersCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		// [ Crack ] ------------------------------------------------------------
		crackCmd := &cobra.Command{
			Use:     consts.CrackStr,
			Short:   "Crack: GPU password cracking",
			Long:    help.GetHelpFor([]string{consts.CrackStr}),
			GroupID: consts.GenericHelpGroup,
			Run: func(cmd *cobra.Command, args []string) {
				crack.CrackCmd(cmd, con, args)
			},
		}
		Flags("", true, crackCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		server.AddCommand(crackCmd)

		crackStationsCmd := &cobra.Command{
			Use:   consts.StationsStr,
			Short: "Manage crackstations",
			Long:  help.GetHelpFor([]string{consts.CrackStr, consts.StationsStr}),
			Run: func(cmd *cobra.Command, args []string) {
				crack.CrackStationsCmd(cmd, con, args)
			},
		}
		crackCmd.AddCommand(crackStationsCmd)

		wordlistsCmd := &cobra.Command{
			Use:   consts.WordlistsStr,
			Short: "Manage wordlists",
			Long:  help.GetHelpFor([]string{consts.CrackStr, consts.WordlistsStr}),
			Run: func(cmd *cobra.Command, args []string) {
				crack.CrackWordlistsCmd(cmd, con, args)
			},
		}
		crackCmd.AddCommand(wordlistsCmd)

		wordlistsAddCmd := &cobra.Command{
			Use:   consts.AddStr,
			Short: "Add a wordlist",
			Run: func(cmd *cobra.Command, args []string) {
				crack.CrackWordlistsAddCmd(cmd, con, args)
			},
		}
		Flags("", false, wordlistsAddCmd, func(f *pflag.FlagSet) {
			f.StringP("name", "n", "", "wordlist name (blank = filename)")
		})
		carapace.Gen(wordlistsAddCmd).PositionalCompletion(carapace.ActionFiles().Usage("path to local wordlist file"))
		wordlistsCmd.AddCommand(wordlistsAddCmd)

		wordlistsRmCmd := &cobra.Command{
			Use:   consts.RmStr,
			Short: "Remove a wordlist",
			Run: func(cmd *cobra.Command, args []string) {
				crack.CrackWordlistsRmCmd(cmd, con, args)
			},
		}
		wordlistsCmd.AddCommand(wordlistsRmCmd)
		carapace.Gen(wordlistsRmCmd).PositionalCompletion(crack.CrackWordlistCompleter(con).Usage("wordlist to remove"))

		rulesCmd := &cobra.Command{
			Use:   consts.RulesStr,
			Short: "Manage rule files",
			Long:  help.GetHelpFor([]string{consts.CrackStr, consts.RulesStr}),
			Run: func(cmd *cobra.Command, args []string) {
				crack.CrackRulesCmd(cmd, con, args)
			},
		}
		crackCmd.AddCommand(rulesCmd)

		rulesAddCmd := &cobra.Command{
			Use:   consts.AddStr,
			Short: "Add a rules file",
			Long:  help.GetHelpFor([]string{consts.CrackStr, consts.RulesStr, consts.AddStr}),
			Run: func(cmd *cobra.Command, args []string) {
				crack.CrackRulesAddCmd(cmd, con, args)
			},
		}
		Flags("", false, rulesAddCmd, func(f *pflag.FlagSet) {
			f.StringP("name", "n", "", "rules name (blank = filename)")
		})
		carapace.Gen(rulesAddCmd).PositionalCompletion(carapace.ActionFiles().Usage("path to local rules file"))
		rulesCmd.AddCommand(rulesAddCmd)

		rulesRmCmd := &cobra.Command{
			Use:   consts.RmStr,
			Short: "Remove rules",
			Long:  help.GetHelpFor([]string{consts.CrackStr, consts.RulesStr, consts.RmStr}),
			Run: func(cmd *cobra.Command, args []string) {
				crack.CrackRulesRmCmd(cmd, con, args)
			},
		}
		carapace.Gen(rulesRmCmd).PositionalCompletion(crack.CrackRulesCompleter(con).Usage("rules to remove"))
		rulesCmd.AddCommand(rulesRmCmd)

		hcstat2Cmd := &cobra.Command{
			Use:   consts.Hcstat2Str,
			Short: "Manage markov hcstat2 files",
			Long:  help.GetHelpFor([]string{consts.CrackStr, consts.Hcstat2Str}),
			Run: func(cmd *cobra.Command, args []string) {
				crack.CrackHcstat2Cmd(cmd, con, args)
			},
		}
		crackCmd.AddCommand(hcstat2Cmd)

		hcstat2AddCmd := &cobra.Command{
			Use:   consts.AddStr,
			Short: "Add a hcstat2 file",
			Long:  help.GetHelpFor([]string{consts.CrackStr, consts.Hcstat2Str, consts.AddStr}),
			Run: func(cmd *cobra.Command, args []string) {
				crack.CrackHcstat2AddCmd(cmd, con, args)
			},
		}
		Flags("", false, hcstat2AddCmd, func(f *pflag.FlagSet) {
			f.StringP("name", "n", "", "hcstat2 name (blank = filename)")
		})
		carapace.Gen(hcstat2AddCmd).PositionalCompletion(carapace.ActionFiles().Usage("path to local hcstat2 file"))
		hcstat2Cmd.AddCommand(hcstat2AddCmd)

		hcstat2RmCmd := &cobra.Command{
			Use:   consts.RmStr,
			Short: "Remove hcstat2 file",
			Long:  help.GetHelpFor([]string{consts.CrackStr, consts.Hcstat2Str, consts.RmStr}),
			Run: func(cmd *cobra.Command, args []string) {
				crack.CrackHcstat2RmCmd(cmd, con, args)
			},
		}
		carapace.Gen(hcstat2RmCmd).PositionalCompletion(crack.CrackHcstat2Completer(con).Usage("hcstat2 to remove"))
		hcstat2Cmd.AddCommand(hcstat2RmCmd)

		// [ Task many ]-----------------------------------------

		taskmanyCmd := &cobra.Command{
			Use:     consts.TaskmanyStr,
			Short:   "Task many beacons or sessions",
			Long:    help.GetHelpFor([]string{consts.TaskmanyStr}),
			GroupID: consts.GenericHelpGroup,
			Run: func(cmd *cobra.Command, args []string) {
				taskmany.TaskmanyCmd(cmd, con, args)
			},
		}
		server.AddCommand(taskmanyCmd)

		// Add the relevant beacon commands as a subcommand to taskmany
		taskmanyCmds := map[string]bool{
			consts.ExecuteStr:     true,
			consts.LsStr:          true,
			consts.CdStr:          true,
			consts.MkdirStr:       true,
			consts.RmStr:          true,
			consts.UploadStr:      true,
			consts.DownloadStr:    true,
			consts.InteractiveStr: true,
			consts.ChmodStr:       true,
			consts.ChownStr:       true,
			consts.ChtimesStr:     true,
			consts.PwdStr:         true,
			consts.CatStr:         true,
			consts.MvStr:          true,
			consts.PingStr:        true,
			consts.NetstatStr:     true,
			consts.PsStr:          true,
			consts.IfconfigStr:    true,
		}

		for _, c := range SliverCommands(con)().Commands() {
			_, ok := taskmanyCmds[c.Use]
			if ok {
				taskmanyCmd.AddCommand(taskmany.WrapCommand(c, con))
			}
		}

		// [ HTTP C2 Profiles ] --------------------------------------------------------------

		ImportC2ProfileCmd := &cobra.Command{
			Use:   consts.ImportC2ProfileStr,
			Short: "Import HTTP C2 profile",
			Long:  help.GetHelpFor([]string{consts.ImportC2ProfileStr}),
			Run: func(cmd *cobra.Command, args []string) {
				c2profiles.ImportC2ProfileCmd(cmd, con, args)
			},
		}
		Flags(consts.ImportC2ProfileStr, true, ImportC2ProfileCmd, func(f *pflag.FlagSet) {
			f.StringP("name", "n", constants.DefaultC2Profile, "HTTP C2 Profile name")
			f.StringP("file", "f", "", "Path to C2 configuration file to import")
			f.BoolP("overwrite", "o", false, "Overwrite profile if it exists")
		})

		C2ProfileCmd := &cobra.Command{
			Use:   consts.C2ProfileStr,
			Short: "Display C2 profile details",
			Long:  help.GetHelpFor([]string{consts.C2ProfileStr}),
			Run: func(cmd *cobra.Command, args []string) {
				c2profiles.C2ProfileCmd(cmd, con, args)
			},
		}
		Flags(consts.C2ProfileStr, true, C2ProfileCmd, func(f *pflag.FlagSet) {
			f.StringP("name", "n", constants.DefaultC2Profile, "HTTP C2 Profile to display")
		})
		FlagComps(C2ProfileCmd, func(comp *carapace.ActionMap) {
			(*comp)["name"] = generate.HTTPC2Completer(con)
		})
		C2ProfileCmd.AddCommand(ImportC2ProfileCmd)
		server.AddCommand(C2ProfileCmd)

		// [ Post-command declaration setup]-----------------------------------------

		// Everything below this line should preferably not be any command binding
		// (unless you know what you're doing). If there are any final modifications
		// to make to the sliver menu command tree, it time to do them here.

		server.InitDefaultHelpCmd()
		server.SetHelpCommandGroupID(consts.GenericHelpGroup)

		// Bind a readline subcommand to the `settings` one, for allowing users to
		// manipulate the shell instance keymaps, bindings, macros and global options.
		settingsCmd.AddCommand(readline.Commands(con.App.Shell()))

		return server
	}

	return serverCommands
}
