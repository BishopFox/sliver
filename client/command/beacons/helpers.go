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
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

var (
	// ErrNoBeacons - No sessions available
	// ErrNoBeacons - No 可用课程
	ErrNoBeacons = errors.New("no beacons")
	// ErrNoSelection - No selection made
	// ErrNoSelection - No 选择
	ErrNoSelection = errors.New("no selection")
	// ErrBeaconNotFound
	ErrBeaconNotFound = errors.New("no beacon found for this ID")
)

// SelectBeacon - Interactive menu for the user to select an session, optionally only display live sessions
// SelectBeacon - Interactive 菜单供用户选择 session，可选择仅显示实时会话
func SelectBeacon(con *console.SliverClient) (*clientpb.Beacon, error) {
	grpcCtx, cancel := con.GrpcContext(nil)
	defer cancel()
	beacons, err := con.Rpc.GetBeacons(grpcCtx, &commonpb.Empty{})
	if err != nil {
		return nil, err
	}
	if len(beacons.Beacons) == 0 {
		return nil, ErrNoBeacons
	}

	beaconsMap := map[string]*clientpb.Beacon{}
	for _, beacon := range beacons.Beacons {
		beaconsMap[beacon.ID] = beacon
	}
	keys := []string{}
	for beaconID := range beaconsMap {
		keys = append(keys, beaconID)
	}
	sort.Strings(keys)

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	for _, key := range keys {
		beacon := beaconsMap[key]
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\n",
			beacon.ID,
			beacon.Name,
			beacon.RemoteAddress,
			beacon.Hostname,
			beacon.Username,
			fmt.Sprintf("%s/%s", beacon.OS, beacon.Arch),
		)
	}
	table.Flush()

	options := strings.Split(outputBuf.String(), "\n")
	options = options[:len(options)-1] // Remove the last empty option
	options = options[:len(options)-1] // Remove 最后一个空选项
	selected := ""
	_ = forms.Select("Select a beacon:", options, &selected)
	if selected == "" {
		return nil, ErrNoSelection
	}

	// Go from the selected option -> index -> session
	// 所选选项中的 Go -> 索引 -> session
	for index, option := range options {
		if option == selected {
			return beaconsMap[keys[index]], nil
		}
	}
	return nil, ErrNoSelection
}

func GetBeacon(con *console.SliverClient, beaconID string) (*clientpb.Beacon, error) {
	grpcCtx, cancel := con.GrpcContext(nil)
	defer cancel()
	beacons, err := con.Rpc.GetBeacons(grpcCtx, &commonpb.Empty{})
	if err != nil {
		return nil, err
	}
	if len(beacons.Beacons) == 0 {
		return nil, ErrNoBeacons
	}
	for _, beacon := range beacons.Beacons {
		if beacon.ID == beaconID || strings.HasPrefix(beacon.ID, beaconID) {
			return beacon, nil
		}
	}
	return nil, ErrBeaconNotFound
}

func GetBeacons(con *console.SliverClient) (*clientpb.Beacons, error) {
	grpcCtx, cancel := con.GrpcContext(nil)
	defer cancel()
	beacons, err := con.Rpc.GetBeacons(grpcCtx, &commonpb.Empty{})
	if err != nil {
		return nil, err
	}
	if len(beacons.Beacons) == 0 {
		return nil, ErrNoBeacons
	}
	return beacons, nil
}
