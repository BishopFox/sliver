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
	"context"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/desertbit/grumble"
)

func listImplantBuilds(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	builds, err := rpc.ImplantBuilds(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	if 0 < len(builds.Configs) {
		displayAllImplantBuilds(builds.Configs)
	} else {
		fmt.Printf(Info + "No implant builds\n")
	}
}

func displayAllImplantBuilds(configs map[string]*clientpb.ImplantConfig) {

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	fmt.Fprintf(table, "Name\tOS/Arch\tDebug\tFormat\tCommand & Control\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("Name")),
		strings.Repeat("=", len("OS/Arch")),
		strings.Repeat("=", len("Debug")),
		strings.Repeat("=", len("Format")),
		strings.Repeat("=", len("Command & Control")),
	)

	for sliverName, config := range configs {
		if 0 < len(config.C2) {
			fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n",
				sliverName,
				fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH),
				fmt.Sprintf("%v", config.Debug),
				config.Format,
				fmt.Sprintf("[1] %s", config.C2[0].URL),
			)
		}
		if 1 < len(config.C2) {
			for index, c2 := range config.C2[1:] {
				fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n",
					"",
					"",
					"",
					"",
					fmt.Sprintf("[%d] %s", index+2, c2.URL),
				)
			}
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n", "", "", "", "", "")
	}
	table.Flush()
	fmt.Printf(outputBuf.String())
}
