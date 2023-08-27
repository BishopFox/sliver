package pipe

/*
   Sliver Implant Framework
   Copyright (C) 2019  Bishop Fox

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
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// Commands returns the `named-pipe` command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	namedPipeCmd := &cobra.Command{
		Use:   consts.NamedPipeStr,
		Short: "Named pipe handlers management",
	}

	listenCmd := &cobra.Command{
		Use:         consts.ListenStr,
		Short:       "Start a named pipe pivot listener",
		Long:        help.GetHelpFor([]string{consts.PivotsStr, consts.NamedPipeStr}),
		Annotations: flags.RestrictTargets(consts.SessionCmdsFilter),
		Run: func(cmd *cobra.Command, args []string) {
			ListenCmd(cmd, con, args)
		},
	}
	namedPipeCmd.AddCommand(listenCmd)

	flags.Bind("", false, listenCmd, func(f *pflag.FlagSet) {
		f.StringP("bind", "b", "", "name of the named pipe to bind pivot listener")
		f.BoolP("allow-all", "a", false, "allow all users to connect")
	})

	return []*cobra.Command{namedPipeCmd}
}
