package https

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

// Commands returns the `https` command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	httpCmd := &cobra.Command{
		Use:     consts.HttpsStr,
		Short:   "HTTPS handlers management",
		GroupID: consts.NetworkHelpGroup,
	}

	// Sliver listeners
	listenCmd := &cobra.Command{
		Use:   consts.ListenStr,
		Short: "Start an HTTPS listener",
		Long:  help.GetHelpFor([]string{consts.HttpsStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ListenCmd(cmd, con, args)
		},
	}
	httpCmd.AddCommand(listenCmd)

	flags.Bind("HTTPS listener", false, listenCmd, func(f *pflag.FlagSet) {
		f.StringP("domain", "d", "", "limit responses to specific domain")
		f.StringP("website", "w", "", "website name (see websites cmd)")
		f.StringP("lhost", "L", "", "interface to bind server to")
		f.Uint32P("lport", "l", generate.DefaultHTTPSLPort, "tcp listen port")
		f.BoolP("disable-otp", "D", false, "disable otp authentication")
		f.StringP("long-poll-timeout", "T", "1s", "server-side long poll timeout")
		f.StringP("long-poll-jitter", "J", "2s", "server-side long poll jitter")

		f.StringP("cert", "c", "", "PEM encoded certificate file")
		f.StringP("key", "k", "", "PEM encoded private key file")
		f.BoolP("lets-encrypt", "e", false, "attempt to provision a let's encrypt certificate")
		f.BoolP("disable-randomized-jarm", "E", false, "disable randomized jarm fingerprints")

		f.BoolP("persistent", "p", false, "make persistent across restarts")
	})
	completers.NewFlagCompsFor(listenCmd, func(comp *carapace.ActionMap) {
		(*comp)["cert"] = carapace.ActionFiles().Tag("certificate file")
		(*comp)["key"] = carapace.ActionFiles().Tag("key file")
	})

	// Staging listeners
	stageCmd := &cobra.Command{
		Use:   consts.ServeStr,
		Short: "Start a stager listener",
		Long:  help.GetHelpFor([]string{consts.StageListenerStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ServeStageCmd(cmd, con, args)
		},
	}
	httpCmd.AddCommand(stageCmd)

	flags.Bind("stage listener", false, stageCmd, func(f *pflag.FlagSet) {
		f.StringP("profile", "p", "", "implant profile name to link with the listener")
		f.StringP("url", "u", "", "URL to which the stager will call back to")
		f.StringP("cert", "c", "", "path to PEM encoded certificate file (HTTPS only)")
		f.StringP("key", "k", "", "path to PEM encoded private key file (HTTPS only)")
		f.BoolP("lets-encrypt", "e", false, "attempt to provision a let's encrypt certificate (HTTPS only)")
		f.String("aes-encrypt-key", "", "encrypt stage with AES encryption key")
		f.String("aes-encrypt-iv", "", "encrypt stage with AES encryption iv")
		f.String("rc4-encrypt-key", "", "encrypt stage with RC4 encryption key")
		f.StringP("compress", "C", "none", "compress the stage before encrypting (zlib, gzip, deflate9, none)")
		f.BoolP("prepend-size", "P", false, "prepend the size of the stage to the payload (to use with MSF stagers)")
	})
	completers.NewFlagCompsFor(stageCmd, func(comp *carapace.ActionMap) {
		(*comp)["profile"] = generate.ProfileNameCompleter(con)
		(*comp)["cert"] = carapace.ActionFiles().Tag("certificate file")
		(*comp)["key"] = carapace.ActionFiles().Tag("key file")
		(*comp)["compress"] = carapace.ActionValues([]string{"zlib", "gzip", "deflate9", "none"}...).Tag("compression formats")
	})

	return []*cobra.Command{httpCmd}
}
