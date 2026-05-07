package shellcodeencoders

import (
	"github.com/bishopfox/sliver/client/command/completers"
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Commands returns the shellcode-encoders command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	shellcodeEncodersCmd := &cobra.Command{
		Use:   consts.ShellcodeEncodersStr,
		Short: "List supported shellcode encoders",
		Long:  help.GetHelpFor([]string{consts.ShellcodeEncodersStr}),
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			ShellcodeEncodersCmd(cmd, con, args)
		},
		GroupID: consts.PayloadsHelpGroup,
	}
	flags.Bind("shellcode-encoders", false, shellcodeEncodersCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	shellcodeEncodersEncodeCmd := &cobra.Command{
		Use:   "encode <shellcode file>...",
		Short: "Encode shellcode using server encoders",
		Long:  help.GetHelpFor([]string{consts.ShellcodeEncodersStr, "encode"}),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ShellcodeEncodersEncodeCmd(cmd, con, args)
		},
	}
	flags.Bind("shellcode-encoders encode", false, shellcodeEncodersEncodeCmd, func(f *pflag.FlagSet) {
		f.StringP("encoder", "e", "", "encoder to use (shikata_ga_nai, xor, xor_dynamic)")
		f.StringP("arch", "a", "amd64", "architecture of the shellcode")
		f.IntP("iterations", "i", 1, "number of encoding iterations")
		f.StringP("bad-chars", "b", "", "hex encoded bad characters to avoid (e.g. 0001)")
		f.StringP("output", "o", "", "output file (or directory when encoding multiple files)")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	flags.BindFlagCompletions(shellcodeEncodersEncodeCmd, func(comp *carapace.ActionMap) {
		(*comp)["encoder"] = ShellcodeEncoderNameCompleter(con)
		(*comp)["arch"] = ShellcodeEncoderArchCompleter(con)
		(*comp)["output"] = carapace.ActionFiles().Tag("output file/directory")
	})
	completers.RegisterLocalFilePathFlagCompletion(shellcodeEncodersEncodeCmd, "output")
	carapace.Gen(shellcodeEncodersEncodeCmd).PositionalCompletion(carapace.ActionFiles().Tag("shellcode file"))
	completers.RegisterLocalFilePathPositionalAnyCompletion(shellcodeEncodersEncodeCmd)
	shellcodeEncodersCmd.AddCommand(shellcodeEncodersEncodeCmd)

	return []*cobra.Command{shellcodeEncodersCmd}
}
