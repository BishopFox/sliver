package server

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
	"fmt"

	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// Operators - List operators and their current status.
type Operators struct{}

// Execute - List operators and their current status.
func (o *Operators) Execute(args []string) (err error) {

	operators, err := transport.RPC.GetOperators(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.RPCError+"%s\n", err)
	} else if 0 < len(operators.Operators) {
		displayOperators(operators.Operators)
	} else {
		fmt.Printf(util.Info + "No remote operators connected\n")
	}

	return
}

func displayOperators(operators []*clientpb.Operator) {

	table := util.NewTable("")

	// Column Headers
	headers := []string{"Operator", "Status"}
	headLen := []int{10, 10}
	table.SetColumns(headers, headLen)

	for _, operator := range operators {
		table.AppendRow([]string{operator.Name, status(operator.Online)})
	}
	table.Output()
}

func status(isOnline bool) string {
	if isOnline {
		return "Online"
	}
	return "Offline"
}
