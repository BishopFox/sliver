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
	"sort"
	"strings"
	"text/tabwriter"

	clientpb "github.com/bishopfox/sliver/protobuf/client"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func sessions(ctx *grumble.Context, rpc RPCServer) {
	interact := ctx.Flags.String("interact")
	if interact != "" {
		sliver := getSliver(interact, rpc)
		if sliver != nil {
			ActiveSliver.SetActiveSliver(sliver)
			fmt.Printf(Info+"Active sliver %s (%d)\n", sliver.Name, sliver.ID)
		} else {
			fmt.Printf(Warn+"Invalid sliver name or session number '%s'\n", ctx.Args[0])
		}
	} else {
		resp := <-rpc(&sliverpb.Envelope{
			Type: clientpb.MsgSessions,
			Data: []byte{},
		}, defaultTimeout)
		if resp.Err != "" {
			fmt.Printf(Warn+"Error: %s\n", resp.Err)
			return
		}
		sessions := &clientpb.Sessions{}
		proto.Unmarshal(resp.Data, sessions)

		slivers := map[uint32]*clientpb.Sliver{}
		for _, sliver := range sessions.Slivers {
			slivers[sliver.ID] = sliver
		}
		if 0 < len(slivers) {
			printSlivers(slivers)
		} else {
			fmt.Printf(Info + "No slivers connected\n")
		}
	}
}

/*
	So this method is a little more complex than you'd maybe think,
	this is because Go's tabwriter aligns columns by counting bytes
	and since we want to modify the color of the active sliver row
	the number of bytes per row won't line up. So we render the table
	into a buffer and note which row the active sliver is in. Then we
	write each line to the term and insert the ANSI codes just before
	we display the row.
*/
func printSlivers(sessions map[uint32]*clientpb.Sliver) {
	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	fmt.Fprintln(table, "ID\tName\tTransport\tRemote Address\tUsername\tOperating System\tLast Check-in\t")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("ID")),
		strings.Repeat("=", len("Name")),
		strings.Repeat("=", len("Transport")),
		strings.Repeat("=", len("Remote Address")),
		strings.Repeat("=", len("Username")),
		strings.Repeat("=", len("Operating System")),
		strings.Repeat("=", len("Last Check-in")))

	// Sort the keys becuase maps have a randomized order
	var keys []int
	for _, sliver := range sessions {
		keys = append(keys, int(sliver.ID))
	}
	sort.Ints(keys) // Fucking Go can't sort int32's, so we convert to/from int's

	activeIndex := -1
	for index, key := range keys {
		sliver := sessions[uint32(key)]
		if ActiveSliver.Sliver != nil && ActiveSliver.Sliver.ID == sliver.ID {
			activeIndex = index + 2 // Two lines for the headers
		}
		fmt.Fprintf(table, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t\n",
			sliver.ID, sliver.Name, sliver.Transport, sliver.RemoteAddress, sliver.Username,
			fmt.Sprintf("%s/%s", sliver.OS, sliver.Arch),
			sliver.LastCheckin)
	}
	table.Flush()

	if activeIndex != -1 {
		lines := strings.Split(outputBuf.String(), "\n")
		for lineNumber, line := range lines {
			if len(line) == 0 {
				continue
			}
			if lineNumber == activeIndex {
				fmt.Printf("%s%s%s\n", green, line, normal)
			} else {
				fmt.Printf("%s\n", line)
			}
		}
	} else {
		fmt.Printf(outputBuf.String())
	}
}

func use(ctx *grumble.Context, rpc RPCServer) {
	if len(ctx.Args) == 0 {
		fmt.Printf(Warn + "Missing sliver name or session number, see `help use`\n")
		return
	}
	sliver := getSliver(ctx.Args[0], rpc)
	if sliver != nil {
		ActiveSliver.SetActiveSliver(sliver)
		fmt.Printf(Info+"Active sliver %s (%d)\n", sliver.Name, sliver.ID)
	} else {
		fmt.Printf(Warn+"Invalid sliver name or session number '%s'\n", ctx.Args[0])
	}
}

func background(ctx *grumble.Context, rpc RPCServer) {
	ActiveSliver.SetActiveSliver(nil)
	fmt.Printf(Info + "Background ...\n")
}

func kill(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	force := ctx.Flags.Bool("force")

	sliver := ActiveSliver.Sliver
	data, _ := proto.Marshal(&sliverpb.KillReq{
		SliverID: sliver.ID,
		Force:    force,
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgKill,
		Data: data,
	}, 5)

	if !force && resp.Err != "" {
		fmt.Printf(Warn+"%s\n", resp.Err)
	} else {
		fmt.Printf(Info+"Killed %s (%d)\n", sliver.Name, sliver.ID)
		ActiveSliver.DisableActiveSliver()
	}
}
