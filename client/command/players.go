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
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"

	clientpb "github.com/bishopfox/sliver/protobuf/client"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func playersCmd(ctx *grumble.Context, rpc RPCServer) {

	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgPlayers,
		Data: []byte{},
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", resp.Err)
		return
	}
	players := &clientpb.Players{}
	proto.Unmarshal(resp.Data, players)

	if 0 < len(players.Players) {
		displayPlayers(players.Players)
	} else {
		fmt.Printf(Info + "No remote players connected\n")
	}
}

func displayPlayers(players []*clientpb.Player) {

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	fmt.Fprintln(table, "Operator\tStatus\t")
	fmt.Fprintf(table, "%s\t%s\t\n",
		strings.Repeat("=", len("Operator")),
		strings.Repeat("=", len("Status")),
	)

	colorRow := []string{"", ""} // Two uncolored rows for the headers
	for _, player := range players {
		fmt.Fprintf(table, "%s\t%s\t\n", player.Client.Operator, status(player.Online))
		if player.Online {
			colorRow = append(colorRow, bold+green)
		} else {
			colorRow = append(colorRow, "")
		}

	}
	table.Flush()

	lines := strings.Split(outputBuf.String(), "\n")
	for lineNumber, line := range lines {
		if len(line) == 0 {
			continue
		}
		fmt.Printf("%s%s%s\n", colorRow[lineNumber], line, normal)
	}
}

func status(isOnline bool) string {
	if isOnline {
		return "Online"
	}
	return "Offline"
}
