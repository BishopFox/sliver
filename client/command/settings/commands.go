package settings

import (
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/reeflective/console/commands/readline"
	"github.com/spf13/cobra"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	settingsCmd := &cobra.Command{
		Use:   consts.SettingsStr,
		Short: "Manage client settings",
		Long:  help.GetHelpFor([]string{consts.SettingsStr}),
		Run: func(cmd *cobra.Command, args []string) {
			SettingsCmd(cmd, con, args)
		},
		GroupID: consts.GenericHelpGroup,
	}
	settingsCmd.AddCommand(&cobra.Command{
		Use:   consts.SaveStr,
		Short: "Save the current settings to disk",
		Long:  help.GetHelpFor([]string{consts.SettingsStr, consts.SaveStr}),
		Run: func(cmd *cobra.Command, args []string) {
			SettingsSaveCmd(cmd, con, args)
		},
	})
	settingsCmd.AddCommand(&cobra.Command{
		Use:   consts.TablesStr,
		Short: "Modify tables setting (style)",
		Long:  help.GetHelpFor([]string{consts.SettingsStr, consts.TablesStr}),
		Run: func(cmd *cobra.Command, args []string) {
			SettingsTablesCmd(cmd, con, args)
		},
	})
	settingsCmd.AddCommand(&cobra.Command{
		Use:   "beacon-autoresults",
		Short: "Automatically display beacon task results when completed",
		Long:  help.GetHelpFor([]string{consts.SettingsStr, "beacon-autoresults"}),
		Run: func(cmd *cobra.Command, args []string) {
			SettingsBeaconsAutoResultCmd(cmd, con, args)
		},
	})
	settingsCmd.AddCommand(&cobra.Command{
		Use:   "autoadult",
		Short: "Automatically accept OPSEC warnings",
		Long:  help.GetHelpFor([]string{consts.SettingsStr, "autoadult"}),
		Run: func(cmd *cobra.Command, args []string) {
			SettingsAutoAdultCmd(cmd, con, args)
		},
	})
	settingsCmd.AddCommand(&cobra.Command{
		Use:   "always-overflow",
		Short: "Disable table pagination",
		Long:  help.GetHelpFor([]string{consts.SettingsStr, "always-overflow"}),
		Run: func(cmd *cobra.Command, args []string) {
			SettingsAlwaysOverflow(cmd, con, args)
		},
	})
	settingsCmd.AddCommand(&cobra.Command{
		Use:   "small-terminal",
		Short: "Set the small terminal width",
		Long:  help.GetHelpFor([]string{consts.SettingsStr, "small-terminal"}),
		Run: func(cmd *cobra.Command, args []string) {
			SettingsSmallTerm(cmd, con, args)
		},
	})
	settingsCmd.AddCommand(&cobra.Command{
		Use:   "user-connect",
		Short: "Enable user connections/disconnections (can be very verbose when they use CLI)",
		Run: func(cmd *cobra.Command, args []string) {
			SettingsUserConnect(cmd, con, args)
		},
	})
	settingsCmd.AddCommand(&cobra.Command{
		Use:   "console-logs",
		Short: "Log console output (toggle)",
		Long:  help.GetHelpFor([]string{consts.SettingsStr, "console-logs"}),
		Run: func(ctx *cobra.Command, args []string) {
			SettingsConsoleLogs(ctx, con)
		},
	})

	// Bind a readline subcommand to the `settings` one, for allowing users to
	// manipulate the shell instance keymaps, bindings, macros and global options.
	settingsCmd.AddCommand(readline.Commands(con.App.Shell()))

	return []*cobra.Command{settingsCmd}
}
