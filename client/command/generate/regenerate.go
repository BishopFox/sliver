package generate

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
	"os"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

// RegenerateCmd - Download an archived implant build/binary
func RegenerateCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	save := ctx.Flags.String("save")
	if save == "" {
		save, _ = os.Getwd()
	}

	regenerate, err := con.Rpc.Regenerate(context.Background(), &clientpb.RegenerateReq{
		ImplantName: ctx.Args.String("implant-name"),
	})
	if err != nil {
		con.PrintErrorf("Failed to regenerate implant %s\n", err)
		return
	}
	if regenerate.File == nil {
		con.PrintErrorf("Failed to regenerate implant (no data)\n")
		return
	}
	saveTo, err := saveLocation(save, regenerate.File.Name)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	err = os.WriteFile(saveTo, regenerate.File.Data, 0700)
	if err != nil {
		con.PrintErrorf("Failed to write to %s\n", err)
		return
	}
	con.PrintInfof("Implant binary saved to: %s\n", saveTo)
}
