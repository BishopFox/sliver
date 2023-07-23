package update

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

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	updateCmd := &cobra.Command{
		Use:   consts.UpdateStr,
		Short: "Check for updates",
		Long:  help.GetHelpFor([]string{consts.UpdateStr}),
		Run: func(cmd *cobra.Command, args []string) {
			UpdateCmd(cmd, con, args)
		},
		GroupID: consts.GenericHelpGroup,
	}
	flags.Bind("update", false, updateCmd, func(f *pflag.FlagSet) {
		f.BoolP("prereleases", "P", false, "include pre-released (unstable) versions")
		f.StringP("proxy", "p", "", "specify a proxy url (e.g. http://localhost:8080)")
		f.StringP("save", "s", "", "save downloaded files to specific directory (default user home dir)")
		f.BoolP("insecure", "I", false, "skip tls certificate validation")
		f.IntP("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	flags.BindFlagCompletions(updateCmd, func(comp *carapace.ActionMap) {
		(*comp)["proxy"] = completers.LocalProxyCompleter()
	})

	versionCmd := &cobra.Command{
		Use:   consts.VersionStr,
		Short: "Display version information",
		Long:  help.GetHelpFor([]string{consts.VersionStr}),
		Run: func(cmd *cobra.Command, args []string) {
			VerboseVersionsCmd(cmd, con, args)
		},
		GroupID: consts.GenericHelpGroup,
	}
	flags.Bind("update", false, versionCmd, func(f *pflag.FlagSet) {
		f.IntP("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	return []*cobra.Command{updateCmd, versionCmd}
}
