package c2profiles

import (
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/command/completers"
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
	consts "github.com/bishopfox/sliver/client/constants"
)

// Commands returns all C2 profile management commands.
func Commands(con *console.SliverClient) []*cobra.Command {
	C2ProfileCmd := &cobra.Command{
		Use:     consts.C2ProfileStr,
		Short:   "Display C2 profile details",
		GroupID: consts.NetworkHelpGroup,
		Long:    help.GetHelpFor([]string{consts.C2ProfileStr}),
		Run: func(cmd *cobra.Command, args []string) {
			C2ProfileCmd(cmd, con, args)
		},
	}
	flags.Bind(consts.C2ProfileStr, true, C2ProfileCmd, func(f *pflag.FlagSet) {
		f.StringP("name", "n", constants.DefaultC2Profile, "HTTP C2 Profile to display")
	})
	completers.NewFlagCompsFor(C2ProfileCmd, func(comp *carapace.ActionMap) {
		(*comp)["name"] = generate.HTTPC2Completer(con)
	})

	ImportC2ProfileCmd := &cobra.Command{
		Use:   consts.ImportC2ProfileStr,
		Short: "Import HTTP C2 profile",
		Long:  help.GetHelpFor([]string{consts.ImportC2ProfileStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ImportC2ProfileCmd(cmd, con, args)
		},
	}
	flags.Bind(consts.ImportC2ProfileStr, true, ImportC2ProfileCmd, func(f *pflag.FlagSet) {
		f.StringP("name", "n", constants.DefaultC2Profile, "HTTP C2 Profile name")
		f.StringP("file", "f", "", "Path to C2 configuration file to import")
		f.BoolP("overwrite", "o", false, "Overwrite profile if it exists")
	})

	C2ProfileCmd.AddCommand(ImportC2ProfileCmd)

	return []*cobra.Command{C2ProfileCmd}
}
