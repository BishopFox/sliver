package crack

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
func Commands(con *console.SliverClient) []*cobra.Command {
	crackCmd := &cobra.Command{
		Use:     consts.CrackStr,
		Short:   "Crack: GPU password cracking",
		Long:    help.GetHelpFor([]string{consts.CrackStr}),
		GroupID: consts.GenericHelpGroup,
		Run: func(cmd *cobra.Command, args []string) {
			CrackCmd(cmd, con, args)
		},
	}
	flags.Bind("", true, crackCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	crackStationsCmd := &cobra.Command{
		Use:   consts.StationsStr,
		Short: "Manage crackstations",
		Long:  help.GetHelpFor([]string{consts.CrackStr, consts.StationsStr}),
		Run: func(cmd *cobra.Command, args []string) {
			CrackStationsCmd(cmd, con, args)
		},
	}
	crackCmd.AddCommand(crackStationsCmd)

	wordlistsCmd := &cobra.Command{
		Use:   consts.WordlistsStr,
		Short: "Manage wordlists",
		Long:  help.GetHelpFor([]string{consts.CrackStr, consts.WordlistsStr}),
		Run: func(cmd *cobra.Command, args []string) {
			CrackWordlistsCmd(cmd, con, args)
		},
	}
	crackCmd.AddCommand(wordlistsCmd)

	wordlistsAddCmd := &cobra.Command{
		Use:   consts.AddStr,
		Short: "Add a wordlist",
		Run: func(cmd *cobra.Command, args []string) {
			CrackWordlistsAddCmd(cmd, con, args)
		},
	}
	flags.Bind("", false, wordlistsAddCmd, func(f *pflag.FlagSet) {
		f.StringP("name", "n", "", "wordlist name (blank = filename)")
	})
	carapace.Gen(wordlistsAddCmd).PositionalCompletion(carapace.ActionFiles().Usage("path to local wordlist file"))
	wordlistsCmd.AddCommand(wordlistsAddCmd)

	wordlistsRmCmd := &cobra.Command{
		Use:   consts.RmStr,
		Short: "Remove a wordlist",
		Run: func(cmd *cobra.Command, args []string) {
			CrackWordlistsRmCmd(cmd, con, args)
		},
	}
	wordlistsCmd.AddCommand(wordlistsRmCmd)
	carapace.Gen(wordlistsRmCmd).PositionalCompletion(CrackWordlistCompleter(con).Usage("wordlist to remove"))

	rulesCmd := &cobra.Command{
		Use:   consts.RulesStr,
		Short: "Manage rule files",
		Long:  help.GetHelpFor([]string{consts.CrackStr, consts.RulesStr}),
		Run: func(cmd *cobra.Command, args []string) {
			CrackRulesCmd(cmd, con, args)
		},
	}
	crackCmd.AddCommand(rulesCmd)

	rulesAddCmd := &cobra.Command{
		Use:   consts.AddStr,
		Short: "Add a rules file",
		Long:  help.GetHelpFor([]string{consts.CrackStr, consts.RulesStr, consts.AddStr}),
		Run: func(cmd *cobra.Command, args []string) {
			CrackRulesAddCmd(cmd, con, args)
		},
	}
	flags.Bind("", false, rulesAddCmd, func(f *pflag.FlagSet) {
		f.StringP("name", "n", "", "rules name (blank = filename)")
	})
	carapace.Gen(rulesAddCmd).PositionalCompletion(carapace.ActionFiles().Usage("path to local rules file"))
	rulesCmd.AddCommand(rulesAddCmd)

	rulesRmCmd := &cobra.Command{
		Use:   consts.RmStr,
		Short: "Remove rules",
		Long:  help.GetHelpFor([]string{consts.CrackStr, consts.RulesStr, consts.RmStr}),
		Run: func(cmd *cobra.Command, args []string) {
			CrackRulesRmCmd(cmd, con, args)
		},
	}
	carapace.Gen(rulesRmCmd).PositionalCompletion(CrackRulesCompleter(con).Usage("rules to remove"))
	rulesCmd.AddCommand(rulesRmCmd)

	hcstat2Cmd := &cobra.Command{
		Use:   consts.Hcstat2Str,
		Short: "Manage markov hcstat2 files",
		Long:  help.GetHelpFor([]string{consts.CrackStr, consts.Hcstat2Str}),
		Run: func(cmd *cobra.Command, args []string) {
			CrackHcstat2Cmd(cmd, con, args)
		},
	}
	crackCmd.AddCommand(hcstat2Cmd)

	hcstat2AddCmd := &cobra.Command{
		Use:   consts.AddStr,
		Short: "Add a hcstat2 file",
		Long:  help.GetHelpFor([]string{consts.CrackStr, consts.Hcstat2Str, consts.AddStr}),
		Run: func(cmd *cobra.Command, args []string) {
			CrackHcstat2AddCmd(cmd, con, args)
		},
	}
	flags.Bind("", false, hcstat2AddCmd, func(f *pflag.FlagSet) {
		f.StringP("name", "n", "", "hcstat2 name (blank = filename)")
	})
	carapace.Gen(hcstat2AddCmd).PositionalCompletion(carapace.ActionFiles().Usage("path to local hcstat2 file"))
	hcstat2Cmd.AddCommand(hcstat2AddCmd)

	hcstat2RmCmd := &cobra.Command{
		Use:   consts.RmStr,
		Short: "Remove hcstat2 file",
		Long:  help.GetHelpFor([]string{consts.CrackStr, consts.Hcstat2Str, consts.RmStr}),
		Run: func(cmd *cobra.Command, args []string) {
			CrackHcstat2RmCmd(cmd, con, args)
		},
	}
	carapace.Gen(hcstat2RmCmd).PositionalCompletion(CrackHcstat2Completer(con).Usage("hcstat2 to remove"))
	hcstat2Cmd.AddCommand(hcstat2RmCmd)

	return []*cobra.Command{crackCmd}
}
