package extensions

import (
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	extCmd := &cobra.Command{
		Use:   consts.ExtensionsStr,
		Short: "List current exts",
		Long:  help.GetHelpFor([]string{consts.ExtensionsStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ExtensionsCmd(cmd, con)
		},
		GroupID: consts.GenericHelpGroup,
	}

	extLoadCmd := &cobra.Command{
		Use:   consts.LoadStr + " [EXT]",
		Short: "Load a command EXT",
		Long:  help.GetHelpFor([]string{consts.ExtensionsStr, consts.LoadStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExtensionLoadCmd(cmd, con, args)
		},
	}
	carapace.Gen(extLoadCmd).PositionalCompletion(
		carapace.ActionDirectories().Tag("ext directory").Usage("path to the ext directory"))
	extCmd.AddCommand(extLoadCmd)

	extInstallCmd := &cobra.Command{
		Use:   consts.InstallStr + " [EXT]",
		Short: "Install a command ext",
		Long:  help.GetHelpFor([]string{consts.ExtensionsStr, consts.InstallStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExtensionsInstallCmd(cmd, con, args)
		},
	}
	carapace.Gen(extInstallCmd).PositionalCompletion(carapace.ActionFiles().Tag("ext file").Usage("path to the extension .tar.gz or directory"))
	extCmd.AddCommand(extInstallCmd)

	extendo := &cobra.Command{
		Use:   consts.RmStr + " [EXT]",
		Short: "Remove an ext",
		Long:  help.GetHelpFor([]string{consts.RmStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// alias.AliasesRemoveCmd(cmd, con, args)
			ExtensionsRemoveCmd(cmd, con, args)
		},
	}
	carapace.Gen(extendo).PositionalCompletion(ExtensionsCommandNameCompleter(con).Usage("the command name of the extension to remove"))
	extCmd.AddCommand(extendo)

	return []*cobra.Command{extCmd}
}
