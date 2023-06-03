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
	"context"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	"github.com/jedib0t/go-pretty/v6/table"
)

// OperatorsCmd - Display operators and current online status
func OperatorsCmd(cmd *cobra.Command, con *console.SliverConsoleClient, args []string) {
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
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"Name",
		"Status",
	})
	for _, operator := range operators {
		tw.AppendRow(table.Row{
			console.Bold + operator.Name + console.Normal,
			status(operator.Online),
		})
	}
	con.Printf("%s\n", tw.Render())
}

func status(isOnline bool) string {
	if isOnline {
		return console.Bold + console.Green + "Online" + console.Normal
	}
	return console.Bold + console.Red + "Offline" + console.Normal
}
