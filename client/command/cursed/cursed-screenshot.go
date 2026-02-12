package cursed

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox
	Copyright (C) 2022 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"fmt"
	"os"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/overlord"
	"github.com/spf13/cobra"
)

func CursedScreenshotCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	curse := selectCursedProcess(con)
	if curse == nil {
		return
	}
	con.Println()
	con.PrintInfof("Querying debug targets ... ")
	targets, err := overlord.QueryDebugTargets(curse.DebugURL().String())
	con.Printf(console.Clearln + "\r")
	if err != nil {
		con.PrintErrorf("Failed to query debug targets: %s\n", err)
		return
	}
	target := selectDebugTarget(targets, con)
	if target == nil {
		return
	}
	con.PrintInfof("Taking a screenshot of '%s' ... \n\n", target.Title)
	quality, _ := cmd.Flags().GetInt64("quality")
	if quality < 1 || quality > 100 {
		con.PrintErrorf("Invalid quality value, must be between 1 and 100\n")
		return
	}
	data, err := overlord.Screenshot(curse, target.WebSocketDebuggerURL, target.ID, quality)
	if err != nil {
		con.PrintErrorf("Failed to take screenshot: %s\n", err)
		return
	}
	saveFile, _ := cmd.Flags().GetString("save")
	if saveFile == "" {
		saveFile = fmt.Sprintf("screenshot-%s.png", time.Now().Format("20060102150405"))
	}
	err = os.WriteFile(saveFile, data, 0o644)
	if err != nil {
		con.PrintErrorf("Failed to save screenshot: %s\n", err)
		return
	}
	con.PrintInfof("Screenshot saved to %s\n", saveFile)
}
