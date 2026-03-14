package harriet

import (
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Commands returns the Harriet command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	harrietCmd := &cobra.Command{
		Use:   consts.HarrietStr,
		Short: "Generate AES-encrypted Harriet stager wrapping Sliver shellcode",
		Long:  help.GetHelpFor([]string{consts.HarrietStr}),
		Run: func(cmd *cobra.Command, args []string) {
			HarrietGenerateCmd(cmd, con, args)
		},
		GroupID: consts.PayloadsHelpGroup,
	}
	flags.Bind("", false, harrietCmd, func(f *pflag.FlagSet) {
		f.StringP("output", "o", "", "output filename (default: payload.exe)")
		f.StringP("harriet-path", "H", "", "path to Harriet repo (default: /opt/Home-Grown-Red-Team/Harriet)")
		f.StringP("format", "f", "exe", "output format: exe or dll")
		f.StringP("method", "m", "aes", "execution method: aes, inject, queueapc, nativeapi, directsyscall")
		f.StringP("listener", "l", "", "Sliver listener host:port for stager shellcode")
		f.StringP("arch", "a", "amd64", "architecture: amd64 or x86")
		f.BoolP("no-sign", "S", false, "skip binary signing")
		f.Int64P("timeout", "t", 120, "grpc timeout in seconds")
	})

	return []*cobra.Command{harrietCmd}
}
