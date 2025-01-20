package extensions

import (
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/command/use"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Commands returns the 'extensions' command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	extensionCmd := &cobra.Command{
		Use:     consts.ExtensionsStr,
		Short:   "Manage extensions",
		Long:    help.GetHelpFor([]string{consts.ExtensionsStr}),
		GroupID: consts.GenericHelpGroup,
		Run: func(cmd *cobra.Command, _ []string) {
			ExtensionsCmd(cmd, con)
		},
	}

	extensionLoadCmd := &cobra.Command{
		Use:   consts.LoadStr,
		Short: "Temporarily load an extension from a local directory",
		Long:  help.GetHelpFor([]string{consts.ExtensionsStr, consts.LoadStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ExtensionLoadCmd(cmd, con, args)
		},
	}
	extensionCmd.AddCommand(extensionLoadCmd)
	carapace.Gen(extensionLoadCmd).PositionalCompletion(carapace.ActionDirectories().Usage("path to the extension directory"))

	extensionInstallCmd := &cobra.Command{
		Use:   consts.InstallStr,
		Short: "Install an extension from a local directory or .tar.gz file",
		Long:  help.GetHelpFor([]string{consts.ExtensionsStr, consts.InstallStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExtensionsInstallCmd(cmd, con, args)
		},
	}
	extensionCmd.AddCommand(extensionInstallCmd)
	carapace.Gen(extensionInstallCmd).PositionalCompletion(carapace.ActionFiles().Usage("path to the extension .tar.gz or directory"))

	extensionRmCmd := &cobra.Command{
		Use:   consts.RmStr,
		Short: "Remove an installed extension",
		Args:  cobra.ExactArgs(1),
		Long:  help.GetHelpFor([]string{consts.ExtensionsStr, consts.RmStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ExtensionsRemoveCmd(cmd, con, args)
		},
	}
	extensionCmd.AddCommand(extensionRmCmd)
	carapace.Gen(extensionRmCmd).PositionalCompletion(ExtensionsCommandNameCompleter(con).Usage("the command name of the extension to remove"))

	return []*cobra.Command{extensionCmd}
}

func SliverCommands(con *console.SliverClient) []*cobra.Command {
	extensionCmd := &cobra.Command{
		Use:         consts.ExtensionsStr,
		Short:       "Manage extensions",
		Long:        help.GetHelpFor([]string{consts.ExtensionsStr}),
		GroupID:     consts.InfoHelpGroup,
		Annotations: flags.RestrictTargets(consts.SessionCmdsFilter), // restrict to session targets since we cannot `list` "loaded" extensions from beacon mode
	}

	listCmd := &cobra.Command{
		Use:   consts.ListStr,
		Short: "List extensions loaded in the current session",
		Long:  help.GetHelpFor([]string{consts.ExtensionsStr, consts.ListStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ExtensionsListCmd(cmd, con, args)
		},
	}
	flags.Bind("use", false, listCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	extensionCmd.AddCommand(listCmd)

	carapace.Gen(listCmd).PositionalCompletion(use.BeaconAndSessionIDCompleter(con))

	return []*cobra.Command{extensionCmd}
}
