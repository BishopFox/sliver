package operators

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

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	"github.com/desertbit/grumble"
)

// OperatorsCmd - Display operators and current online status
func OperatorsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	operators, err := con.Rpc.GetOperators(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else if 0 < len(operators.Operators) {
		displayOperators(operators.Operators, con)
	} else {
		con.PrintInfof("No remote operators connected\n")
	}
}

func displayOperators(operators []*clientpb.Operator, con *console.SliverConsoleClient) {

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
		// This is the CA, but I guess you could also name an operator
		// "multiplayer" and it'll never show up in the list
		if operator.Name == "multiplayer" {
			continue
		}
		fmt.Fprintf(table, "%s\t%s\t\n", operator.Name, status(operator.Online))
		if operator.Online {
			colorRow = append(colorRow, console.Bold+console.Green)
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
		con.Printf("%s%s%s\n", colorRow[lineNumber], line, console.Normal)
	}
}

func status(isOnline bool) string {
	if isOnline {
		return "Online"
	}
	return "Offline"
}
