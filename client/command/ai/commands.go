package ai

import (
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/spf13/cobra"
)

// Commands returns the ai command.
func Commands(con *console.SliverClient) []*cobra.Command {
	return []*cobra.Command{newAICommand(consts.SliverCoreHelpGroup, con)}
}

// ServerCommands returns the ai command for the top-level client REPL.
func ServerCommands(con *console.SliverClient) []*cobra.Command {
	return []*cobra.Command{newAICommand(consts.GenericHelpGroup, con)}
}

func newAICommand(groupID string, con *console.SliverClient) *cobra.Command {
	return &cobra.Command{
		Use:     consts.AIStr,
		Short:   "Open the Sliver AI TUI layout preview",
		Long:    help.GetHelpFor([]string{consts.AIStr}),
		Args:    cobra.NoArgs,
		GroupID: groupID,
		Run: func(cmd *cobra.Command, args []string) {
			AICmd(cmd, con, args)
		},
	}
}
