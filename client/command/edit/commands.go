package edit

import (
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Commands returns the edit command.
func Commands(con *console.SliverClient) []*cobra.Command {
	editCmd := &cobra.Command{
		Use:   consts.EditStr,
		Short: "Edit a remote text file",
		Long:  help.GetHelpFor([]string{consts.EditStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			EditCmd(cmd, con, args)
		},
		GroupID: consts.FilesystemHelpGroup,
	}

	flags.Bind("", false, editCmd, func(f *pflag.FlagSet) {
		f.String("syntax", "", "syntax highlighting lexer name (optional)")
		f.Bool("syntax-select", false, "prompt to select syntax highlighting lexer")
		f.Bool("line-numbers", false, "show line numbers")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	carapace.Gen(editCmd).PositionalCompletion(
		carapace.ActionValues().Usage("path to the remote file to edit"),
	)

	return []*cobra.Command{editCmd}
}
