package triggers

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Commands returns the "triggers" command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	triggersCmd := &cobra.Command{
		Use:     consts.TriggersStr,
		Short:   "List generated trigger implants",
		Long:    help.GetHelpFor([]string{consts.TriggersStr}),
		GroupID: consts.SliverHelpGroup,
		Run: func(cmd *cobra.Command, args []string) {
			TriggersCmd(cmd, con, args)
		},
	}
	flags.Bind("triggers", true, triggersCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	// triggers target <index> <ip>
	triggersTargetCmd := &cobra.Command{
		Use:   "target <index> <ip>",
		Short: "Associate a target IP with a trigger implant by index",
		Long: `Store a target IP for a trigger implant so that 'trigger send <index> <intent>'
can automatically resolve the target. The mapping is stored client-side
in ~/.sliver-client/triggers.json.

Examples:
  triggers target 1 192.168.1.42
  triggers target 2 10.0.0.5`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			TriggerTargetCmd(cmd, con, args)
		},
	}
	triggersCmd.AddCommand(triggersTargetCmd)

	return []*cobra.Command{triggersCmd}
}
