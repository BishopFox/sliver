package generate

import (
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	// [ Generate ] --------------------------------------------------------------
	generateCmd := &cobra.Command{
		Use:   consts.GenerateStr,
		Short: "Generate an implant binary",
		Long:  help.GetHelpFor([]string{consts.GenerateStr}),
		Run: func(cmd *cobra.Command, args []string) {
			GenerateCmd(cmd, con, args)
		},
		GroupID: consts.PayloadsHelpGroup,
	}
	flags.Bind("generate", true, generateCmd, func(f *pflag.FlagSet) {
		f.IntP("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	// Session flags and completions.
	coreImplantFlags("session", generateCmd)
	compileImplantFlags("session", generateCmd)
	coreImplantFlagCompletions(generateCmd, con)

	generateBeaconCmd := &cobra.Command{
		Use:   consts.BeaconStr,
		Short: "Generate a beacon binary",
		Long:  help.GetHelpFor([]string{consts.GenerateStr, consts.BeaconStr}),
		Run: func(cmd *cobra.Command, args []string) {
			GenerateBeaconCmd(cmd, con, args)
		},
	}

	// Beacon flags and completions.
	coreImplantFlags("beacon", generateBeaconCmd)
	compileImplantFlags("beacon", generateBeaconCmd)
	coreBeaconFlags("beacon", generateBeaconCmd)
	coreImplantFlagCompletions(generateBeaconCmd, con)

	generateCmd.AddCommand(generateBeaconCmd)

	generateInfoCmd := &cobra.Command{
		Use:   consts.CompilerInfoStr,
		Short: "Get information about the server's compiler",
		Long:  help.GetHelpFor([]string{consts.CompilerInfoStr}),
		Run: func(cmd *cobra.Command, args []string) {
			GenerateInfoCmd(cmd, con, args)
		},
	}
	generateCmd.AddCommand(generateInfoCmd)

	// Traffic Encoder SubCommands
	trafficEncodersCmd := &cobra.Command{
		Use:   consts.TrafficEncodersStr,
		Short: "Manage implant traffic encoders",
		Long:  help.GetHelpFor([]string{consts.GenerateStr, consts.TrafficEncodersStr}),
		Run: func(cmd *cobra.Command, args []string) {
			TrafficEncodersCmd(cmd, con, args)
		},
	}
	generateCmd.AddCommand(trafficEncodersCmd)

	trafficEncodersAddCmd := &cobra.Command{
		Use:   consts.AddStr,
		Short: "Add a new traffic encoder to the server from the local file system",
		Long:  help.GetHelpFor([]string{consts.GenerateStr, consts.TrafficEncodersStr, consts.AddStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			TrafficEncodersAddCmd(cmd, con, args)
		},
	}
	flags.Bind("", false, trafficEncodersAddCmd, func(f *pflag.FlagSet) {
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
			TrafficEncodersRemoveCmd(cmd, con, args)
		},
	}
	carapace.Gen(trafficEncodersRmCmd).PositionalCompletion(TrafficEncodersCompleter(con).Usage("traffic encoder to remove"))
	trafficEncodersCmd.AddCommand(trafficEncodersRmCmd)

	// [ Regenerate ] --------------------------------------------------------------

	regenerateCmd := &cobra.Command{
		Use:   consts.RegenerateStr,
		Short: "Regenerate an implant",
		Long:  help.GetHelpFor([]string{consts.RegenerateStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			RegenerateCmd(cmd, con, args)
		},
		GroupID: consts.PayloadsHelpGroup,
	}
	flags.Bind("regenerate", false, regenerateCmd, func(f *pflag.FlagSet) {
		f.StringP("save", "s", "", "directory/file to the binary to")
	})
	flags.BindFlagCompletions(regenerateCmd, func(comp *carapace.ActionMap) {
		(*comp)["save"] = carapace.ActionFiles().Tag("directory/file to save implant")
	})
	carapace.Gen(regenerateCmd).PositionalCompletion(ImplantBuildNameCompleter(con))

	// [ Profiles ] --------------------------------------------------------------

	profilesCmd := &cobra.Command{
		Use:   consts.ProfilesStr,
		Short: "List existing profiles",
		Long:  help.GetHelpFor([]string{consts.ProfilesStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ProfilesCmd(cmd, con, args)
		},
		GroupID: consts.PayloadsHelpGroup,
	}
	flags.Bind("profiles", true, profilesCmd, func(f *pflag.FlagSet) {
		f.IntP("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	profilesGenerateCmd := &cobra.Command{
		Use:   consts.GenerateStr,
		Short: "Generate implant from a profile",
		Long:  help.GetHelpFor([]string{consts.ProfilesStr, consts.GenerateStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ProfilesGenerateCmd(cmd, con, args)
		},
	}
	flags.Bind("profiles", false, profilesGenerateCmd, func(f *pflag.FlagSet) {
		f.StringP("save", "s", "", "directory/file to the binary to")
		f.BoolP("disable-sgn", "G", false, "disable shikata ga nai shellcode encoder")
	})
	flags.BindFlagCompletions(profilesGenerateCmd, func(comp *carapace.ActionMap) {
		(*comp)["save"] = carapace.ActionFiles().Tag("directory/file to save implant")
	})
	carapace.Gen(profilesGenerateCmd).PositionalCompletion(ProfileNameCompleter(con))
	profilesCmd.AddCommand(profilesGenerateCmd)

	profilesNewCmd := &cobra.Command{
		Use:   consts.NewStr,
		Short: "Create a new implant profile (interactive session)",
		Long:  help.GetHelpFor([]string{consts.ProfilesStr, consts.NewStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ProfilesNewCmd(cmd, con, args)
		},
	}
	profilesCmd.AddCommand(profilesNewCmd)

	profilesStageCmd := &cobra.Command{
		Use:   consts.StageStr,
		Short: "Generate implant from a profile and encode or encrypt it",
		Args:  cobra.ExactArgs(1),
		Long:  help.GetHelpFor([]string{consts.ProfilesStr, consts.StageStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ProfilesStageCmd(cmd, con, args)
		},
	}
	flags.Bind("profiles", false, profilesStageCmd, func(f *pflag.FlagSet) {
		f.StringP("save", "s", "", "directory/file to the binary to")
		f.StringP("name", "n", "", "Implant name")
		f.StringP("aes-encrypt-key", "k", "", "AES Encryption Key")
		f.StringP("aes-encrypt-iv", "i", "", "AES Encryption IV")
		f.StringP("rc4-encrypt-key", "r", "", "RC4 encryption key")
		f.BoolP("prepend-size", "p", false, "Prepend stage size")
		f.StringP("compress", "c", "", "Compress stage (zlib, gzip, deflate9 or deflate)")
	})

	carapace.Gen(profilesStageCmd).PositionalCompletion(ProfileNameCompleter(con))
	profilesCmd.AddCommand(profilesStageCmd)

	// Session flags and completions.
	coreImplantFlags("session", profilesNewCmd)
	compileImplantFlags("session", profilesNewCmd)
	coreImplantFlagCompletions(profilesNewCmd, con)

	profilesNewBeaconCmd := &cobra.Command{
		Use:   consts.BeaconStr,
		Short: "Create a new implant profile (beacon)",
		Long:  help.GetHelpFor([]string{consts.ProfilesStr, consts.NewStr, consts.BeaconStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ProfilesNewBeaconCmd(cmd, con, args)
		},
	}
	profilesNewCmd.AddCommand(profilesNewBeaconCmd)

	// Beacon flags and completions.
	coreImplantFlags("beacon", profilesNewBeaconCmd)
	compileImplantFlags("beacon", profilesNewBeaconCmd)
	coreBeaconFlags("beacon", profilesNewBeaconCmd)
	coreImplantFlagCompletions(profilesNewBeaconCmd, con)

	profilesRmCmd := &cobra.Command{
		Use:   consts.RmStr,
		Short: "Remove a profile",
		Long:  help.GetHelpFor([]string{consts.ProfilesStr, consts.RmStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ProfilesRmCmd(cmd, con, args)
		},
	}
	carapace.Gen(profilesRmCmd).PositionalCompletion(ProfileNameCompleter(con))
	profilesCmd.AddCommand(profilesRmCmd)

	profilesInfoCmd := &cobra.Command{
		Use:   consts.InfoStr,
		Short: "Details about a profile",
		Long:  help.GetHelpFor([]string{consts.ProfilesStr, consts.RmStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			PrintProfileInfo(args[0], con)
		},
	}
	carapace.Gen(profilesInfoCmd).PositionalCompletion(ProfileNameCompleter(con))
	profilesCmd.AddCommand(profilesInfoCmd)

	// [ Implants ] --------------------------------------------------------------

	implantBuildsCmd := &cobra.Command{
		Use:   consts.ImplantBuildsStr,
		Short: "List implant builds",
		Long:  help.GetHelpFor([]string{consts.ImplantBuildsStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ImplantsCmd(cmd, con, args)
		},
		GroupID: consts.PayloadsHelpGroup,
	}
	flags.Bind("implants", true, implantBuildsCmd, func(f *pflag.FlagSet) {
		f.IntP("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	flags.Bind("implants", false, implantBuildsCmd, func(f *pflag.FlagSet) {
		f.StringP("os", "o", "", "filter builds by operating system")
		f.StringP("arch", "a", "", "filter builds by cpu architecture")
		f.StringP("format", "f", "", "filter builds by artifact format")
		f.BoolP("only-sessions", "s", false, "filter interactive sessions")
		f.BoolP("only-beacons", "b", false, "filter beacons")
		f.BoolP("no-debug", "d", false, "filter builds by debug flag")
	})
	flags.BindFlagCompletions(implantBuildsCmd, func(comp *carapace.ActionMap) {
		(*comp)["os"] = OSCompleter(con)
		(*comp)["arch"] = ArchCompleter(con)
		(*comp)["format"] = FormatCompleter()
	})

	implantsRmCmd := &cobra.Command{
		Use:   consts.RmStr,
		Short: "Remove implant build",
		Long:  help.GetHelpFor([]string{consts.ImplantBuildsStr, consts.RmStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ImplantsRmCmd(cmd, con, args)
		},
	}
	carapace.Gen(implantsRmCmd).PositionalCompletion(ImplantBuildNameCompleter(con))
	implantBuildsCmd.AddCommand(implantsRmCmd)

	implantsStageCmd := &cobra.Command{
		Use:   consts.StageStr,
		Short: "Serve a previously generated build",
		Long:  help.GetHelpFor([]string{consts.ImplantBuildsStr, consts.StageStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ImplantsStageCmd(cmd, con, args)
		},
	}
	implantBuildsCmd.AddCommand(implantsStageCmd)

	canariesCmd := &cobra.Command{
		Use:   consts.CanariesStr,
		Short: "List previously generated canaries",
		Long:  help.GetHelpFor([]string{consts.CanariesStr}),
		Run: func(cmd *cobra.Command, args []string) {
			CanariesCmd(cmd, con, args)
		},
		GroupID: consts.PayloadsHelpGroup,
	}
	flags.Bind("canaries", false, canariesCmd, func(f *pflag.FlagSet) {
		f.BoolP("burned", "b", false, "show only triggered/burned canaries")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	return []*cobra.Command{generateCmd, regenerateCmd, profilesCmd, implantBuildsCmd}
}

// coreImplantFlags binds all flags common to all sliver implant types.
// This is used by all sliver compilation and profiles generation commands.
func coreImplantFlags(name string, cmd *cobra.Command) {
	flags.Bind(name, false, cmd, func(f *pflag.FlagSet) {
		// Core compile
		f.StringP("os", "o", "windows", "operating system")
		f.StringP("arch", "a", "amd64", "cpu architecture")
		f.StringP("name", "N", "", "agent name") //
		f.BoolP("debug", "d", false, "enable debug features")
		f.StringP("debug-file", "O", "", "path to debug output")
		f.BoolP("evasion", "e", false, "enable evasion features (e.g. overwrite user space hooks)")
		f.BoolP("skip-symbols", "l", false, "skip symbol obfuscation")
		f.BoolP("disable-sgn", "G", false, "disable shikata ga nai shellcode encoder")

		f.StringP("canary", "c", "", "canary domain(s)")

		// C2 channels
		f.StringP("mtls", "m", "", "mtls connection strings")
		f.StringP("wg", "g", "", "wg connection strings")
		f.StringP("http", "b", "", "http(s) connection strings")
		f.StringP("dns", "n", "", "dns connection strings")
		f.StringP("named-pipe", "p", "", "named-pipe connection strings")
		f.StringP("tcp-pivot", "i", "", "tcp-pivot connection strings")

		f.Uint32P("key-exchange", "X", DefaultWGKeyExPort, "wg key-exchange port")
		f.Uint32P("tcp-comms", "T", DefaultWGNPort, "wg c2 comms port")

		f.BoolP("run-at-load", "R", false, "run the implant entrypoint from DllMain/Constructor (shared library only)")
		f.BoolP("netgo", "q", false, "force the use of netgo")
		f.StringP("traffic-encoders", "A", "", "comma separated list of traffic encoders to enable")

		f.StringP("strategy", "Z", "", "specify a connection strategy (r = random, rd = random domain, s = sequential)")
		f.Int64P("reconnect", "j", DefaultReconnect, "attempt to reconnect every n second(s)")
		f.Int64P("poll-timeout", "P", DefaultPollTimeout, "long poll request timeout")
		f.Uint32P("max-errors", "k", DefaultMaxErrors, "max number of connection errors")
		f.StringP("c2profile", "C", consts.DefaultC2Profile, "HTTP C2 profile to use")

		// Limits
		f.StringP("limit-datetime", "w", "", "limit execution to before datetime")
		f.BoolP("limit-domainjoined", "x", false, "limit execution to domain joined machines")
		f.StringP("limit-username", "y", "", "limit execution to specified username")
		f.StringP("limit-hostname", "z", "", "limit execution to specified hostname")
		f.StringP("limit-fileexists", "F", "", "limit execution to hosts with this file in the filesystem")
		f.StringP("limit-locale", "L", "", "limit execution to hosts that match this locale")

		f.StringP("format", "f", "exe", "Specifies the output formats, valid values are: 'exe', 'shared' (for dynamic libraries), 'service' (see: `psexec` for more info) and 'shellcode' (windows only)")
	})
}

// coreImplantFlagCompletions binds completions to flags registered in coreImplantFlags.
func coreImplantFlagCompletions(cmd *cobra.Command, con *console.SliverClient) {
	flags.BindFlagCompletions(cmd, func(comp *carapace.ActionMap) {
		(*comp)["debug-file"] = carapace.ActionFiles()
		(*comp)["os"] = OSCompleter(con)
		(*comp)["arch"] = ArchCompleter(con)
		(*comp)["strategy"] = carapace.ActionValuesDescribed([]string{"r", "random", "rd", "random domain", "s", "sequential"}...).Tag("C2 strategy")
		(*comp)["format"] = FormatCompleter()
		(*comp)["save"] = carapace.ActionFiles().Tag("directory/file to save implant")
		(*comp)["traffic-encoders"] = TrafficEncodersCompleter(con).UniqueList(",")
	})
}

// coreBeaconFlags binds all flags specific to beacon implants (profiles or compiled).
func coreBeaconFlags(name string, cmd *cobra.Command) {
	flags.Bind(name, false, cmd, func(f *pflag.FlagSet) {
		f.Int64P("days", "D", 0, "beacon interval days")
		f.Int64P("hours", "H", 0, "beacon interval hours")
		f.Int64P("minutes", "M", 0, "beacon interval minutes")
		f.Int64P("seconds", "S", 60, "beacon interval seconds")
		f.Int64P("jitter", "J", 30, "beacon interval jitter in seconds")
	})
}

// compileImplantFlags binds all flags used when actually compiling an implant (not when creating a profile).
func compileImplantFlags(name string, cmd *cobra.Command) {
	flags.Bind(name, false, cmd, func(f *pflag.FlagSet) {
		f.StringP("name", "N", "", "agent name")
		f.StringP("template", "I", "sliver", "implant code template")
		f.BoolP("external-builder", "E", false, "use an external builder")
		f.StringP("save", "s", "", "directory/file to the binary to")
	})
}
