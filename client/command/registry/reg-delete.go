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
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// RegDeleteKeyCmd - Remove a Windows registry key
// RegDeleteKeyCmd - Remove 一个 Windows 注册表项
func RegDeleteKeyCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	targetOS := getOS(session, beacon)
	if targetOS != "windows" {
		con.PrintErrorf("Registry operations can only target Windows\n")
		return
	}

	hostname, _ := cmd.Flags().GetString("hostname")
	hive, _ := cmd.Flags().GetString("hive")
	if err := checkHive(hive); err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	regPath := args[0]
	if regPath == "" {
		con.PrintErrorf("You must provide a path\n")
		return
	}
	if strings.Contains(regPath, "/") {
		regPath = strings.ReplaceAll(regPath, "/", "\\")
	}
	pathBaseIdx := strings.LastIndex(regPath, `\`)
	if pathBaseIdx < 0 {
		con.PrintErrorf("invalid path: %s", regPath)
		return
	}
	if len(regPath) < pathBaseIdx+1 {
		con.PrintErrorf("invalid path: %s", regPath)
		return
	}
	finalPath := regPath[:pathBaseIdx]
	key := regPath[pathBaseIdx+1:]

	deleteKey, err := con.Rpc.RegistryDeleteKey(context.Background(), &sliverpb.RegistryDeleteKeyReq{
		Hive:     hive,
		Path:     finalPath,
		Key:      key,
		Hostname: hostname,
		Request:  con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if deleteKey.Response != nil && deleteKey.Response.Async {
		con.AddBeaconCallback(deleteKey.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, deleteKey)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintDeleteKey(deleteKey, finalPath, key, con)
		})
		con.PrintAsyncResponse(deleteKey.Response)
	} else {
		PrintDeleteKey(deleteKey, finalPath, key, con)
	}
}

// PrintDeleteKey - Print the results of the delete key command
// PrintDeleteKey - Print 删除键盘命令的结果
func PrintDeleteKey(deleteKey *sliverpb.RegistryDeleteKey, regPath string, key string, con *console.SliverClient) {
	if deleteKey.Response != nil && deleteKey.Response.Err != "" {
		con.PrintErrorf("%s", deleteKey.Response.Err)
		return
	}
	con.PrintInfof("Key removed at %s\\%s", regPath, key)
}
