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
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// GetPrivsCmd - Get the current process privileges (Windows only)
// GetPrivsCmd - Get 当前进程权限（仅限 Windows）
func GetPrivsCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	targetOS := getOS(session, beacon)
	if targetOS != "windows" {
		con.PrintErrorf("Command only supported on Windows.\n")
		return
	}

	privs, err := con.Rpc.GetPrivs(context.Background(), &sliverpb.GetPrivsReq{
		Request: con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	pid := getPID(session, beacon)
	if privs.Response != nil && privs.Response.Async {
		con.AddBeaconCallback(privs.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, privs)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintGetPrivs(privs, pid, con)
			err = updateBeaconIntegrityInformation(con, beacon.ID, privs.ProcessIntegrity)
			if err != nil {
				con.PrintWarnf("Could not save integrity information for the beacon: %s\n", err)
				return
			}
		})
		con.PrintAsyncResponse(privs.Response)
	} else {
		PrintGetPrivs(privs, pid, con)
	}
}

// PrintGetPrivs - Print the results of the get privs command
// PrintGetPrivs - Print get privs 命令的结果
func PrintGetPrivs(privs *sliverpb.GetPrivs, pid int32, con *console.SliverClient) {
	// Response is the Envelope (see RPC API), Err is part of it.
	// Response 是 Envelope（请参阅 RPC API），Err 是 it. 的一部分
	if privs.Response != nil && privs.Response.Err != "" {
		con.PrintErrorf("\nNOTE: Information may be incomplete due to an error:\n")
		con.PrintErrorf("%s\n", privs.Response.Err)
	}
	if privs.PrivInfo == nil {
		return
	}

	var processName string = "Current Process"
	if privs.ProcessName != "" {
		processName = privs.ProcessName
	}

	// To make things look pretty, figure out the longest name and description
	// To 让事情看起来更漂亮，找出最长的名称和描述
	// for column width
	// 对于列宽
	var nameColumnWidth int = 0
	var descriptionColumnWidth int = 0
	var introWidth int = 34 + len(processName) + len(strconv.Itoa(int(pid)))

	for _, entry := range privs.PrivInfo {
		if len(entry.Name) > nameColumnWidth {
			nameColumnWidth = len(entry.Name)
		}
		if len(entry.Description) > descriptionColumnWidth {
			descriptionColumnWidth = len(entry.Description)
		}
	}

	// Give one more space
	// Give 多一个空间
	nameColumnWidth += 1
	descriptionColumnWidth += 1

	con.Printf("\nPrivilege Information for %s (PID: %d)\n", processName, pid)
	con.Println(strings.Repeat("-", introWidth))
	con.Printf("\nProcess Integrity Level: %s\n\n", privs.ProcessIntegrity)
	con.Printf("%-*s\t%-*s\t%s\n", nameColumnWidth, "Name", descriptionColumnWidth, "Description", "Attributes")
	con.Printf("%-*s\t%-*s\t%s\n", nameColumnWidth, "====", descriptionColumnWidth, "===========", "==========")
	for _, entry := range privs.PrivInfo {
		con.Printf("%-*s\t%-*s\t", nameColumnWidth, entry.Name, descriptionColumnWidth, entry.Description)
		if entry.Enabled {
			con.Printf("Enabled")
		} else {
			con.Printf("Disabled")
		}
		if entry.EnabledByDefault {
			con.Printf(", Enabled by Default")
		}
		if entry.Removed {
			con.Printf(", Removed")
		}
		if entry.UsedForAccess {
			con.Printf(", Used for Access")
		}
		con.Printf("\n")
	}
}

func getOS(session *clientpb.Session, beacon *clientpb.Beacon) string {
	if session != nil {
		return session.OS
	}
	if beacon != nil {
		return beacon.OS
	}
	panic("no session or beacon")
}

func getPID(session *clientpb.Session, beacon *clientpb.Beacon) int32 {
	if session != nil {
		return session.PID
	}
	if beacon != nil {
		return beacon.PID
	}
	panic("no session or beacon")
}

func updateBeaconIntegrityInformation(con *console.SliverClient, beaconID string, integrity string) error {
	_, err := con.Rpc.UpdateBeaconIntegrityInformation(context.Background(), &clientpb.BeaconIntegrity{BeaconID: beaconID,
		Integrity: integrity})

	return err
}
