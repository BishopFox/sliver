package pivots

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox
	Copyright (C) 2021 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/rsteube/carapace"
)

// SelectPivotListener - Interactive menu to select a pivot listener.
// SelectPivotListener - Interactive 菜单用于选择枢轴 listener.
func SelectPivotListener(listeners []*sliverpb.PivotListener, con *console.SliverClient) (*sliverpb.PivotListener, error) {
	// Render selection table
	// Render选型表
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
	err := forms.Select("Select a beacon task:", options, &selected)
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
// PivotIDCompleter 完成枢轴侦听器的 IDs.
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
