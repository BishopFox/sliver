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
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/tabwriter"

	// "time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"github.com/desertbit/grumble"
)

var (
	// Stylizes known processes in the `ps` command
	knownProcs = map[string]string{
		"ccSvcHst.exe":    console.Red, // SEP
		"cb.exe":          console.Red, // Carbon Black
		"MsMpEng.exe":     console.Red, // Windows Defender
		"smartscreen.exe": console.Red, // Windows Defender Smart Screen
		"bdservicehost.exe": console.Red, // Bitdefender (Total Security)
		"bdagent.exe": console.Red, // Bitdefender (Total Security)
		"bdredline.exe": console.Red, // Bitdefender Redline Update Service (Source https://community.bitdefender.com/en/discussion/82135/bdredline-exe-bitdefender-total-security-2020)
	}
)

// PsCmd - List processes on the remote system
func PsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	pidFilter := ctx.Flags.Int("pid")
	exeFilter := ctx.Flags.String("exe")
	ownerFilter := ctx.Flags.String("owner")

	ps, err := con.Rpc.Ps(context.Background(), &sliverpb.PsReq{
		Request: con.ActiveSession.Request(ctx),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', tabwriter.DiscardEmptyColumns)

	switch session.GetOS() {
	case "windows":
		fmt.Fprintf(table, "pid\tppid\towner\texecutable\tsession\n")
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n",
			strings.Repeat("=", len("pid")),
			strings.Repeat("=", len("ppid")),
			strings.Repeat("=", len("owner")),
			strings.Repeat("=", len("executable")),
			strings.Repeat("=", len("session")),
		)
	case "darwin":
		fallthrough
	case "linux":
		fmt.Fprintf(table, "pid\tppid\towner\texecutable\t\n")
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t\n",
			strings.Repeat("=", len("pid")),
			strings.Repeat("=", len("ppid")),
			strings.Repeat("=", len("owner")),
			strings.Repeat("=", len("executable")),
		)
	default:
		fmt.Fprintf(table, "pid\tppid\towner\texecutable\t\n")
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t\n",
			strings.Repeat("=", len("pid")),
			strings.Repeat("=", len("ppid")),
			strings.Repeat("=", len("owner")),
			strings.Repeat("=", len("executable")),
		)
	}
	cmdLine := ctx.Flags.Bool("print-cmdline")
	lineColors := []string{}
	for _, proc := range ps.Processes {
		var lineColor = ""
		if pidFilter != -1 && proc.Pid == int32(pidFilter) {
			lineColor = printProcInfo(table, proc, cmdLine, con)
		}
		if exeFilter != "" && strings.HasPrefix(proc.Executable, exeFilter) {
			lineColor = printProcInfo(table, proc, cmdLine, con)
		}
		if ownerFilter != "" && strings.HasPrefix(proc.Owner, ownerFilter) {
			lineColor = printProcInfo(table, proc, cmdLine, con)
		}
		if pidFilter == -1 && exeFilter == "" && ownerFilter == "" {
			lineColor = printProcInfo(table, proc, cmdLine, con)
		}

		// Should be set to normal/green if we rendered the line
		if lineColor != "" {
			lineColors = append(lineColors, lineColor)
		}
	}
	table.Flush()

	for index, line := range strings.Split(outputBuf.String(), "\n") {
		if len(line) == 0 {
			continue
		}
		// We need to account for the two rows of column headers
		if 0 < len(line) && 2 <= index {
			lineColor := lineColors[index-2]
			con.Printf("%s%s%s\n", lineColor, line, console.Normal)
		} else {
			con.Printf("%s\n", line)
		}
	}

}

// printProcInfo - Stylizes the process information
func printProcInfo(table *tabwriter.Writer, proc *commonpb.Process, cmdLine bool, con *console.SliverConsoleClient) string {
	color := console.Normal
	if modifyColor, ok := knownProcs[proc.Executable]; ok {
		color = modifyColor
	}
	session := con.ActiveSession.GetInteractive()
	if session != nil && proc.Pid == session.PID {
		color = console.Green
	}
	switch session.GetOS() {
	case "windows":
		fmt.Fprintf(table, "%d\t%d\t%s\t%s\t%d\t\n", proc.Pid, proc.Ppid, proc.Owner, proc.Executable, proc.SessionID)
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
			fmt.Fprintf(table, "%d\t%d\t%s\t%s\t\n", proc.Pid, proc.Ppid, proc.Owner, args)
		} else {
			fmt.Fprintf(table, "%d\t%d\t%s\t%s\t\n", proc.Pid, proc.Ppid, proc.Owner, proc.Executable)
		}
	}
	return color
}

// GetPIDByName - Get a PID by name from the active session
func GetPIDByName(ctx *grumble.Context, name string, con *console.SliverConsoleClient) int {
	ps, err := con.Rpc.Ps(context.Background(), &sliverpb.PsReq{
		Request: con.ActiveSession.Request(ctx),
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
