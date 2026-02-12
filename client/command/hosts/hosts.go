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
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var (
	// ErrNoHosts - No hosts in database.
	// ErrNoHosts - No 在 database. 托管
	ErrNoHosts = errors.New("no hosts")
	// ErrNoIOCs - No IOCs in database
	// 数据库中的 ErrNoIOCs - No IOCs
	ErrNoIOCs = errors.New("no IOCs in database for selected host")
	// ErrNoSelection - No selection made
	// ErrNoSelection - No 选择
	ErrNoSelection = errors.New("no selection")
)

// HostsCmd - Main hosts command.
// HostsCmd - Main 主办 command.
func HostsCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	allHosts, err := con.Rpc.Hosts(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}
	if 0 < len(allHosts.Hosts) {
		con.Printf("%s\n", hostsTable(allHosts.Hosts, con))
	} else {
		con.PrintInfof("No hosts\n")
	}
}

func hostsTable(hosts []*clientpb.Host, con *console.SliverClient) string {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"ID",
		"Hostname",
		"Operating System",
		"Locale",
		"Sessions",
		"Beacons",
		"IOCs",
		"Extensions",
		"First Contact",
	})
	for _, host := range hosts {
		var shortID string
		if len(host.HostUUID) < 8 {
			shortID = host.HostUUID[:len(host.HostUUID)]
		} else {
			shortID = host.HostUUID[:8]
		}
		tw.AppendRow(table.Row{
			shortID,
			host.Hostname,
			host.OSVersion,
			host.Locale,
			hostSessions(host.HostUUID, con),
			hostBeacons(host.HostUUID, con),
			len(host.IOCs),
			len(host.ExtensionData),
			con.FormatDateDelta(time.Unix(host.FirstContact, 0), true, false),
		})
	}
	return tw.Render()
}

func hostSessions(hostUUID string, con *console.SliverClient) string {
	hostSessions := SessionsForHost(hostUUID, con)
	if len(hostSessions) == 0 {
		return "None"
	}
	sessionIDs := []string{}
	for _, hostSession := range hostSessions {
		sessionIDs = append(sessionIDs, strings.Split(hostSession.ID, "-")[0])
	}
	return fmt.Sprintf("%d", len(sessionIDs))
}

func hostBeacons(hostUUID string, con *console.SliverClient) string {
	beacons, err := con.Rpc.GetBeacons(context.Background(), &commonpb.Empty{})
	if err != nil {
		return "Error"
	}
	count := 0
	for _, beacon := range beacons.Beacons {
		if beacon.UUID == hostUUID {
			count++
		}
	}
	if count == 0 {
		return "None"
	} else {
		return fmt.Sprintf("%d", count)
	}
}

// SessionsForHost - Find session for a given host by id.
// id. 给定主机的 SessionsForHost - Find session
func SessionsForHost(hostUUID string, con *console.SliverClient) []*clientpb.Session {
	sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		return []*clientpb.Session{}
	}
	hostSessions := []*clientpb.Session{}
	for _, session := range sessions.Sessions {
		if session.UUID == hostUUID {
			hostSessions = append(hostSessions, session)
		}
	}
	return hostSessions
}

// SelectHost - Interactively select a host from the database.
// SelectHost - Interactively 从 database. 中选择一个主机
func SelectHost(con *console.SliverClient) (*clientpb.Host, error) {
	allHosts, err := con.Rpc.Hosts(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, err
	}
	// Sort the keys because maps have a randomized order, these keys must be ordered for the selection
	// Sort 键，因为地图具有随机顺序，这些键必须按顺序进行选择
	// to work properly since we rely on the index of the user's selection to find the session in the map
	// 正常工作，因为我们依赖用户选择的索引来查找地图中的 session
	var keys []string
	hostMap := make(map[string]*clientpb.Host)
	for _, host := range allHosts.Hosts {
		keys = append(keys, host.HostUUID)
		hostMap[host.HostUUID] = host
	}
	sort.Strings(keys)

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	for _, key := range keys {
		host := hostMap[key]
		fmt.Fprintf(table, "%s\t%s\t\n",
			host.Hostname,
			host.OSVersion,
		)
	}
	table.Flush()

	options := strings.Split(outputBuf.String(), "\n")
	options = options[:len(options)-1] // Remove the last empty option
	options = options[:len(options)-1] // Remove 最后一个空选项
	selected := ""
	_ = forms.Select("Select a host:", options, &selected)
	if selected == "" {
		return nil, ErrNoSelection
	}

	// Go from the selected option -> index -> host
	// Go 来自所选选项 -> 索引 -> 主机
	for index, option := range options {
		if option == selected {
			return hostMap[keys[index]], nil
		}
	}
	return nil, ErrNoSelection
}
