/*
	Sliver Implant Framework
	Copyright (C) 2024  Bishop Fox

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

package filesystem

import (
	"context"
	"fmt"
	"math"

	"google.golang.org/protobuf/proto"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// Drive mappings for Windows
var driveTypeMap = map[string]string{
	"0": "Unknown",
	"1": "Root Path invalid (no volume mounted for path)",
	"2": "Removable",
	"3": "Fixed disk",
	"4": "Remote / network drive",
	"5": "CD-ROM",
	"6": "RAM disk",
}

// MountCmd - Print information about mounted filesystems
func MountCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	mount, err := con.Rpc.Mount(context.Background(), &sliverpb.MountReq{
		Request: con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	os := getOS(session, beacon)
	if mount.Response != nil && mount.Response.Async {
		con.AddBeaconCallback(mount.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, mount)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintMount(os, mount, con)
		})
		con.PrintAsyncResponse(mount.Response)
	} else {
		PrintMount(os, mount, con)
	}
}

func getOS(session *clientpb.Session, beacon *clientpb.Beacon) string {
	if session != nil {
		return session.OS
	} else if beacon != nil {
		return beacon.OS
	}
	return ""
}

func reduceSpaceMetric(numberOfBytes float64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}

	if numberOfBytes < 1 {
		return fmt.Sprintf("0 %s", units[0])
	}

	base := 1024.0

	exp := math.Floor(math.Log(numberOfBytes) / math.Log(base))
	index := int(math.Min(exp, float64(len(units)-1)))
	divisor := math.Pow(base, float64(index))

	value := numberOfBytes / divisor

	return fmt.Sprintf("%.2f %s", value, units[index])
}

// PrintMount - Print a table containing information on mounted filesystems
func PrintMount(os string, mount *sliverpb.Mount, con *console.SliverClient) {
	if mount.Response != nil && mount.Response.Err != "" {
		con.PrintErrorf("%s\n", mount.Response.Err)
		return
	}
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))

	switch os {
	case "windows":
		tw.AppendHeader(table.Row{"Volume", "Volume Type", "Mount Point", "Label", "Filesystem", "Used Space", "Free Space", "Total Space"})
	case "darwin":
		fallthrough
	case "linux":
		fallthrough
	default:
		tw.AppendHeader(table.Row{"Source", "Mount Point", "Mount Root", "Filesystem Type", "Options", "Total Space"})
	}

	for _, mountPoint := range mount.Info {
		row := mountRow(os, mountPoint)
		tw.AppendRow(row)
	}

	settings.PaginateTable(tw, 0, false, false, con)
}

func mountRow(os string, mountInfo *sliverpb.MountInfo) table.Row {
	var row table.Row

	switch os {
	case "windows":
		// Translate VolumeType
		volType, ok := driveTypeMap[mountInfo.VolumeType]
		if !ok {
			volType = driveTypeMap["0"]
		}
		row = table.Row{mountInfo.VolumeName,
			volType,
			mountInfo.MountPoint,
			mountInfo.Label,
			mountInfo.FileSystem,
			reduceSpaceMetric(float64(mountInfo.UsedSpace)),
			reduceSpaceMetric(float64(mountInfo.FreeSpace)),
			reduceSpaceMetric(float64(mountInfo.TotalSpace)),
		}
	case "darwin":
		fallthrough
	case "linux":
		fallthrough
	default:
		row = table.Row{mountInfo.VolumeName,
			mountInfo.MountPoint,
			mountInfo.Label,
			mountInfo.VolumeType,
			mountInfo.MountOptions,
			reduceSpaceMetric(float64(mountInfo.TotalSpace)),
		}
	}
	return row
}
