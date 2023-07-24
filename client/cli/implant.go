package cli

import (
	"github.com/bishopfox/sliver/client/command"
	"github.com/bishopfox/sliver/client/command/use"
	client "github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/reeflective/console"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func implantCmd(con *client.SliverClient, sliverCmds console.Commands) *cobra.Command {
	implantCmd := sliverCmds()
	implantCmd.Use = constants.ImplantMenu

	implantFlags := pflag.NewFlagSet(constants.ImplantMenu, pflag.ContinueOnError)
	implantFlags.StringP("use", "s", "", "interact with a session")
	implantCmd.Flags().AddFlagSet(implantFlags)

	command.BindFlagCompletions(implantCmd, func(comp *carapace.ActionMap) {
		(*comp)["use"] = carapace.ActionCallback(func(c carapace.Context) carapace.Action {
			return use.SessionIDCompleter(con)
		})
	})

	return implantCmd
}
