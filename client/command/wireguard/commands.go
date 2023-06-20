package wireguard

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
	wgConfigCmd := &cobra.Command{
		Use:   consts.WgConfigStr,
		Short: "Generate a new WireGuard client config",
		Long:  help.GetHelpFor([]string{consts.WgConfigStr}),
		Run: func(cmd *cobra.Command, args []string) {
			WGConfigCmd(cmd, con, args)
		},
		GroupID: consts.NetworkHelpGroup,
	}

	flags.Bind("wg-config", true, wgConfigCmd, func(f *pflag.FlagSet) {
		f.IntP("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	flags.Bind("wg-config", false, wgConfigCmd, func(f *pflag.FlagSet) {
		f.StringP("save", "s", "", "save configuration to file (.conf)")
	})
	flags.BindFlagCompletions(wgConfigCmd, func(comp *carapace.ActionMap) {
		(*comp)["save"] = carapace.ActionFiles().Tag("directory/file to save config")
	})

	return []*cobra.Command{wgConfigCmd}
}
