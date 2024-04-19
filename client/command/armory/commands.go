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
	flags.Bind("", false, armoryInstallCmd, func(f *pflag.FlagSet) {
		f.BoolP("force", "f", false, "force installation of package, overwriting the package if it exists")
		f.StringP("armory", "a", "", "name of armory to install package from")
	})
	armoryCmd.AddCommand(armoryInstallCmd)
	flags.NewCompletions(armoryInstallCmd).PositionalCompletion(
		AliasExtensionOrBundleCompleter().Usage("name of the extension or alias to install"),
	)

	armoryUpdateCmd := &cobra.Command{
		Use:   consts.UpdateStr,
		Short: "Update installed aliases and extensions",
		Long:  help.GetHelpFor([]string{consts.ArmoryStr, consts.UpdateStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ArmoryUpdateCmd(cmd, con, args)
		},
	}
	flags.Bind("", false, armoryUpdateCmd, func(f *pflag.FlagSet) {
		f.StringP("armory", "a", "", "name of armory to get updates from")
	})
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

	armoryInfoCmd := &cobra.Command{
		Use:   consts.InfoStr,
		Short: "View configured armories or details about a specific armory or package",
		Args:  cobra.RangeArgs(0, 1),
		Long:  help.GetHelpFor([]string{consts.ArmoryStr, consts.InfoStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ArmoryInfoCommand(cmd, con, args)
		},
	}
	armoryCmd.AddCommand(armoryInfoCmd)
	carapace.Gen(armoryInfoCmd).PositionalCompletion(
		carapace.ActionValues().Usage("armory or package name"),
	)

	armoryAddCmd := &cobra.Command{
		Use:   consts.AddStr,
		Short: "Add a new armory",
		Long:  help.GetHelpFor([]string{consts.ArmoryStr, consts.AddStr}),
		Args:  cobra.ExactArgs(1), // the name of the armory
		Run: func(cmd *cobra.Command, args []string) {
			AddArmoryCmd(cmd, con, args)
		},
	}
	flags.Bind("", false, armoryAddCmd, func(f *pflag.FlagSet) {
		f.StringP("url", "u", "", "The URL of the armory index (required)")
		f.StringP("pubkey", "k", "", "The public key for the armory (required)")
		f.StringP("auth", "a", "", "Authorization details / credentials for the armory")
		f.StringP("authcmd", "x", "", "Local command to run for authorization to the armory")
		f.BoolP("no-save", "e", false, "Do not save this armory configuration to disk")
	})
	armoryAddCmd.MarkFlagRequired("url")
	armoryAddCmd.MarkFlagRequired("pubkey")
	carapace.Gen(armoryAddCmd).PositionalCompletion(
		carapace.ActionValues().Usage("name of the armory"),
	)
	armoryCmd.AddCommand(armoryAddCmd)

	armoryRemoveCmd := &cobra.Command{
		Use:   consts.RmStr,
		Short: "Remove an armory",
		Long:  help.GetHelpFor([]string{consts.ArmoryStr, consts.RmStr}),
		Args:  cobra.ExactArgs(1), // the name of the armory
		Run: func(cmd *cobra.Command, args []string) {
			RemoveArmoryCmd(cmd, con, args)
		},
	}
	carapace.Gen(armoryAddCmd).PositionalCompletion(
		carapace.ActionValues().Usage("name of the armory"),
	)
	armoryCmd.AddCommand(armoryRemoveCmd)

	armorySaveCmd := &cobra.Command{
		Use:   consts.SaveStr,
		Short: "Save the armory configuration to disk",
		Long:  help.GetHelpFor([]string{consts.ArmoryStr, consts.SaveStr}),
		Run: func(cmd *cobra.Command, args []string) {
			SaveArmories(con)
		},
	}
	armoryCmd.AddCommand(armorySaveCmd)

	armoryEnableCmd := &cobra.Command{
		Use:   consts.EnableStr,
		Short: "Enable an armory",
		Long:  help.GetHelpFor([]string{consts.ArmoryStr, consts.EnableStr}),
		Args:  cobra.ExactArgs(1), // The name of the armory
		Run: func(cmd *cobra.Command, args []string) {
			ChangeArmoryEnabledState(cmd, con, args, true)
		},
	}
	carapace.Gen(armoryEnableCmd).PositionalCompletion(
		carapace.ActionValues().Usage("name of the armory"),
	)
	armoryCmd.AddCommand(armoryEnableCmd)

	armoryDisableCmd := &cobra.Command{
		Use:   consts.DisableStr,
		Short: "Disable an armory",
		Long:  help.GetHelpFor([]string{consts.ArmoryStr, consts.DisableStr}),
		Args:  cobra.ExactArgs(1), // The name of the armory
		Run: func(cmd *cobra.Command, args []string) {
			ChangeArmoryEnabledState(cmd, con, args, false)
		},
	}
	carapace.Gen(armoryDisableCmd).PositionalCompletion(
		carapace.ActionValues().Usage("name of the armory"),
	)
	armoryCmd.AddCommand(armoryDisableCmd)

	armoryModifyCmd := &cobra.Command{
		Use:   consts.ModifyStr,
		Short: "Modify an armory configuration",
		Long:  help.GetHelpFor([]string{consts.ArmoryStr, consts.ModifyStr}),
		Args:  cobra.ExactArgs(1), // The name of the armory
		Run: func(cmd *cobra.Command, args []string) {
			ModifyArmoryCmd(cmd, con, args)
		},
	}
	flags.Bind("", false, armoryModifyCmd, func(f *pflag.FlagSet) {
		f.StringP("name", "n", "", "new name for the armory")
		f.StringP("pubkey", "k", "", "new public key for the armory")
		f.StringP("url", "u", "", "new URL for the armory")
		f.StringP("auth", "a", "", "new authorization details / credentials for the armory")
		f.StringP("authcmd", "x", "", "new local command to run for authorization to the armory")
		f.BoolP("no-save", "e", false, "do not save armory configuration to armory configuration file")
	})
	carapace.Gen(armoryModifyCmd).PositionalCompletion(
		carapace.ActionValues().Usage("name of the armory"),
	)
	armoryCmd.AddCommand(armoryModifyCmd)

	armoryRefreshCmd := &cobra.Command{
		Use:   consts.RefreshStr,
		Short: "Refresh armory indexes",
		Long:  help.GetHelpFor([]string{consts.ArmoryStr, consts.RefreshStr}),
		Run: func(cmd *cobra.Command, args []string) {
			RefreshArmories(cmd, con)
		},
	}
	armoryCmd.AddCommand(armoryRefreshCmd)

	armoryResetCmd := &cobra.Command{
		Use:   consts.ResetStr,
		Short: "Reset armory configuration",
		Long:  help.GetHelpFor([]string{consts.ArmoryStr, consts.ResetStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ResetArmoryConfig(cmd, con)
		},
	}
	armoryCmd.AddCommand(armoryResetCmd)

	return []*cobra.Command{armoryCmd}
}
