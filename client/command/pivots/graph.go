package pivots

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"encoding/json"
	"fmt"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/spf13/cobra"
)

// PivotsGraphCmd - Display pivots for all sessions.
func PivotsGraphCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	graph, err := con.Rpc.PivotGraph(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", con.UnwrapServerErr(err))
		return
	}

	data, err := json.MarshalIndent(graph.Children, "", "  ")
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	fmt.Printf("%s\n", string(data))
}
