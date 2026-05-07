package aka

import (
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/spf13/cobra"
)

func ServerCommands(con *console.SliverClient) []*cobra.Command {
	return commands(con, constants.GenericHelpGroup)
}

func ImplantCommands(con *console.SliverClient) []*cobra.Command {
	return commands(con, constants.SliverCoreHelpGroup)
}

func commands(con *console.SliverClient, group string) []*cobra.Command {
	akaCommand := &cobra.Command{
		Use:   "aka",
		Short: "Manage command aliases (aka's)",
		Run: func(cmd *cobra.Command, args []string) {
			AkaListCmd(cmd, con, args)
		},
		GroupID: group,
	}

	createAka := &cobra.Command{
		Use:                   "create <alias> <command> [args...]",
		Short:                 "Create a new command alias (aka)",
		Args:                  cobra.MinimumNArgs(2),
		DisableFlagParsing:    true, // No flags with aka create so we capture flags to other commands
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			AkaCreateCmd(cmd, con, args)
		},
	}
	createAka.InitDefaultHelpFlag()
	createAka.Flags().Lookup("help").Hidden = true
	akaCommand.AddCommand(createAka)

	deleteAka := &cobra.Command{
		Use:   "delete <alias>",
		Short: "Delete a command alias (aka)",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			AkaDeleteCmd(cmd, con, args)
		},
	}
	akaCommand.AddCommand(deleteAka)

	con.App.PreCmdRunLineHooks = append(con.App.PreCmdRunLineHooks, Cmdhook)

	err := LoadAkaAliases()
	if err != nil {
		con.PrintErrorf("Failed to load aka alises: %s\n", err)
	}

	return []*cobra.Command{akaCommand}
}
