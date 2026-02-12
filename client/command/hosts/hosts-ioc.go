package hosts

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
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// HostsIOCCmd - Remove a host from the database.
// HostsIOCCmd - Remove 来自 database. 的主机
func HostsIOCCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	host, err := SelectHost(con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if 0 < len(host.IOCs) {
		con.Printf("%s\n", hostIOCsTable(host, con))
	} else {
		con.Println()
		con.PrintInfof("No IOCs tracked on host\n")
	}
}

func hostIOCsTable(host *clientpb.Host, con *console.SliverClient) string {
	tw := table.NewWriter()
	tw.SetStyle(table.StyleBold)
	tw.AppendHeader(table.Row{"File Path", "SHA-256"})
	for _, ioc := range host.IOCs {
		tw.AppendRow(table.Row{
			ioc.Path,
			ioc.FileHash,
		})
	}
	return tw.Render()
}

func SelectHostIOC(host *clientpb.Host, con *console.SliverClient) (*clientpb.IOC, error) {
	// Sort the keys because maps have a randomized order, these keys must be ordered for the selection
	// Sort 键，因为地图具有随机顺序，这些键必须按顺序进行选择
	// to work properly since we rely on the index of the user's selection to find the session in the map
	// 正常工作，因为我们依赖用户选择的索引来查找地图中的 session
	var keys []string
	iocMap := make(map[string]*clientpb.IOC)
	for _, ioc := range host.IOCs {
		keys = append(keys, ioc.ID)
		iocMap[ioc.ID] = ioc
	}
	sort.Strings(keys)

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	for _, key := range keys {
		ioc := iocMap[key]
		fmt.Fprintf(table, "%s\t%s\t\n",
			ioc.Path,
			ioc.FileHash,
		)
	}
	table.Flush()

	options := strings.Split(outputBuf.String(), "\n")
	options = options[:len(options)-1] // Remove the last empty option
	options = options[:len(options)-1] // Remove 最后一个空选项
	selected := ""
	_ = forms.Select("Select an IOC:", options, &selected)
	if selected == "" {
		return nil, ErrNoSelection
	}

	// Go from the selected option -> index -> host
	// Go 来自所选选项 -> 索引 -> 主机
	for index, option := range options {
		if option == selected {
			return iocMap[keys[index]], nil
		}
	}
	return nil, ErrNoSelection
}
