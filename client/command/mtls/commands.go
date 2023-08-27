package mtls

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
	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// Commands returns the `mtls` command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	mtlsCmd := &cobra.Command{
		Use:     consts.MtlsStr,
		Short:   "mTLS handlers management",
		GroupID: consts.NetworkHelpGroup,
	}

	listenCmd := &cobra.Command{
		Use:   consts.ListenStr,
		Short: "Start an mTLS listener",
		Long:  help.GetHelpFor([]string{consts.MtlsStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ListenCmd(cmd, con, args)
		},
	}
	mtlsCmd.AddCommand(listenCmd)

	flags.Bind("mTLS listener", false, listenCmd, func(f *pflag.FlagSet) {
		f.StringP("lhost", "L", "", "interface to bind server to")
		f.Uint32P("lport", "l", generate.DefaultMTLSLPort, "tcp listen port")
		f.BoolP("persistent", "p", false, "make persistent across restarts")
	})

	return []*cobra.Command{mtlsCmd}
}
