package beacons

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
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/bishopfox/sliver/client/command/kill"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// BeaconsCmd - Display/interact with beacons
// BeaconsCmd - Display/interact 带信标
func BeaconsCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	killFlag, _ := cmd.Flags().GetString("kill")
	killAll, _ := cmd.Flags().GetBool("kill-all")
	interact, _ := cmd.Flags().GetString("interact")

	// Handle kill
	// Handle 杀死
	if killFlag != "" {
		beacon, err := GetBeacon(con, killFlag)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		err = kill.KillBeacon(beacon, cmd, con)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		con.Println()
		con.PrintInfof("Killed %s (%s)\n", beacon.Name, beacon.ID)
	}

	if killAll {
		beacons, err := GetBeacons(con)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		for _, beacon := range beacons.Beacons {
			err = kill.KillBeacon(beacon, cmd, con)
			if err != nil {
				con.PrintErrorf("%s\n", err)
				return
			}
			con.Println()
			con.PrintInfof("Killed %s (%s)\n", beacon.Name, beacon.ID)
		}
	}

	if interact != "" {
		beacon, err := GetBeacon(con, interact)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		con.ActiveTarget.Set(nil, beacon)
		con.PrintInfof("Active beacon %s (%s)\n", beacon.Name, strings.Split(beacon.ID, "-")[0])
		return
	}

	filter, _ := cmd.Flags().GetString("filter")
	var filterRegex *regexp.Regexp
	if filterRe, _ := cmd.Flags().GetString("filter-re"); filterRe != "" {
		var err error
		filterRegex, err = regexp.Compile(filterRe)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}

	grpcCtx, cancel := con.GrpcContext(cmd)
	defer cancel()
	beacons, err := con.Rpc.GetBeacons(grpcCtx, &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	PrintBeacons(beacons.Beacons, filter, filterRegex, con)
}

// PrintBeacons - Display a list of beacons
// PrintBeacons - Display 信标列表
func PrintBeacons(beacons []*clientpb.Beacon, filter string, filterRegex *regexp.Regexp, con *console.SliverClient) {
	if len(beacons) == 0 {
		con.PrintInfof("No beacons 🙁\n")
		return
	}
	tw := renderBeacons(beacons, filter, filterRegex, con)
	con.Printf("%s\n", tw.Render())
}

func renderBeacons(beacons []*clientpb.Beacon, filter string, filterRegex *regexp.Regexp, con *console.SliverClient) table.Writer {
	width, _, err := term.GetSize(0)
	if err != nil {
		width = 999
	}

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	wideTermWidth := con.Settings.SmallTermWidth < width
	windowsBeaconInList := false
	for _, beacon := range beacons {
		if beacon.OS == "windows" {
			windowsBeaconInList = true
		}
	}
	if wideTermWidth {
		if windowsBeaconInList {
			tw.AppendHeader(table.Row{
				"ID",
				"Name",
				"Tasks",
				"Transport",
				"Remote Address",
				"Hostname",
				"Username",
				"Process (PID)",
				"Integrity",
				"Operating System",
				"Locale",
				"Last Check-in",
				"Next Check-in",
			})
		} else {
			tw.AppendHeader(table.Row{
				"ID",
				"Name",
				"Tasks",
				"Transport",
				"Remote Address",
				"Hostname",
				"Username",
				"Process (PID)",
				"Operating System",
				"Locale",
				"Last Check-in",
				"Next Check-in",
			})
		}
	} else {
		tw.AppendHeader(table.Row{
			"ID",
			"Name",
			"Transport",
			"Hostname",
			"Username",
			"Operating System",
			"Last Check-in",
			"Next Check-in",
		})
	}

	for _, beacon := range beacons {
		style := console.StyleNormal
		activeBeacon := con.ActiveTarget.GetBeacon()
		if activeBeacon != nil && activeBeacon.ID == beacon.ID {
			style = console.StyleGreen
		}
		if beacon.Integrity == "" {
			beacon.Integrity = "-"
		}

		// We need a slice of strings so we can apply filters
		// We 需要一段字符串，以便我们可以应用过滤器
		var rowEntries []string

		if wideTermWidth {
			rowEntries = []string{
				style.Render(strings.Split(beacon.ID, "-")[0]),
				style.Render(beacon.Name),
				style.Render(fmt.Sprintf("%d/%d", beacon.TasksCountCompleted, beacon.TasksCount)),
				style.Render(beacon.Transport),
				style.Render(beacon.RemoteAddress),
				style.Render(beacon.Hostname),
				style.Render(strings.TrimPrefix(beacon.Username, beacon.Hostname+"\\")),
				style.Render(fmt.Sprintf("%s (%d)", beacon.Filename, beacon.PID)),
			}

			if windowsBeaconInList {
				rowEntries = append(rowEntries, style.Render(beacon.Integrity))
			}

			rowEntries = append(rowEntries, []string{
				style.Render(fmt.Sprintf("%s/%s", beacon.OS, beacon.Arch)),
				style.Render(beacon.Locale),
				con.FormatDateDelta(time.Unix(beacon.LastCheckin, 0), wideTermWidth, false),
				con.FormatDateDelta(time.Unix(beacon.NextCheckin, 0), wideTermWidth, true),
			}...)
		} else {
			rowEntries = []string{
				style.Render(strings.Split(beacon.ID, "-")[0]),
				style.Render(beacon.Name),
				style.Render(beacon.Transport),
				style.Render(beacon.Hostname),
				style.Render(strings.TrimPrefix(beacon.Username, beacon.Hostname+"\\")),
				style.Render(fmt.Sprintf("%s/%s", beacon.OS, beacon.Arch)),
				con.FormatDateDelta(time.Unix(beacon.LastCheckin, 0), wideTermWidth, false),
				con.FormatDateDelta(time.Unix(beacon.NextCheckin, 0), wideTermWidth, true),
			}
		}
		// Build the row struct
		// Build 行结构
		row := table.Row{}
		for _, entry := range rowEntries {
			row = append(row, entry)
		}
		// Apply filters if any
		// Apply 过滤器（如果有）
		if filter == "" && filterRegex == nil {
			tw.AppendRow(row)
		} else {
			for _, rowEntry := range rowEntries {
				if filter != "" {
					if strings.Contains(rowEntry, filter) {
						tw.AppendRow(row)
						break
					}
				}
				if filterRegex != nil {
					if filterRegex.MatchString(rowEntry) {
						tw.AppendRow(row)
						break
					}
				}
			}
		}
	}
	return tw
}
