package commands

import (
	"context"
	"fmt"
	"path/filepath"

	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

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

// ChangeDirectory - Change the working directory of the client console
type ChangeDirectory struct {
	Positional struct {
		Path string `description:"Local path" required:"1"`
	} `positional-args:"yes" required:"yes"`
}

// Execute - Handler for ChangeDirectory
func (cd *ChangeDirectory) Execute(args []string) (err error) {

	path := cd.Positional.Path
	if (path == "~" || path == "~/") && cctx.Context.Sliver.OS == "linux" {
		path = filepath.Join("/home", cctx.Context.Sliver.Username)
	}

	pwd, err := transport.RPC.Cd(context.Background(), &sliverpb.CdReq{
		Request: &commonpb.Request{
			SessionID: cctx.Context.Sliver.ID,
		},
		Path: path,
	})
	if err != nil {
		fmt.Printf(util.RPCError+"%s\n", err)
	} else {
		fmt.Printf(util.Info+"%s\n", pwd.Path)
		cctx.Context.Sliver.WorkingDir = pwd.Path
	}

	return
}
