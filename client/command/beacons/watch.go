package beacons

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
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// BeaconsWatchCmd - Watch your beacons in real-ish time
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
