package aka

import (
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/spf13/cobra"
)

func ServerCommands(con *console.SliverClient) []*cobra.Command {
  akaCommand := &cobra.Command{
    Use: "aka",
    Short: "Manage command aliases (aka's)",
    Run: func(cmd *cobra.Command, args []string) {
      AkaListCmd(cmd, con, args)
    },
    GroupID: constants.GenericHelpGroup,
  }

  createAka := &cobra.Command{
    Use: "create <alias> <command> [args...]",
    Short: "Create a new command alias (aka)",
    Args: cobra.MinimumNArgs(2),
    Run: func(cmd *cobra.Command, args []string) {
      AkaCreateCmd(cmd, con, args)
    },
  }
  akaCommand.AddCommand(createAka)

  con.App.PreCmdRunLineHooks = append(con.App.PreCmdRunLineHooks, Cmdhook)

  return []*cobra.Command{akaCommand}
}

func ImplantCommands(con *console.SliverClient) []*cobra.Command {
  akaCommand := &cobra.Command{
    Use: "aka",
    Short: "Manage command aliases (aka's)",
    DisableFlagParsing: true,
    Run: func(cmd *cobra.Command, args []string) {
      AkaListCmd(cmd, con, args)
    },
    GroupID: constants.SliverCoreHelpGroup,
  }

  createAka := &cobra.Command{
    Use: "create <alias> <command> [args...]",
    Short: "Create a new command alias (aka)",
    Args: cobra.MinimumNArgs(2),
    DisableFlagParsing: true, // No flags with aka create so we capture flags to other commands
    DisableFlagsInUseLine: true,
    Run: func(cmd *cobra.Command, args []string) {
      AkaCreateCmd(cmd, con, args)
    },
  }
  createAka.InitDefaultHelpFlag()
  createAka.Flags().Lookup("help").Hidden = true
  createAka.Flags().ParseErrorsWhitelist.UnknownFlags = true
  akaCommand.AddCommand(createAka)

  con.App.PreCmdRunLineHooks = append(con.App.PreCmdRunLineHooks, Cmdhook)

  return []*cobra.Command{akaCommand}
}
