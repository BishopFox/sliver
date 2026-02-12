package certificates

/*
	Sliver Implant Framework
	Copyright (C) 2024  Bishop Fox
	Copyright (C) 2024 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *console.SliverClient) []*cobra.Command {
	certificatesCmd := &cobra.Command{
		Use:   consts.CertificatesStr,
		Short: "Certificate management",
		Long:  help.GetHelpFor([]string{consts.CertificatesStr}),
		Run: func(cmd *cobra.Command, args []string) {
			CertificateInfoCmd(cmd, con, args)
		},
		GroupID: consts.GenericHelpGroup,
	}
	flags.Bind(consts.CertificatesStr, false, certificatesCmd, func(f *pflag.FlagSet) {
		f.BoolP("mtls", "m", false, "Show MTLS certificates")
		f.BoolP("https", "p", false, "Show HTTPS certificates")
		f.BoolP("implant", "i", false, "Show implant certificates")
		f.BoolP("server", "s", false, "Show server certificates")
		f.StringP("cn", "c", "", "Show certificate information for a provided common name")
	})
	flags.BindFlagCompletions(certificatesCmd, func(comp *carapace.ActionMap) {
		(*comp)["cn"] = CertificateCommonNameCompleter(con)
	})
	registerCertificateCNFlagCompletion(certificatesCmd, "cn", con)

	authoritiesCmd := &cobra.Command{
		Use:   "authorities",
		Short: "List certificate authorities",
		Run: func(cmd *cobra.Command, args []string) {
			CertificateAuthoritiesCmd(cmd, con, args)
		},
	}
	certificatesCmd.AddCommand(authoritiesCmd)

	return []*cobra.Command{
		certificatesCmd,
	}
}
