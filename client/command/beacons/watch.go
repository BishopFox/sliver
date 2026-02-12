package beacons

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox
	Copyright (C) 2021 Bishop Fox

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
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// BeaconsWatchCmd - Watch your beacons in real-ish time
// BeaconsWatchCmd - Watch 您在 real__PH0__ 时间内的信标
func BeaconsWatchCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	done := waitForInput()
	defer func() {
		con.Printf(console.UpN+console.Clearln+"\r", 1)
		con.Printf(console.UpN+console.Clearln+"\r", 1)
	}()
	for {
		select {
		case <-done:
			return
		case <-time.After(time.Second):
			beacons, err := con.Rpc.GetBeacons(context.Background(), &commonpb.Empty{})
			if err != nil {
				panic(err) // If we return we may leak the waiting goroutine, so we panic instead
				panic(err) // If 我们返回，我们可能会泄漏等待的 goroutine，所以我们会恐慌
			}
			tw := renderBeacons(beacons.Beacons, "", nil, con)
			lines := strings.Split(tw.Render(), "\n")
			for _, line := range lines {
				con.Printf(console.Clearln+"\r%s\n", line)
			}
			con.Printf("\nPress enter to stop.\n")
			con.Printf(console.UpN+"\r", len(lines)+2)
		}
	}
}

func waitForInput() <-chan bool {
	done := make(chan bool, 1)
	go func() {
		defer close(done)
		fmt.Scanf("\n")
		done <- true
	}()
	return done
}
