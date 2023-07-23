package pivots

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"errors"
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/rsteube/carapace"
)

// SelectPivotListener - Interactive menu to select a pivot listener.
func SelectPivotListener(listeners []*sliverpb.PivotListener, con *console.SliverClient) (*sliverpb.PivotListener, error) {
	// Render selection table
	buf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(buf, 0, 2, 2, ' ', 0)
	for _, listener := range listeners {
		fmt.Fprintf(table, "%d\t%s\t%s\t\n", listener.ID, PivotTypeToString(listener.Type), listener.BindAddress)
	}
	table.Flush()
	options := strings.Split(buf.String(), "\n")
	options = options[:len(options)-1]
	if len(options) == 0 {
		return nil, errors.New("no task to select from")
	}

	selected := ""
	prompt := &survey.Select{
		Message: "Select a beacon task:",
		Options: options,
	}
	err := survey.AskOne(prompt, &selected)
	if err != nil {
		return nil, err
	}
	for index, value := range options {
		if value == selected {
			return listeners[index], nil
		}
	}
	return nil, errors.New("task not found")
}

// PivotIDCompleter completes pivot listeners' IDs.
func PivotIDCompleter(con *console.SliverClient) carapace.Action {
	callback := func(_ carapace.Context) carapace.Action {
		results := make([]string, 0)

		pivotListeners, err := con.Rpc.PivotSessionListeners(context.Background(), &sliverpb.PivotListenersReq{
			Request: con.ActiveTarget.Request(con.App.ActiveMenu().Root()),
		})
		if err != nil {
			return carapace.ActionMessage("failed to get remote pivots: %s", err.Error())
		}

		for _, listener := range pivotListeners.Listeners {
			results = append(results, strconv.Itoa(int(listener.ID)))
			results = append(results, fmt.Sprintf("[%s] %s (%d pivots)", listener.Type, listener.BindAddress, len(listener.Pivots)))
		}

		if len(results) == 0 {
			return carapace.ActionMessage("no pivot listeners")
		}

		return carapace.ActionValuesDescribed(results...).Tag("pivot listeners")
	}

	return carapace.ActionCallback(callback)
}
