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

// `slivers` command impl
func listSliverBuilds(ctx *grumble.Context, rpc RPCServer) {

	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgListSliverBuilds,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"%s\n", resp.Err)
		return
	}

	builds := &clientpb.SliverBuilds{}
	proto.Unmarshal(resp.Data, builds)
	if 0 < len(builds.Configs) {
		displayAllSliverBuilds(builds.Configs)
	} else {
		fmt.Printf(Info + "No sliver builds\n")
	}
}

func displayAllSliverBuilds(configs map[string]*clientpb.SliverConfig) {

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	fmt.Fprintf(table, "Name\tOS/Arch\tDebug\tFormat\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("Name")),
		strings.Repeat("=", len("OS/Arch")),
		strings.Repeat("=", len("Debug")),
		strings.Repeat("=", len("Format")),
	)

	for sliverName, config := range configs {
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t\n",
			sliverName,
			fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH),
			fmt.Sprintf("%v", config.Debug),
			config.Format,
		)
	}
	table.Flush()
	fmt.Printf(outputBuf.String())
}
