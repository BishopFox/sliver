package creds

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
	credsCmd := &cobra.Command{
		Use:     consts.CredsStr,
		Short:   "Manage the database of credentials",
		Long:    help.GetHelpFor([]string{consts.CredsStr}),
		GroupID: consts.GenericHelpGroup,
		Run: func(cmd *cobra.Command, args []string) {
			CredsCmd(cmd, con, args)
		},
	}
	flags.Bind("creds", true, credsCmd, func(f *pflag.FlagSet) {
		f.IntP("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	credsAddCmd := &cobra.Command{
		Use:   consts.AddStr,
		Short: "Add a credential to the database",
		Long:  help.GetHelpFor([]string{consts.CredsStr, consts.AddStr}),
		Run: func(cmd *cobra.Command, args []string) {
			CredsAddCmd(cmd, con, args)
		},
	}
	flags.Bind("", false, credsAddCmd, func(f *pflag.FlagSet) {
		f.StringP("collection", "c", "", "name of collection")
		f.StringP("username", "u", "", "username for the credential")
		f.StringP("plaintext", "p", "", "plaintext for the credential")
		f.StringP("hash", "P", "", "hash of the credential")
		f.StringP("hash-type", "H", "", "hash type of the credential")
	})
	flags.BindFlagCompletions(credsAddCmd, func(comp *carapace.ActionMap) {
		(*comp)["hash-type"] = CredsHashTypeCompleter(con)
	})
	credsCmd.AddCommand(credsAddCmd)

	credsAddFileCmd := &cobra.Command{
		Use:   consts.FileStr,
		Short: "Add a credential to the database",
		Long:  help.GetHelpFor([]string{consts.CredsStr, consts.AddStr, consts.FileStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			CredsAddHashFileCmd(cmd, con, args)
		},
	}
	flags.Bind("", false, credsAddFileCmd, func(f *pflag.FlagSet) {
		f.StringP("collection", "c", "", "name of collection")
		f.StringP("file-format", "F", HashNewlineFormat, "file format of the credential file")
		f.StringP("hash-type", "H", "", "hash type of the credential")
	})
	flags.BindFlagCompletions(credsAddFileCmd, func(comp *carapace.ActionMap) {
		(*comp)["collection"] = CredsCollectionCompleter(con)
		(*comp)["file-format"] = CredsHashFileFormatCompleter(con)
		(*comp)["hash-type"] = CredsHashTypeCompleter(con)
	})
	carapace.Gen(credsAddFileCmd).PositionalCompletion(carapace.ActionFiles().Tag("credential file"))
	credsAddCmd.AddCommand(credsAddFileCmd)

	credsRmCmd := &cobra.Command{
		Use:   consts.RmStr,
		Short: "Remove a credential to the database",
		Long:  help.GetHelpFor([]string{consts.CredsStr, consts.RmStr}),
		Run: func(cmd *cobra.Command, args []string) {
			CredsRmCmd(cmd, con, args)
		},
	}
	carapace.Gen(credsRmCmd).PositionalCompletion(CredsCredentialIDCompleter(con).Usage("id of credential to remove (leave empty to select)"))
	credsCmd.AddCommand(credsRmCmd)

	return []*cobra.Command{credsCmd}
}
