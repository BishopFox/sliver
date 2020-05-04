package command

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
	"io/ioutil"
	"path"
	"time"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"github.com/desertbit/grumble"
)

func screenshot(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	if session.OS != "windows" && session.OS != "linux" {
		fmt.Printf(Warn+"Not implemented for %s\n", session.OS)
		return
	}

	screenshot, err := rpc.Screenshot(context.Background(), &sliverpb.ScreenshotReq{
		Request: ActiveSession.Request(ctx),
	})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	timestamp := time.Now().Format("20060102150405")
	tmpFileName := path.Base(fmt.Sprintf("screenshot_%s_%d_%s_*.png", session.Name, session.ID, timestamp))
	tmpFile, err := ioutil.TempFile("", tmpFileName)
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}
	err = ioutil.WriteFile(tmpFile.Name(), screenshot.Data, 0600)
	if err != nil {
		fmt.Printf(Warn+"Error writting file: %s\n", err)
		return
	}
	fmt.Printf(bold+"Screenshot written to %s\n", tmpFile.Name())
}
