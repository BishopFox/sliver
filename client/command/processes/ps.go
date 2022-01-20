package processes

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
	"strings"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
	"github.com/jedib0t/go-pretty/v6/table"
	"google.golang.org/protobuf/proto"
)

var (
	// Stylizes known processes in the `ps` command
	knownProcs = map[string]string{
		"ccSvcHst.exe":          console.Red, // Symantec Endpoint Protection (SEP)
		"cb.exe":                console.Red, // Carbon Black
		"MsMpEng.exe":           console.Red, // Windows Defender
		"smartscreen.exe":       console.Red, // Windows Defender Smart Screen
		"CSFalconService.exe":   console.Red, // Crowdstrike Falcon Service
		"CSFalconContainer.exe": console.Red, // CrowdStrike Falcon Container Security
		"bdservicehost.exe": console.Red, // Bitdefender (Total Security)
		"bdagent.exe":       console.Red, // Bitdefender (Total Security)
		"bdredline.exe":     console.Red, // Bitdefender Redline Update Service (Source https://community.bitdefender.com/en/discussion/82135/bdredline-exe-bitdefender-total-security-2020)
	
	}
)

// PsCmd - List processes on the remote system
func PsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	ps, err := con.Rpc.Ps(context.Background(), &sliverpb.PsReq{
		Request: con.ActiveTarget.Request(ctx),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	os := getOS(session, beacon)
	if ps.Response != nil && ps.Response.Async {
		con.AddBeaconCallback(ps.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, ps)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintPS(os, ps, false, ctx, con)
		})
		con.PrintAsyncResponse(ps.Response)
	} else {
		PrintPS(os, ps, true, ctx, con)
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

// PrintPS - Prints the process list
func PrintPS(os string, ps *sliverpb.Ps, interactive bool, ctx *grumble.Context, con *console.SliverConsoleClient) {

	pidFilter := ctx.Flags.Int("pid")
	exeFilter := ctx.Flags.String("exe")
	ownerFilter := ctx.Flags.String("owner")
	overflow := ctx.Flags.Bool("overflow")
	skipPages := ctx.Flags.Int("skip-pages")

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))

	switch os {
	case "windows":
		tw.AppendHeader(table.Row{"pid", "ppid", "owner", "executable", "session"})
	case "darwin":
		fallthrough
	case "linux":
		tw.AppendHeader(table.Row{"pid", "ppid", "owner", "executable"})
	default:
		tw.AppendHeader(table.Row{"pid", "ppid", "owner", "executable"})
	}

	cmdLine := ctx.Flags.Bool("print-cmdline")
	for _, proc := range ps.Processes {
		if pidFilter != -1 && proc.Pid != int32(pidFilter) {
			continue
		}
		if exeFilter != "" && !strings.Contains(strings.ToLower(proc.Executable), strings.ToLower(exeFilter)) {
			continue
		}
		if ownerFilter != "" && !strings.Contains(strings.ToLower(proc.Owner), strings.ToLower(ownerFilter)) {
			continue
		}
		procRow(tw, proc, cmdLine, con)
	}
	tw.SortBy([]table.SortBy{
		{Name: "pid", Mode: table.Asc},
		{Name: "ppid", Mode: table.Asc},
	})

	settings.PaginateTable(tw, skipPages, overflow, interactive, con)
}

// procRow - Stylizes the process information
func procRow(tw table.Writer, proc *commonpb.Process, cmdLine bool, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()

	color := console.Normal
	if modifyColor, ok := knownSecurityTools[proc.Executable]; ok {
		color = modifyColor
	}
	if session != nil && proc.Pid == session.PID {
		color = console.Green
	}
	if beacon != nil && proc.Pid == beacon.PID {
		color = console.Green
	}

	var row table.Row
	switch session.GetOS() {
	case "windows":
		if cmdLine {
			var args string
			if len(proc.CmdLine) >= 1 {
				args = strings.Join(proc.CmdLine, " ")
			} else {
				args = proc.Executable
			}
			row = table.Row{
				fmt.Sprintf(color+"%d"+console.Normal, proc.Pid),
				fmt.Sprintf(color+"%d"+console.Normal, proc.Ppid),
				fmt.Sprintf(color+"%s"+console.Normal, proc.Owner),
				fmt.Sprintf(color+"%s"+console.Normal, args),
				fmt.Sprintf(color+"%d"+console.Normal, proc.SessionID),
			}
		} else {
			row = table.Row{
				fmt.Sprintf(color+"%d"+console.Normal, proc.Pid),
				fmt.Sprintf(color+"%d"+console.Normal, proc.Ppid),
				fmt.Sprintf(color+"%s"+console.Normal, proc.Owner),
				fmt.Sprintf(color+"%s"+console.Normal, proc.Executable),
				fmt.Sprintf(color+"%d"+console.Normal, proc.SessionID),
			}
		}
	case "darwin":
		fallthrough
	case "linux":
		fallthrough
	default:
		if cmdLine {
			var args string
			if len(proc.CmdLine) >= 2 {
				args = strings.Join(proc.CmdLine, " ")
			} else {
				args = proc.Executable
			}
			row = table.Row{
				fmt.Sprintf(color+"%d"+console.Normal, proc.Pid),
				fmt.Sprintf(color+"%d"+console.Normal, proc.Ppid),
				fmt.Sprintf(color+"%s"+console.Normal, proc.Owner),
				fmt.Sprintf(color+"%s"+console.Normal, args),
			}
		} else {
			row = table.Row{
				fmt.Sprintf(color+"%d"+console.Normal, proc.Pid),
				fmt.Sprintf(color+"%d"+console.Normal, proc.Ppid),
				fmt.Sprintf(color+"%s"+console.Normal, proc.Owner),
				fmt.Sprintf(color+"%s"+console.Normal, proc.Executable),
			}
		}
	}
	tw.AppendRow(row)
}

// GetPIDByName - Get a PID by name from the active session
func GetPIDByName(ctx *grumble.Context, name string, con *console.SliverConsoleClient) int {
	ps, err := con.Rpc.Ps(context.Background(), &sliverpb.PsReq{
		Request: con.ActiveTarget.Request(ctx),
	})
	if err != nil {
		return -1
	}
	for _, proc := range ps.Processes {
		if proc.Executable == name {
			return int(proc.Pid)
		}
	}
	return -1
}
