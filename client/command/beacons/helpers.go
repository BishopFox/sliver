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
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/AlecAivazis/survey/v2"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

var (
	// ErrNoBeacons - No sessions available
	ErrNoBeacons = errors.New("no beacons")
	// ErrNoSelection - No selection made
	ErrNoSelection = errors.New("no selection")
	// ErrBeaconNotFound
	ErrBeaconNotFound = errors.New("no beacon found for this ID")
)

// SelectBeacon - Interactive menu for the user to select an session, optionally only display live sessions
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
	prompt := &survey.Select{
		Message: "Select a beacon:",
		Options: options,
	}
	selected := ""
	survey.AskOne(prompt, &selected)
	if selected == "" {
		return nil, ErrNoSelection
	}

	// Go from the selected option -> index -> session
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
