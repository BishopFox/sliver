package cursed

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
	cursedCmd := &cobra.Command{
		Use:         consts.Cursed,
		Short:       "Chrome/electron post-exploitation tool kit (∩｀-´)⊃━☆ﾟ.*･｡ﾟ",
		Long:        help.GetHelpFor([]string{consts.Cursed}),
		GroupID:     consts.ExecutionHelpGroup,
		Annotations: flags.RestrictTargets(consts.SessionCmdsFilter),
		Run: func(cmd *cobra.Command, args []string) {
			CursedCmd(cmd, con, args)
		},
	}
	flags.Bind("", true, cursedCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	cursedRmCmd := &cobra.Command{
		Use:   consts.RmStr,
		Short: "Remove a Curse from a process",
		Long:  help.GetHelpFor([]string{consts.Cursed, consts.CursedConsole}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			CursedRmCmd(cmd, con, args)
		},
	}
	cursedCmd.AddCommand(cursedRmCmd)
	flags.Bind("", false, cursedRmCmd, func(f *pflag.FlagSet) {
		f.BoolP("kill", "k", false, "kill the process after removing the curse")
	})
	carapace.Gen(cursedRmCmd).PositionalCompletion(carapace.ActionValues().Usage("bind port of the Cursed process to stop"))

	cursedConsoleCmd := &cobra.Command{
		Use:   consts.CursedConsole,
		Short: "Start a JavaScript console connected to a debug target",
		Long:  help.GetHelpFor([]string{consts.Cursed, consts.CursedConsole}),
		Run: func(cmd *cobra.Command, args []string) {
			CursedConsoleCmd(cmd, con, args)
		},
	}
	cursedCmd.AddCommand(cursedConsoleCmd)
	flags.Bind("", false, cursedConsoleCmd, func(f *pflag.FlagSet) {
		f.IntP("remote-debugging-port", "r", 0, "remote debugging tcp port (0 = random)`")
	})

	cursedChromeCmd := &cobra.Command{
		Use:   consts.CursedChrome,
		Short: "Automatically inject a Cursed Chrome payload into a remote Chrome extension",
		Long:  help.GetHelpFor([]string{consts.Cursed, consts.CursedChrome}),
		Run: func(cmd *cobra.Command, args []string) {
			CursedChromeCmd(cmd, con, args)
		},
	}
	cursedCmd.AddCommand(cursedChromeCmd)
	flags.Bind("", false, cursedChromeCmd, func(f *pflag.FlagSet) {
		f.IntP("remote-debugging-port", "r", 0, "remote debugging tcp port (0 = random)")
		f.BoolP("restore", "R", true, "restore the user's session after process termination")
		f.StringP("exe", "e", "", "chrome/chromium browser executable path (blank string = auto)")
		f.StringP("user-data", "u", "", "user data directory (blank string = auto)")
		f.StringP("payload", "p", "", "cursed chrome payload file path (.js)")
		f.BoolP("keep-alive", "k", false, "keeps browser alive after last browser window closes")
		f.BoolP("headless", "H", false, "start browser process in headless mode")
	})
	flags.BindFlagCompletions(cursedChromeCmd, func(comp *carapace.ActionMap) {
		(*comp)["payload"] = carapace.ActionFiles("js").Tag("javascript files")
	})
	cursedChromeCmd.Flags().ParseErrorsWhitelist.UnknownFlags = true
	carapace.Gen(cursedChromeCmd).PositionalAnyCompletion(carapace.ActionValues().Usage("additional Chrome CLI arguments"))

	cursedEdgeCmd := &cobra.Command{
		Use:   consts.CursedEdge,
		Short: "Automatically inject a Cursed Chrome payload into a remote Edge extension",
		Long:  help.GetHelpFor([]string{consts.Cursed, consts.CursedEdge}),
		Run: func(cmd *cobra.Command, args []string) {
			CursedEdgeCmd(cmd, con, args)
		},
	}
	cursedCmd.AddCommand(cursedEdgeCmd)
	flags.Bind("", false, cursedEdgeCmd, func(f *pflag.FlagSet) {
		f.IntP("remote-debugging-port", "r", 0, "remote debugging tcp port (0 = random)")
		f.BoolP("restore", "R", true, "restore the user's session after process termination")
		f.StringP("exe", "e", "", "edge browser executable path (blank string = auto)")
		f.StringP("user-data", "u", "", "user data directory (blank string = auto)")
		f.StringP("payload", "p", "", "cursed chrome payload file path (.js)")
		f.BoolP("keep-alive", "k", false, "keeps browser alive after last browser window closes")
		f.BoolP("headless", "H", false, "start browser process in headless mode")
	})
	flags.BindFlagCompletions(cursedEdgeCmd, func(comp *carapace.ActionMap) {
		(*comp)["payload"] = carapace.ActionFiles("js").Tag("javascript files")
	})
	cursedEdgeCmd.Flags().ParseErrorsWhitelist.UnknownFlags = true
	carapace.Gen(cursedEdgeCmd).PositionalAnyCompletion(carapace.ActionValues().Usage("additional Edge CLI arguments"))

	cursedElectronCmd := &cobra.Command{
		Use:   consts.CursedElectron,
		Short: "Curse a remote Electron application",
		Long:  help.GetHelpFor([]string{consts.Cursed, consts.CursedElectron}),
		Run: func(cmd *cobra.Command, args []string) {
			CursedElectronCmd(cmd, con, args)
		},
	}
	cursedCmd.AddCommand(cursedElectronCmd)
	flags.Bind("", false, cursedElectronCmd, func(f *pflag.FlagSet) {
		f.StringP("exe", "e", "", "remote electron executable absolute path")
		f.IntP("remote-debugging-port", "r", 0, "remote debugging tcp port (0 = random)")
	})
	cursedElectronCmd.Flags().ParseErrorsWhitelist.UnknownFlags = true
	carapace.Gen(cursedElectronCmd).PositionalAnyCompletion(carapace.ActionValues().Usage("additional Electron CLI arguments"))

	CursedCookiesCmd := &cobra.Command{
		Use:   consts.CursedCookies,
		Short: "Dump all cookies from cursed process",
		Long:  help.GetHelpFor([]string{consts.Cursed, consts.CursedCookies}),
		Run: func(cmd *cobra.Command, args []string) {
			CursedCookiesCmd(cmd, con, args)
		},
	}
	cursedCmd.AddCommand(CursedCookiesCmd)
	flags.Bind("", false, CursedCookiesCmd, func(f *pflag.FlagSet) {
		f.StringP("save", "s", "", "save to file")
	})

	cursedScreenshotCmd := &cobra.Command{
		Use:   consts.ScreenshotStr,
		Short: "Take a screenshot of a cursed process debug target",
		Long:  help.GetHelpFor([]string{consts.Cursed, consts.ScreenshotStr}),
		Run: func(cmd *cobra.Command, args []string) {
			CursedScreenshotCmd(cmd, con, args)
		},
	}
	cursedCmd.AddCommand(cursedScreenshotCmd)
	flags.Bind("", false, cursedScreenshotCmd, func(f *pflag.FlagSet) {
		f.Int64P("quality", "q", 100, "screenshot quality (1 - 100)")
		f.StringP("save", "s", "", "save to file")
	})

	return []*cobra.Command{cursedCmd}
}
