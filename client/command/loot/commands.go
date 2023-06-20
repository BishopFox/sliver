package loot

import (
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverConsoleClient) []*cobra.Command {
	lootCmd := &cobra.Command{
		Use:   consts.LootStr,
		Short: "Manage the server's loot store",
		Long:  help.GetHelpFor([]string{consts.LootStr}),
		Run: func(cmd *cobra.Command, args []string) {
			LootCmd(cmd, con, args)
		},
		GroupID: consts.SliverHelpGroup,
	}
	flags.Bind("loot", true, lootCmd, func(f *pflag.FlagSet) {
		f.IntP("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	flags.Bind("loot", false, lootCmd, func(f *pflag.FlagSet) {
		f.StringP("filter", "f", "", "filter based on loot type")
	})

	lootAddCmd := &cobra.Command{
		Use:   consts.LootLocalStr,
		Short: "Add a local file to the server's loot store",
		Long:  help.GetHelpFor([]string{consts.LootStr, consts.LootLocalStr}),
		Run: func(cmd *cobra.Command, args []string) {
			LootAddLocalCmd(cmd, con, args)
		},
		Args: cobra.ExactArgs(1),
	}
	lootCmd.AddCommand(lootAddCmd)
	flags.Bind("loot", false, lootAddCmd, func(f *pflag.FlagSet) {
		f.StringP("name", "n", "", "name of this piece of loot")
		f.StringP("type", "T", "", "force a specific loot type (file/cred)")
		f.StringP("file-type", "F", "", "force a specific file type (binary/text)")
	})
	flags.BindFlagCompletions(lootAddCmd, func(comp *carapace.ActionMap) {
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
			LootAddRemoteCmd(cmd, con, args)
		},
		Args: cobra.ExactArgs(1),
	}
	lootCmd.AddCommand(lootRemoteCmd)
	flags.Bind("loot", false, lootRemoteCmd, func(f *pflag.FlagSet) {
		f.StringP("name", "n", "", "name of this piece of loot")
		f.StringP("type", "T", "", "force a specific loot type (file/cred)")
		f.StringP("file-type", "F", "", "force a specific file type (binary/text)")
	})
	flags.BindFlagCompletions(lootRemoteCmd, func(comp *carapace.ActionMap) {
		(*comp)["type"] = carapace.ActionValues("file", "cred").Tag("loot type")
		(*comp)["file-type"] = carapace.ActionValues("binary", "text").Tag("loot file type")
	})
	carapace.Gen(lootRemoteCmd).PositionalCompletion(carapace.ActionValues().Usage("The file path on the remote host to the loot"))

	lootRenameCmd := &cobra.Command{
		Use:   consts.RenameStr,
		Short: "Re-name a piece of existing loot",
		Long:  help.GetHelpFor([]string{consts.LootStr, consts.RenameStr}),
		Run: func(cmd *cobra.Command, args []string) {
			LootRenameCmd(cmd, con, args)
		},
	}
	lootCmd.AddCommand(lootRenameCmd)

	lootFetchCmd := &cobra.Command{
		Use:   consts.FetchStr,
		Short: "Fetch a piece of loot from the server's loot store",
		Long:  help.GetHelpFor([]string{consts.LootStr, consts.FetchStr}),
		Run: func(cmd *cobra.Command, args []string) {
			LootFetchCmd(cmd, con, args)
		},
	}
	lootCmd.AddCommand(lootFetchCmd)
	flags.Bind("loot", false, lootFetchCmd, func(f *pflag.FlagSet) {
		f.StringP("save", "s", "", "save loot to a local file")
		f.StringP("filter", "f", "", "filter based on loot type")
	})
	flags.BindFlagCompletions(lootFetchCmd, func(comp *carapace.ActionMap) {
		(*comp)["save"] = carapace.ActionFiles().Tag("directory/file to save loot")
	})

	lootRmCmd := &cobra.Command{
		Use:   consts.RmStr,
		Short: "Remove a piece of loot from the server's loot store",
		Long:  help.GetHelpFor([]string{consts.LootStr, consts.RmStr}),
		Run: func(cmd *cobra.Command, args []string) {
			LootRmCmd(cmd, con, args)
		},
	}
	lootCmd.AddCommand(lootRmCmd)
	flags.Bind("loot", false, lootRmCmd, func(f *pflag.FlagSet) {
		f.StringP("filter", "f", "", "filter based on loot type")
	})

	return []*cobra.Command{lootCmd}
}
