package beacons

import (
	"bytes"
	"context"
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
)

// SelectBeacon - Interactive menu for the user to select an session, optionally only display live sessions
func SelectBeacon(con *console.SliverConsoleClient) (*clientpb.Beacon, error) {
	beacons, err := con.Rpc.GetBeacons(context.Background(), &commonpb.Empty{})
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
	for beaconID, _ := range beaconsMap {
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
		Message: "Select a session:",
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
