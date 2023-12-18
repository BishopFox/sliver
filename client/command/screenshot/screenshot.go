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
	"os"
	"path/filepath"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
)

// ScreenshotCmd - Take a screenshot of the remote system
func ScreenshotCmd(cmd *cobra.Command, con *console.SliverConsoleClient, args []string) {
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
		Request: con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	saveLoot, _ := cmd.Flags().GetBool("loot")
	lootName, _ := cmd.Flags().GetString("name")
	saveTo, _ := cmd.Flags().GetString("save")

	hostname := getHostname(session, beacon)
	if screenshot.Response != nil && screenshot.Response.Async {
		con.AddBeaconCallback(screenshot.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, screenshot)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			if saveLoot {
				if len(screenshot.Data) > 0 {
					LootScreenshot(screenshot, lootName, hostname, con)
				} else {
					con.PrintErrorf("Cannot loot screenshot because it contained no data")
				}
			}

			if !saveLoot || saveTo != "" {
				PrintScreenshot(screenshot, hostname, cmd, con)
			}
		})
		con.PrintAsyncResponse(screenshot.Response)
	} else {
		if saveLoot {
			if len(screenshot.Data) > 0 {
				LootScreenshot(screenshot, lootName, hostname, con)
			} else {
				con.PrintErrorf("Cannot loot screenshot because it contained no data")
			}
		}

		if !saveLoot || saveTo != "" {
			PrintScreenshot(screenshot, hostname, cmd, con)
		}
	}
}

// PrintScreenshot - Handle the screenshot command response
func PrintScreenshot(screenshot *sliverpb.Screenshot, hostname string, cmd *cobra.Command, con *console.SliverConsoleClient) {
	timestamp := time.Now().Format("20060102150405")

	saveTo, _ := cmd.Flags().GetString("save")
	var saveToFile *os.File
	var err error
	if saveTo == "" {
		tmpFileName := filepath.Base(fmt.Sprintf("screenshot_%s_%s_*.png", filepath.Base(hostname), timestamp))
		saveToFile, err = os.CreateTemp("", tmpFileName)
		if err != nil {
			con.PrintErrorf("%s\n", err)
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
	var n int
	n, err = saveToFile.Write(screenshot.Data)
	if err != nil {
		con.PrintErrorf("Error writing file: %s\n", err)
		return
	}

	con.PrintInfof("Screenshot written to %s (%s)\n", saveToFile.Name(), util.ByteCountBinary(int64(n)))
}

func LootScreenshot(screenshot *sliverpb.Screenshot, lootName string, hostName string, con *console.SliverConsoleClient) {
	timeNow := time.Now().UTC()
	screenshotFileName := fmt.Sprintf("screenshot_%s_%s.png", hostName, timeNow.Format("20060102150405"))

	if lootName == "" {
		lootName = screenshotFileName
	}

	lootMessage := loot.CreateLootMessage(con.ActiveTarget.GetHostUUID(), screenshotFileName, lootName, clientpb.FileType_BINARY, screenshot.GetData())
	loot.SendLootMessage(lootMessage, con)
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
