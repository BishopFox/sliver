package creds

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"context"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/spf13/cobra"
)

// CredsCmd - Add new credentials.
func CredsRmCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var id string
	if len(args) > 0 {
		id = args[0]
	}
	if id == "" {
		credential, err := SelectCredential(false, clientpb.HashType_INVALID, con)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		id = credential.ID
	}
	_, err := con.Rpc.CredsRm(context.Background(), &clientpb.Credentials{
		Credentials: []*clientpb.Credential{
			{
				ID: id,
			},
		},
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	creds, err := con.Rpc.Creds(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if len(creds.Credentials) == 0 {
		con.PrintInfof("No credentials 🙁\n")
		return
	}
	PrintCreds(creds.Credentials, con)
}
