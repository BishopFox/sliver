package sgn

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
	shikataGaNaiCmd := &cobra.Command{
		Use:   consts.ShikataGaNai,
		Short: "Polymorphic binary shellcode encoder (ノ ゜Д゜)ノ ︵ 仕方がない",
		Long:  help.GetHelpFor([]string{consts.ShikataGaNai}),
		Run: func(cmd *cobra.Command, args []string) {
			ShikataGaNaiCmd(cmd, con, args)
		},
		Args:    cobra.ExactArgs(1),
		GroupID: consts.PayloadsHelpGroup,
	}
	flags.Bind("shikata ga nai", false, shikataGaNaiCmd, func(f *pflag.FlagSet) {
		f.StringP("save", "s", "", "save output to local file")
		f.StringP("arch", "a", "amd64", "architecture of shellcode")
		f.IntP("iterations", "i", 1, "number of iterations")
		f.StringP("bad-chars", "b", "", "hex encoded bad characters to avoid (e.g. 0001)")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	flags.BindFlagCompletions(shikataGaNaiCmd, func(comp *carapace.ActionMap) {
		(*comp)["arch"] = carapace.ActionValues("386", "amd64").Tag("shikata-ga-nai architectures")
		(*comp)["save"] = carapace.ActionFiles().Tag("directory/file to save shellcode")
	})
	carapace.Gen(shikataGaNaiCmd).PositionalCompletion(carapace.ActionFiles().Tag("shellcode file"))

	return []*cobra.Command{shikataGaNaiCmd}
}
