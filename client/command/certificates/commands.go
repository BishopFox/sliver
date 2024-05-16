package certificates

/*
	Sliver Implant Framework
	Copyright (C) 2024  Bishop Fox

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
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/spf13/cobra"
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

	return []*cobra.Command{
		certificatesCmd,
	}
}
