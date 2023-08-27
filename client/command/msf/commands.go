package msf

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
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/command/completers"
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// Commands returns the `msf` command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	msfCmd := &cobra.Command{
		Use:     consts.MsfStr,
		Short:   "Generic transport preparation & management",
		GroupID: consts.ExecutionHelpGroup,
	}

	msfExecCmd := &cobra.Command{
		Use:   consts.ExecuteStr,
		Short: "Execute an MSF payload in the current process",
		Long:  help.GetHelpFor([]string{consts.MsfStr}),
		Run: func(cmd *cobra.Command, args []string) {
			MsfCmd(cmd, con, args)
		},
	}
	msfCmd.AddCommand(msfExecCmd)
	flags.Bind("", false, msfExecCmd, func(f *pflag.FlagSet) {
		f.StringP("payload", "m", "meterpreter_reverse_https", "msf payload")
		f.StringP("lhost", "L", "", "listen host")
		f.IntP("lport", "l", 4444, "listen port")
		f.StringP("encoder", "e", "", "msf encoder")
		f.IntP("iterations", "i", 1, "iterations of the encoder")

		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	completers.NewFlagCompsFor(msfExecCmd, func(comp *carapace.ActionMap) {
		(*comp)["encoder"] = generate.MsfEncoderCompleter(con)
		(*comp)["payload"] = generate.MsfPayloadCompleter(con)
	})

	msfInjectCmd := &cobra.Command{
		Use:   consts.MsfInjectStr,
		Short: "Inject an MSF payload into a process",
		Long:  help.GetHelpFor([]string{consts.MsfInjectStr}),
		Run: func(cmd *cobra.Command, args []string) {
			MsfInjectCmd(cmd, con, args)
		},
	}
	msfCmd.AddCommand(msfInjectCmd)
	flags.Bind("", false, msfInjectCmd, func(f *pflag.FlagSet) {
		f.IntP("pid", "p", -1, "pid to inject into")
		f.StringP("payload", "m", "meterpreter_reverse_https", "msf payload")
		f.StringP("lhost", "L", "", "listen host")
		f.IntP("lport", "l", 4444, "listen port")
		f.StringP("encoder", "e", "", "msf encoder")
		f.IntP("iterations", "i", 1, "iterations of the encoder")

		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	completers.NewFlagCompsFor(msfInjectCmd, func(comp *carapace.ActionMap) {
		(*comp)["encoder"] = generate.MsfEncoderCompleter(con)
	})

	return []*cobra.Command{msfCmd}
}
