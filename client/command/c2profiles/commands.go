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
	flags.BindFlagCompletions(exportC2ProfileCmd, func(comp *carapace.ActionMap) {
		(*comp)["name"] = generate.HTTPC2Completer(con)
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

	generateC2ProfileCmd := &cobra.Command{
		Use:   consts.C2GenerateStr,
		Short: "Generate a C2 Profile from a list of urls",
		Long:  help.GetHelpFor([]string{consts.C2ProfileStr + "." + consts.C2GenerateStr}),
		Run: func(cmd *cobra.Command, args []string) {
			GenerateC2ProfileCmd(cmd, con, args)
		},
	}

	flags.Bind(consts.GenerateStr, false, generateC2ProfileCmd, func(f *pflag.FlagSet) {
		f.StringP("file", "f", "", "Path to file containing URL list, /hello/there.txt one per line")
		f.BoolP("import", "i", false, "Import the generated profile after creation")
		f.StringP("name", "n", "", "HTTP C2 Profile name to save C2Profile as")
		f.StringP("template", "t", consts.DefaultC2Profile, "HTTP C2 Profile to use as a template for the new profile")
	})

	flags.BindFlagCompletions(generateC2ProfileCmd, func(comp *carapace.ActionMap) {
		(*comp)["template"] = generate.HTTPC2Completer(con)
	})

	C2ProfileCmd.AddCommand(generateC2ProfileCmd)
	C2ProfileCmd.AddCommand(importC2ProfileCmd)
	C2ProfileCmd.AddCommand(exportC2ProfileCmd)

	return []*cobra.Command{
		C2ProfileCmd,
	}
}
