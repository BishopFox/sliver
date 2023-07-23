package reaction

import (
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
	reactionCmd := &cobra.Command{
		Use:   consts.ReactionStr,
		Short: "Manage automatic reactions to events",
		Long:  help.GetHelpFor([]string{consts.ReactionStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ReactionCmd(cmd, con, args)
		},
		GroupID: consts.SliverHelpGroup,
	}

	reactionSetCmd := &cobra.Command{
		Use:   consts.SetStr,
		Short: "Set a reaction to an event",
		Long:  help.GetHelpFor([]string{consts.ReactionStr, consts.SetStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ReactionSetCmd(cmd, con, args)
		},
	}
	reactionCmd.AddCommand(reactionSetCmd)
	flags.Bind("reactions", false, reactionSetCmd, func(f *pflag.FlagSet) {
		f.StringP("event", "e", "", "specify the event type to react to")
	})

	flags.BindFlagCompletions(reactionSetCmd, func(comp *carapace.ActionMap) {
		(*comp)["event"] = carapace.ActionValues(
			consts.SessionOpenedEvent,
			consts.SessionClosedEvent,
			consts.SessionUpdateEvent,
			consts.BeaconRegisteredEvent,
			consts.CanaryEvent,
			consts.WatchtowerEvent,
		)
	})

	reactionUnsetCmd := &cobra.Command{
		Use:   consts.UnsetStr,
		Short: "Unset an existing reaction",
		Long:  help.GetHelpFor([]string{consts.ReactionStr, consts.UnsetStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ReactionUnsetCmd(cmd, con, args)
		},
	}
	reactionCmd.AddCommand(reactionUnsetCmd)
	flags.Bind("reactions", false, reactionUnsetCmd, func(f *pflag.FlagSet) {
		f.IntP("id", "i", 0, "the id of the reaction to remove")
	})
	flags.BindFlagCompletions(reactionUnsetCmd, func(comp *carapace.ActionMap) {
		(*comp)["id"] = ReactionIDCompleter(con)
	})

	reactionSaveCmd := &cobra.Command{
		Use:   consts.SaveStr,
		Short: "Save current reactions to disk",
		Long:  help.GetHelpFor([]string{consts.ReactionStr, consts.SaveStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ReactionSaveCmd(cmd, con, args)
		},
	}
	reactionCmd.AddCommand(reactionSaveCmd)

	reactionReloadCmd := &cobra.Command{
		Use:   consts.ReloadStr,
		Short: "Reload reactions from disk, replaces the running configuration",
		Long:  help.GetHelpFor([]string{consts.ReactionStr, consts.ReloadStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ReactionReloadCmd(cmd, con, args)
		},
	}
	reactionCmd.AddCommand(reactionReloadCmd)

	return []*cobra.Command{reactionCmd}
}
