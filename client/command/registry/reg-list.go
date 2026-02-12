package registry

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

// RegListSubKeysCmd - List sub registry keys
// RegListSubKeysCmd - List 子注册表项
func RegListSubKeysCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	targetOS := getOS(session, beacon)
	if targetOS != "windows" {
		con.PrintErrorf("Registry operations can only target Windows\n")
		return
	}

	regPath := args[0]
	hive, _ := cmd.Flags().GetString("hive")
	hostname, _ := cmd.Flags().GetString("hostname")

	regList, err := con.Rpc.RegistryListSubKeys(context.Background(), &sliverpb.RegistrySubKeyListReq{
		Hive:     hive,
		Hostname: hostname,
		Path:     regPath,
		Request:  con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if regList.Response != nil && regList.Response.Async {
		con.AddBeaconCallback(regList.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, regList)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintListSubKeys(regList, hive, regPath, con)
		})
		con.PrintAsyncResponse(regList.Response)
	} else {
		PrintListSubKeys(regList, hive, regPath, con)
	}
}

// PrintListSubKeys - Print the list sub keys command result
// PrintListSubKeys - Print 列表子键命令结果
func PrintListSubKeys(regList *sliverpb.RegistrySubKeyList, hive string, regPath string, con *console.SliverClient) {
	if regList.Response != nil && regList.Response.Err != "" {
		con.PrintErrorf("%s\n", regList.Response.Err)
		return
	}
	if 0 < len(regList.Subkeys) {
		con.PrintInfof("Sub keys under %s:\\%s:\n", hive, regPath)
	}
	for _, subKey := range regList.Subkeys {
		con.Println(subKey)
	}
}

// RegListValuesCmd - List registry values
// RegListValuesCmd - List 注册表值
func RegListValuesCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	regPath := args[0]
	hive, _ := cmd.Flags().GetString("hive")
	hostname, _ := cmd.Flags().GetString("hostname")

	regList, err := con.Rpc.RegistryListValues(context.Background(), &sliverpb.RegistryListValuesReq{
		Hive:     hive,
		Hostname: hostname,
		Path:     regPath,
		Request:  con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if regList.Response != nil && regList.Response.Async {
		con.AddBeaconCallback(regList.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, regList)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintListValues(regList, hive, regPath, con)
		})
		con.PrintAsyncResponse(regList.Response)
	} else {
		PrintListValues(regList, hive, regPath, con)
	}
}

// PrintListValues - Print the registry list values
// PrintListValues - Print 注册表列表值
func PrintListValues(regList *sliverpb.RegistryValuesList, hive string, regPath string, con *console.SliverClient) {
	if regList.Response != nil && regList.Response.Err != "" {
		con.PrintErrorf("%s\n", regList.Response.Err)
		return
	}
	if 0 < len(regList.ValueNames) {
		con.PrintInfof("Values under %s:\\%s:\n", hive, regPath)
	}
	for _, val := range regList.ValueNames {
		con.Println(val)
	}
}
