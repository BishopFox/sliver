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
	"context"
	"text/tabwriter"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"

	"github.com/desertbit/grumble"
)

func operatorsCmd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	operators, err := rpc.GetOperators(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	} else if 0 < len(operators.Operators) {
		displayOperators(operators.Operators)
	} else {
		fmt.Printf(Info + "No remote operators connected\n")
	}
}

func displayOperators(operators []*clientpb.Operator) {

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	fmt.Fprintln(table, "Operator\tStatus\t")
	fmt.Fprintf(table, "%s\t%s\t\n",
		strings.Repeat("=", len("Operator")),
		strings.Repeat("=", len("Status")),
	)

	colorRow := []string{"", ""} // Two uncolored rows for the headers
	for _, operator := range operators {
		fmt.Fprintf(table, "%s\t%s\t\n", operator.Name, status(operator.Online))
		if operator.Online {
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
