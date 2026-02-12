package creds

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox
	Copyright (C) 2022 Bishop Fox

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
	"context"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/spf13/cobra"
)

// CredsCmd - Add new credentials.
// CredsCmd - Add 新 credentials.
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
