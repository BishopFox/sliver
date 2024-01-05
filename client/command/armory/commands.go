package armory

import (
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/command/completers"
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// Commands returns the `armory` command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	armoryCmd := &cobra.Command{
		Use:   consts.ArmoryStr,
		Short: "Automatically download and install extensions/aliases",
		Long:  help.GetHelpFor([]string{consts.ArmoryStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ArmoryCmd(cmd, con, args)
		},
		GroupID: consts.GenericHelpGroup,
	}
	flags.Bind("connection", true, armoryCmd, func(f *pflag.FlagSet) {
		f.BoolP("insecure", "I", false, "skip tls certificate validation")
		f.StringP("proxy", "p", "", "specify a proxy url (e.g. http://localhost:8080)")
		f.BoolP("ignore-cache", "c", false, "ignore metadata cache, force refresh")
		f.StringP("timeout", "t", "15m", "download timeout")
	})
	flags.BindFlagCompletions(armoryCmd, func(comp *carapace.ActionMap) {
		(*comp)["proxy"] = completers.LocalProxyCompleter()
	})

	armoryInstallCmd := &cobra.Command{
		Use:   consts.InstallStr,
		Short: "Install an alias or extension",
		Long:  help.GetHelpFor([]string{consts.ArmoryStr, consts.InstallStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ArmoryInstallCmd(cmd, con, args)
		},
	}
	armoryCmd.AddCommand(armoryInstallCmd)
	flags.NewCompletions(armoryInstallCmd).PositionalCompletion(
		AliasExtensionOrBundleCompleter().Usage("name of the extension or alias to install"),
	)

	armoryUpdateCmd := &cobra.Command{
		Use:   consts.UpdateStr,
		Short: "Update installed an aliases and extensions",
		Long:  help.GetHelpFor([]string{consts.ArmoryStr, consts.UpdateStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ArmoryUpdateCmd(cmd, con, args)
		},
	}
	armoryCmd.AddCommand(armoryUpdateCmd)

	armorySearchCmd := &cobra.Command{
		Use:   consts.SearchStr,
		Short: "Search for aliases and extensions by name (regex)",
		Long:  help.GetHelpFor([]string{consts.ArmoryStr, consts.SearchStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ArmorySearchCmd(cmd, con, args)
		},
	}
	armoryCmd.AddCommand(armorySearchCmd)
	flags.NewCompletions(armorySearchCmd).PositionalCompletion(carapace.ActionValues().Usage("a name regular expression"))

	return []*cobra.Command{armoryCmd}
}
