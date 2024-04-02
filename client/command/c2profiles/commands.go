package c2profiles

import (
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {

	importC2ProfileCmd := &cobra.Command{
		Use:   consts.ImportC2ProfileStr,
		Short: "Import HTTP C2 profile",
		Long:  help.GetHelpFor([]string{consts.ImportC2ProfileStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ImportC2ProfileCmd(cmd, con, args)
		},
	}
	flags.Bind(consts.ImportC2ProfileStr, false, importC2ProfileCmd, func(f *pflag.FlagSet) {
		f.StringP("name", "n", consts.DefaultC2Profile, "HTTP C2 Profile name")
		f.StringP("file", "f", "", "Path to C2 configuration file to import")
		f.BoolP("overwrite", "o", false, "Overwrite profile if it exists")
	})

	exportC2ProfileCmd := &cobra.Command{
		Use:   consts.ExportC2ProfileStr,
		Short: "Export HTTP C2 profile",
		Long:  help.GetHelpFor([]string{consts.ExportC2ProfileStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ExportC2ProfileCmd(cmd, con, args)
		},
	}
	flags.Bind(consts.ExportC2ProfileStr, false, exportC2ProfileCmd, func(f *pflag.FlagSet) {
		f.StringP("file", "f", "", "Path to file to export C2 configuration to")
		f.StringP("name", "n", consts.DefaultC2Profile, "HTTP C2 Profile name")

	})

	C2ProfileCmd := &cobra.Command{
		Use:   consts.C2ProfileStr,
		Short: "Display C2 profile details",
		Long:  help.GetHelpFor([]string{consts.C2ProfileStr}),
		Run: func(cmd *cobra.Command, args []string) {
			C2ProfileCmd(cmd, con, args)
		},
		GroupID: consts.NetworkHelpGroup,
	}
	flags.Bind(consts.C2ProfileStr, true, C2ProfileCmd, func(f *pflag.FlagSet) {
		f.StringP("name", "n", consts.DefaultC2Profile, "HTTP C2 Profile to display")
	})

	flags.BindFlagCompletions(C2ProfileCmd, func(comp *carapace.ActionMap) {
		(*comp)["name"] = generate.HTTPC2Completer(con)
	})
	C2ProfileCmd.AddCommand(importC2ProfileCmd)
	C2ProfileCmd.AddCommand(exportC2ProfileCmd)

	return []*cobra.Command{
		C2ProfileCmd,
	}
}
