package screenshot

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

	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"github.com/desertbit/grumble"
)

// ScreenshotCmd - Take a screenshot of the remote system
func ScreenshotCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}

	if session.OS != "windows" && session.OS != "linux" {
		con.PrintErrorf("Not implemented for %s\n", session.OS)
		return
	}

	screenshot, err := con.Rpc.Screenshot(context.Background(), &sliverpb.ScreenshotReq{
		Request: con.ActiveTarget.Request(ctx),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	timestamp := time.Now().Format("20060102150405")
	tmpFileName := path.Base(fmt.Sprintf("screenshot_%s_%d_%s_*.png", session.Name, session.ID, timestamp))
	tmpFile, err := ioutil.TempFile("", tmpFileName)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	err = ioutil.WriteFile(tmpFile.Name(), screenshot.Data, 0600)
	if err != nil {
		con.PrintErrorf("Error writting file: %s\n", err)
		return
	}
	con.Printf(console.Bold+"Screenshot written to %s\n", tmpFile.Name())

	if ctx.Flags.Bool("loot") && 0 < len(screenshot.Data) {
		err = loot.AddLootFile(con.Rpc, fmt.Sprintf("[screenshot] %s", timestamp), tmpFileName, screenshot.Data, false)
		if err != nil {
			con.PrintErrorf("Failed to save output as loot: %s\n", err)
		} else {
			con.PrintInfof("Output saved as loot\n")
		}
	}
}
