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
	"io/ioutil"
	"path"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

// ProcdumpCmd - Dump the memory of a remote process
func ProcdumpCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	pid := ctx.Flags.Int("pid")
	name := ctx.Flags.String("name")

	if pid == -1 && name != "" {
		pid = GetPIDByName(ctx, name, con)
	}
	if pid == -1 {
		con.PrintErrorf("Invalid process target\n")
		return
	}

	if ctx.Flags.Int("timeout") < 1 {
		con.PrintErrorf("Invalid timeout argument\n")
		return
	}

	ctrl := make(chan bool)
	con.SpinUntil("Dumping remote process memory ...", ctrl)
	dump, err := con.Rpc.ProcessDump(context.Background(), &sliverpb.ProcessDumpReq{
		Request: con.ActiveSession.Request(ctx),
		Pid:     int32(pid),
		Timeout: int32(ctx.Flags.Int("timeout") - 1),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	hostname := session.Hostname
	tmpFileName := path.Base(fmt.Sprintf("procdump_%s_%d_*", hostname, pid))
	tmpFile, err := ioutil.TempFile("", tmpFileName)
	if err != nil {
		con.PrintErrorf("Error creating temporary file: %v\n", err)
		return
	}
	tmpFile.Write(dump.GetData())
	con.PrintInfof("Process dump stored in: %s\n", tmpFile.Name())
}
